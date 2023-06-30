package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

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

func (atBlock *ATBlock) attchmentBlock(file *os.File, address int64) {

	const BLOCK_SIZE = 96

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
