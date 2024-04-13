package CA

import (
	"os"

	"github.com/LincolnG4/GoMDF/blocks"
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
	ComparisonQuatity []int64
	CcAxisConvertion  []int64
	Axis              []int64
}

type Data struct {
	Type            uint8
	Storage         uint8
	Ndim            uint16
	Flags           uint32
	ByteOffsetBase  int32
	InvalBitPosBase uint32
	DimSize         []uint64
	AxisValue       []float64
	CycleCount      []uint64
}

func New(file *os.File, startAdress int64) *Block {
	var b Block
	var err error

	b.Header = blocks.Header{}

	b.Header, err = blocks.GetHeader(file, startAdress, blocks.CcID)
	if err != nil {
		return b.BlankBlock()
	}

	return &b
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.CaID),
			Reserved:  [4]byte{},
			Length:    blocks.FhblockSize,
			LinkCount: 2,
		},
		Link: Link{},
		Data: Data{},
	}
}
