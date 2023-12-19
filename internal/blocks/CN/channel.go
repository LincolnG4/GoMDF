package CN

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
	Next         int64
	Composition  int64
	TxName       int64
	SiSource     int64
	CcConvertion int64
	Data         int64
	MdUnit       int64
	MdComment    int64
	//Version 4.1
	AtReference int64
	//Version 4.1
	DefaultX [3]int64
}

type Data struct {
	Type        uint8
	SyncType    uint8
	DataType    uint8
	BitOffset   uint8
	ByteOffset  uint32
	BitCount    uint32
	Flags       uint32
	InvalBitPos uint32
	Precision   uint8
	// Use [1]byte for versions >= 4.1
	Reserved uint8
	//Version 4.1
	AttachmentCount uint16
	ValRangeMin     float64
	ValRangeMax     float64
	LimitMin        float64
	LimitMax        float64
	LimitExtMin     float64
	LimitExtMax     float64
}

const blockID string = blocks.CnID

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

	// Handle version-specific fields in Link based on the version
	if version >= 420 {
		linkFields = append(linkFields, blocks.ReadInt64FromBinary(file))
	}

	b.Link = Link{
		Next:         linkFields[0],
		Composition:  linkFields[1],
		TxName:       linkFields[2],
		SiSource:     linkFields[3],
		CcConvertion: linkFields[4],
		Data:         linkFields[5],
		MdUnit:       linkFields[6],
		MdComment:    linkFields[7],
	}
	fmt.Printf("%+v\n", b.Link)
	//Calculates size of Data Block
	blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	buf = blocks.LoadBuffer(file, blockSize)

	// Create a buffer based on block size
	if err := binary.Read(buf, binary.LittleEndian, &b.Data); err != nil {
		fmt.Println("ERROR", err)
	}

	if version < 410 {
		return &b
	}

	//Handling versions >= 4.10
	for i := 0; i < int(b.Data.AttachmentCount); i++ {
		b.Link.AtReference = linkFields[8]
	}

	if b.Data.Flags == 12 {
		b.Link.DefaultX = [3]int64{linkFields[9], linkFields[10], linkFields[11]}
		fmt.Println("DefaultX Flagged")
	}
	fmt.Printf("%+v\n", b.Data)
	return &b
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'C', 'N'},
			Reserved:  [4]byte{},
			Length:    blocks.CnblockSize,
			LinkCount: 0,
		},
		Link: Link{},
		Data: Data{},
	}
}

func (b *Block) GetSignalData(file *os.File, startAdress uint64, recordsize uint8, size uint64) {

}

func (b *Block) IsAllValuesInvalid() bool {
	// Bit 0 corresponds to the "all values invalid" flag
	// Check if bit 0 is set
	return blocks.IsBitSet(int(b.Data.Flags), 0)
}

func (b *Block) GetChannelName(f *os.File) string {
	return TX.GetText(f, b.getTxName())
}

func (b *Block) getTxName() int64 {
	return b.Link.TxName
}

func (b *Block) GetCommentMd() int64 {
	return b.Link.MdComment
}

func (b *Block) Next() int64 {
	return b.Link.Next
}
