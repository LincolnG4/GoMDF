package blocks

import (
	"encoding/binary"
	"fmt"
	"os"
)

type CG struct {
	ID          [4]byte
	Reserved    [4]byte
	Length      uint64
	LinkCount   uint64
	CGNext      int64
	CNNext      int64
	TxAcqName   int64
	SiAcqSource uint64
	SrFirst     uint64
	MDComment   uint64
	RecordId    uint64
	CycleCount  uint64
	Flags       uint16
	Reserved1   [6]byte
	DataBytes   uint32
	InvalBytes  uint32
}

type Group struct {
	DataGroup               *DG
	Channels                []uint64
	ChannelDependencies     []uint64
	SignalData              []uint64
	Record                  int
	Trigger                 int
	StringDtypes            int
	DataBlocks              []uint64
	SingleChannelDtype      int
	UsesId                  bool
	ReadSplitCount          int
	DataBlocksInfoGenerator []uint64
	ChannelGroup            CG
	RecordSize              map[uint64]uint32
	Sorted                  bool
}

func (b *CG) NewBlock(file *os.File, startAdress int64, BLOCK_SIZE int) {
	buffer := NewBuffer(file, startAdress, BLOCK_SIZE)
	BinaryError := binary.Read(buffer, binary.LittleEndian, b)

	if string(b.ID[:]) != CG_ID {
		fmt.Printf("ERROR NOT %s", CG_ID)
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		b.BlankBlock()
	}

}

func (b *CG) BlankBlock() CG {
	return CG{
		ID:          [4]byte{'#', '#', 'C', 'G'},
		Reserved:    [4]byte{},
		Length:      0,
		LinkCount:   0,
		CGNext:      0,
		CNNext:      0,
		TxAcqName:   0,
		SiAcqSource: 0,
		SrFirst:     0,
		MDComment:   0,
		RecordId:    0,
		CycleCount:  0,
		Flags:       0,
		Reserved1:   [6]byte{},
		DataBytes:   0,
		InvalBytes:  0,
	}
}

// func (CG *CG) Copy(dgBlock *DGBlock) {
// 	CG.DataGroup = &dgBlock
// 	CG.Channels = []uint64{}
// 	CG.ChannelDependencies = []uint64{}
// 	CG.SignalData = []uint64{}
// 	CG.Record = 0
// 	CG.Trigger = 0
// 	CG.StringDtypes = 0
// 	CG.DataBlocks = []uint64{}
// 	CG.SingleChannelDtype = 0
// 	CG.UsesId = false
// 	CG.ReadSplitCount = 0
// 	CG.DataBlocksInfoGenerator = []uint64{}

// }
