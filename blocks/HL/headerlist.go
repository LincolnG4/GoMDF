package HL

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
	//Pointer first DLBlock
	DlFirst int64
}

type Data struct {
	//bit 0 set equals Equal length
	//
	//bit 1 set equals Time values
	//
	//bit 2 set equals Angle values
	//
	//bit 3 set equals Distance values
	Flags uint16

	//Zip algorithm
	ZipType uint8

	Reserved [5]byte
}

func New(file *os.File, startAddress int64) (*Block, error) {
	var b Block
	var err error

	b.Header = blocks.Header{}

	// Load Header
	b.Header, err = blocks.GetHeader(file, startAddress, blocks.HlID)
	if err != nil {
		return b.BlankBlock(), err // Early return on error
	}
	b.Link = Link{}

	// Read the Link block directly into b.Link
	if err := binary.Read(file, binary.LittleEndian, &b.Link); err != nil {
		fmt.Println("error reading link section dgblock:", err)
		return b.BlankBlock(), err
	}

	b.Data = Data{}

	// Read the Data block directly into b.Data
	if err := binary.Read(file, binary.LittleEndian, &b.Data); err != nil {
		fmt.Println("error reading data section dgblock:", err)
		return b.BlankBlock(), err
	}

	return &b, nil
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.HlID),
			Reserved:  [4]byte{},
			Length:    24,
			LinkCount: 0,
		},
	}
}
