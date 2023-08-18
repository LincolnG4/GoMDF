package blocks

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/davecgh/go-spew/spew"
)

type Header struct {
	ID        [4]byte
	Reserved  [4]byte
	Length    uint64
	LinkCount uint64
}

type Link struct {
	TxData []byte
}

type TX struct {
	Header *Header
	Link   *Link
}

func (b *TX) NewBlock(file *os.File, startAdress int64, BLOCK_SIZE int) {
	b.Header = &Header{}
	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	
	if string(b.Header.ID[:]) != TX_ID {
		fmt.Printf("ERROR NOT %s ", TX_ID)
		panic(BinaryError)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

	b.Link = &Link{}
	_, err := file.Seek(startAdress + 24, io.SeekStart)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, int64(b.Header.Length-24))
	n, err := file.Read(buf[:cap(buf)])
	buf = buf[:n]
	if err != nil {
		if err != io.EOF {
			panic(err)
		}
	}
	b.Link.TxData = buf
	
	spew.Dump(&b)
	fmt.Printf("%+v\n",string(buf))

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
