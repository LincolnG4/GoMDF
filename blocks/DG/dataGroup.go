package DG

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
	Next      int64
	CgFirst   int64
	Data      int64
	MdComment int64
}

type Data struct {
	//Number of Bytes used for record IDs in the datablock.
	RecIDSize uint8
	Reserved  [7]byte
}

func New(file *os.File, startAddress int64) *Block {
	var b Block

	// Initialize header
	var err error
	b.Header = blocks.Header{}

	// Load Header
	b.Header, err = blocks.GetHeader(file, startAddress, blocks.DgID)
	if err != nil {
		return b.BlankBlock() // Early return on error
	}

	b.Link = Link{}

	// Read the Link block directly into b.Link
	if err := binary.Read(file, binary.LittleEndian, &b.Link); err != nil {
		fmt.Println("error reading link section dgblock:", err)
		return b.BlankBlock()
	}

	b.Data = Data{}

	// Read the Data block directly into b.Data
	if err := binary.Read(file, binary.LittleEndian, &b.Data); err != nil {
		fmt.Println("error reading data section dgblock:", err)
		return b.BlankBlock()
	}

	return &b
}

// BytesOfRecordIDSize returns number of Bytes used for record IDs in the data
// block.
func (b *Block) BytesOfRecordIDSize(f *os.File, buf []byte) (uint64, error) {
	switch b.RecordIDSize() {
	case 0:
		return 0, nil // Sorted record
	case 1:
		return uint64(buf[0]), nil
	case 2:
		return uint64(binary.LittleEndian.Uint16(buf[:2])), nil
	case 4:
		return uint64(binary.LittleEndian.Uint32(buf[:4])), nil
	case 8:
		return binary.LittleEndian.Uint64(buf[:8]), nil
	default:
		return 0, fmt.Errorf("invalid number of bytes for record IDs: %d", b.RecordIDSize())
	}
}

// IsSorted checks if is Sorted `True`. Else `False` if it is Unsorted
func (b *Block) IsSorted() bool {
	return b.RecordIDSize() == 0
}

func (b *Block) RecordIDSize() uint8 {
	return b.Data.RecIDSize
}

func (b *Block) MetadataComment() int64 {
	return b.Link.MdComment
}

func (b *Block) FirstChannelGroup() int64 {
	return b.Link.CgFirst
}

func (b *Block) Next() int64 {
	return b.Link.Next
}

func (b *Block) HeaderID() string {
	return string(b.Header.ID[:])
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.DgID),
			Reserved:  [4]byte{},
			Length:    blocks.DgblockSize,
			LinkCount: 4,
		},
		Link: Link{},
		Data: Data{},
	}
}
