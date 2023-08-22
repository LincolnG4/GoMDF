package CG

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
	Next        int64
	CnNext      int64
	TxAcqName   int64
	SiAcqSource uint64
	SrFirst     uint64
	MDComment   uint64
}

type Data struct {
	RecordId   uint64
	CycleCount uint64
	Flags      uint16
	Reserved1  [6]byte
	DataBytes  uint32
	InvalBytes uint32
}

func (b *Block) New(file *os.File, startAdress int64, BLOCK_SIZE int) {
	//Read Header Section
	b.Header = &blocks.Header{}
	buffer := blocks.NewBuffer(file, startAdress, blocks.HeaderSize)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	fmt.Println(string(b.Header.ID[:]))
	if string(b.Header.ID[:]) != blocks.CgID {
		fmt.Printf("ERROR NOT %s", blocks.CgID)
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
}

func (b *Block) BlankBlock() Block {
	return Block{
		Header: &blocks.Header{
			ID:        [4]byte{'#', '#', 'C', 'G'},
			Reserved:  [4]byte{},
			Length:    0,
			LinkCount: 0,
		},
		Link: &Link{
			Next:        0,
			CnNext:      0,
			TxAcqName:   0,
			SiAcqSource: 0,
			SrFirst:     0,
			MDComment:   0,
		},
		Data: &Data{
			RecordId:   0,
			CycleCount: 0,
			Flags:      0,
			Reserved1:  [6]byte{},
			DataBytes:  0,
			InvalBytes: 0,
		},
	}
}
