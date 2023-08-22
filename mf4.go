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
	"github.com/davecgh/go-spew/spew"
)

type MF4 struct {
	Identification *ID.Block
	FileHistory    []*FH.Block
	AT             map[int]*AT.Block
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
	var startAddress int64 = 0

	//fileInfo, _ := file.Stat()
	//fileSize := fileInfo.Size()

	mf4File := MF4{}

	//Get IDBLOCK
	idBlock := ID.Block{}
	idBlock.New(file, startAddress)

	mf4File.Identification = &idBlock

	fmt.Printf("%s %s %s %s %d %s \n", idBlock.File,
		idBlock.Version,
		idBlock.Program,
		idBlock.Reserved1,
		idBlock.VersionNumber,
		idBlock.Reserved2)

	//Create MF4 struct from the file

	//Get HDBLOCK
	hdBlock := HD.Block{}
	hdBlock.New(file, blocks.IdblockSize, blocks.HdblockSize)

	fmt.Printf("%s\n", hdBlock.Header.ID)
	fmt.Printf("%+v\n\n", hdBlock.Header)
	fmt.Printf("%+v\n\n", hdBlock.Link)
	fmt.Printf("%+v\n\n", hdBlock.Data)

	//From HDBLOCK read File History
	startAddressFH := hdBlock.Link.FHFirst

	//Get all File History
	fmt.Println("##FH")

	mf4File.LoadFileHistory(file, startAddressFH)

	//From HDBLOCK read Attachments
	fmt.Println("##AT")

	//Get all AT
	startAddressAT := hdBlock.Link.ATFirst
	mf4File.LoadAttachmemt(file, startAddressAT)
	fmt.Printf("%+v\n\n", mf4File.AT)

	//From HDBLOCK read DataGroup
	NextAddressDG := hdBlock.Link.DGFirst
	index := 0

	mf4File.Groups = make(map[string]*Group)

	var cnBlock CN.Block
	var cgBlock CG.Block

	//Get all DataGroup
	for NextAddressDG != 0 {

		//Store group
		grp := Group{}

		dgBlock := DG.Block{}
		grp.DataGroup = &dgBlock

		dgBlock.New(file, NextAddressDG, blocks.DgblockSize)

		fmt.Printf("%s\n", dgBlock.Header.ID)
		fmt.Printf("%+v\n", dgBlock.Header)
		fmt.Printf("%+v\n", dgBlock.Link)
		fmt.Printf("%+v\n", dgBlock.Data)

		//From DGBLOCK read ChannelGroup
		NextAddressCG := dgBlock.Link.CGNext
		indexCG := 0
		mapCG := make(map[int]*CG.Block)

		for NextAddressCG != 0 {

			cgBlock = CG.Block{}
			grp.ChannelGroup = &cgBlock

			cgBlock.New(file, NextAddressCG, blocks.CgblockSize)

			mapCG[indexCG] = &cgBlock

			fmt.Printf("\n%s\n", cgBlock.Header.ID)
			fmt.Printf("%+v\n", cgBlock.Header)
			fmt.Printf("%+v\n", cgBlock.Link)
			fmt.Printf("%+v\n\n", cgBlock.Data)

			//debug(file, cgBlock.TxAcqName, 88)
			//From CGBLOCK read Channel
			nextAddressCN := cgBlock.Link.CnNext
			indexCN := 0

			// mapCN := make(map[int]*blocks.CG)
			for nextAddressCN != 0 {
				cnBlock = CN.Block{}
				grp.ChannelGroup = &cgBlock

				cnBlock.New(file, nextAddressCN)
				fmt.Printf("\n%s\n", cnBlock.Header.ID)
				fmt.Printf("%+v\n", cnBlock.Header)
				fmt.Printf("%+v\n", cnBlock.Link)
				fmt.Printf("%+v\n\n", cnBlock.Data)

				txBlock := blocks.TX{}

				txBlock.New(file, int64(cnBlock.Link.TxName), 50)
				channelName := string(txBlock.Link.TxData)
				channelMap := make(map[string]*CN.Block)
				channelMap[channelName] = &cnBlock

				grp.Channels = &channelMap
				//fmt.Println(channelName)

				//Get XML comments
				MdCommentAdress := cnBlock.Link.MdComment
				if getXML && MdCommentAdress != 0 {
					mdBlock := blocks.MD{}
					mdBlock.New(file, int64(MdCommentAdress), 50)
					//mdComment := string(mdBlock.MdData.Value)
					//fmt.Print(mdComment,"\n")
				} else {
					mdBlock := (&blocks.MD{}).BlankBlock()
					mdComment := ""
					fmt.Print(mdComment, mdBlock, "\n")
				}

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

				fmt.Println(grp)

				nextAddressCN = cnBlock.Link.Next
				indexCN++

			}

			fmt.Println("\n##############################")

			NextAddressCG = cgBlock.Link.Next
			indexCG++
		}

		//Read data

		//dataAddress := dgBlock.Data

		NextAddressDG = dgBlock.Link.Next
		index++
	}

	return &mf4File
}

func (m *MF4) LoadAttachmemt(file *os.File, startAddressAT int64) {
	var index int = 0
	mapAT := make(map[int]*AT.Block)
	nextAddressAT := startAddressAT

	for nextAddressAT != 0 {
		atBlock := AT.Block{}
		atBlock.New(file, nextAddressAT, blocks.AtblockSize)

		mapAT[index] = &atBlock
		m.AT = mapAT
		fmt.Printf("%s\n", atBlock.Header.ID)
		fmt.Printf("%+v\n\n", atBlock)

		nextAddressAT = atBlock.Link.Next
		index++
	}

}

func (m *MF4) LoadFileHistory(file *os.File, startAddressFH int64) {
	var index int = 0
	FHarray := make([]*FH.Block, 0)
	nextAddressFH := startAddressFH

	for nextAddressFH != 0 {
		fhBlock := FH.Block{}
		fhBlock.New(file, nextAddressFH, blocks.FhblockSize)

		FHarray = append(FHarray, &fhBlock)

		fmt.Printf("%+v\n\n", fhBlock)

		nextAddressFH = fhBlock.Link.Next
		index++
	}
	m.FileHistory = FHarray
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
