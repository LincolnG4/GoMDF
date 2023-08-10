package blocks

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

type IDBlock struct {
	IDFile          [8]byte
	IDVersion       [8]byte
	IDProgram       [8]byte
	IDReserved1     [4]byte
	IDVersionNumber uint16
	IDReserved2     [34]byte
}

func (idBlock *IDBlock) NewBlock(file *os.File) {

	var ADDRESS int64 = 0
	const IDBLOCK_SIZE = 64

	bytesValue := seekBinaryByAddress(file, ADDRESS, IDBLOCK_SIZE)
	buffer := bytes.NewBuffer(bytesValue)
	BinaryError := binary.Read(buffer, binary.LittleEndian, idBlock)

	//fmt.Println(string(bytesValue))

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		copy(idBlock.IDFile[:], []byte("MDF     "))
		copy(idBlock.IDVersion[:], []byte("4.00    "))
		copy(idBlock.IDProgram[:], []byte("GoMDF1.0"))
		copy(idBlock.IDReserved1[:], bytes.Repeat([]byte{0}, 4))
		idBlock.IDVersionNumber = 400
		copy(idBlock.IDReserved2[:], bytes.Repeat([]byte{0}, 34))
	}
}
