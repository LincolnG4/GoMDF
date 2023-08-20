package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type DT struct {
	Header *Header
	Samples []byte
}

func (b *DT) NewBlock(file *os.File, startAdress int64, BLOCK_SIZE int) {
	b.Header = &Header{}
	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	if string(b.Header.ID[:]) != DtID {
		fmt.Printf("ERROR NOT %s ", DtID)
		panic(BinaryError)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}
	fmt.Printf("%+v",b.Header)
	
	

}

func (b *DT) BlankBlock() DT {
	return DT{
		Header: &Header{
			ID:        [4]byte{'#', '#', 'D', 'T'},
			Reserved:  [4]byte{},
			Length:    64,
			LinkCount: 4,
		},
		Samples: []byte{},
	}
}
