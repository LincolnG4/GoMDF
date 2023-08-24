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
	CgMaster    int64 //Version4.2
}

type Data struct {
	RecordId      uint64
	CycleCount    uint64
	Flags         uint16
	PathSeparator uint16
	Reserved      [4]byte
	DataBytes     uint32
	InvalBytes    uint32
}

const blockID string = blocks.CgID

func New(file *os.File, version uint16, startAdress int64) *Block {
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
	buffEach := make([]byte, blockSize)

	// Read the Link section from the binary file
	BinaryError = binary.Read(file, binary.LittleEndian, &buffEach)
	if BinaryError != nil {
		fmt.Println("Error reading Link section:", BinaryError)
	}
	
	// Populate the Link fields
	b.Link.Next 		= int64(binary.LittleEndian.Uint64(buffEach[0:8]))
	b.Link.CnFirst  	= int64(binary.LittleEndian.Uint64(buffEach[8:16]))
	b.Link.TxAcqName 	= int64(binary.LittleEndian.Uint64(buffEach[16:24]))
	b.Link.SiAcqSource 	= int64(binary.LittleEndian.Uint64(buffEach[24:32]))
	b.Link.SrFirst 		= int64(binary.LittleEndian.Uint64(buffEach[32:40]))
	b.Link.MdComment 	= int64(binary.LittleEndian.Uint64(buffEach[40:48]))

	if version >= 420 {
		BinaryError = binary.Read(file, binary.LittleEndian, &b.Link.CgMaster)
		if BinaryError != nil {
			fmt.Println("Error reading cg_cg_master:", BinaryError)
		}
	}
	fmt.Printf("%+v\n", b.Link)

	//Calculates size of Data Block
	blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	b.Data = Data{}
	buf = blocks.LoadBuffer(file, blockSize)
	
	// if version >= 410{
	// 	b.Data.Reserved = [4]byte{}
	// }
	
	//Create a buffer based on blocksize
	BinaryError = binary.Read(buf, binary.LittleEndian, &b.Data)
	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
	}

	


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
		Link: Link{
			Next:        0,
			CnFirst:     0,
			TxAcqName:   0,
			SiAcqSource: 0,
			SrFirst:     0,
			MdComment:   0,
		},
		Data: Data{},
	}
}
