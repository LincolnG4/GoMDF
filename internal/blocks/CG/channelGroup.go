package CG

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
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

func New(file *os.File, version uint16, startAdress int64) *Block {
	var b Block
	var err error

	b.Header = blocks.Header{}

	b.Header, err = blocks.GetHeader(file, startAdress, blocks.CgID)
	if err != nil {
		return b.BlankBlock()
	}

	//Calculates size of Link Block
	blockSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	buffEach := make([]byte, blockSize)

	// Read the Link section from the binary file
	if err := binary.Read(file, binary.LittleEndian, &buffEach); err != nil {
		fmt.Println("Error reading Link section:", err)
	}

	// Populate the Link fields dynamically based on version
	linkFields := []int64{}
	for i := 0; i < len(buffEach)/8; i++ {
		linkFields = append(linkFields, int64(binary.LittleEndian.Uint64(buffEach[i*8:(i+1)*8])))
	}

	b.Link = Link{
		Next:        linkFields[0],
		CnFirst:     linkFields[1],
		TxAcqName:   linkFields[2],
		SiAcqSource: linkFields[3],
		SrFirst:     linkFields[4],
		MdComment:   linkFields[5],
	}

	if version >= blocks.Version420 {
		linkFields = append(linkFields, blocks.ReadInt64FromBinary(file))
		b.Link.CgMaster = linkFields[6]
	}

	fmt.Printf("%+v\n", b.Link)

	//Calculates size of Data Block
	blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	buf := blocks.LoadBuffer(file, blockSize)

	// Create a buffer based on block size
	if err := binary.Read(buf, binary.LittleEndian, &b.Data); err != nil {
		fmt.Println("ERROR ddd", err)
	}

	fmt.Printf("%+v\n", b.Data)

	return &b
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
