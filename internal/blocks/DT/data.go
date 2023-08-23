package DT

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

type Block struct {
	Header *blocks.Header
	Link   *Link
	Data   *Data
}

type Link struct {
}

type Data struct {
	Data []byte
}

func (b *Block) New(file *os.File, startAdress int64) {
	//Read Header Section
	b.Header = &blocks.Header{}
	buffer := blocks.NewBuffer(file, startAdress, blocks.HeaderSize)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	if string(b.Header.ID[:]) != blocks.DtID {
		fmt.Printf("ERROR NOT %s", blocks.DtID)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

	//Read Data Section
	dataAddress := startAdress + 24
	dataSize := int(b.Header.Length - 24)

	b.Data = &Data{}
	buffer = blocks.NewBuffer(file, dataAddress, dataSize)
	BinaryError = binary.Read(buffer, binary.LittleEndian, b.Data)

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
	}

}

func (b *Block) BlankBlock() Block {
	return Block{
		Header: &blocks.Header{
			ID:        [4]byte{'#', '#', 'D', 'T'},
			Reserved:  [4]byte{},
			Length:    64,
			LinkCount: 4,
		},
		Link: &Link{},
		Data: &Data{},
	}
}
