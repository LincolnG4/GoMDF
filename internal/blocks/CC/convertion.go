package CC

import (
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

type Block struct {
	Header blocks.Header
	Link   Link
	Data   Data
}

type Link struct {
	TxName    int64
	MdUnit    int64
	MdComment int64
	CcInverse int64
	CcRef     []int64
}

type Data struct {
	Type        uint8
	Precision   uint8
	Flags       uint16
	RefCount    uint16
	ValCount    uint16
	PhyRangeMin float64
	PhyRangeMax float64
	Val         []float64
}

const blockID string = blocks.CcID

func New(file *os.File, version uint16, startAdress int64) *Block {
	return &Block{}
}

func (b *Block) BlankBlock() *Block {
	return &Block{}
}

func (b *Block) GetSignalData(file *os.File, startAdress uint64, recordsize uint8, size uint64) {

}
