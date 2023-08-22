package HD

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
	DGFirst   int64
	FHFirst   int64
	CHFirst   int64
	ATFirst   int64
	EVFirst   int64
	MDComment int64
}

type Data struct {
	StartTime  uint64
	TZOffset   int16
	DSTOffset  int16
	TimeFlags  uint8
	TimeClass  uint8
	Flags      uint8
	Reserved2  uint8
	StartAngle float32
	StartDist  float32
}

func (b *Block) New(file *os.File, startAdress int64, BLOCK_SIZE int) {
	//Read Header Section
	b.Header = &blocks.Header{}
	buffer := blocks.NewBuffer(file, startAdress, blocks.HeaderSize)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	if string(b.Header.ID[:]) != blocks.HdID {
		fmt.Printf("ERROR NOT %s", blocks.HdID)
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
			ID:        [4]byte{'#', '#', 'H', 'D'},
			Length:    104,
			LinkCount: 6,
		},
		Link: &Link{
			DGFirst:   0,
			FHFirst:   0,
			CHFirst:   0,
			ATFirst:   0,
			EVFirst:   0,
			MDComment: 0,
		},
		Data: &Data{
			StartTime:  0,
			TZOffset:   0,
			DSTOffset:  0,
			TimeFlags:  0,
			TimeClass:  0,
			Flags:      0,
			Reserved2:  0,
			StartAngle: 0,
			StartDist:  0,
		},
	}

}
