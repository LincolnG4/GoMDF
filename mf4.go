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
	"github.com/LincolnG4/GoMDF/internal/blocks/DT"
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
	Groups         map[string]*Group
}

type Group struct {
	DataGroup               *DG.Block
	ChannelGroup            *CG.Block
	Channels                *map[string]*CN.Block
	SignalData              []uint64
	Record                  int
	Trigger                 int
	StringDtypes            int
	DataBlocks              []uint64
	SingleChannelDtype      int
	UsesId                  bool
	ReadSplitCount          int
	DataBlocksInfoGenerator []uint64
	RecordSize              map[uint64]uint32
	Sorted                  bool
}

func ReadFile(file *os.File, getXML bool) *MF4 {
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

	if idBlock.VersionNumber >= 410 {
		mf4File.read(getXML)
	}

	if idBlock.VersionNumber >= 420 {
		fmt.Print("ADDING COLUMN STORE")
		//ADD COLUMN STORAGE
	}

	return &mf4File
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

func (m *MF4) read(getXML bool) {
	var file *os.File = m.File

	if m.Identification.UnfinalizedFlag != 0 {
		panic("NOT FINALIZED, CODE NOTE PREPARE FOR IT")
	}

	//Create MF4 struct from the file
	//Get HDBLOCK
	hdBlock := HD.New(file, blocks.IdblockSize)

	//From HDBLOCK read File History
	m.LoadFileHistory(file, hdBlock.Link.FhFirst, getXML)

	//From HDBLOCK read Attachments
	//Get all AT
	m.LoadAttachmemt(file, hdBlock.Link.AtFirst)

	//From HDBLOCK read DataGroup
	NextAddressDG := hdBlock.Link.DgFirst
	index := 0

	m.Groups = make(map[string]*Group)

	var cnBlock CN.Block
	

	//Get all DataGroup
	for NextAddressDG != 0 {

		//Store group
		grp := Group{}

		//Creat DGBlock
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
			cgBlock := CG.New(file,m.Identification.VersionNumber,NextAddressCG)

			//From CGBLOCK read Channel
			nextAddressCN := cgBlock.Link.CnFirst
			indexCN := 0

			// mapCN := make(map[int]*blocks.CG)
			for nextAddressCN != 0 {
				cnBlock = CN.Block{}
				

				cnBlock.New(file, nextAddressCN)
				fmt.Printf("%s\n", cnBlock.Header.ID)
				fmt.Printf("%+v\n", cnBlock.Header)
				fmt.Printf("%+v\n", cnBlock.Link)
				fmt.Printf("%+v\n\n", cnBlock.Data)

				txBlock := TX.New(file, int64(cnBlock.Link.TxName))
				channelName := string(txBlock.Data.TxData)
				channelMap := make(map[string]*CN.Block)
				channelMap[channelName] = &cnBlock

				grp.Channels = &channelMap
				//fmt.Println(channelName)

				//Get XML comments
				MdCommentAdress := cnBlock.Link.MdComment
				if getXML && MdCommentAdress != 0 {
					mdBlock := MD.Block{}
					mdBlock.New(file, MdCommentAdress)
					//mdComment := string(mdBlock.MdData.Value)

					fmt.Printf("%+v\n", mdBlock.Header)
					fmt.Printf("%+s\n\n", mdBlock.Data)

					//debug(file,MdCommentAdress,500)
				} else {
					mdBlock := (&MD.Block{}).BlankBlock()
					mdComment := ""
					fmt.Print(mdComment, mdBlock, "\n")
				}

				//debug(file, int64(dgBlock.Data), 1000)
				fmt.Printf("%+v", cnBlock)

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
				dtBlock := DT.Block{}

				dataAddress := dgBlock.Link.Data
				dtBlock.New(file, dataAddress)

				//CnDataType := cnBlock.Data.DataType
				numberBits := cnBlock.Data.BitCount
				offsetByte := cnBlock.Data.ByteOffset
				//offsetBit := cnBlock.Data.BitOffset
				//invalidBit := cnBlock.Data.InvalBitPos

				//databytesLeng := cgBlock.Data.DataBytes
				skipSignalAddr := dataAddress + int64(dgBlock.Data.RecIDSize) + int64(offsetByte)

				buf := make([]byte, numberBits)
				_, errs := file.Seek(skipSignalAddr, 0)
				if errs != nil {
					if errs != io.EOF {
						fmt.Println(errs)
					}

				}
				reader := bytes.NewReader(buf)
				debug(file, skipSignalAddr, int(numberBits))
				for {
					var value uint32
					err := binary.Read(reader, binary.LittleEndian, &value)
					if err != nil {
						// Exit the loop if no more data can be read
						break
					}
					fmt.Printf("Read value: %d\n", value)
				}

				nextAddressCN = cnBlock.Link.Next
				indexCN++

			}

			fmt.Println("\n##############################")

			NextAddressCG = cgBlock.Link.Next
			indexCG++
		}

		//Read data

		// dataAddress := dgBlock.Link.Data
		// dtBlock := DT.Block{}
		// dtBlock.New(file, dataAddress, 50)

		// fmt.Printf("\n%s\n", dtBlock.Header.ID)
		// fmt.Printf("%+v\n", dtBlock.Header)
		// fmt.Printf("%+v\n", dtBlock.Link)
		// fmt.Printf("%+v\n\n", dtBlock.Data)

		NextAddressDG = dgBlock.Link.Next
		index++
	}

}

// iterate over all AT blocks and append array to MF4 object
func (m *MF4) LoadAttachmemt(file *os.File, startAddressAT int64) {
	var index int = 0
	array := make([]*AT.Block, 0)
	nextAddressAT := startAddressAT

	for nextAddressAT != 0 {
		atBlock := AT.New(file, nextAddressAT)
		
		txBlock := TX.New(file,atBlock.Link.TXFilename) 
		filename := txBlock.Data.TxData
		fmt.Printf("Filename attached: %s\n",filename)

		txBlock = TX.New(file,atBlock.Link.TXMimetype) 
		mime := txBlock.Data.TxData
		fmt.Printf("Mime attached: %s\n",mime)
		

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

// iterate over all FH blocks and append array to MF4 object
func (m *MF4) LoadFileHistory(file *os.File, startAddressFH int64, getXML bool) {
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
