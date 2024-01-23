package EV

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/blocks/MD"
	"github.com/LincolnG4/GoMDF/internal/blocks/TX"
)

type Block struct {
	Header blocks.Header
	Link   Link
	Data   Data
}

type Link struct {
	Next        int64
	Parent      int64
	Range       int64
	TxName      int64
	MdComment   int64
	Scope       []int64
	ATReference []int64
	//Version 4.2
	TxGroupName int64
}

type Data struct {
	Type      uint8
	SyncType  uint8
	RangeType uint8
	Cause     uint8
	//Version 4.2
	Flags           uint8
	Reserved1       [3]byte
	ScopeCount      uint32
	AttachmentCount uint16
	CreatorIndex    uint16
	SyncBaseValue   int64
	SyncFactor      float64
}

type Event struct {
	Name    string
	Comment string
	block   *Block
}

func (b *Block) getID() [4]byte {
	return b.Header.ID
}

func (b *Block) getLinkCount() uint64 {
	return b.Header.LinkCount
}

// Creates a new Block struct and initializes it by reading data from
// the provided file.
func New(file *os.File, version uint16, startAdress int64) (*Block, error) {
	var b Block

	blockID := blocks.EvID
	b.Header = blocks.Header{}
	_, err := file.Seek(startAdress, 0)
	if err != nil {
		if err != io.EOF {
			return b.BlankBlock(), fmt.Errorf("failed to seek to memory address: %v", err)
		}
	}

	// Read the header block.
	err = b.readHeader(file)
	if err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading header: %v", err)
	}

	// Check if it is a valid block ID.
	id := b.getID()
	if string(id[:]) != blockID {
		return b.BlankBlock(), fmt.Errorf("invalid block ID: expected %s, got %s", blockID, id)
	}

	// Read the link block.
	linkBytes, err := b.readLink(file)
	if err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading link section: %v", err)
	}

	// Read the data block.
	err = b.readData(file)
	if err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading data section: %v", err)
	}

	// Extract data from the link block.
	linkFields := []int64{}
	for i := 0; i < len(linkBytes)/blocks.Byte; i++ {
		linkFields = append(linkFields, int64(binary.LittleEndian.Uint64(linkBytes[i*blocks.Byte:(i+1)*blocks.Byte])))
	}

	b.Link = Link{
		Next:      linkFields[0],
		Parent:    linkFields[1],
		Range:     linkFields[2],
		TxName:    linkFields[3],
		MdComment: linkFields[4],
	}
	if b.Data.ScopeCount != 0 {
		endIdx := 5 + b.Data.ScopeCount
		b.Link.Scope = linkFields[5:endIdx]
	}
	if b.Data.AttachmentCount != 0 {
		startIdx := 5 + b.Data.ScopeCount
		endIdx := 5 + int(b.Data.ScopeCount) + int(b.Data.AttachmentCount)
		b.Link.ATReference = linkFields[startIdx:endIdx]
	}
	if b.Data.Flags == 1 && version >= blocks.Version420 {
		linkFields = append(linkFields, blocks.ReadInt64FromBinary(file))
		b.Link.TxGroupName = linkFields[len(linkFields)-1]
	}

	return &b, nil
}

func (b *Block) readHeader(file *os.File) error {
	blockSize := blocks.HeaderSize
	buf := blocks.LoadBuffer(file, blockSize)
	err := binary.Read(buf, binary.LittleEndian, &b.Header)
	if err != nil {
		return fmt.Errorf("failed to read header: %v", err)
	}
	return nil
}

func (b *Block) readLink(file *os.File) ([]byte, error) {
	linkCount := b.getLinkCount()
	blockSize := blocks.CalculateLinkSize(linkCount)
	buffEach := make([]byte, blockSize)
	if err := binary.Read(file, binary.LittleEndian, &buffEach); err != nil {
		return nil, fmt.Errorf("error reading link section: %v", err)
	}
	return buffEach, nil
}

func (b *Block) readData(file *os.File) error {
	blockSize := blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	buf := blocks.LoadBuffer(file, blockSize)
	// Create a buffer based on block size
	if err := binary.Read(buf, binary.LittleEndian, &b.Data); err != nil {
		return fmt.Errorf("error reading data section: %v", err)
	}
	return nil
}

func (b *Block) LoadEvent(f *os.File) *Event {
	var n, c string
	var err error

	if b.Link.TxName != 0 {
		n, err = TX.GetText(f, b.Link.TxName)
		if err != nil {
			n = ""
		}
	}

	if b.Link.MdComment != 0 {
		c = MD.New(f, b.Link.MdComment)
	}

	return &Event{
		Name:    n,
		Comment: c,
		block:   b,
	}
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'E', 'V'},
			Reserved:  [4]byte{},
			Length:    blocks.EvblockSize,
			LinkCount: 0,
		},
		Link: Link{},
		Data: Data{},
	}
}

func (b *Block) Next() int64 {
	return b.Link.Next
}
