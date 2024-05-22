package SR

import (
	"encoding/binary"
	"os"

	"github.com/LincolnG4/GoMDF/blocks"
)

type Block struct {
	Header blocks.Header
	Link   Link
	Data   Data
}

type Link struct {
	// Pointer to next SRBLOCK
	Next int64

	// Pointer to sample reduction records or data list block
	Data int64
}

type Data struct {
	// Number of cycles
	CycleCount uint64

	// Length of sample interval
	Interval float64

	// Time == 1, Angle == 2, Distance == 3, Index == 4
	SyncType uint8

	// Valid for MDF 4.2.0
	Flags    uint8
	Reserved [6]byte
}

func New(file *os.File, version uint16, startAdress int64) (*Block, error) {
	var b Block
	var err error

	b.Header = blocks.Header{}

	b.Header, err = blocks.GetHeader(file, startAdress, blocks.SrID)
	if err != nil {
		return b.BlankBlock(), err
	}

	//Calculates size of Link Block
	blockSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	b.Link = Link{}
	buf := blocks.LoadBuffer(file, blockSize)

	//Create a buffer based on blocksize
	BinaryError := binary.Read(buf, binary.LittleEndian, &b.Link)
	if BinaryError != nil {
		return b.BlankBlock(), BinaryError
	}

	//Calculates size of Data Block
	blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	b.Data = Data{}
	buf = blocks.LoadBuffer(file, blockSize)

	//Create a buffer based on blocksize
	BinaryError = binary.Read(buf, binary.LittleEndian, &b.Data)
	if BinaryError != nil {
		return b.BlankBlock(), BinaryError
	}

	return &b, nil
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.SrID),
			Reserved:  [4]byte{},
			Length:    blocks.FhblockSize,
			LinkCount: 2,
		},
		Link: Link{},
		Data: Data{},
	}
}
