package HD

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/LincolnG4/GoMDF/blocks"
)

type Block struct {
	Header blocks.Header
	Link   Link
	Data   Data
}

type Link struct {
	DgFirst   int64
	FhFirst   int64
	ChFirst   int64
	AtFirst   int64
	EvFirst   int64
	MdComment int64
}

type Data struct {
	StartTimeNs   uint64
	TZOffsetMin   int16
	DSTOffsetMin  int16
	TimeFlags     uint8
	TimeClass     uint8
	Flags         uint8
	Reserved      uint8
	StartAngleRad float64
	StartDistM    float64
}

const blockID string = blocks.HdID

// New() seek and read to Block struct based on startAddress and blockSize.
//
// The HDBLOCK always begins at file position 64. It contains general information about the
// contents of the measured data file and is the root for the block hierarchy.
func New(file *os.File, startAdress int64) *Block {
	var b Block
	var err error

	b.Header = blocks.Header{}

	b.Header, err = blocks.GetHeader(file, startAdress, blocks.HdID)
	if err != nil {
		return b.BlankBlock()
	}

	//Calculates size of Link Block
	blockSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	b.Link = Link{}
	buf := blocks.LoadBuffer(file, blockSize)

	//Create a buffer based on blocksize
	BinaryError := binary.Read(buf, binary.LittleEndian, &b.Link)
	if BinaryError != nil {
		fmt.Println("error", BinaryError)
	}

	//Calculates size of Data Block
	blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	b.Data = Data{}
	buf = blocks.LoadBuffer(file, blockSize)

	//Create a buffer based on blocksize
	BinaryError = binary.Read(buf, binary.LittleEndian, &b.Data)
	if BinaryError != nil {
		fmt.Println("error", BinaryError)
	}

	return &b
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.HdID),
			Length:    blocks.HdblockSize,
			LinkCount: 6,
		},
		Link: Link{},
		Data: Data{},
	}

}
