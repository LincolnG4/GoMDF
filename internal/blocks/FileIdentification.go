package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type ID struct {
	File          [8]byte
	Version       [8]byte
	Program       [8]byte
	Reserved1     [4]byte
	VersionNumber uint16
	Reserved2     [34]byte
}

func (b *ID) NewBlock(file *os.File, startAdress int64, BLOCK_SIZE int) {
	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b)

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}
}

func (b *ID) BlankBlock() ID {
	return ID{
		File:          [8]byte{'M', 'D', 'F', ' ', ' ', ' ', ' ', ' '},
		Version:       [8]byte{'4', '.', '0', '0', ' ', ' ', ' ', ' '},
		Program:       [8]byte{'G', 'o', 'M', 'D', 'F', '1', '.', '0'},
		Reserved1:     [4]byte{},
		VersionNumber: 400,
		Reserved2:     [34]byte{},
	}
}
