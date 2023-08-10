package mdf

import (
	"fmt"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)
const (
		FH_BLOCK_SIZE = 56
		AT_BLOCK_SIZE = 96
		DG_BLOCK_SIZE = 64
		CG_BLOCK_SIZE = 104
		COMMON_SIZE = 24
	)

func OpenFile(file *os.File) {

	
	
  cgMAP := make(map[int64]int, 0)

	
	fileInfo, _ := file.Stat()

	fileSize := fileInfo.Size()

	

	//Create IDBLOCK
	idBlock := blocks.IDBlock{}
	idBlock.NewBlock(file)
	
	fmt.Printf("%+v\n", idBlock)

	if idBlock.IDVersionNumber > 400 {
		//Create HDBLOCK
		hdBlock := blocks.HDBlock{}
		hdBlock.NewBlock(file)

		// read file history
		fileHistoryAddr := hdBlock.HDFHFirst
		fileHistory := make([]blocks.FHBlock, 0)
		i := 0
		for fileHistoryAddr != 0 {
			if (fileHistoryAddr + FH_BLOCK_SIZE) > fileSize {
				fmt.Println("File history address", fileHistoryAddr, "is outside the file size", fileSize)
				break
			}
			fhBlock := blocks.FHBlock{}

			fhBlock.HistoryBlock(file, fileHistoryAddr)
			fileHistory = append(fileHistory, fhBlock)
			fileHistoryAddr = fhBlock.FHNext

			i++
		}

		// read file history
		attachmentAddr := hdBlock.HDATFirst
		attachmentArray := make([]blocks.ATBlock, 0)

		i = 0
		for attachmentAddr != 0 {
			if (attachmentAddr + AT_BLOCK_SIZE) > fileSize {
				fmt.Println("File history address", attachmentAddr, "is outside the file size", fileSize)
				break
			}
			atBlock := blocks.ATBlock{}

			atBlock.AttchmentBlock(file, attachmentAddr)
			attachmentArray = append(attachmentArray, atBlock)
			attachmentAddr = atBlock.ATNext

			i++
		}

		datagroupAddress := hdBlock.HDDGFirst
		datagroupArray := make([]blocks.DGBlock, 0)

		i = 0
		for datagroupAddress != 0 {
			if (datagroupAddress + DG_BLOCK_SIZE) > fileSize {
				fmt.Println("File history address", datagroupAddress, "is outside the file size", fileSize)
				break
			}
			dgBlock := blocks.DGBlock{}
			dgBlock.NewBlock(file, datagroupAddress)
			recordIDNr := dgBlock.RecIDSize

			// go to first channel group of the current data group
			chanelgroupAddress := dgBlock.CGNext
			firstCGAddress := chanelgroupAddress

			cgNR := 0
			cgSize := make(map[uint64]uint32, 0)
			dgCount := 0
			currentCgIndex := 0
			for chanelgroupAddress != 0 {
				if (chanelgroupAddress + CG_BLOCK_SIZE) > fileSize {
					fmt.Println("File history address", chanelgroupAddress, "is outside the file size", fileSize)
					break
				}
				cgNR += 1
				//if chanelgroupAddress == firstCGAddress {
				grp := blocks.Group{
					DataGroup:               &dgBlock,
					Channels:                []uint64{},
					ChannelDependencies:     []uint64{},
					SignalData:              []uint64{},
					Record:                  0,
					Trigger:                 0,
					StringDtypes:            0,
					DataBlocks:              []uint64{},
					SingleChannelDtype:      0,
					UsesId:                  false,
					ReadSplitCount:          0,
					DataBlocksInfoGenerator: []uint64{},
					ChannelGroup:            blocks.CGBlock{},
					RecordSize:              map[uint64]uint32{},
					Sorted:                  false,
				}

				//}
				fmt.Println(recordIDNr, firstCGAddress, cgSize, datagroupArray)

				channelBlock := blocks.CGBlock{}
				channelBlock.ChannelBlock(file, chanelgroupAddress)

				cgMAP[chanelgroupAddress] = dgCount

				grp.ChannelGroup = channelBlock
				channelGroup := grp.ChannelGroup
				fmt.Println(channelGroup)
				grp.RecordSize = cgSize

				if channelGroup.Flags&1 != 0 {
					// VLDS flag
					recordID := channelGroup.RecordId
					cgSize[recordID] = 0
				} else {
					// In case no `cg_flags` are set
					samplesSize := channelGroup.DataBytes
					invalSize := channelGroup.InvalBytes
					recordID := channelGroup.RecordId
					cgSize[recordID] = samplesSize + invalSize
				}

				if recordIDNr != 0 {
					grp.Sorted = false
				} else {
					grp.Sorted = true
				}

				channelAddress := channelGroup.CNNext
				chCount := 0

				fmt.Println(channelAddress, chCount)

				readChannels(channelAddress ,grp ,dgCount, chCount, fileSize)
			
				chanelgroupAddress = channelGroup.CGNext
				
				dgCount += 1
				currentCgIndex += 1
				fmt.Print(chanelgroupAddress)
				break
			}
			//break
		}

	}

}


func readChannels(channelAddress uint64,grp blocks.Group,dgCount int, chCount int, fileSize int64) {
	//channels := grp.Channels
	//dependencies := grp.ChannelDependencies

	for channelAddress != 0 {

		if (channelAddress + COMMON_SIZE) > uint64(fileSize) {
			fmt.Println("Channel address is outside the file size ")
                break
		}
             
	}
}