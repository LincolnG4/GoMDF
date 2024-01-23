package DT

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

type Block struct {
	Header blocks.Header
	Data   []byte
}

func New(file *os.File, startAdress int64) *Block {
	var blockSize uint64 = blocks.HeaderSize
	var b Block

	_, errs := file.Seek(startAdress, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs, "memory addr out of size")
		}
	}

	b.Header = blocks.Header{}

	//Create a buffer based on blocksize
	buf := blocks.LoadBuffer(file, blockSize)

	//Read header
	BinaryError := binary.Read(buf, binary.LittleEndian, &b.Header)
	if BinaryError != nil {
		fmt.Println("error", BinaryError)
		b.BlankBlock()
	}

	return &b
}

func (b *Block) DataBlockType() string {
	return string(b.Header.ID[:])
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.DtID),
			Reserved:  [4]byte{},
			Length:    64,
			LinkCount: 4,
		},
		Data: []byte{},
	}
}
