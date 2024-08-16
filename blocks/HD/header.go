package HD

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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
func New(file *os.File, startAddress int64) (*Block, error) {
	var b Block

	// Initialize the Header
	b.Header = blocks.Header{}

	// Get the header from the file
	var err error
	b.Header, err = blocks.GetHeader(file, startAddress, blocks.HdID)
	if err != nil {
		return b.BlankBlock(), err
	}

	// Calculate size and read the Link Block
	linkBlockSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	linkBuffer := make([]byte, linkBlockSize)
	if _, err := io.ReadFull(file, linkBuffer); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading link block: %w", err)
	}

	// Read Link Block from buffer
	linkReader := bytes.NewReader(linkBuffer)
	if err := binary.Read(linkReader, binary.LittleEndian, &b.Link); err != nil {
		return b.BlankBlock(), fmt.Errorf("error decoding link block: %w", err)
	}

	// Calculate size and read the Data Block
	dataBlockSize := blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	dataBuffer := make([]byte, dataBlockSize)
	if _, err := io.ReadFull(file, dataBuffer); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading data block: %w", err)
	}

	// Read Data Block from buffer
	dataReader := bytes.NewReader(dataBuffer)
	if err := binary.Read(dataReader, binary.LittleEndian, &b.Data); err != nil {
		return b.BlankBlock(), fmt.Errorf("error decoding data block: %w", err)
	}

	return &b, nil

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
