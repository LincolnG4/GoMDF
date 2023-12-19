package DG

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
	Next      int64
	CgFirst   int64
	Data      int64
	MdComment int64
}

type Data struct {
	RecIDSize uint8
	Reserved  [7]byte
}

const blockID string = blocks.DgID

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

// IsSorted checks if is Sorted `True`. Else `False` if it is Unsorted
func (b *Block) IsSorted() bool {
	return b.GetRecordID() == 0
}

func (b *Block) GetRecordID() uint8 {
	return b.Data.RecIDSize
}

func (b *Block) MetadataComment() int64 {
	return b.Link.MdComment
}

func (b *Block) FirstChannelGroup() int64 {
	return b.Link.CgFirst
}

func (b *Block) Next() int64 {
	return b.Link.Next
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'D', 'G'},
			Reserved:  [4]byte{},
			Length:    blocks.DgblockSize,
			LinkCount: 4,
		},
		Link: Link{},
		Data: Data{},
	}
}
