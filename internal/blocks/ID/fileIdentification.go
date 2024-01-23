package ID

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type Block struct {
	File                  [8]byte
	Version               [8]byte
	Program               [8]byte
	Reserved1             [4]byte
	VersionNumber         uint16
	Reserved2             [30]byte
	UnfinalizedFlag       uint16
	CustomUnfinalizedFlag uint16
}

func New(file *os.File, startAdress int64) *Block {
	var b Block

	_, errs := file.Seek(startAdress, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs, "memory addr out of size")
		}

	}

	err := binary.Read(file, binary.LittleEndian, &b)
	if err != nil {
		fmt.Println("binary.Read failed:", err)
		return b.BlankBlock()
	}

	return &b
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		File:                  [8]byte{'M', 'D', 'F', ' ', ' ', ' ', ' ', ' '},
		Version:               [8]byte{'4', '.', '0', '0', ' ', ' ', ' ', ' '},
		Program:               [8]byte{'G', 'o', 'M', 'D', 'F', '1', '.', '0'},
		Reserved1:             [4]byte{},
		VersionNumber:         400,
		Reserved2:             [30]byte{},
		UnfinalizedFlag:       0,
		CustomUnfinalizedFlag: 0,
	}
}
