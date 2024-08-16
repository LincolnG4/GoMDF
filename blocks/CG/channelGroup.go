package CG

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
	//Pointer to next channel group block (CGBLOCK)
	Next int64

	//Pointer to first channel block (CNBLOCK)
	CnFirst int64

	//Pointer to acquisition name (TXBLOCK)
	TxAcqName int64

	//Pointer to acquisition source (SIBLOCK)
	SiAcqSource int64

	//Pointer to first sample reduction block (SRBLOCK)
	SrFirst int64

	//Pointer to comment and additional information (TXBLOCK or MDBLOCK)
	MdComment int64

	// Version 4.2
	CgMaster int64
}

type Data struct {
	RecordId   uint64
	CycleCount uint64
	Flags      uint16
	// Version 4.1
	PathSeparator uint16
	Reserved      [4]byte
	DataBytes     uint32
	InvalBytes    uint32
}

const blockID string = blocks.CgID

func New(file *os.File, version uint16, startAddress int64) (*Block, error) {
	var b Block

	// Initialize the header
	b.Header = blocks.Header{}

	// Read the header
	var err error
	b.Header, err = blocks.GetHeader(file, startAddress, blocks.CgID)
	if err != nil {
		return b.BlankBlock(), err
	}

	// Calculate size of Link Block
	linkBlockSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	linkBuffer := make([]byte, linkBlockSize)

	// Read the Link section into the buffer
	if _, err := io.ReadFull(file, linkBuffer); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading link section channelgroup: %w", err)
	}

	// Extract Link fields from the buffer
	linkFields := make([]int64, linkBlockSize/8)
	for i := range linkFields {
		linkFields[i] = int64(binary.LittleEndian.Uint64(linkBuffer[i*8 : (i+1)*8]))
	}

	// Populate the Link fields
	b.Link = Link{
		Next:        linkFields[0],
		CnFirst:     linkFields[1],
		TxAcqName:   linkFields[2],
		SiAcqSource: linkFields[3],
		SrFirst:     linkFields[4],
		MdComment:   linkFields[5],
	}

	if version >= blocks.Version420 {
		// Read additional field for versions >= 420
		masterField := blocks.ReadInt64FromBinary(file)
		b.Link.CgMaster = masterField
	}

	// Calculate size of Data Block
	dataBlockSize := blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	dataBuffer := make([]byte, dataBlockSize)

	// Read the Data section into the buffer
	if _, err := io.ReadFull(file, dataBuffer); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading data section channelgroup: %w", err)
	}

	// Populate the Data struct
	if err := binary.Read(bytes.NewReader(dataBuffer), binary.LittleEndian, &b.Data); err != nil {
		return b.BlankBlock(), fmt.Errorf("error parsing data section channelgroup: %w", err)
	}

	return &b, nil
}

func (b *Block) getFlag() uint16 {
	return b.Data.Flags
}

func (b *Block) IsVLSD() bool {
	return blocks.IsBitSet(int(b.getFlag()), 0)
}

func (b *Block) GetDataBytes() uint32 {
	return b.Data.DataBytes
}

func (b *Block) RowSize() int64 {
	return int64(b.Data.DataBytes)
}

func (b *Block) Type(version uint16) []string {
	t := []string{}
	f := int(b.Data.Flags)

	if b.IsVLSD() {
		t = append(t, blocks.VlsdEvent)
	}

	if version < blocks.Version410 {
		return t
	}

	//BUS EVENT FLAG
	if blocks.IsBitSet(f, 1) && blocks.IsBitSet(f, 2) {
		t = append(t, blocks.PlainBusEvent)
	} else if blocks.IsBitSet(f, 1) {
		t = append(t, blocks.BusEvent)
	}

	if version < 420 {
		return t
	}

	//REMOTE MASTER
	if blocks.IsBitSet(f, 3) {
		t = append(t, blocks.RemoteMaster)
	}

	//EVENT
	if blocks.IsBitSet(f, 4) {
		t = append(t, blocks.Event)
	}

	return t
}

func (b *Block) PathSeparator() string {
	return string(rune(b.Data.PathSeparator))
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.CgID),
			Reserved:  [4]byte{},
			Length:    blocks.CgblockSize,
			LinkCount: 0,
		},
		Link: Link{},
		Data: Data{},
	}
}

// Pointer to first channel block (CNBLOCK)
func (b *Block) FirstChannel() int64 {
	return b.Link.CnFirst
}

// Pointer to next channel group block (CGBLOCK)
func (b *Block) Next() int64 {
	return b.Link.Next
}
