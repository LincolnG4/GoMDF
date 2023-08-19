package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type HD struct {
	Header     Header
	DGFirst    int64
	FHFirst    int64
	CHFirst    int64
	ATFirst    int64
	EVFirst    int64
	MDComment  int64
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

func (b *HD) NewBlock(file *os.File, startAdress int64, BLOCK_SIZE int) {
	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b)

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

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
