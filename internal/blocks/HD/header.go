package HD

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
	DgFirst   int64
	FhFirst   int64
	ChFirst   int64
	AtFirst   int64
	EvFirst   int64
	MdComment int64
}

type Data struct {
	StartTimeNs   uint64
	TZOffsetMin   int16
	DSTOffsetMin  int16
	TimeFlags     uint8
	TimeClass     uint8
	Flags         uint8
	Reserved      uint8
	StartAngleRad float64
	StartDistM    float64
}

const blockID string = blocks.HdID

// New() seek and read to Block struct based on startAddress and blockSize
func New(file *os.File, startAdress int64) *Block {
	var blockSize uint64 = blocks.HeaderSize
	var b Block

	_, errs := file.Seek(startAdress, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs, "Memory Addr out of size")
		}
	}

	b.Header = blocks.Header{}

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

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'H', 'D'},
			Length:    blocks.HdblockSize,
			LinkCount: 6,
		},
		Link: Link{},
		Data: Data{},
	}

}
