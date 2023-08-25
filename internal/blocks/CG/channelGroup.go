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
	Next        int64
	CnFirst     int64
	TxAcqName   int64
	SiAcqSource int64
	SrFirst     int64
	MdComment   int64
	CgMaster    int64 // Version 4.2
}

type Data struct {
	RecordId      uint64
	CycleCount    uint64
	Flags         uint16
	PathSeparator uint16  // Version 4.1
	Reserved      uint8
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
