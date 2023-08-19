package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type XML struct {
	Value []byte
}

type MD struct {
	Header *Header
	MdData *XML
}

func (b *MD) NewBlock(file *os.File, startAdress int64, BLOCK_SIZE int) {

	b.Header = &Header{}
	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	if string(b.Header.ID[:]) != MdID {
		fmt.Printf("ERROR NOT %s ", MdID)
		panic(BinaryError)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}
	b.MdData = &XML{}
	buf := make([]byte, int64(b.Header.Length-24))

	b.MdData.Value = getText(file, startAdress, buf, true)

}

func (b *MD) BlankBlock() MD {
	return MD{
		&Header{
			ID:        [4]byte{'#', '#', 'M', 'D'},
			Reserved:  [4]byte{},
			Length:    64,
			LinkCount: 4,
		},
		&XML{},
	}
}
