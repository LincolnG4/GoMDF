package TX

import (
	"bytes"
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

func GetText(file *os.File, startAdress int64) string {
	var blockSize uint64 = blocks.HeaderSize
	var b Block

	_, errs := file.Seek(startAdress, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs, "Memory Addr out of size")
		}
	}

	b.Header = blocks.Header{}
	buf := blocks.LoadBuffer(file, blockSize)
	BinaryError := binary.Read(buf, binary.LittleEndian, &b.Header)
	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}
	if string(b.Header.ID[:]) != blocks.TxID && string(b.Header.ID[:]) != blocks.MdID {
		fmt.Printf("ERROR NOT %s or %s", blocks.TxID, blocks.MdID)
	}

	blockSize = b.Header.Length - blockSize
	b.Data = Data{}
	buff := make([]byte, blockSize)
	t := blocks.GetText(file, startAdress, buff, true)
	result := string(bytes.Trim(t, "\x00"))
	return result
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
