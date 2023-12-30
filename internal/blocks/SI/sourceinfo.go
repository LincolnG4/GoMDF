package SI

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/blocks/TX"
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

type SourceInfo struct {
	Name    string
	Path    string
	Comment string
	Type    string
	BusType string
	Flag    string
}

func New(file *os.File, version uint16, startAdress int64) *Block {
	var blockSize uint64 = blocks.HeaderSize
	var b Block

	b.Header = blocks.Header{}

	_, errs := file.Seek(startAdress, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs, "Memory Addr out of size")
		}
	}

	//Create a buffer based on blocksize
	buf := blocks.LoadBuffer(file, blockSize)

	//Read header
	BinaryError := binary.Read(buf, binary.LittleEndian, &b.Header)
	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

	if string(b.Header.ID[:]) != blocks.SiID {
		fmt.Printf("ERROR NOT %s", blocks.SiID)
		return b.BlankBlock()
	}

	fmt.Printf("\n%s\n", b.Header.ID)
	fmt.Printf("%+v\n", b.Header)

	//Calculates size of Link Block
	blockSize = blocks.CalculateLinkSize(b.Header.LinkCount)
	b.Link = Link{}
	buf = blocks.LoadBuffer(file, blockSize)

	//Create a buffer based on blocksize
	BinaryError = binary.Read(buf, binary.LittleEndian, &b.Link)
	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
	}

	fmt.Printf("%+v\n", b.Link)

	//Calculates size of Data Block
	blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	b.Data = Data{}
	buf = blocks.LoadBuffer(file, blockSize)

	//Create a buffer based on blocksize
	BinaryError = binary.Read(buf, binary.LittleEndian, &b.Data)
	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
	}

	fmt.Printf("%+v\n", b.Data)

	return &b
}

// GetPath returns human readable string containing additional
// information about the source
func (b *Block) GetPath(file *os.File) string {
	if b.Link.TxPath == 0 {
		return ""
	}
	return TX.GetText(file, b.Link.TxPath)
}

func (b *Block) GetName(file *os.File) string {
	if b.Link.TxName == 0 {
		return ""
	}
	return TX.GetText(file, b.Link.TxName)
}

func (b *Block) GetComment(file *os.File) string {
	if b.Link.TxName == 0 {
		return ""
	}
	return TX.GetText(file, b.Link.TxName)
}

func GetSourceInfo(file *os.File, version uint16, address int64) SourceInfo {
	b := New(file, version, address)
	return SourceInfo{
		Name:    b.GetName(file),
		Path:    b.GetPath(file),
		Comment: b.GetComment(file),
		Type:    b.GetType(),
		BusType: b.GetBusType(),
		Flag:    b.GetFlag(),
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
func (b *Block) GetType() string {
	i := b.getDataType()
	if i == 0 {
		return ""
	}

	return blocks.SourceTypeMap[i]
}

// GetType returns classification of used bus
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
func (b *Block) GetBusType() string {
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
func (b *Block) GetFlag() string {
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
			ID:        [4]byte{'#', '#', 'S', 'I'},
			Reserved:  [4]byte{},
			Length:    blocks.FhblockSize,
			LinkCount: 2,
		},
		Link: Link{},
		Data: Data{},
	}
}
