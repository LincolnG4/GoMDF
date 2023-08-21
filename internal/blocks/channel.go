package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type CN struct {
	Header       Header
	Next         Link
	Composition  Link
	TxName       Link
	SiSource     Link
	CcConvertion Link
	Data         Link
	MdUnit       Link
	MdComment    Link
	Type         uint8
	SyncType     uint8
	DataType     uint8
	BitOffset    uint8
	ByteOffset   uint32
	BitCount     uint32
	Flags        uint32
	Precision    uint8
	Reserved1    [3]byte
	ValRangeMin  float32
	ValRangeMax  float32
	LimitMin     float32
	LimitExtMin  uint32
	LimitExtMax  float32
}

func (b *CN) New(file *os.File, startAdress Link) {
	buffer := NewBuffer(file, startAdress, CnblockSize)
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

func (b *CN) GetSignalData(file *os.File, startAdress Link, recordsize uint8, size uint64) {

}
