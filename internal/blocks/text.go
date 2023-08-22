package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type Link struct {
	TxData []byte
}

type TX struct {
	Header *Header
	Link   *Link
}

func (b *TX) New(file *os.File, startAdress int64, BLOCK_SIZE int) {
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
	b.Link = &Link{}
	buf := make([]byte, int64(b.Header.Length-24))

	b.Link.TxData = getText(file, startAdress, buf, true)

}

func (b *TX) BlankBlock() TX {
	return TX{
		&Header{
			ID:        [4]byte{'#', '#', 'T', 'X'},
			Reserved:  [4]byte{},
			Length:    64,
			LinkCount: 4,
		},
		&Link{
			TxData: []byte{},
		},
	}
}
