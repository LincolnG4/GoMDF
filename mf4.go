package mf4

import (
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/davecgh/go-spew/spew"
)

type MF4 struct {
	FH map[int]*blocks.FH
	AT map[int]*blocks.AT
	//EV map[int]*blocks.EVblock
	//CH map[int]*blocks.CHBlock
	DG map[int]*blocks.DG
	CG map[int]*blocks.CG
}

func (mf4 *MF4) ReadFile(file *os.File, getXML bool) {
	var startAddress int64 = 0
	var previousBlock int
	//fileInfo, _ := file.Stat()
	//fileSize := fileInfo.Size()

	//Get IDBLOCK
	idBlock := blocks.ID{}
	idBlock.NewBlock(file, startAddress, blocks.IdblockSize)

	previousBlock = blocks.IdblockSize

	fmt.Printf("%s %s %s %s %d %s \n", idBlock.File,
		idBlock.Version,
		idBlock.Program,
		idBlock.Reserved1,
		idBlock.VersionNumber,
		idBlock.Reserved2)

	//Create MF4 struct from the file
	mf4File := MF4{}

	//Get HDBLOCK
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
	startAddressAT := hdBlock.ATFirst

	//Get all File History
	mf4File.LoadAttachmemt(file, startAddressAT)
	fmt.Printf("%+v\n\n", mf4File.AT)

	//From HDBLOCK read DataGroup
	NextAddressDG := hdBlock.DGFirst
	index := 0
	mapDG := make(map[int]*blocks.DG)

	//Get all DataGroup
	for NextAddressDG != 0 {

		dgBlock := blocks.DG{}
		dgBlock.NewBlock(file, NextAddressDG, blocks.DgblockSize)

		mapDG[index] = &dgBlock
		mf4File.DG = mapDG
		fmt.Printf("%s\n", dgBlock.Header.ID)
		fmt.Printf("%+v\n\n", dgBlock)

		//From DGBLOCK read ChannelGroup
		NextAddressCG := dgBlock.CGNext
		indexCG := 0
		mapCG := make(map[int]*blocks.CG)

		for NextAddressCG != 0 {
			cgBlock := blocks.CG{}
			cgBlock.NewBlock(file, NextAddressCG, blocks.CgblockSize)

			mapCG[indexCG] = &cgBlock
			mf4File.CG = mapCG
			fmt.Printf("%s\n", cgBlock.Header.ID)
			fmt.Printf("%+v\n\n", cgBlock)

			//debug(file, cgBlock.TxAcqName, 88)
			//From CGBLOCK read Channel
			nextAddressCN := cgBlock.CNNext
			indexCN := 0

			// mapCN := make(map[int]*blocks.CG)
			for nextAddressCN != 0 {
				cnBlock := blocks.CN{}

				cnBlock.NewBlock(file, nextAddressCN)
				fmt.Printf("%+v\n\n", cnBlock)

				txBlock := blocks.TX{}
				txBlock.NewBlock(file, int64(cnBlock.TxName), 50)
				channelName := string(txBlock.Link.TxData)

				fmt.Println(channelName)

				if getXML && cnBlock.MdComment != 0 {
					mdBlock := blocks.MD{}
					mdBlock.NewBlock(file, int64(cnBlock.MdComment), 50)
					mdComment := string(mdBlock.MdData.Value)
					fmt.Print(mdComment)
				} else {

					mdBlock := (&blocks.MD{}).BlankBlock()
					mdComment := ""
					fmt.Print(mdComment, mdBlock)
				}

				// debug(file, int64(cnBlock.MdComment), 100)
				fmt.Printf("%s\n", cnBlock.Header.ID)
				fmt.Printf("%+v\n\n", cnBlock)

				nextAddressCN = cnBlock.CnNext
				indexCN++
			}

			NextAddressCG = cgBlock.CGNext
			indexCG++
		}

		NextAddressDG = dgBlock.DGNext
		index++
	}
	fmt.Printf("%+v\n", mf4File.DG)

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
	mapFH := make(map[int]*blocks.FH)
	nextAddressFH := startAddressFH

	for nextAddressFH != 0 {
		fhBlock := blocks.FH{}
		fhBlock.NewBlock(file, nextAddressFH, blocks.FhblockSize)

		mapFH[index] = &fhBlock
		m.FH = mapFH
		fmt.Printf("%+v\n\n", fhBlock)

		nextAddressFH = fhBlock.FHNext
		index++
	}

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
