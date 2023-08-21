package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type CG struct {
	Header      Header
	Next        Link
	CNNext      Link
	TxAcqName   Link
	SiAcqSource Link
	SrFirst     Link
	MDComment   Link
	RecordId    uint64
	CycleCount  uint64
	Flags       uint16
	Reserved1   [6]byte
	DataBytes   uint32
	InvalBytes  uint32
}

func (b *CG) New(file *os.File, startAdress Link, BLOCK_SIZE int) {
	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b)

	if string(b.Header.ID[:]) != CgID {
		fmt.Printf("ERROR NOT %s", CgID)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

}

func (b *CG) BlankBlock() CG {
	return CG{
		Header: Header{
			ID:        [4]byte{'#', '#', 'C', 'G'},
			Reserved:  [4]byte{},
			Length:    0,
			LinkCount: 0,
		},
		Next:        0,
		CNNext:      0,
		TxAcqName:   0,
		SiAcqSource: 0,
		SrFirst:     0,
		MDComment:   0,
		RecordId:    0,
		CycleCount:  0,
		Flags:       0,
		Reserved1:   [6]byte{},
		DataBytes:   0,
		InvalBytes:  0,
	}
}
