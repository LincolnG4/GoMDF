package AT

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

type Block struct {
	Header       blocks.Header
	Link         Link
	Data         Data

}

type Link struct {
	Next       int64
	TXFilename int64
	TXMimetype int64
	MDComment  int64
}

type Data struct {
	Flags        uint16
	CreatorIndex uint16
	Reserved     [4]byte
	MD5Checksum  [16]byte
	OriginalSize uint64
	EmbeddedSize uint64
	EmbeddedData []byte
}



const blockID string = blocks.AtID

func New(file *os.File, startAdress int64) *Block {
	var b Block
	var blockSize uint64 = blocks.HeaderSize

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
	buffEach := make([]byte, blockSize)

	// Read the Link section from the binary file
	if err := binary.Read(file, binary.LittleEndian, &buffEach); err != nil {
		fmt.Println("Error reading Link section:", err)
	}
	
	
	var fixedArray16 [16]byte
	
	b.Data = Data{}
	b.Data.Flags        = binary.LittleEndian.Uint16(buffEach[0:2]) 
	b.Data.CreatorIndex = binary.LittleEndian.Uint16(buffEach[2:4])
	
	//md5CheckSum
	md5CheckSum := buffEach[8:24]
	copy(fixedArray16[:], md5CheckSum[:])
	b.Data.MD5Checksum = fixedArray16

	b.Data.OriginalSize = binary.LittleEndian.Uint64(buffEach[24:32])
	b.Data.EmbeddedSize = binary.LittleEndian.Uint64(buffEach[32:40])
	b.Data.EmbeddedSize = binary.LittleEndian.Uint64(buffEach[40:b.Data.EmbeddedSize])
	
	return &b
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'A', 'T'},
			Reserved:  [4]byte{},
			Length:    blocks.AtblockSize,
			LinkCount: 2,
		},
		Link:         Link{},
		Data:         Data{},
		
	}
}