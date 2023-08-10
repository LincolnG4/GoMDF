package blocks

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

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

func (fhBlock *FHBlock) HistoryBlock(file *os.File, address int64) {

	const FHBLOCK_SIZE = 56

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
