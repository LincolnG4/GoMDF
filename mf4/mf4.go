package mf4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"time"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/blocks/AT"
	"github.com/LincolnG4/GoMDF/internal/blocks/CG"
	"github.com/LincolnG4/GoMDF/internal/blocks/CN"
	"github.com/LincolnG4/GoMDF/internal/blocks/DG"
	"github.com/LincolnG4/GoMDF/internal/blocks/EV"
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
	//Address to First File History Block
	FileHistory  int64
	ChannelGroup []*ChannelGroup
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
	fileVersion := mf4File.MdfVersion()
	if fileVersion < 400 {
		return nil, fmt.Errorf("file version is not >= 4.00")
	}

	if fileVersion >= 400 {
		mf4File.loadHeader()
		mf4File.loadFirstFileHistory()
		mf4File.loadEvents()
		mf4File.read(getXML)
	}
	return &mf4File, nil
}

func (m *MF4) loadEvents() {
	if m.getFirstEvent() != 0 {
		nextEvent := m.getFirstEvent()
		for nextEvent != 0 {
			event, err := EV.New(m.File, m.MdfVersion(), nextEvent)
			if err != nil {
				fmt.Println(err)
			}
			nextEvent = event.Next()
		}

	}
}

func (m *MF4) loadHeader() {
	m.Header = HD.New(m.File, blocks.IdblockSize)
}

func (m *MF4) read(getXML bool) {
	var file *os.File = m.File

	if !m.IsFinalized() {
		panic("NOT FINALIZED MF4, PACKAGE IS NOT PREPARED")
	}

	version := m.MdfVersion()
	NextAddressDG := m.firstDataGroup()
	for NextAddressDG != 0 {
		dgBlock := DG.New(file, NextAddressDG)
		mdCommentAddr := dgBlock.MetadataComment()
		if mdCommentAddr != 0 {
			comment := MD.New(file, mdCommentAddr)
			fmt.Printf("%s\n", comment)
		}

		NextAddressCG := dgBlock.FirstChannelGroup()
		for NextAddressCG != 0 {
			cgBlock := CG.New(file, version, NextAddressCG)
			channelGroup := &ChannelGroup{
				Block:     cgBlock,
				Channels:  make(map[string]*CN.Block),
				Datagroup: dgBlock,
			}

			nextAddressCN := cgBlock.FirstChannel()
			for nextAddressCN != 0 {
				cnBlock := CN.New(file, version, nextAddressCN)
				channelName := cnBlock.GetChannelName(m.File)
				channelGroup.Channels[channelName] = cnBlock
				MdCommentAdress := cnBlock.GetCommentMd()
				if getXML && MdCommentAdress != 0 {
					comment := MD.New(file, MdCommentAdress)
					fmt.Println(comment)
				} else {
					mdBlock := (&MD.Block{}).BlankBlock()
					mdComment := ""
					fmt.Print(mdComment, mdBlock, "\n")
				}
				nextAddressCN = cnBlock.Next()
			}
			m.ChannelGroup = append(m.ChannelGroup, channelGroup)
			NextAddressCG = cgBlock.Next()
		}
		fmt.Println("\n##############################")
		NextAddressDG = dgBlock.Next()
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
				return nil, fmt.Errorf("error during parsing channel: %s ", err)
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

// GetAttachmemts iterates over all AT blocks and return to an array
func (m *MF4) GetAttachmemts() []AT.AttFile {
	return AT.Get(m.File, m.getFirstAttachment())
}

// Saves attachment file input to output path
func (m *MF4) SaveAttachment(a AT.AttFile, op string) AT.AttFile {
	return a.Save(m.File, op)
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

// Version method returns the MDF file version
func (m *MF4) Version() string {
	return string(m.Identification.Version[:])
}

// ID method returns the MDF file ID
func (m *MF4) ID() string {
	return string(m.Identification.File[:])
}

// CreatedBy method returns the MDF Program identifier
func (m *MF4) CreatedBy() string {
	return string(m.Identification.Program[:])
}

// VersionNumber method returns the Version number of the MDF format, i.e. 420
func (m *MF4) MdfVersion() uint16 {
	return m.Identification.VersionNumber
}

// isUnfinalized method returns Standard flags for unfinalized MDF
func (m *MF4) IsFinalized() bool {
	return m.Identification.UnfinalizedFlag == 0
}

func (m *MF4) firstDataGroup() int64 {
	return m.Header.Link.DgFirst
}

func (m *MF4) loadFirstFileHistory() {
	m.FileHistory = m.Header.Link.FhFirst
}

func (m *MF4) getFirstAttachment() int64 {
	return m.Header.Link.AtFirst
}

func (m *MF4) getFirstEvent() int64 {
	return m.Header.Link.EvFirst
}

// Start angle in radians at the beginning of the measurement serves as the
// reference point for angle synchronous measurements.
func (m *MF4) StartAngleRad() (float64, error) {
	if !m.isDistanceValid() {
		return 0, fmt.Errorf("start angle rad is not valid for this file")
	}
	return m.getStartAngleRad(), nil
}

// Start distance in meters in meters at the beginning of the measurement serves
// as the reference point for distance synchronous measurements.
func (m *MF4) StartDistanceM() (float64, error) {
	if m.isDistanceValid() {
		return 0, fmt.Errorf("start distance meters is not valid for this file")
	}
	return m.getStartDistanceM(), nil
}
func (m *MF4) getHDTimezoneOffsetMin() int16 {
	return m.Header.Data.TZOffsetMin
}

func (m *MF4) getTimeFlag() uint8 {
	return m.Header.Data.TimeFlags
}

func (m *MF4) geStartTimeNs() uint64 {
	return m.Header.Data.StartTimeNs
}

func (m *MF4) getStartAngleRad() float64 {
	return m.Header.Data.StartAngleRad
}

func (m *MF4) getStartDistanceM() float64 {
	return m.Header.Data.StartDistM
}

func (m *MF4) isDistanceValid() bool {
	return m.Header.Data.Flags == 1
}

func (m *MF4) getTimeClass() uint8 {
	return m.Header.Data.TimeClass
}

func (m *MF4) GetMeasureComment() string {
	if m.getHeaderMdComment() == 0 {
		return ""
	}
	return TX.GetText(m.File, m.getHeaderMdComment())
}

func (m *MF4) getHeaderMdComment() int64 {
	return m.Header.Link.MdComment
}

func (m *MF4) ReadChangeLog() {
	nextAddressFH := m.getFileHistory()
	for nextAddressFH != 0 {
		fhBlock := FH.New(m.File, nextAddressFH)
		c := fhBlock.GetChangeLog(m.File)
		t := fhBlock.GetTimeNs()
		f := fhBlock.GetTimeFlag()

		fmt.Println(m.formatLog(t, f, c))

		nextAddressFH = fhBlock.Next()
	}
}

// StartTimeNs returns the start timestamp of measurement in nanoseconds
func (m *MF4) StartTimeNs() int64 {
	t := m.geStartTimeNs()
	tzo := uint64(m.getHDTimezoneOffsetMin())
	dlo := uint64(m.getDaylightOffsetMin())
	tf := m.getTimeFlag()
	return m.GetTimeNs(t, tzo, dlo, tf)
}

func (m *MF4) StartTimeLT() time.Time {
	return m.formatTimeLT(m.StartTimeNs())
}

func (m *MF4) getFileHistory() int64 {
	return m.FileHistory
}

func (m *MF4) formatLog(t int64, f uint8, c string) string {
	ts := m.formatTimeLT(t)
	return fmt.Sprint(ts, c)
}
