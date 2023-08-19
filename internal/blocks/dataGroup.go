package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type DG struct {
	Header     Header
	DGNext     int64
	CGNext     int64
	DATA       uint64
	MDComment  uint16
	RecIDSize  uint8
	DGReserved [7]byte
}

func (b *DG) NewBlock(file *os.File, startAdress int64, BLOCK_SIZE int) {
	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b)

	if string(b.Header.ID[:]) != DgID {
		fmt.Printf("ERROR NOT %s", DgID)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

}

func (b *DG) BlankBlock() DG {
	return DG{
		Header: Header{
			ID:        [4]byte{'#', '#', 'D', 'G'},
			Reserved:  [4]byte{},
			Length:    64,
			LinkCount: 4,
		},
		DGNext:     0,
		CGNext:     0,
		DATA:       0,
		MDComment:  0,
		RecIDSize:  0,
		DGReserved: [7]byte{},
	}
}
