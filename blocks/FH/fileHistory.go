package FH

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/blocks"
	"github.com/LincolnG4/GoMDF/blocks/TX"
)

type Block struct {
	Header blocks.Header
	Link   Link
	Data   Data
}

type Link struct {
	Next      int64
	MDComment int64
}

type Data struct {
	TimeNS       uint64
	TZOffsetMin  int16
	DSTOffsetMin int16
	TimeFlags    uint8
	Reserved     [3]byte
}

func New(file *os.File, startAddress int64) (*Block, error) {
	var b Block

	// Seek to the start address
	if _, err := file.Seek(startAddress, io.SeekStart); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to seek to address %d: %w", startAddress, err)
	}

	// Read and decode the header
	headerBuf := make([]byte, blocks.HeaderSize)
	if _, err := io.ReadFull(file, headerBuf); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to read header: %w", err)
	}
	if err := binary.Read(bytes.NewReader(headerBuf), binary.LittleEndian, &b.Header); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to decode header: %w", err)
	}

	// Validate block ID
	if string(b.Header.ID[:]) != blocks.FhID {
		return b.BlankBlock(), fmt.Errorf("invalid block ID: expected %s, got %s", blocks.FhID, b.Header.ID)
	}

	// Read and decode the link block
	linkSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	linkBuf := make([]byte, linkSize)
	if _, err := io.ReadFull(file, linkBuf); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to read link block: %w", err)
	}
	if err := binary.Read(bytes.NewReader(linkBuf), binary.LittleEndian, &b.Link); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to decode link block: %w", err)
	}

	// Read and decode the data block
	dataSize := blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	dataBuf := make([]byte, dataSize)
	if _, err := io.ReadFull(file, dataBuf); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to read data block: %w", err)
	}
	if err := binary.Read(bytes.NewReader(dataBuf), binary.LittleEndian, &b.Data); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to decode data block: %w", err)
	}

	return &b, nil
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

func (b *Block) GetChangeLog(file *os.File) string {
	t, err := TX.GetText(file, b.GetMdComment())
	if err != nil {
		return ""
	}

	return t
}

func (b *Block) GetMdComment() int64 {
	return b.Link.MDComment
}

func (b *Block) GetTimeNs() int64 {
	return int64(b.Data.TimeNS)
}

func (b *Block) GetTimeFlag() uint8 {
	return b.Data.TimeFlags
}

func (b *Block) Next() int64 {
	return b.Link.Next
}
