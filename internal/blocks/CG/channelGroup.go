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

type LinkV42 struct {
	Next        int64
	CnFirst     int64
	TxAcqName   int64
	SiAcqSource int64
	SrFirst     int64
	MDComment   int64
	CgMaster 	int64
}

type LinkV41 struct {
	Next        int64
	CnFirst     int64
	TxAcqName   int64
	SiAcqSource int64
	SrFirst     int64
	MDComment   int64
	CgMaster 	int64
}


type LinkV400 struct {
	Next        int64
	CnFirst     int64
	TxAcqName   int64
	SiAcqSource int64
	SrFirst     int64
	MDComment   int64
	CgMaster 	int64
}

type Data struct {
	RecordId      uint64
	CycleCount    uint64
	Flags         uint16
	PathSeparator uint16
	Reserved1     [4]byte
	DataBytes     uint32
	InvalBytes    uint32
}

const blockID string = blocks.CgID

func initializeBlockVersion(version uint16) *Block {
	b := Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'C', 'G'},
			Reserved:  [4]byte{},
			Length:    0,
			LinkCount: 0,
		},
		Link: Link{
			Next:        0,
			CnFirst:     0,
			TxAcqName:   0,
			SiAcqSource: 0,
			SrFirst:     0,
			MDComment:   0,
		},
		Data: Data{},
	}

	if version >= 420{
		b.Link.CgMaster = 0
	}

	return &b
}

func New(file *os.File, version uint16, startAdress int64) *Block {
	var blockSize uint64 = blocks.HeaderSize
	var b *Block

	b = initializeBlockVersion(version)
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

	return b
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'C', 'G'},
			Reserved:  [4]byte{},
			Length:    blocks.CgblockSize,
			LinkCount: 0,
		},
		Link: Link{
			Next:        0,
			CnFirst:     0,
			TxAcqName:   0,
			SiAcqSource: 0,
			SrFirst:     0,
			MDComment:   0,
		},
		Data: Data{},
	}
}