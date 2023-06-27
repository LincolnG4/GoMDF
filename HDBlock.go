package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

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
