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

	//fileInfo, _ := file.Stat()
	//fileSize := fileInfo.Size()

	mf4File := MF4{}

	//Get IDBLOCK
	idBlock := blocks.ID{}
	idBlock.New(file, startAddress, blocks.IdblockSize)

	mf4File.Identification = &idBlock

	fmt.Printf("%s %s %s %s %d %s \n", idBlock.File,
		idBlock.Version,
		idBlock.Program,
		idBlock.Reserved1,
		idBlock.VersionNumber,
		idBlock.Reserved2)

	//Create MF4 struct from the file

	//Get HDBLOCK
	startAddress = blocks.IdblockSize

	hdBlock := blocks.HD{}
	hdBlock.New(file, startAddress, blocks.HdblockSize)

	mf4File.Header = &hdBlock

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
	fmt.Printf("%+v\n\n", mf4File.Attachments)

	//From HDBLOCK read DataGroup
	NextAddressDG := hdBlock.DGFirst
	index := 0

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
			indexCN := 0

			// mapCN := make(map[int]*blocks.CG)
			for nextAddressCN != 0 {
				cnBlock = blocks.CN{}

				cnBlock.New(file, nextAddressCN)
				// fmt.Printf("%+v\n\n", cnBlock)

				txBlock := blocks.TX{}

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
		
				//Read data


		
				

			

				nextAddressCN = cnBlock.Next
				indexCN++

			}

			

			NextAddressCG = cgBlock.Next
			indexCG++
		}
		
		fmt.Println("\n##############################")
		dtBlock := blocks.DT{}
		dtBlock.New(file,dgBlock.Data,100)

		fmt.Printf("%+v \n",dtBlock.Header)

		NextAddressDG = dgBlock.Next
		index++
	}

	return &mf4File
}

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

		nextAddressAT = atBlock.ATNext
		index++
	}
	m.Attachments = arrayAT
}

func (m *MF4) LoadFileHistory(file *os.File, startAddressFH blocks.Link) {
	var index int = 0
	FHarray := make([]*blocks.FH, 0)
	nextAddressFH := startAddressFH

	for nextAddressFH != 0 {
		fhBlock := blocks.FH{}
		fhBlock.New(file, nextAddressFH, blocks.FhblockSize)

		FHarray = append(FHarray, &fhBlock)

		fmt.Printf("%+v\n\n", fhBlock)

		nextAddressFH = fhBlock.FHNext
		index++
	}
	m.FileHistory = FHarray
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
