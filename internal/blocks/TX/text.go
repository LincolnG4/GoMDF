package TX

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

type Data struct {
	TxData []byte
}

type Block struct {
	Header blocks.Header
	Data   Data
}

const blockID string = blocks.TxID

func New(file *os.File, startAdress int64) *Block {
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

	if string(b.Header.ID[:]) != blockID {
		fmt.Printf("ERROR NOT %s", blockID)
	}


	blockSize = b.Header.Length-uint64(blocks.HeaderSize)

	b.Data = Data{}
	buff := make([]byte, blockSize)

	b.Data.TxData = blocks.GetText(file, startAdress, buff, true)
	return &b
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'T', 'X'},
			Reserved:  [4]byte{},
			Length:    64,
			LinkCount: 4,
		},
		Data: Data{
			TxData: []byte{},
		},
	}
}
