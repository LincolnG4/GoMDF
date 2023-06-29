package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

type DGBlock struct {
	ID         [4]byte
	Reserved   [4]byte
	Length     uint64
	LinkCount  uint64
	DGNext     int64
	CGNext     int64
	DATA       uint64
	MDComment  uint16
	RecIDSize  uint8
	DGReserved [7]byte
}

func (dgBlock *DGBlock) dataBlock(file *os.File, address int64) {

	BLOCK_SIZE := 64

	bytesValue := seekBinaryByAddress(file, address, BLOCK_SIZE)
	buffer := bytes.NewBuffer(bytesValue)

	BinaryError := binary.Read(buffer, binary.LittleEndian, dgBlock)

	fmt.Println(string(bytesValue))
	fmt.Printf("%+v\n", dgBlock)

	if string(dgBlock.ID[:]) != "##DG" {
		fmt.Println("ERROR NOT AT")
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		copy(dgBlock.ID[:], []byte("##DG"))
		copy(dgBlock.Reserved[:], bytes.Repeat([]byte{0}, 4))
		dgBlock.Length = 64
		dgBlock.LinkCount = 4
		dgBlock.DGNext = 0
		dgBlock.CGNext = 0
		dgBlock.DATA = 0
		dgBlock.MDComment = 0
		dgBlock.RecIDSize = 0
		copy(dgBlock.DGReserved[:], bytes.Repeat([]byte{0}, 7))
	}

}
