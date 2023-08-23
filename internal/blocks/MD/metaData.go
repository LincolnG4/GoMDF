package MD

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

type Data struct {
	Value []byte
}

type Block struct {
	Header *blocks.Header
	Data   *Data
}

func (b *Block) New(file *os.File, startAdress int64) {

	//Read Header Section
	b.Header = &blocks.Header{}
	buffer := blocks.NewBuffer(file, startAdress, blocks.HeaderSize)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b.Header)

	blockID := string(b.Header.ID[:])
	if blockID != blocks.MdID && blockID != blocks.TxID {
		fmt.Printf("ERROR NOT %s\n", blocks.MdID)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

	//If block is ##MD
	if blockID == blocks.MdID {
		b.Data = &Data{}
		buf := make([]byte, int64(b.Header.Length-blocks.HeaderSize))

		b.Data.Value = blocks.GetText(file, startAdress, buf, true)

	} else { //If block is ##TX
		b.Data = &Data{}
		buf := make([]byte, int64(b.Header.Length-blocks.HeaderSize))

		b.Data.Value = blocks.GetText(file, startAdress, buf, true)

	}

}

func (b *Block) BlankBlock() Block {
	return Block{
		&blocks.Header{
			ID:        [4]byte{'#', '#', 'M', 'D'},
			Reserved:  [4]byte{},
			Length:    64,
			LinkCount: 4,
		},
		&Data{},
	}
}
