package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func main() {
	const (
		FH_BLOCK_SIZE = 56
		AT_BLOCK_SIZE = 96
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
		fmt.Println(fileHistoryAddr)
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
		fmt.Println(attachmentAddr)
		attachmentArray := make([]ATBlock, 0)
		fmt.Println("READING ATTACHMENTS")
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

	}

}

type IDBlock struct {
	IDFile          [8]byte
	IDVersion       [8]byte
	IDProgram       [8]byte
	IDReserved1     [4]byte
	IDVersionNumber uint16
	IDReserved2     [34]byte
}

type HDBlock struct {
	ID          [4]byte
	Reserved    [4]byte
	Length      uint64
	LinkCount   uint64
	HDDGFirst   int64
	HDFHFirst   int64
	HDCHFirst   int64
	HDATFirst   int64
	HDEVFirst   int64
	HDMDComment int64
	StartTime   uint64
	TZOffset    int16
	DSTOffset   int16
	TimeFlags   uint8
	TimeClass   uint8
	Flags       uint8
	Reserved2   uint8
	StartAngle  float32
	StartDist   float32
	Reserved3   byte
}

type FHBlock struct {
	ID             [4]byte
	Reserved       [4]byte
	Length         uint64
	LinkCount      uint64
	FHNext         int64
	MDComment      uint64
	FHTimeNS       uint64
	FHTZOffsetMin  int16
	FHDSTOffsetMin int16
	FHTimeFlags    uint8
	FHReserved     [3]byte
}

type ATBlock struct {
	ID           [4]byte
	Reserved     [4]byte
	Length       uint64
	LinkCount    uint64
	ATNext       int64
	TXFilename   uint64
	TXMimetype   uint64
	MDComment    uint16
	Flags        uint16
	CreatorIndex uint16
	ATReserved   [4]byte
	MD5Checksum  [16]byte
	OriginalSize uint64
	EmbeddedSize uint64
	EmbeddedData []byte
}

func (idBlock *IDBlock) init(file *os.File) {

	var ADDRESS int64 = 0
	IDBLOCK_SIZE := 64

	bytesValue := seekBinaryByAddress(file, ADDRESS, IDBLOCK_SIZE)
	buffer := bytes.NewBuffer(bytesValue)
	BinaryError := binary.Read(buffer, binary.LittleEndian, idBlock)

	//fmt.Println(string(bytesValue))

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		copy(idBlock.IDFile[:], []byte("MDF     "))
		copy(idBlock.IDVersion[:], []byte("4.00    "))
		copy(idBlock.IDProgram[:], []byte("GoMDF1.0"))
		copy(idBlock.IDReserved1[:], bytes.Repeat([]byte{0}, 4))
		idBlock.IDVersionNumber = 400
		copy(idBlock.IDReserved2[:], bytes.Repeat([]byte{0}, 34))
	}
}

func (hdBlock *HDBlock) init(file *os.File) {
	var ADDRESS int64 = 64
	HDBLOCK_SIZE := 104

	bytesValue := seekBinaryByAddress(file, ADDRESS, HDBLOCK_SIZE)
	buffer := bytes.NewBuffer(bytesValue)
	BinaryError := binary.Read(buffer, binary.LittleEndian, hdBlock)
	//fmt.Println(string(bytesValue))

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		copy(hdBlock.ID[:], []byte("##HD"))
		hdBlock.Length = 104
		hdBlock.LinkCount = 6
		hdBlock.HDDGFirst = 0
		hdBlock.HDFHFirst = 0
		hdBlock.HDCHFirst = 0
		hdBlock.HDATFirst = 0
		hdBlock.HDEVFirst = 0
		hdBlock.HDMDComment = 0
		hdBlock.StartTime = 0
		hdBlock.TZOffset = 0
		hdBlock.DSTOffset = 0
		hdBlock.TimeFlags = 0
		hdBlock.TimeClass = 0
		hdBlock.Flags = 0
		hdBlock.Reserved2 = 0
		hdBlock.StartAngle = 0
		hdBlock.StartDist = 0
		hdBlock.Reserved3 = 0
	}

}

func (fhBlock *FHBlock) historyBlock(file *os.File, address int64) {

	FHBLOCK_SIZE := 56

	bytesValue := seekBinaryByAddress(file, address, FHBLOCK_SIZE)
	buffer := bytes.NewBuffer(bytesValue)
	BinaryError := binary.Read(buffer, binary.LittleEndian, fhBlock)
	fmt.Println(string(bytesValue))
	fmt.Printf("%+v\n", fhBlock)

	if string(fhBlock.ID[:]) != "##FH" {
		fmt.Println("ERROR NOT FH")
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		copy(fhBlock.ID[:], []byte("##FH"))
		copy(fhBlock.Reserved[:], bytes.Repeat([]byte{0}, 4))
		fhBlock.Length = 56
		fhBlock.LinkCount = 2
		fhBlock.FHNext = 0
		fhBlock.MDComment = 0
		fhBlock.FHTimeNS = 0
		fhBlock.FHTZOffsetMin = 0
		fhBlock.FHTimeFlags = 0
		fhBlock.FHDSTOffsetMin = 0
		copy(fhBlock.FHReserved[:], bytes.Repeat([]byte{0}, 3))
	}

}

func (atBlock *ATBlock) attchmentBlock(file *os.File, address int64) {

	BLOCK_SIZE := 96

	bytesValue := seekBinaryByAddress(file, address, BLOCK_SIZE)
	buffer := bytes.NewBuffer(bytesValue)
	fmt.Println(string(bytesValue))
	BinaryError := binary.Read(buffer, binary.LittleEndian, atBlock)
	fmt.Println(string(bytesValue))
	fmt.Printf("%+v\n", atBlock)

	if string(atBlock.ID[:]) != "##AT" {
		fmt.Println("ERROR NOT AT")
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		copy(atBlock.ID[:], []byte("##AT"))
		copy(atBlock.Reserved[:], bytes.Repeat([]byte{0}, 4))
		atBlock.Length = 96
		atBlock.LinkCount = 2
		atBlock.ATNext = 0

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
