package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type FH struct {
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

func (b *FH) NewBlock(file *os.File, startAdress int64, BLOCK_SIZE int) {

	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b)

	if string(b.ID[:]) != "##FH" {
		fmt.Println("ERROR NOT FH")
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

}

func (b *FH) BlankBlock() FH {
	return FH{
		ID:             [4]byte{'#', '#', 'F', 'H'},
		Reserved:       [4]byte{},
		Length:         56,
		LinkCount:      2,
		FHNext:         0,
		MDComment:      0,
		FHTimeNS:       0,
		FHTZOffsetMin:  0,
		FHDSTOffsetMin: 0,
		FHTimeFlags:    0,
		FHReserved:     [3]byte{},
	}
}
