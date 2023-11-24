package MD

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/blocks/TX"
)

type Block struct {
	Header blocks.Header
	Data   []byte
}

func New(file *os.File, startAdress int64) string {

	var blockSize uint64 = blocks.HeaderSize
	var b Block

	_, errs := file.Seek(startAdress, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs, "Memory Addr out of size")
		}
	}

	b.Header = blocks.Header{}

	//Create a buffer based on blocksize
	buf := blocks.LoadBuffer(file, blockSize)

	//Read header
	BinaryError := binary.Read(buf, binary.LittleEndian, &b.Header)
	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

	//If block is ##MD
	if string(b.Header.ID[:]) == blocks.MdID {
		blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
		buf := make([]byte, blockSize)
		b.Data = blocks.GetText(file, startAdress, buf, true)
		r := string(b.Data)
		return r
	}

	return TX.GetText(file, startAdress)
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'M', 'D'},
			Reserved:  [4]byte{},
			Length:    64,
			LinkCount: 4,
		},
		Data: []byte{},
	}
}
