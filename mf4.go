package mf4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/blocks/AT"
	"github.com/LincolnG4/GoMDF/internal/blocks/CG"
	"github.com/LincolnG4/GoMDF/internal/blocks/CN"
	"github.com/LincolnG4/GoMDF/internal/blocks/DG"
	"github.com/LincolnG4/GoMDF/internal/blocks/FH"
	"github.com/LincolnG4/GoMDF/internal/blocks/HD"
	"github.com/LincolnG4/GoMDF/internal/blocks/ID"
	"github.com/LincolnG4/GoMDF/internal/blocks/MD"
	"github.com/LincolnG4/GoMDF/internal/blocks/TX"
	"github.com/davecgh/go-spew/spew"
)

type MF4 struct {
	File           *os.File
	Header         *HD.Block
	Identification *ID.Block
	FileHistory    []*FH.Block
	Attachments    []*AT.Block
	ChannelGroup   []*ChannelGroup
}

type ChannelGroup struct {
	Block     *CG.Block
	Channels  map[string]*CN.Block
	Datagroup *DG.Block
}

func ReadFile(file *os.File, getXML bool) (*MF4, error) {
	var address int64 = 0

	mf4File := MF4{
		File:           file,
		Identification: ID.New(file, address),
	}

	fmt.Printf("%#v \n", mf4File.Identification)

	fileVersion := mf4File.MdfVersion()
	if fileVersion < 400 {
		return nil, fmt.Errorf("file version is not >= 4.00")
	}

	if fileVersion >= 400 {
		mf4File.getHeader()
		mf4File.loadFileHistory(getXML)
		mf4File.loadAttachmemt()
		mf4File.read(getXML)
	}

	return &mf4File, nil
}

func (m *MF4) getHeader() {
	m.Header = HD.New(m.File, blocks.IdblockSize)
}

func (m *MF4) read(getXML bool) {
	var file *os.File = m.File

	if !m.IsFinalized() {
		panic("NOT FINALIZED MF4, PACKAGE IS NOT PREPARED")
	}

	index := 0
	version := m.MdfVersion()
	NextAddressDG := m.firstDataGroup()

	//Get all DataGroup
	for NextAddressDG != 0 {
		dgBlock := DG.New(file, NextAddressDG)

		//Read MdBlocks inside
		mdCommentAddr := dgBlock.MetadataComment()
		if mdCommentAddr != 0 {
			comment := *MD.New(file, mdCommentAddr)
			fmt.Printf("%s\n", comment)
		}

		//From DGBLOCK read ChannelGroup
		indexCG := 0
		NextAddressCG := dgBlock.FirstChannelGroup()

		for NextAddressCG != 0 {
			cgBlock := CG.New(file, version, NextAddressCG)

			//Save Informations
			channelGroup := &ChannelGroup{
				Block:     cgBlock,
				Channels:  make(map[string]*CN.Block),
				Datagroup: dgBlock,
			}

			//From CGBLOCK read Channel
			indexCN := 0
			nextAddressCN := cgBlock.FirstChannel()
			for nextAddressCN != 0 {
				cnBlock := CN.New(file, version, nextAddressCN)
				channelName := *TX.GetText(file, int64(cnBlock.Link.TxName))

				//Remove 00 bytes from the name
				channelGroup.Channels[channelName] = cnBlock

				//Get XML comments
				MdCommentAdress := cnBlock.Link.MdComment
				if getXML && MdCommentAdress != 0 {
					comment := MD.New(file, MdCommentAdress)
					fmt.Println(comment)
				} else {
					mdBlock := (&MD.Block{}).BlankBlock()
					mdComment := ""
					fmt.Print(mdComment, mdBlock, "\n")
				}
				nextAddressCN = cnBlock.Link.Next
				indexCN++
			}

			m.ChannelGroup = append(m.ChannelGroup, channelGroup)
			NextAddressCG = cgBlock.Link.Next
			indexCG++
		}

		fmt.Println("\n##############################")

		NextAddressDG = dgBlock.Next()
		index++
	}

}

/*
ChannelNames returns a map of channels of each datagroup

	map[key]value
	key = Datagroup number
	value = array with channel names
*/
func (m *MF4) ChannelNames() map[int][]string {
	channelMap := make(map[int][]string, 0)

	for i, cg := range m.ChannelGroup {
		channelNames := make([]string, 0)
		for name := range cg.Channels {
			channelNames = append(channelNames, name)
		}
		channelMap[i] = channelNames
	}

	return channelMap
}

// GetChannelSample loads sample by DataGroupName and ChannelName
func (m *MF4) GetChannelSample(dgName int, channelName string) ([]interface{}, error) {
	var byteOrder binary.ByteOrder
	file := m.File

	//for each Channel Group, read channel
	for i, cgrp := range m.ChannelGroup {
		if i != dgName {
			continue
		}
		cn, ok := cgrp.Channels[channelName]
		if !ok {
			continue
		}
		dg := cgrp.Datagroup
		cg := cgrp.Block

		//Get channel with compositon Structure or Array
		comp := cn.Link.Composition
		if comp != 0 {
			id := make([]byte, 4)
			seekRead(file, comp, id)

			if string(id) == blocks.CaID {
				readArrayBlock(file, comp)
			}
		}

		dataType := cn.Data.DataType

		readAddr := blocks.HeaderSize + dg.Link.Data + int64(dg.Data.RecIDSize) + int64(cn.Data.ByteOffset)
		size := (cn.Data.BitCount + uint32(cn.Data.BitOffset)) / 8
		data := make([]byte, size)
		sample := make([]interface{}, 0)

		rowSize := int64(cg.Data.DataBytes)

		if dataType == 0 || dataType == 2 || dataType == 4 || dataType == 8 || dataType == 15 {
			byteOrder = binary.LittleEndian
		} else {
			byteOrder = binary.BigEndian
		}

		dtype := loadDataType(dataType, len(data))

		// Create a new instance of the data type using reflection
		sliceElemType := reflect.TypeOf(dtype)
		sliceElem := reflect.New(sliceElemType).Interface()

		for i := uint64(0); i < cg.Data.CycleCount; i += 1 {
			seekRead(file, readAddr, data)

			buf := bytes.NewBuffer(data)
			err := binary.Read(buf, byteOrder, sliceElem)
			if err != nil {
				fmt.Println("Error reading:", err)
				return nil, errors.New("parsing channel error")
			}
			sample = append(sample, reflect.ValueOf(sliceElem).Elem().Interface())
			readAddr += rowSize
		}

		return sample, nil
	}
	return nil, errors.New("channel doen't exist")
}

func readArrayBlock(file *os.File, addr int64) {
	//debug(file,addr,400)
}

func seekRead(file *os.File, readAddr int64, data []byte) {
	_, errs := file.Seek(readAddr, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs, "Memory Addr out of size")
		}
	}
	_, err := file.Read(data)
	if err != nil {
		if err != io.EOF {
			fmt.Println("LoadBuffer error: ", err)
		}
	}

}

func loadDataType(dataType uint8, lenSize int) interface{} {
	var dtype interface{}
	switch dataType {
	case 0, 1:
		switch lenSize {
		case 1:
			dtype = uint8(0)
		case 2:
			dtype = uint16(0)
		case 4:
			dtype = uint32(0)
		case 8:
			dtype = uint64(0)
		}
	case 2, 3:
		switch lenSize {
		case 1:
			dtype = int8(0)
		case 2:
			dtype = int16(0)
		case 4:
			dtype = int32(0)
		case 8:
			dtype = int64(0)
		}

	case 4, 5:
		switch lenSize {
		case 4:
			dtype = float32(0)
		case 8:
			dtype = float64(0)

		}

	}

	return dtype
}

// loadAttachmemt iterates over all AT blocks and append array to MF4 object
func (m *MF4) loadAttachmemt() {
	var index int = 0
	array := make([]*AT.Block, 0)
	nextAddressAT := m.firstAttachment()
	file := m.File
	for nextAddressAT != 0 {
		atBlock := AT.New(file, nextAddressAT)

		fileName := TX.GetText(file, atBlock.Link.TxFilename)
		fmt.Printf("Filename attached: %s\n", *fileName)

		mimeType := TX.GetText(file, atBlock.Link.TxMimetype)
		fmt.Printf("Mime attached: %s\n", *mimeType)

		//Read MDComment
		MdCommentAdress := atBlock.Link.MDComment
		if MdCommentAdress != 0 {
			comment := MD.New(file, MdCommentAdress)
			fmt.Printf("%s\n", *comment)
		}

		array = append(array, atBlock)
		nextAddressAT = atBlock.Link.Next
		index++
	}
	m.Attachments = array
}

// LoadFileHistory iterates over all FH blocks and append array to MF4 object
func (m *MF4) loadFileHistory(getXML bool) {
	var index int = 0

	array := make([]*FH.Block, 0)
	nextAddressFH := m.firstFileHistory()

	//iterate over all FH blocks
	for nextAddressFH != 0 {
		fhBlock := FH.New(m.File, nextAddressFH)
		MdCommentAdress := fhBlock.Link.MDComment

		//Read MDComment
		if MdCommentAdress != 0 {
			comment := *MD.New(m.File, MdCommentAdress)
			fmt.Printf("%s\n", comment)
		}

		array = append(array, fhBlock)

		nextAddressFH = fhBlock.Link.Next
		index++

	}
	m.FileHistory = array

}

func debug(file *os.File, offset int64, size int) {
	_, err := file.Seek(int64(offset), io.SeekStart)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, size)
	n, err := file.Read(buf[:cap(buf)])
	buf = buf[:n]
	if err != nil {
		if err != io.EOF {
			panic(err)
		}
	}
	spew.Dump(buf)
}
