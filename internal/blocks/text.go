package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type TX struct {
	Header *Header
	TxData []byte
}

func (b *TX) New(file *os.File, startAdress Link, BLOCK_SIZE int) {
	b.Header = &Header{}
	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	if string(b.Header.ID[:]) != TxID {
		fmt.Printf("ERROR NOT %s ", TxID)
		panic(BinaryError)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}
	b.TxData = []byte{}
	buf := make([]byte, int64(b.Header.Length-24))

	b.TxData = getText(file, startAdress, buf, true)

}

func (b *TX) BlankBlock() TX {
	return TX{
		Header: &Header{
			ID:        [4]byte{'#', '#', 'T', 'X'},
			Reserved:  [4]byte{},
			Length:    64,
			LinkCount: 4,
		},
		TxData: []byte{},
	}
}
