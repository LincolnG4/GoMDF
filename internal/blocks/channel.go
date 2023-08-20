package blocks

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type CN struct {
	Header        Header
	CnNext        int64
	CnComposition uint64
	TxName        uint64
	SiSource      uint64
	CcConvertion  uint64
	Data      	  uint64
	MdUnit        uint64
	MdComment     uint64
	Type          uint8
	SyncType      uint8
	DataType      uint8
	BitOffset     uint8
	ByteOffset    uint32
	BitCount      uint32
	Flags         uint32
	Precision     uint8
	Reserved1     [3]byte
	ValRangeMin   float32
	ValRangeMax   float32
	LimitMin      float32
	LimitExtMin   uint32
	LimitExtMax   float32
}

func (b *CN) NewBlock(file *os.File, startAdress int64) {
	buffer := NewBufferCN(file, startAdress, CnblockSize)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b)
	fmt.Println(string(b.Header.ID[:]))
	if string(b.Header.ID[:]) != CnID {
		fmt.Printf("ERROR NOT %s", CnID)
		panic(BinaryError)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

}

func (b *CN) BlankBlock() CN {
	return CN{}
}

func NewBufferCN(file *os.File, startAdress int64, BLOCK_SIZE int64) *bytes.Buffer {
	bytesValue := seekBinaryByAddressCN(file, startAdress, BLOCK_SIZE)
	return bytes.NewBuffer(bytesValue)
}

func seekBinaryByAddressCN(file *os.File, address int64, block_size int64) []byte {
	buf := make([]byte, block_size)
	_, errs := file.Seek(address, 0)
	if errs != nil {
		if errs != io.EOF {
			fmt.Println(errs)
		}

	}
	_, err := file.Read(buf)
	if err != nil {
		if err != io.EOF {
			fmt.Println(err)
		}

	}
	return buf
}

func (b *CN) GetSignalData(file *os.File, startAdress uint64, recordsize uint8, size uint64) {
	
}