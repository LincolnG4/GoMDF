package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type FH struct {
	Header       Header
	FHNext       Link
	MDComment    Link
	TimeNS       uint64
	TZOffsetMin  int16
	DSTOffsetMin int16
	TimeFlags    uint8
	Reserved     [3]byte
}

func (b *FH) New(file *os.File, startAdress Link, BLOCK_SIZE int) {

	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b)

	if string(b.Header.ID[:]) != FgID {
		fmt.Println("ERROR NOT FH")
		panic(BinaryError)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

}

func (b *FH) BlankBlock() FH {
	return FH{
		Header: Header{
			ID:        [4]byte{'#', '#', 'F', 'H'},
			Reserved:  [4]byte{},
			Length:    56,
			LinkCount: 2,
		},
		FHNext:       0,
		MDComment:    0,
		TimeNS:       0,
		TZOffsetMin:  0,
		DSTOffsetMin: 0,
		TimeFlags:    0,
		Reserved:     [3]byte{},
	}
}
