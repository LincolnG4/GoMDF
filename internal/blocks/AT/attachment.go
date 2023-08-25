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
	EmbeddedData DynamicData
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
	Reserved     uint8
	MD5Checksum  uint8
	OriginalSize uint64
	EmbeddedSize uint64
}

type DynamicData struct {
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
	b.Data = Data{}
	//Create a buffer based on blocksize
	buf = blocks.LoadBuffer(file, blockSize)

	BinaryError = binary.Read(buf, binary.LittleEndian, &b.Data)
	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
	}
	fmt.Printf("%+v\n\n", b.Data)

	//Calculates size of DynamicData Block
	blockSize = b.Data.EmbeddedSize

	b.EmbeddedData = DynamicData{EmbeddedData: make([]byte, b.Data.EmbeddedSize)}
	buff := make([]byte, blockSize)

	_, err := file.Read(buff)
	if err != nil {
		if err != io.EOF {
			fmt.Println("LoadBuffer error: ", err)
		}
	}

	BinaryError = binary.Read(buf, binary.LittleEndian, &b.EmbeddedData.EmbeddedData)
	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
	}

	fmt.Printf("%s\n\n", string(b.EmbeddedData.EmbeddedData))

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
		EmbeddedData: DynamicData{},
	}
}
