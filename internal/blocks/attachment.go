package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type AT struct {
	Header       Header
	ATNext       Link
	TXFilename   Link
	TXMimetype   Link
	MDComment    Link
	Flags        uint16
	CreatorIndex uint16
	ATReserved   [4]byte
	MD5Checksum  [16]byte
	OriginalSize uint64
	EmbeddedSize uint64
	EmbeddedData []byte
}

func (b *AT) New(file *os.File, startAdress Link, BLOCK_SIZE int) {

	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b)

	if string(b.Header.ID[:]) != AtID {
		fmt.Printf("ERROR NOT %s", AtID)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

}

func (b *AT) BlankBlock() AT {
	return AT{
		Header: Header{
			ID:        [4]byte{'#', '#', 'A', 'T'},
			Reserved:  [4]byte{},
			Length:    96,
			LinkCount: 2,
		},
		ATNext: 0,
	}
}
