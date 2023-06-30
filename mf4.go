package main

import (
	"fmt"
	"io"
	"os"
)

func main() {

	const (
		FH_BLOCK_SIZE = 56
		AT_BLOCK_SIZE = 96
		DG_BLOCK_SIZE = 64
		CG_BLOCK_SIZE = 104
	)

	cgMAP := make(map[int64]int, 0)

	file, err := os.Open("/home/lincolng/Downloads/sample4.mf4")
	fileInfo, _ := file.Stat()

	fileSize := fileInfo.Size()

	if err != nil {
		if err != io.EOF {
			fmt.Println(err)
		}

	}

	defer file.Close()

	//Create IDBLOCK
	idBlock := IDBlock{}
	idBlock.init(file)

	fmt.Printf("%+v\n", idBlock)

	if idBlock.IDVersionNumber > 400 {
		//Create HDBLOCK
		hdBlock := HDBlock{}
		hdBlock.init(file)

		// read file history
		fileHistoryAddr := hdBlock.HDFHFirst
		fileHistory := make([]FHBlock, 0)
		i := 0
		for fileHistoryAddr != 0 {
			if (fileHistoryAddr + FH_BLOCK_SIZE) > fileSize {
				fmt.Println("File history address", fileHistoryAddr, "is outside the file size", fileSize)
				break
			}
			fhBlock := FHBlock{}

			fhBlock.historyBlock(file, fileHistoryAddr)
			fileHistory = append(fileHistory, fhBlock)
			fileHistoryAddr = fhBlock.FHNext

			i++
		}

		// read file history
		attachmentAddr := hdBlock.HDATFirst
		attachmentArray := make([]ATBlock, 0)

		i = 0
		for attachmentAddr != 0 {
			if (attachmentAddr + AT_BLOCK_SIZE) > fileSize {
				fmt.Println("File history address", attachmentAddr, "is outside the file size", fileSize)
				break
			}
			atBlock := ATBlock{}

			atBlock.attchmentBlock(file, attachmentAddr)
			attachmentArray = append(attachmentArray, atBlock)
			attachmentAddr = atBlock.ATNext

			i++
		}

		datagroupAddress := hdBlock.HDDGFirst
		datagroupArray := make([]DGBlock, 0)

		i = 0
		for datagroupAddress != 0 {
			if (datagroupAddress + DG_BLOCK_SIZE) > fileSize {
				fmt.Println("File history address", datagroupAddress, "is outside the file size", fileSize)
				break
			}
			dgBlock := DGBlock{}
			dgBlock.dataBlock(file, datagroupAddress)
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
				grp := Group{
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
					ChannelGroup:            CGBlock{},
					RecordSize:              map[uint64]uint32{},
					Sorted:                  false,
				}

				//}
				fmt.Println(recordIDNr, firstCGAddress, cgSize, datagroupArray)

				channelBlock := CGBlock{}
				channelBlock.channelBlock(file, chanelgroupAddress)

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
				chCounter := 0

				fmt.Println(channelAddress, chCounter)
				// self._read_channels(
				//     ch_addr, grp, stream, dg_cntr, ch_cntr, mapped=mapped
				// )

				chanelgroupAddress = channelGroup.CGNext

				dgCount += 1
				currentCgIndex += 1

				break
			}
			break
		}

	}

}
