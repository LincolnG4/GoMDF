package ID

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

type Block struct {
	File          [8]byte
	Version       [8]byte
	Program       [8]byte
	Reserved1     [4]byte
	VersionNumber uint16
	Reserved2     [34]byte
}

func (b *Block) New(file *os.File, startAdress int64) {
	buffer := blocks.NewBuffer(file, startAdress, blocks.IdblockSize)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b)

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}
}

func (b *Block) BlankBlock() Block {
	return Block{
		File:          [8]byte{'M', 'D', 'F', ' ', ' ', ' ', ' ', ' '},
		Version:       [8]byte{'4', '.', '0', '0', ' ', ' ', ' ', ' '},
		Program:       [8]byte{'G', 'o', 'M', 'D', 'F', '1', '.', '0'},
		Reserved1:     [4]byte{},
		VersionNumber: 400,
		Reserved2:     [34]byte{},
	}
}
