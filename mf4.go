package mf4

import (
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/davecgh/go-spew/spew"
)

type MF4 struct {
	Identification *blocks.ID
	FileHistory    []*blocks.FH
	AT             map[int]*blocks.AT
	//EV map[int]*blocks.EVblock
	//CH map[int]*blocks.CHBlock
	Groups map[string]*blocks.Group
}

func ReadFile(file *os.File, getXML bool) *MF4 {
	var startAddress int64 = 0
	var previousBlock int
	//fileInfo, _ := file.Stat()
	//fileSize := fileInfo.Size()

	mf4File := MF4{}

	//Get IDBLOCK
	idBlock := blocks.ID{}
	idBlock.NewBlock(file, startAddress, blocks.IdblockSize)

	mf4File.Identification = &idBlock

	fmt.Printf("%s %s %s %s %d %s \n", idBlock.File,
		idBlock.Version,
		idBlock.Program,
		idBlock.Reserved1,
		idBlock.VersionNumber,
		idBlock.Reserved2)

	//Create MF4 struct from the file

	//Get HDBLOCK
	previousBlock = blocks.IdblockSize
	startAddress += int64(previousBlock)

	hdBlock := blocks.HD{}
	hdBlock.NewBlock(file, startAddress, blocks.HdblockSize)

	fmt.Printf("%s\n", hdBlock.Header.ID)
	fmt.Printf("%+v\n\n", hdBlock)

	//From HDBLOCK read File History
	startAddressFH := hdBlock.FHFirst

	//Get all File History
	fmt.Println("##FH")
	mf4File.LoadFileHistory(file, startAddressFH)

	//From HDBLOCK read Attachments
	fmt.Println("##AT")

	//Get all AT
	startAddressAT := hdBlock.ATFirst
	mf4File.LoadAttachmemt(file, startAddressAT)
	fmt.Printf("%+v\n\n", mf4File.AT)

	//From HDBLOCK read DataGroup
	NextAddressDG := hdBlock.DGFirst
	index := 0

	mf4File.Groups = make(map[string]*blocks.Group)

	var cnBlock blocks.CN
	var cgBlock blocks.CG

	//Get all DataGroup
	for NextAddressDG != 0 {

		//Store group
		grp := blocks.Group{}

		dgBlock := blocks.DG{}
		grp.DataGroup = &dgBlock

		dgBlock.NewBlock(file, NextAddressDG, blocks.DgblockSize)

		fmt.Printf("%s\n", dgBlock.Header.ID)
		fmt.Printf("%+v\n\n", dgBlock)

		//From DGBLOCK read ChannelGroup
		NextAddressCG := dgBlock.CGNext
		indexCG := 0
		mapCG := make(map[int]*blocks.CG)

		for NextAddressCG != 0 {

			cgBlock = blocks.CG{}
			grp.ChannelGroup = &cgBlock

			cgBlock.NewBlock(file, NextAddressCG, blocks.CgblockSize)

			mapCG[indexCG] = &cgBlock

			// fmt.Printf("%s\n", cgBlock.Header.ID)
			// fmt.Printf("%+v\n\n", cgBlock)

			//debug(file, cgBlock.TxAcqName, 88)
			//From CGBLOCK read Channel
			nextAddressCN := cgBlock.CNNext
			indexCN := 0

			// mapCN := make(map[int]*blocks.CG)
			for nextAddressCN != 0 {
				cnBlock = blocks.CN{}
				grp.ChannelGroup = &cgBlock

				cnBlock.NewBlock(file, nextAddressCN)
				// fmt.Printf("%+v\n\n", cnBlock)

				txBlock := blocks.TX{}

				txBlock.NewBlock(file, int64(cnBlock.TxName), 50)
				channelName := string(txBlock.Link.TxData)
				channelMap := make(map[string]*blocks.CN)
				channelMap[channelName] = &cnBlock

				grp.Channels = &channelMap
				//fmt.Println(channelName)

				//Get XML comments
				if getXML && cnBlock.MdComment != 0 {
					mdBlock := blocks.MD{}
					mdBlock.NewBlock(file, int64(cnBlock.MdComment), 50)
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

				nextAddressCN = cnBlock.CnNext
				indexCN++

			}

			fmt.Println("\n##############################")

			NextAddressCG = cgBlock.CGNext
			indexCG++
		}

		//Read data

		//dataAddress := dgBlock.Data

		NextAddressDG = dgBlock.DGNext
		index++
	}

	return &mf4File
}

func (m *MF4) LoadAttachmemt(file *os.File, startAddressAT int64) {
	var index int = 0
	mapAT := make(map[int]*blocks.AT)
	nextAddressAT := startAddressAT

	for nextAddressAT != 0 {
		atBlock := blocks.AT{}
		atBlock.NewBlock(file, nextAddressAT, blocks.AtblockSize)

		mapAT[index] = &atBlock
		m.AT = mapAT
		fmt.Printf("%s\n", atBlock.Header.ID)
		fmt.Printf("%+v\n\n", atBlock)

		nextAddressAT = atBlock.ATNext
		index++
	}

}

func (m *MF4) LoadFileHistory(file *os.File, startAddressFH int64) {
	var index int = 0
	FHarray := make([]*blocks.FH, 0)
	nextAddressFH := startAddressFH

	for nextAddressFH != 0 {
		fhBlock := blocks.FH{}
		fhBlock.NewBlock(file, nextAddressFH, blocks.FhblockSize)

		FHarray = append(FHarray, &fhBlock)

		fmt.Printf("%+v\n\n", fhBlock)

		nextAddressFH = fhBlock.FHNext
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
