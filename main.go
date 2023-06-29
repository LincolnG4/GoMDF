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
			cgSize := make([]int, 0)

			for chanelgroupAddress != 0 {
				if (chanelgroupAddress + CG_BLOCK_SIZE) > fileSize {
					fmt.Println("File history address", chanelgroupAddress, "is outside the file size", fileSize)
					break
				}
				cgNR += 1
				//if cg_addr == firstCGAddress {
				//	grp = Group(group)
				//} else {
				//	grp = Group(group.copy())
				//}
				fmt.Println(recordIDNr, firstCGAddress, cgSize, datagroupArray)
				break
			}
			break
		}

	}

}

func seekBinaryByAddress(file *os.File, address int64, block_size int) []byte {
	buf := make([]byte, block_size)
	_, errs := file.Seek(address, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs)
		}

	}
	_, err := file.Read(buf)
	if err != nil {
		if err != io.EOF {
			fmt.Println(err)
		}

	}
	return buf
}
