package DG

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
	Next      int64
	CGNext    int64
	Data      int64
	MDComment int64
}

type Data struct {
	RecIDSize uint8
	Reserved  [7]byte
}

func (b *Block) New(file *os.File, startAdress int64, BLOCK_SIZE int) {
	//Read Header Section
	b.Header = &blocks.Header{}
	buffer := blocks.NewBuffer(file, startAdress, blocks.HeaderSize)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	if string(b.Header.ID[:]) != blocks.DgID {
		fmt.Printf("ERROR NOT %s", blocks.DgID)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

	//Read Link Section
	linkAddress := startAdress + blocks.HeaderSize
	linkSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	b.Link = &Link{}
	buffer = blocks.NewBuffer(file, linkAddress, int(linkSize))
	BinaryError = binary.Read(buffer, binary.LittleEndian, b.Link)

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
	}

	//Read Data Section
	dataAddress := linkAddress + int64(linkSize)
	dataSize := blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)

	b.Data = &Data{}
	buffer = blocks.NewBuffer(file, dataAddress, int(dataSize))
	BinaryError = binary.Read(buffer, binary.LittleEndian, b.Data)

	if BinaryError != nil {
		fmt.Println("ERROR ", BinaryError)
	}

}

func (b *Block) BlankBlock() Block {
	return Block{
		Header: &blocks.Header{
			ID:        [4]byte{'#', '#', 'D', 'G'},
			Reserved:  [4]byte{},
			Length:    64,
			LinkCount: 4,
		},
		Link: &Link{},
		Data: &Data{},
	}
}
