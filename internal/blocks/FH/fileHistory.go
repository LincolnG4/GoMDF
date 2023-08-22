package FH

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
	MDComment uint64
}

type Data struct {
	TimeNS       uint64
	TZOffsetMin  int16
	DSTOffsetMin int16
	TimeFlags    uint8
	Reserved     [3]byte
}

func (b *Block) New(file *os.File, startAdress int64, BLOCK_SIZE int) {

	//Read Header Section
	b.Header = &blocks.Header{}
	buffer := blocks.NewBuffer(file, startAdress, blocks.HeaderSize)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	if string(b.Header.ID[:]) != blocks.FhID {
		fmt.Printf("ERROR NOT %s", blocks.FhID)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

	//Read Link Section
	linkAddress := startAdress + blocks.HeaderSize
	linkSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	b.Link = &Link{}
	buffer = blocks.NewBuffer(file, linkAddress, linkSize)
	BinaryError = binary.Read(buffer, binary.LittleEndian, b.Link)

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
	}

	//Read Data Section
	dataAddress := linkAddress + int64(linkSize)
	dataSize := blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)

	b.Data = &Data{}
	buffer = blocks.NewBuffer(file, dataAddress, dataSize)
	BinaryError = binary.Read(buffer, binary.LittleEndian, b.Data)

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
	}

	fmt.Println(&b.Header)
}

func (b *Block) BlankBlock() Block {
	return Block{
		Header: &blocks.Header{
			ID:        [4]byte{'#', '#', 'F', 'H'},
			Reserved:  [4]byte{},
			Length:    56,
			LinkCount: 2,
		},
		Link: &Link{},
		Data: &Data{},
	}
}
