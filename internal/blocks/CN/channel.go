package CN

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
	Next          int64
	CnComposition uint64
	TxName        uint64
	SiSource      uint64
	CcConvertion  uint64
	Data          uint64
	MdUnit        uint64
	MdComment     uint64
}

type Data struct {
	Type        uint8
	SyncType    uint8
	DataType    uint8
	BitOffset   uint8
	ByteOffset  uint32
	BitCount    uint32
	Flags       uint32
	Precision   uint8
	Reserved1   [3]byte
	ValRangeMin float32
	ValRangeMax float32
	LimitMin    float32
	LimitExtMin uint32
	LimitExtMax float32
}

func (b *Block) New(file *os.File, startAdress int64) {
	//Read Header Section
	b.Header = &blocks.Header{}
	buffer := blocks.NewBuffer(file, startAdress, blocks.HeaderSize)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	if string(b.Header.ID[:]) != blocks.CnID {
		fmt.Printf("ERROR NOT %s", blocks.CnID)
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
	return Block{}
}

func (b *Block) GetSignalData(file *os.File, startAdress uint64, recordsize uint8, size uint64) {

}
