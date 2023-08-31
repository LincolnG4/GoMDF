package mf4

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

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
<<<<<<< HEAD
	Identification *blocks.ID
	Header         *blocks.HD
	FileHistory    []*blocks.FH
	Attachments    []*blocks.AT
	Events         []*blocks.EV
	Groups         map[string]*blocks.Group
}

// Read the MF4 file and return the MF4 object with all informations
func ReadFile(file *os.File, getXML bool) *MF4 {

	var cnBlock blocks.CN
	var cgBlock blocks.CG

	var startAddress blocks.Link = 0
=======
	File           *os.File
	Identification *ID.Block
	FileHistory    []*FH.Block
	Attachments    []*AT.Block
	Channels       map[string]*Channel
}

type Channel struct {
	Block        *CN.Block
	DataGroup    *DG.Block
	ChannelGroup *CG.Block
}

func ReadFile(file *os.File, getXML bool) (*MF4, error) {
	var address int64 = 0
>>>>>>> main

	//fileInfo, _ := file.Stat()
	//fileSize := fileInfo.Size()

	mf4File := MF4{File: file}

<<<<<<< HEAD
	//Get IDBLOCK
	idBlock := blocks.ID{}
	idBlock.New(file, startAddress, blocks.IdblockSize)

	mf4File.Identification = &idBlock
=======
	//Load Identification IDBlock
	idBlock := ID.New(file, address)
	mf4File.Identification = idBlock
>>>>>>> main

	fmt.Printf("%s %s %s %s %d %s \n", idBlock.File,
		idBlock.Version,
		idBlock.Program,
		idBlock.Reserved1,
		idBlock.VersionNumber,
		idBlock.Reserved2)

	fileVersion := idBlock.VersionNumber

	if fileVersion < 400 {
		return nil, &VersionError{}
	}

	if fileVersion >= 400 {
		mf4File.read(getXML)
	}

	return &mf4File, nil
}

func (m *MF4) read(getXML bool) {
	var file *os.File = m.File
	m.Channels = make(map[string]*Channel)

	if m.Identification.UnfinalizedFlag != 0 {
		panic("NOT FINALIZED, CODE NOTE PREPARE FOR IT")
	}

	//Create MF4 struct from the file
	//Get HDBLOCK
<<<<<<< HEAD
	startAddress = blocks.IdblockSize

	hdBlock := blocks.HD{}
	hdBlock.New(file, startAddress, blocks.HdblockSize)

	mf4File.Header = &hdBlock

	fmt.Printf("%s\n", hdBlock.Header.ID)
	fmt.Printf("%+v\n\n", hdBlock)
=======
	hdBlock := HD.New(file, blocks.IdblockSize)
>>>>>>> main

	//From HDBLOCK read File History
	m.loadFileHistory(file, hdBlock.Link.FhFirst, getXML)
	version := m.Identification.VersionNumber

	//From HDBLOCK read Attachments
	//Get all AT
<<<<<<< HEAD
	startAddressAT := hdBlock.ATFirst
	mf4File.LoadAttachmemt(file, startAddressAT)
	fmt.Printf("%+v\n\n", mf4File.Attachments)
=======
	m.loadAttachmemt(file, hdBlock.Link.AtFirst)
>>>>>>> main

	//From HDBLOCK read DataGroup
	NextAddressDG := hdBlock.Link.DgFirst
	index := 0

<<<<<<< HEAD
	mf4File.Groups = make(map[string]*blocks.Group)

	//Get all DataGroup
	for NextAddressDG != 0 {
		//Store group
		grp := blocks.Group{}

		dgBlock := blocks.DG{}
		grp.DataGroup = &dgBlock

		dgBlock.New(file, NextAddressDG, blocks.DgblockSize)

		fmt.Printf("%s\n", dgBlock.Header.ID)
		fmt.Printf("%+v\n\n", dgBlock)

		//From DGBLOCK read ChannelGroup
		NextAddressCG := dgBlock.CGFirst
		indexCG := 0
		arrayCG := make([]*blocks.CG, 0)

		for NextAddressCG != 0 {
			cgBlock = blocks.CG{}
			arrayCG = append(arrayCG, &cgBlock)
			grp.ChannelGroup = arrayCG
			

			cgBlock.New(file, NextAddressCG, blocks.CgblockSize)

			fmt.Printf("%s\n", cgBlock.Header.ID)
			fmt.Printf("%+v\n\n", cgBlock)

			//debug(file, cgBlock.TxAcqName, 88)
			//From CGBLOCK read Channel

			nextAddressCN := cgBlock.CNNext
=======
	//Get all DataGroup
	for NextAddressDG != 0 {
		//Create DGBlock and append to MF4
		dgBlock := DG.New(file, NextAddressDG)

		//Read MdBlocks inside
		mdCommentAddr := dgBlock.Link.MdComment
		if mdCommentAddr != 0 {
			mdBlock := MD.ReadMdComment(file, mdCommentAddr)
			comment := mdBlock.Data.Value
			fmt.Printf("%s\n", comment)
		}

		//From DGBLOCK read ChannelGroup
		NextAddressCG := dgBlock.Link.CgFirst
		indexCG := 0

		for NextAddressCG != 0 {
			cgBlock := CG.New(file, version, NextAddressCG)

			//From CGBLOCK read Channel
			nextAddressCN := cgBlock.Link.CnFirst
>>>>>>> main
			indexCN := 0

			for nextAddressCN != 0 {
<<<<<<< HEAD
				cnBlock = blocks.CN{}

				cnBlock.New(file, nextAddressCN)
				// fmt.Printf("%+v\n\n", cnBlock)
=======
				cnBlock := CN.New(file, version, nextAddressCN)

				//Save Informations
				channel := &Channel{
					Block:        cnBlock,
					DataGroup:    dgBlock,
					ChannelGroup: cgBlock,
				}
>>>>>>> main

				txBlock := TX.New(file, int64(cnBlock.Link.TxName))

<<<<<<< HEAD
				txBlock.New(file, cnBlock.TxName, 50)
				channelName := string(txBlock.TxData)
				channelMap := make(map[string]*blocks.CN)
				channelMap[channelName] = &cnBlock

				grp.Channels = &channelMap
				fmt.Println(channelName)

				//Get XML comments
				if getXML && cnBlock.MdComment != 0 {
					mdBlock := blocks.MD{}
					mdBlock.New(file, cnBlock.MdComment, 50)
					//mdComment := string(mdBlock.MdData.Value)
					//fmt.Print(mdComment,"\n")
=======
				channelName := string(bytes.Trim(txBlock.Data.TxData, "\x00"))
				m.Channels[channelName] = channel

				//Get XML comments
				MdCommentAdress := cnBlock.Link.MdComment
				if getXML && MdCommentAdress != 0 {
					mdBlock := MD.Block{}
					mdBlock.New(file, MdCommentAdress)
>>>>>>> main
				} else {
					mdBlock := (&MD.Block{}).BlankBlock()
					mdComment := ""
					fmt.Print(mdComment, mdBlock, "\n")
				}

<<<<<<< HEAD
				//debug(file, int64(dgBlock.Data), 1000)
				fmt.Printf("%+v", cnBlock)

				// signal data

				// // signal data
				// if cnBlock.Data != 0 {
				// 	cnBlock.GetSignalData(file,dgBlock.Data,dgBlock.RecIDSize, dgBlock.Header.Length)
				// }else{
				// 	fmt.Println("")
				// }

				// if cnBlock.CnComposition != 0 {
				// 	cnBlock.GetSignalData(file)
				// }else{
				// 	fmt.Println("")
				// }
		
				//Read data


		
				

			

				nextAddressCN = cnBlock.Next
=======
				nextAddressCN = cnBlock.Link.Next
>>>>>>> main
				indexCN++

			}

			

<<<<<<< HEAD
			NextAddressCG = cgBlock.Next
=======
			NextAddressCG = cgBlock.Link.Next
>>>>>>> main
			indexCG++
		}
		
		fmt.Println("\n##############################")
		dtBlock := blocks.DT{}
		dtBlock.New(file,dgBlock.Data,100)

<<<<<<< HEAD
		fmt.Printf("%+v \n",dtBlock.Header)

		NextAddressDG = dgBlock.Next
=======
		NextAddressDG = dgBlock.Link.Next
>>>>>>> main
		index++
	}

}

<<<<<<< HEAD
func (m *MF4) LoadAttachmemt(file *os.File, startAddressAT blocks.Link) {
	var index int = 0
	arrayAT := make([]*blocks.AT, 0)
	nextAddressAT := startAddressAT

	for nextAddressAT != 0 {
		atBlock := blocks.AT{}
		atBlock.New(file, nextAddressAT, blocks.AtblockSize)

		arrayAT = append(arrayAT, &atBlock)

		fmt.Printf("%s\n", atBlock.Header.ID)
		fmt.Printf("%+v\n\n", atBlock)
=======
// ChannelNames returns the sample data from a signal
func (m *MF4) ChannelNames() []string {
	channelNames := make([]string, 0)
	for key := range m.Channels {
		channelNames = append(channelNames, key)
	}
	return channelNames
}

func (m *MF4) GetChannelSample(channelName string) {
	cn := m.Channels[channelName]
	dg := cn.DataGroup
	cg := cn.ChannelGroup
	file := m.File

	//cnType := cn.Block.Data.Type
	//cnSyncType := cn.Block.Data.SyncType
	dataType := cn.Block.Data.DataType

	readAddr := blocks.HeaderSize + dg.Link.Data + int64(dg.Data.RecIDSize) + int64(cn.Block.Data.ByteOffset)
	size := (cn.Block.Data.BitCount + uint32(cn.Block.Data.BitOffset)) / 8
	data := make([]byte, size)

	rowSize := int64(cg.Data.DataBytes)

	var byteOrder binary.ByteOrder

	if dataType == 0 || dataType == 2 || dataType == 4 || dataType == 8 || dataType == 15 {
		byteOrder = binary.LittleEndian
	} else {
		byteOrder = binary.BigEndian
	}
	dtype:=loadDataType(dataType, len(data))
	fmt.Println(channelName)
	switch dtype.(type) {
	case uint8:
		var i uint64
		var value uint8
		
		for i = 0; i < cg.Data.CycleCount; i += 1 {
			seekRead(file, readAddr, data)

			buf := bytes.NewBuffer(data)
			err := binary.Read(buf, byteOrder, &value)
			if err != nil {
				fmt.Println("Error reading uint32:", err)
				return
			}
			fmt.Print("--Value:", value,"--")

			readAddr += rowSize
		}
	case uint16:
		var value uint16
		var i uint64
		
		for i = 0; i < cg.Data.CycleCount; i += 1 {
			seekRead(file, readAddr, data)

			buf := bytes.NewBuffer(data)
			err := binary.Read(buf, byteOrder, &value)
			if err != nil {
				fmt.Println("Error reading uint32:", err)
				return
			}
			fmt.Print("--Value:", value,"--")

			readAddr += rowSize
		}
	case uint32:
		var value uint32
		var i uint64
		
		for i = 0; i < cg.Data.CycleCount; i += 1 {
			seekRead(file, readAddr, data)

			buf := bytes.NewBuffer(data)
			err := binary.Read(buf, byteOrder, &value)
		
			if err != nil {
				fmt.Println("Error reading uint32:", err)
				return
			}
			fmt.Print("--Value:", value,"--")

			readAddr += rowSize
		}
	case uint64:
		var value uint64
		var i uint64
		
		for i = 0; i < cg.Data.CycleCount; i += 1 {
			seekRead(file, readAddr, data)

			buf := bytes.NewBuffer(data)
			err := binary.Read(buf, byteOrder, &value)
			if err != nil {
				fmt.Println("Error reading uint32:", err)
				return
			}
			fmt.Print("--Value:", value,"--")

			readAddr += rowSize
		}
	case int8:
		var value int8
		var i uint64
		
		for i = 0; i < cg.Data.CycleCount; i += 1 {
			seekRead(file, readAddr, data)

			buf := bytes.NewBuffer(data)
			err := binary.Read(buf, byteOrder, &value)
			if err != nil {
				fmt.Println("Error reading uint32:", err)
				return
			}
			fmt.Print("--Value:", value,"--")

			readAddr += rowSize
		}
	case int16:
		var value int16
		var i uint64
		
		for i = 0; i < cg.Data.CycleCount; i += 1 {
			seekRead(file, readAddr, data)

			buf := bytes.NewBuffer(data)
			err := binary.Read(buf, byteOrder, &value)
			if err != nil {
				fmt.Println("Error reading uint32:", err)
				return
			}
			fmt.Print("--Value:", value,"--")

			readAddr += rowSize
		}
	case int32:
		var value int32
		var i uint64
		
		for i = 0; i < cg.Data.CycleCount; i += 1 {
			seekRead(file, readAddr, data)

			buf := bytes.NewBuffer(data)
			err := binary.Read(buf, byteOrder, &value)
			if err != nil {
				fmt.Println("Error reading uint32:", err)
				return
			}
			fmt.Print("--Value:", value,"--")
			readAddr += rowSize
		}
	case int64:
		var value int64
		var i uint64
		
		for i = 0; i < cg.Data.CycleCount; i += 1 {
			seekRead(file, readAddr, data)

			buf := bytes.NewBuffer(data)
			err := binary.Read(buf, byteOrder, &value)
			if err != nil {
				fmt.Println("Error reading uint32:", err)
				return
			}
			fmt.Print("--Value:", value,"--")

			readAddr += rowSize
		}
	case float32:
		var value float32
		var i uint64
		
		for i = 0; i < cg.Data.CycleCount; i += 1 {
			seekRead(file, readAddr, data)

			buf := bytes.NewBuffer(data)
			err := binary.Read(buf, byteOrder, &value)
			if err != nil {
				fmt.Println("Error reading uint32:", err)
				return
			}
			fmt.Print("--Value:", value,"--")

			readAddr += rowSize
		}
	case float64:
		var value float64
		var i uint64
		
		for i = 0; i < cg.Data.CycleCount; i += 1 {
			seekRead(file, readAddr, data)

			buf := bytes.NewBuffer(data)
			err := binary.Read(buf, byteOrder, &value)
			if err != nil {
				fmt.Println("Error reading uint32:", err)
				return
			}
			fmt.Print("--Value:", value,"--")

			readAddr += rowSize
		}
						
	}

	// switch cnType {
	// case 0:

	// case 1:
	// 	fmt.Print("1")
	// case 2:
	// 	switch cnSyncType {
	// 	case 0:
	// 		fmt.Println("Normal")
	// 	case 1:
	// 		fmt.Println("Time")
	// 	case 2:
	// 		fmt.Println("Angle")
	// 	case 3:
	// 		fmt.Println("Distance")
	// 	case 4:
	// 		fmt.Println("Index")
	// 	}
	// case 3:
	// 	fmt.Print("Virtual Master channel")
	// case 4:
	// 	fmt.Print("Virtual Master channel")
	// case 5:
	// 	fmt.Print("Virtual Master channel")
	// case 6:
	// 	fmt.Print("Virtual Master channel")
	// }

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
func (m *MF4) loadAttachmemt(file *os.File, startAddressAT int64) {
	var index int = 0
	array := make([]*AT.Block, 0)
	nextAddressAT := startAddressAT

	for nextAddressAT != 0 {
		atBlock := AT.New(file, nextAddressAT)

		txBlock := TX.New(file, atBlock.Link.TXFilename)
		filename := txBlock.Data.TxData
		fmt.Printf("Filename attached: %s\n", filename)
>>>>>>> main

		txBlock = TX.New(file, atBlock.Link.TXMimetype)
		mime := txBlock.Data.TxData
		fmt.Printf("Mime attached: %s\n", mime)

		//Read MDComment
		MdCommentAdress := atBlock.Link.MDComment
		if MdCommentAdress != 0 {
			mdBlock := MD.ReadMdComment(file, MdCommentAdress)
			comment := mdBlock.Data.Value
			fmt.Printf("%s\n", comment)
		}

		array = append(array, atBlock)
		nextAddressAT = atBlock.Link.Next
		index++
	}
<<<<<<< HEAD
	m.Attachments = arrayAT
}

func (m *MF4) LoadFileHistory(file *os.File, startAddressFH blocks.Link) {
=======
	m.Attachments = array

}

// LoadFileHistory iterates over all FH blocks and append array to MF4 object
func (m *MF4) loadFileHistory(file *os.File, startAddressFH int64, getXML bool) {
>>>>>>> main
	var index int = 0
	array := make([]*FH.Block, 0)
	nextAddressFH := startAddressFH

	//iterate over all FH blocks
	for nextAddressFH != 0 {
<<<<<<< HEAD
		fhBlock := blocks.FH{}
		fhBlock.New(file, nextAddressFH, blocks.FhblockSize)
=======
		fhBlock := FH.New(file, nextAddressFH)
		MdCommentAdress := fhBlock.Link.MDComment
>>>>>>> main

		//Read MDComment
		if MdCommentAdress != 0 {
			comment := MD.ReadMdComment(file, MdCommentAdress)
			fmt.Printf("%s\n", comment.Data)
		}

		array = append(array, fhBlock)

		nextAddressFH = fhBlock.Link.Next
		index++

	}
	m.FileHistory = array

}

// Version method returns the MDF file version
func (m *MF4) Version() string {
	return string(m.Identification.Version[:])
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
