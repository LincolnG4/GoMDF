package DL

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/blocks/DT"
)

type Block struct {
	Header blocks.Header
	Link   Link
	Data   Data
}

type Link struct {
	Next int64
	Data []int64
}

type Data struct {
	Flags          uint8
	Reserved       [3]byte
	Count          uint32
	EqualLegth     uint64
	Offset         []uint64
	TimeValues     []float64
	AngleValues    []float64
	DistanceValues []float64
}

const (
	EqualLegth = iota
	//version 4.2.0
	Time
	Angle
	Distance
)

func New(file *os.File, version uint16, startAdress int64) (*Block, error) {
	var b Block
	var err error

	b.Header = blocks.Header{}

	b.Header, err = blocks.GetHeader(file, startAdress, blocks.DlID)
	if err != nil {
		return b.BlankBlock(), err
	}

	//Calculates size of Link Block
	blockSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	buffEach := make([]byte, blockSize)

	// Read the Link section from the binary file
	if err := binary.Read(file, binary.LittleEndian, &buffEach); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading link section channelgroup %s", err)
	}

	// Populate the Link fields dynamically
	linkFields := []int64{}
	for i := 0; i < len(buffEach)/8; i++ {
		linkFields = append(linkFields, int64(binary.LittleEndian.Uint64(buffEach[i*8:(i+1)*8])))
	}

	b.Link = Link{
		Next: linkFields[0],
		Data: linkFields[1:],
	}

	//Calculates size of Data Block
	blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	b.Data = Data{}
	buf := blocks.LoadBuffer(file, blockSize)

	err = binary.Read(buf, binary.LittleEndian, &b.Data.Flags)
	if err != nil {
		return b.BlankBlock(), err
	}

	err = binary.Read(buf, binary.LittleEndian, &b.Data.Reserved)
	if err != nil {
		return b.BlankBlock(), err
	}

	err = binary.Read(buf, binary.LittleEndian, &b.Data.Count)
	if err != nil {
		return b.BlankBlock(), err
	}

	if blocks.IsBitSet(int(b.Data.Flags), EqualLegth) {
		err = binary.Read(buf, binary.LittleEndian, &b.Data.EqualLegth)
		if err != nil {
			return b.BlankBlock(), err
		}
	} else {
		// Only present if "equal length" flag (bit 0 in dl_flags) is not set.
		err = binary.Read(buf, binary.LittleEndian, &b.Data.Offset)
		if err != nil {
			return b.BlankBlock(), err
		}
	}

	if version < 420 {
		return &b, nil
	}

	// iterate over all fields and extract if bit is set
	var flagsArray [3]int = [3]int{Time, Angle, Distance}
	copy := [3]*[]float64{&b.Data.TimeValues, &b.Data.AngleValues, &b.Data.DistanceValues}
	for index, field := range copy {
		if blocks.IsBitSet(int(b.Data.Flags), flagsArray[index]) {
			err = binary.Read(buf, binary.LittleEndian, &field)
			if err != nil {
				return b.BlankBlock(), err
			}
		}
	}

	return &b, nil
}

func (b *Block) Concatenate(file *os.File) *DT.Block {
	samples := make([]byte, 0)
	for i := 0; i < int(b.Data.Count)-1; i++ {
		dt := DT.New(file, b.Link.Data[i])
		samples = append(samples, dt.Data...)
	}
	return &DT.Block{
		Header: blocks.Header{
			ID:        [4]byte{'#', '#', 'D', 'T'},
			Reserved:  [4]byte{},
			Length:    24 + uint64(b.Data.Count)*(b.Data.EqualLegth-24),
			LinkCount: 0,
		},
		Data: samples,
	}
}

// DataSectionLength returns offset based on datablock. If DTblock has EqualLegth, `variableOffsetIndex`
// will be ignored.
func (b *Block) DataSectionLength(variableOffsetIndex int) uint64 {
	if len(b.Data.Offset) > 0 {
		return b.Data.Offset[variableOffsetIndex]
	}
	return b.Data.EqualLegth / 16
}

func (b *Block) DataBlockType() string {
	return string(b.Header.ID[:])
}

func (b *Block) Next() int64 {
	return b.Link.Next
}

func (b *Block) BlankBlock() *Block {
	return &Block{}
}
