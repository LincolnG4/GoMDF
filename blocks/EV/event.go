package EV

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/blocks"
	"github.com/LincolnG4/GoMDF/blocks/MD"
	"github.com/LincolnG4/GoMDF/blocks/TX"
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
	Name     string
	Comment  string
	Block    *Block
	Previous *Block
	Cause    string
}

const (
	//cause unknown or not fit into categories.
	OTHER uint8 = iota
	//event was caused by some error.
	ERROR
	//event was caused by tool-internal condition,
	TOOL
	//event was caused by a scripting command.
	SCRIPT
	//event was caused directly by user
	USER
)

func (b *Block) getID() [4]byte {
	return b.Header.ID
}

func (b *Block) getLinkCount() uint64 {
	return b.Header.LinkCount
}

// Creates a new Block struct and initializes it by reading data from
// the provided file.
func New(file *os.File, version uint16, startAddress int64) (*Block, error) {
	var b Block
	blockID := blocks.EvID

	// Seek to the start address
	if _, err := file.Seek(startAddress, io.SeekStart); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to seek to memory address %d: %w", startAddress, err)
	}

	// Read and decode the header
	if err := b.readHeader(file); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading header: %w", err)
	}

	// Validate block ID
	if id := b.getID(); string(id[:]) != blockID {
		return b.BlankBlock(), fmt.Errorf("invalid block ID: expected %s, got %s", blockID, id)
	}

	// Read the link block
	linkBytes, err := b.readLink(file)
	if err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading link section: %w", err)
	}

	// Process the link block
	linkFields := extractLinkFields(linkBytes)

	// Assign extracted data to Link
	b.Link = Link{
		Next:      linkFields[0],
		Parent:    linkFields[1],
		Range:     linkFields[2],
		TxName:    linkFields[3],
		MdComment: linkFields[4],
	}

	// Handle Scope and Attachment references
	if b.Data.ScopeCount > 0 {
		b.Link.Scope = linkFields[5 : 5+b.Data.ScopeCount]
	}
	if b.Data.AttachmentCount > 0 {
		b.Link.ATReference = linkFields[5+b.Data.ScopeCount : 5+b.Data.ScopeCount+uint32(b.Data.AttachmentCount)]
	}

	// Handle version-specific fields
	if b.Data.Flags == 1 && version >= blocks.Version420 {
		if extraField := blocks.ReadInt64FromBinary(file); err == nil {
			b.Link.TxGroupName = extraField
		} else {
			return b.BlankBlock(), fmt.Errorf("error reading TxGroupName: %w", err)
		}
	}

	return &b, nil
}

// Helper function to extract link fields from the byte slice
func extractLinkFields(linkBytes []byte) []int64 {
	numFields := len(linkBytes) / blocks.Byte
	linkFields := make([]int64, numFields)
	for i := 0; i < numFields; i++ {
		linkFields[i] = int64(binary.LittleEndian.Uint64(linkBytes[i*blocks.Byte : (i+1)*blocks.Byte]))
	}
	return linkFields
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

func (b *Block) Load(mf4File *os.File) *Event {
	var n, c string
	var err error

	if b.Link.TxName != 0 {
		n, err = TX.GetText(mf4File, b.Link.TxName)
		if err != nil {
			n = ""
		}
	}

	if b.Link.MdComment != 0 {
		c = MD.New(mf4File, b.Link.MdComment)
	}

	return &Event{
		Name:    n,
		Comment: c,
		Block:   b,
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
