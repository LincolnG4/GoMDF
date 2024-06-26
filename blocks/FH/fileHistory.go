package FH

import (
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
	Next      int64
	MDComment int64
}

type Data struct {
	TimeNS       uint64
	TZOffsetMin  int16
	DSTOffsetMin int16
	TimeFlags    uint8
	Reserved     [3]byte
}

func New(file *os.File, startAdress int64) *Block {
	var blockSize uint64 = blocks.HeaderSize
	var b Block

	_, errs := file.Seek(startAdress, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs, "memory addr out of size")
		}
	}

	//Read Header Section
	b.Header = blocks.Header{}

	//Create a buffer based on blocksize
	buf := blocks.LoadBuffer(file, blockSize)

	//Read header
	BinaryError := binary.Read(buf, binary.LittleEndian, &b.Header)
	if BinaryError != nil {
		fmt.Println("error", BinaryError)
		b.BlankBlock()
	}

	if string(b.Header.ID[:]) != blocks.FhID {
		fmt.Printf("error not %s", blocks.FhID)
	}

	//Calculates size of Link Block
	blockSize = blocks.CalculateLinkSize(b.Header.LinkCount)
	b.Link = Link{}
	buf = blocks.LoadBuffer(file, blockSize)

	//Create a buffer based on blocksize
	BinaryError = binary.Read(buf, binary.LittleEndian, &b.Link)
	if BinaryError != nil {
		fmt.Println("error", BinaryError)
	}

	//Calculates size of Data Block
	blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	b.Data = Data{}
	buf = blocks.LoadBuffer(file, blockSize)

	//Create a buffer based on blocksize
	BinaryError = binary.Read(buf, binary.LittleEndian, &b.Data)
	if BinaryError != nil {
		fmt.Println("error", BinaryError)
	}

	return &b
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'F', 'H'},
			Reserved:  [4]byte{},
			Length:    blocks.FhblockSize,
			LinkCount: 2,
		},
		Link: Link{},
		Data: Data{},
	}
}

func (b *Block) GetChangeLog(file *os.File) string {
	t, err := TX.GetText(file, b.GetMdComment())
	if err != nil {
		return ""
	}

	return t
}

func (b *Block) GetMdComment() int64 {
	return b.Link.MDComment
}

func (b *Block) GetTimeNs() int64 {
	return int64(b.Data.TimeNS)
}

func (b *Block) GetTimeFlag() uint8 {
	return b.Data.TimeFlags
}

func (b *Block) Next() int64 {
	return b.Link.Next
}
