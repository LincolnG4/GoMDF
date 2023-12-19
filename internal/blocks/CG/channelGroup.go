package CG

import (
	"encoding/binary"
	"fmt"
	"io"
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

	if string(b.Header.ID[:]) != blockID {
		fmt.Printf("ERROR NOT %s", blockID)
	}

	fmt.Printf("\n%s\n", b.Header.ID)
	fmt.Printf("%+v\n", b.Header)

	//Calculates size of Link Block
	blockSize = blocks.CalculateLinkSize(b.Header.LinkCount)
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

	if version >= 420 {
		linkFields = append(linkFields, blocks.ReadInt64FromBinary(file))
		b.Link.CgMaster = linkFields[6]
	}

	fmt.Printf("%+v\n", b.Link)

	//Calculates size of Data Block
	blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	buf = blocks.LoadBuffer(file, blockSize)

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

func (b *Block) Type(version uint16) []string {
	t := []string{}
	f := int(b.Data.Flags)

	if b.IsVLSD() {
		t = append(t, "VLSD")
	}

	if version < 410 {
		return t
	}

	//BUS EVENT FLAG
	if blocks.IsBitSet(f, 1) && blocks.IsBitSet(f, 2) {
		t = append(t, "PLAIN_BUS_EVENT")
	} else if blocks.IsBitSet(f, 1) {
		t = append(t, "BUS_EVENT")
	}

	if version < 420 {
		return t
	}

	//REMOTE MASTER
	if blocks.IsBitSet(f, 3) {
		t = append(t, "REMOTE_MASTER")
	}

	//EVENT
	if blocks.IsBitSet(f, 4) {
		t = append(t, "EVENT")
	}

	return t
}

func (b *Block) PathSeparator() string {
	return string(rune(b.Data.PathSeparator))
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'C', 'G'},
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
