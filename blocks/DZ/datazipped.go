package DZ

import (
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
	OrgBlockTyṕe [2]byte

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

	// Seek to the start address
	if _, err := file.Seek(startAddress, io.SeekStart); err != nil {
		if err != io.EOF {
			return b.BlankBlock(), fmt.Errorf("error seeking to address %d: %w", startAddress, err)
		}
		return b.BlankBlock(), fmt.Errorf("EOF reached while seeking to address %d", startAddress)
	}

	// Calculate the size of the Header and read it directly
	size := int64(blocks.HeaderSize) + DataOrgBlockTypeSize + DataZipTypeSize + DataReservedSize + DataZipParameterSize + DataOrgDataLengthSize + DataLengthSize

	buf := make([]byte, size)
	if _, err := io.ReadFull(file, buf); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading header: %w", err)
	}

	// Parse the header
	b.Header = blocks.Header{
		ID:        [4]byte(buf[HeaderIDOffset : HeaderIDOffset+HeaderIDSize]),
		Reserved:  [4]byte(buf[HeaderReservedOffset : HeaderReservedOffset+HeaderReservedSize]),
		Length:    binary.LittleEndian.Uint64(buf[HeaderLengthOffset : HeaderLengthOffset+HeaderLengthSize]),
		LinkCount: binary.LittleEndian.Uint64(buf[HeaderLinkCountOffset : HeaderLinkCountOffset+HeaderLinkCountSize]),
	}

	// Parse the data
	b.Data = Data{
		OrgBlockTyṕe:  [2]byte(buf[DataOrgBlockTypeOffset : DataOrgBlockTypeOffset+DataOrgBlockTypeSize]),
		ZipType:       buf[DataZipTypeOffset],
		Reserved:      [1]byte(buf[DataReservedOffset : DataReservedOffset+DataReservedSize]),
		ZipParameter:  binary.LittleEndian.Uint32(buf[DataZipParameterOffset : DataZipParameterOffset+DataZipParameterSize]),
		OrgDataLenght: binary.LittleEndian.Uint64(buf[DataOrgDataLengthOffset : DataOrgDataLengthOffset+DataOrgDataLengthSize]),
		DataLenght:    binary.LittleEndian.Uint64(buf[DataLengthOffset : DataLengthOffset+DataLengthSize]),
	}

	buf = make([]byte, b.Data.DataLenght)
	if _, err := io.ReadFull(file, buf); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading header: %w", err)
	}
	b.Data.Data = buf

	return &b, nil
}

// ID original block ("DT", "SD", "RD" or "DV", "DI", "RV", "RI")
func (b *Block) BlockType() string {
	return string(b.Data.OrgBlockTyṕe[:])
}

// ID modified block with `#` ("##DT", "##SD", "##RD" or "#D#V", "##DI", "##RV", "##RI")
func (b *Block) BlockTypeModified() string {
	return "##" + string(b.Data.OrgBlockTyṕe[:])
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
