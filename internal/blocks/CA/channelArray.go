package CA

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

type Block struct {
	Header blocks.Header
	Link   Link
	Data   Data
}

type Link struct {
	Composition       int64
	Data              []int64
	DynamicSize       []int64
	InputQuality      []int64
	OutputQuality     []int64
	ComparisonQuality []int64
	CcAxisConvertion  []int64
	Axis              []int64
}

type Data struct {
	Type            uint8
	Storage         uint8
	Ndim            uint16
	Flags           uint32
	ByteOffsetBase  int32
	InvalBitposBase uint32
	DimSize         []uint64
	AxisValue       []float64
	CycleCount      []uint64
}

const blockID string = blocks.CaID

func New(file *os.File, startAdress int64) *Block {
	var blockSize uint64 = blocks.HeaderSize
	var b Block

	_, errs := file.Seek(startAdress, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs, "Memory Addr out of size")
		}
	}

	//Read Header Section
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
	fmt.Printf("\n%s\n", b.Header.ID)
	fmt.Printf("%+v\n", b.Header)
	return &b
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'F', 'H'},
			Reserved:  [4]byte{},
			Length:    blocks.FhblockSize,
			LinkCount: 2,
		},
		Link: Link{},
		Data: Data{},
	}
}
