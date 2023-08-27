package mf4

import (
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

func ReadFile(file *os.File, getXML bool) (*MF4,error) {
	var address int64 = 0

	//fileInfo, _ := file.Stat()
	//fileSize := fileInfo.Size()

	mf4File := MF4{File: file}

	//Load Identification IDBlock
	idBlock := ID.New(file, address)
	mf4File.Identification = idBlock

	fmt.Printf("%s %s %s %s %d %s \n", idBlock.File,
		idBlock.Version,
		idBlock.Program,
		idBlock.Reserved1,
		idBlock.VersionNumber,
		idBlock.Reserved2)

	fileVersion := idBlock.VersionNumber 

	if fileVersion < 400{
		return nil,&VersionError{}
	}
	
	if fileVersion >= 410 {
		mf4File.read(getXML)
	}

	if fileVersion >= 420 {
		fmt.Print("ADDING COLUMN STORE")
		//ADD COLUMN STORAGE
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
	hdBlock := HD.New(file, blocks.IdblockSize)

	//From HDBLOCK read File History
	m.loadFileHistory(file, hdBlock.Link.FhFirst, getXML)
	version := m.Identification.VersionNumber

	//From HDBLOCK read Attachments
	//Get all AT
	m.loadAttachmemt(file, hdBlock.Link.AtFirst)

	//From HDBLOCK read DataGroup
	NextAddressDG := hdBlock.Link.DgFirst
	index := 0

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
			indexCN := 0

			// mapCN := make(map[int]*blocks.CG)
			for nextAddressCN != 0 {
				cnBlock := CN.New(file, version, nextAddressCN)

				//Save Informations
				channel := &Channel{
					Block:        cnBlock,
					DataGroup:    dgBlock,
					ChannelGroup: cgBlock,
				}
				txBlock := TX.New(file, int64(cnBlock.Link.TxName))

				channelName := string(txBlock.Data.TxData)
				m.Channels[channelName] = channel

				//Get XML comments
				MdCommentAdress := cnBlock.Link.MdComment
				if getXML && MdCommentAdress != 0 {
					mdBlock := MD.Block{}
					mdBlock.New(file, MdCommentAdress)
				} else {
					mdBlock := (&MD.Block{}).BlankBlock()
					mdComment := ""
					fmt.Print(mdComment, mdBlock, "\n")
				}

				// var i uint64

				// readAddr := blocks.HeaderSize + dgBlock.Link.Data + int64(dgBlock.Data.RecIDSize) + int64(cnBlock.Data.ByteOffset)
				// size := (cnBlock.Data.BitCount + uint32(cnBlock.Data.BitOffset)) / 8

				// dataSlice := make([]byte, size)

				// if cnBlock.Data.Type == 0 || cnBlock.Data.Type == 2 {
				// 	if cnBlock.Data.DataType == 0 || cnBlock.Data.DataType == 2 || cnBlock.Data.DataType == 4 {
				// 		//Start read signals
				// 		for i = 0; i <= cgBlock.Data.CycleCount; i += 1 {
				// 			_, errs := file.Seek(readAddr, 0)
				// 			if errs != nil {
				// 				if errs != io.EOF {
				// 					fmt.Println(errs, "Memory Addr out of size")
				// 				}
				// 			}
				// 			_, err := file.Read(dataSlice)
				// 			if err != nil {
				// 				if err != io.EOF {
				// 					fmt.Println("LoadBuffer error: ", err)
				// 				}
				// 			}

				// 			fmt.Print("")
				// 			for _, x := range dataSlice {
				// 				fmt.Printf("%X ", x)
				// 			}
				// 			fmt.Print(" -- ")
				// 			measure := binary.LittleEndian.Uint16(dataSlice)
				// 			fmt.Println(measure)
				// 			readAddr += int64((cgBlock.Data.CycleCount + 16) / 8)
				// 		}
				// 	}
				// 	if cnBlock.Data.DataType != 0 {
				// 		//Start read signals
				// 		for i = 0; i <= cgBlock.Data.CycleCount; i += 1 {
				// 			_, errs := file.Seek(readAddr, 0)
				// 			if errs != nil {
				// 				if errs != io.EOF {
				// 					fmt.Println(errs, "Memory Addr out of size")
				// 				}
				// 			}
				// 			_, err := file.Read(dataSlice)
				// 			if err != nil {
				// 				if err != io.EOF {
				// 					fmt.Println("LoadBuffer error: ", err)
				// 				}
				// 			}

				// 			fmt.Print("")
				// 			for _, x := range dataSlice {
				// 				fmt.Printf("%X ", x)
				// 			}
				// 			fmt.Print("--")

				// 			readAddr += int64((cgBlock.Data.CycleCount + 16) / 8)
				// 		}
				// 	}

				// }

				// //End read signals

				nextAddressCN = cnBlock.Link.Next
				indexCN++

			}

			fmt.Println("\n##############################")

			NextAddressCG = cgBlock.Link.Next
			indexCG++
		}

		NextAddressDG = dgBlock.Link.Next
		index++
	}

}

// ChannelNames returns the sample data from a signal
func (m *MF4) ChannelNames() []string {
	channelNames := make([]string, 0)
	for key := range m.Channels {
		channelNames = append(channelNames, key)
	}
	return channelNames
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
	m.Attachments = array

}

// LoadFileHistory iterates over all FH blocks and append array to MF4 object
func (m *MF4) loadFileHistory(file *os.File, startAddressFH int64, getXML bool) {
	var index int = 0
	array := make([]*FH.Block, 0)
	nextAddressFH := startAddressFH

	//iterate over all FH blocks
	for nextAddressFH != 0 {
		fhBlock := FH.New(file, nextAddressFH)
		MdCommentAdress := fhBlock.Link.MDComment

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
