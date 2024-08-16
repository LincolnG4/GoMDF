package SI

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
	//Pointer to TXBLOCK that represent indentification name
	TxName int64

	//Pointer to TXBLOCK path
	TxPath int64

	//Pointer to comment and additional information (TXBLOCK or MDBLOCK)
	MdComment int64
}

type Data struct {
	//Source type classification
	Type uint8

	//Bus type classification of used bus
	BusType  uint8
	Flags    uint8
	Reserved [5]byte
}

// SourceInfo describes the source of an acquisition mode or of a signal
type SourceInfo struct {
	Name    string
	Path    string
	Comment string
	Type    string
	BusType string
	Flag    string
}

func New(file *os.File, version uint16, startAddress int64) (*Block, error) {
	var b Block

	// Seek to the start address
	if _, err := file.Seek(startAddress, io.SeekStart); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to seek to address %d: %w", startAddress, err)
	}

	// Read and decode the header
	b.Header = blocks.Header{}
	headerBuf := make([]byte, blocks.HeaderSize)
	if _, err := io.ReadFull(file, headerBuf); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to read header: %w", err)
	}
	if err := binary.Read(bytes.NewReader(headerBuf), binary.LittleEndian, &b.Header); err != nil {
		return b.BlankBlock(), fmt.Errorf("failed to decode header: %w", err)
	}

	// Validate the block ID
	if string(b.Header.ID[:]) != blocks.SiID {
		return b.BlankBlock(), fmt.Errorf("invalid block ID: expected %s, got %s", blocks.SiID, b.Header.ID)
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

// GetPath returns human readable string containing additional
// information about the source
func (b *Block) Path(file *os.File) string {
	if b.Link.TxPath == 0 {
		return ""
	}

	t, err := TX.GetText(file, b.Link.TxPath)
	if err != nil {
		return ""
	}

	return t
}

func (b *Block) Name(file *os.File) string {
	if b.Link.TxName == 0 {
		return ""
	}

	t, err := TX.GetText(file, b.Link.TxName)
	if err != nil {
		return ""
	}

	return t
}

func (b *Block) Comment(file *os.File) string {
	if b.Link.TxName == 0 {
		return ""
	}

	t, err := TX.GetText(file, b.Link.MdComment)
	if err != nil {
		return ""
	}

	return t
}

func Get(file *os.File, version uint16, address int64) SourceInfo {
	b, err := New(file, version, address)
	if err != nil {
		return SourceInfo{
			Name:    "",
			Path:    "",
			Comment: "",
			Type:    "",
			BusType: "",
			Flag:    "",
		}
	}
	return SourceInfo{
		Name:    b.Name(file),
		Path:    b.Path(file),
		Comment: b.Comment(file),
		Type:    b.Type(),
		BusType: b.BusType(),
		Flag:    b.Flag(),
	}
}

func (b *Block) getDataType() uint8 {
	return b.Data.Type
}

// GetType returns classification of source
//
// - OTHER: unknown or not fit
//
// - ECU: ECU
//
// - BUS: Bus
//
// - I/O: I/O device
//
// - TOOL: software generated
//
// - USER: user interaction/input
func (b *Block) Type() string {
	i := b.getDataType()
	if i == 0 {
		return ""
	}

	return blocks.SourceTypeMap[i]
}

// GetBusType returns classification of used bus
// (should be "NONE" for si_type ≥ 3)
//
// - NONE: no bus
//
// - OTHER: unknown or not fit
//
// - CAN,
//
// - LIN,
//
// - MOST,
//
// - FLEXRAY,
//
// - K_LINE,
//
// - ETHERNET,
//
// - USB
func (b *Block) BusType() string {
	bt := b.Data.BusType
	if b.getDataType() >= 3 {
		return blocks.BusTypeMap[0]
	}

	if bt == 0 {
		return ""
	}

	return blocks.BusTypeMap[bt]
}

// GetFlag returns if source is a simulation
func (b *Block) Flag() string {
	f := int(b.Data.Flags)
	if f == 0 {
		return ""
	}

	if b.getDataType() == 4 {
		return ""
	}

	if blocks.IsBitSet(f, 0) {
		return "SIMULATED SOURCE"
	}

	return "REAL SOURCE"
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.SiID),
			Reserved:  [4]byte{},
			Length:    blocks.FhblockSize,
			LinkCount: 2,
		},
		Link: Link{},
		Data: Data{},
	}
}
