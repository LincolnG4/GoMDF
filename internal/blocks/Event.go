package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type EV struct {
	ID              [4]byte
	Reserved        [4]byte
	Length          uint64
	LinkCount       uint64
	EVNext          int64
	EVParent        uint64
	EVRange         uint64
	TXName          uint16
	MDComment       uint16
	Scope           uint16
	ATReference     [4]byte
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

func (b *EV) NewBlock(file *os.File, startAdress int64, BLOCK_SIZE int) {

	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b)

	if string(b.ID[:]) != EV_ID {
		fmt.Printf("ERROR NOT %s", EV_ID)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

}

func (b *EV) BlankBlock() EV {
	return EV{
		// ID:              [4]byte{'#', '#', 'E', 'V'},
		// Reserved:        [4]byte,
		// Length:          0,
		// LinkCount:       0,
		// EVNext:          0,
		// EVParent:        0,
		// EVRange:         0,
		// TXName:          0,
		// MDComment:       0,
		// Scope:           0,
		// ATReference:     0,
		// Type:            0,
		// SyncType:        0,
		// RangeType:       0,
		// Flags:           0,
		// Reserved1:       0,
		// ScopeCount:      0,
		// AttachmentCount: 0,
		// CreatorIndex:    0,
		// SyncBaseValue:   0,
		// SyncFactor:      0,
	}
}
