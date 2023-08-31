package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type EV struct {
	Header          Header
<<<<<<< HEAD
	Next            Link
	EVParent        Link
	EVRange         Link
	TXName          Link
	MDComment       Link
	Scope           Link
	ATReference     Link
=======
	EVNext          int64
	EVParent        int64
	EVRange         int64
	TXName          int64
	MDComment       int64
	Scope           int64
	ATReference     [4]byte
>>>>>>> main
	Type            uint8
	SyncType        uint8
	RangeType       uint8
	Cause           uint8
	Flags           uint8
	Reserved1       [3]byte
	ScopeCount      uint32
	AttachmentCount uint16
	CreatorIndex    uint16
	SyncBaseValue   int64
	SyncFactor      float32
}

<<<<<<< HEAD
func (b *EV) New(file *os.File, startAdress Link, BLOCK_SIZE int) {
=======
func (b *EV) New(file *os.File, startAdress int64, BLOCK_SIZE int) {
>>>>>>> main

	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b)

	if string(b.Header.ID[:]) != EvID {
		fmt.Printf("ERROR NOT %s", EvID)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

}

func (b *EV) BlankBlock() EV {
	return EV{
		Header: Header{
			ID: [4]byte{'#', '#', 'E', 'V'},
			// Reserved:  [4]byte{},
			// Length:    0,
			// LinkCount: 0,
		},
		// EVNext:          0,
		// EVParent:        0,
		// EVRange:         0,
		// TXName:          0,
		// MDComment:       0,
		// Scope:           0,
		// ATReference:     [4]byte{},
		// Type:            0,
		// SyncType:        0,
		// RangeType:       0,
		// Flags:           0,
		// Reserved1:       [3]byte{},
		// ScopeCount:      0,
		// AttachmentCount: 0,
		// CreatorIndex:    0,
		// SyncBaseValue:   0,
		// SyncFactor:      0,
	}
}
