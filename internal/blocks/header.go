package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type HD struct {
	Header     Header
	DGFirst    Link
	FHFirst    Link
	CHFirst    Link
	ATFirst    Link
	EVFirst    Link
	MDComment  Link
	StartTime  uint64
	TZOffset   int16
	DSTOffset  int16
	TimeFlags  uint8
	TimeClass  uint8
	Flags      uint8
	Reserved2  uint8
	StartAngle float32
	StartDist  float32
}

func (b *HD) New(file *os.File, startAdress Link, BLOCK_SIZE int) {
	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

	buffer = NewBuffer(file, startAdress+24, int(b.Header.Length))
	BinaryError = binary.Read(buffer, binary.LittleEndian, b)
	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
	}
	fmt.Println(b)
}

func (b *HD) BlankBlock() HD {
	return HD{
		Header: Header{
			ID:        [4]byte{'#', '#', 'H', 'D'},
			Length:    104,
			LinkCount: 6,
		},
		DGFirst:    0,
		FHFirst:    0,
		CHFirst:    0,
		ATFirst:    0,
		EVFirst:    0,
		MDComment:  0,
		StartTime:  0,
		TZOffset:   0,
		DSTOffset:  0,
		TimeFlags:  0,
		TimeClass:  0,
		Flags:      0,
		Reserved2:  0,
		StartAngle: 0,
		StartDist:  0,
	}

}
