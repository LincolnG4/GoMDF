package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func main() {

	file, err := os.Open("samples/sample1.mf4")

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

		fmt.Printf("%+v\n", hdBlock)
		fmt.Printf("%d \n", hdBlock.HDFHFirst)

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
	HDDGFirst   uint64
	HDFHFirst   uint64
	HDCHFirst   uint64
	HDATFirst   uint64
	HDEVFirst   uint64
	HDMDComment uint64
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

func (idBlock *IDBlock) init(file *os.File) {

	var ADDRESS int64 = 0
	IDBLOCK_SIZE := 64

	bytesValue := seekBinaryByAddress(file, ADDRESS, IDBLOCK_SIZE)
	buffer := bytes.NewBuffer(bytesValue)
	BinaryError := binary.Read(buffer, binary.LittleEndian, &(*idBlock))

	fmt.Println(string(bytesValue))

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
	BinaryError := binary.Read(buffer, binary.LittleEndian, &(*hdBlock))
	fmt.Println(string(bytesValue))

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
