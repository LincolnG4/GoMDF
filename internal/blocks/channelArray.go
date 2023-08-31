package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type CA struct {
	Header            *Header
	Composition       Link
	Data              Link
	DynamicSize       Link
	InputQuality      Link
	OutputQuality     Link
	ComparisonQuality Link
	CcAxisConvertion  Link
	Axis              Link
	Type              uint8
	Storage           uint8
	Ndim              uint16
	Flags             uint32
	ByteOffsetBase    uint32
	InvalBitposBase   uint32
	DimSize           uint64
	AxisValue         float64
	CycleCount        uint64
}

func (b *CA) New(file *os.File, startAdress int64, BLOCK_SIZE int) {
	b.Header = &Header{}
	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	fmt.Println(string(b.Header.ID[:]))

	if string(b.Header.ID[:]) != CaID {
		fmt.Printf("ERROR NOT %s ", CaID)
		panic(BinaryError)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)

	}

}

func (b *CA) BlankBlock() CA {
	return CA{
		Header: &Header{
			ID:        [4]byte{'#', '#', 'C', 'A'},
			Reserved:  [4]byte{},
			Length:    64,
			LinkCount: 4,
		},
	}
}
