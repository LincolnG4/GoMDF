package DZ

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/LincolnG4/GoMDF/blocks"
)

type ZipBlock interface {
	Decompress() ([]byte, error)
}

type Block struct {
	Header blocks.Header
	Data   Data
}

type Data struct {
	//ID original block ("DT", "SD", "RD" or "DV", "DI", "RV", "RI")
	OrgBlockType [2]byte

	//Zip algorithm (Deflate or Transposition + Deflate)
	ZipType uint8

	Reserved [1]byte

	//Parameter for zip algorithm
	ZipParameter uint32

	//Data section's length
	OrgDataLenght uint64

	//Lenght stored in dz_data (bytes)
	DataLenght uint64

	//Stored value
	Data []byte
}

const (
	// Header offsets
	HeaderIDOffset        = 0
	HeaderReservedOffset  = 4
	HeaderLengthOffset    = 8
	HeaderLinkCountOffset = 16

	// Data section offsets
	DataOrgBlockTypeOffset  = 24
	DataZipTypeOffset       = 26
	DataReservedOffset      = 27
	DataZipParameterOffset  = 28
	DataOrgDataLengthOffset = 32
	DataLengthOffset        = 40
	DataOffset              = 48
)

const (
	// Sizes of fields in bytes
	HeaderIDSize        = 4
	HeaderReservedSize  = 4
	HeaderLengthSize    = 8
	HeaderLinkCountSize = 8

	DataOrgBlockTypeSize  = 2
	DataZipTypeSize       = 1
	DataReservedSize      = 1
	DataZipParameterSize  = 4
	DataOrgDataLengthSize = 8
	DataLengthSize        = 8
)

func New(file *os.File, startAddress int64) (*Block, error) {
	var b Block

	// Initialize the header
	b.Header = blocks.Header{}

	// Read the header
	var err error
	b.Header, err = blocks.GetHeader(file, startAddress, blocks.DzID)
	if err != nil {
		return b.BlankBlock(), err
	}

	// Calculate the size of the Data Block and read it directly
	dataBlockSize := blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	dataBuffer := make([]byte, dataBlockSize)
	if _, err := io.ReadFull(file, dataBuffer); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading data section: %w", err)
	}

	// Create a reader for the data buffer
	dataReader := bytes.NewReader(dataBuffer)

	// Define pointers to the fields to be read
	fields := []interface{}{
		&b.Data.OrgBlockType,
		&b.Data.ZipType,
		&b.Data.Reserved,
		&b.Data.ZipParameter,
		&b.Data.OrgDataLenght,
		&b.Data.DataLenght,
	}

	// Read data fields into the Data struct
	for _, field := range fields {
		if err := binary.Read(dataReader, binary.LittleEndian, field); err != nil {
			return b.BlankBlock(), fmt.Errorf("error loading data from ccblock: %w", err)
		}
	}

	// Read the actual compressed data based on DataLenght
	buf := make([]byte, b.Data.DataLenght)
	if _, err := io.ReadFull(dataReader, buf); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading compressed data: %w", err)
	}
	b.Data.Data = buf

	return &b, nil
}

// ID original block ("DT", "SD", "RD" or "DV", "DI", "RV", "RI")
func (b *Block) BlockType() string {
	return string(b.Data.OrgBlockType[:])
}

// ID modified block with `#` ("##DT", "##SD", "##RD" or "##DV", "##DI", "##RV", "##RI")
func (b *Block) BlockTypeModified() string {
	return "##" + string(b.Data.OrgBlockType[:])
}

func (b *Block) NewCompressType() (ZipBlock, error) {
	switch b.Data.ZipType {
	case 0:
		return &Flate{
			DecompressedID:     b.BlockTypeModified(),
			CompressType:       b.Data.ZipType,
			DecompressedLength: b.Data.OrgDataLenght,
			CompressedLength:   b.Data.DataLenght,
			Datablock:          &b.Data.Data,
		}, nil
	case 1:
		return &Transposition{
			DecompressedID:     b.BlockTypeModified(),
			CompressType:       b.Data.ZipType,
			DecompressedLength: b.Data.OrgDataLenght,
			CompressedLength:   b.Data.DataLenght,
			Parameter:          b.Data.ZipParameter,
			Datablock:          &b.Data.Data,
		}, nil
	default:
		return nil, fmt.Errorf("invalid decompress type %d", b.Data.ZipType)
	}
}

// decompress the zip block based on the type of compresion
func (b *Block) decompress() ([]byte, error) {
	zip, err := b.NewCompressType()
	if err != nil {
		return nil, err
	}
	return zip.Decompress()
}

// Read returns the value inside of the block
func (b *Block) Read() ([]byte, error) {
	return b.decompress()
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.DtID),
			Reserved:  [4]byte{},
			Length:    24,
			LinkCount: 0,
		},
	}
}
