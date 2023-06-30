package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

type CGBlock struct {
	ID          [4]byte
	Reserved    [4]byte
	Length      uint64
	LinkCount   uint64
	CGNext      int64
	CNNext      uint64
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
	DataGroup               *DGBlock
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
	ChannelGroup            CGBlock
	RecordSize              map[uint64]uint32
	Sorted                  bool
}

func (cgBlock *CGBlock) channelBlock(file *os.File, address int64) {

	const BLOCK_SIZE = 104

	bytesValue := seekBinaryByAddress(file, address, BLOCK_SIZE)
	buffer := bytes.NewBuffer(bytesValue)

	BinaryError := binary.Read(buffer, binary.LittleEndian, cgBlock)

	fmt.Println(string(bytesValue))

	if string(cgBlock.ID[:]) != "##CG" {
		fmt.Println("ERROR NOT CG")
	}

	if BinaryError != nil {
		fmt.Println("ERROR", BinaryError)
		// copy(cgBlock.ID[:], []byte("##DG"))
		// copy(cgBlock.Reserved[:], bytes.Repeat([]byte{0}, 4))
		// cgBlock.Length = 64
		// cgBlock.LinkCount = 4
		// cgBlock.DGNext = 0
		// cgBlock.CGNext = 0
		// cgBlock.DATA = 0
		// cgBlock.MDComment = 0
		// cgBlock.RecIDSize = 0
		// copy(cgBlock.DGReserved[:], bytes.Repeat([]byte{0}, 7))
	}

}

// func (cgBlock *CGBlock) Copy(dgBlock *DGBlock) {
// 	cgBlock.DataGroup = &dgBlock
// 	cgBlock.Channels = []uint64{}
// 	cgBlock.ChannelDependencies = []uint64{}
// 	cgBlock.SignalData = []uint64{}
// 	cgBlock.Record = 0
// 	cgBlock.Trigger = 0
// 	cgBlock.StringDtypes = 0
// 	cgBlock.DataBlocks = []uint64{}
// 	cgBlock.SingleChannelDtype = 0
// 	cgBlock.UsesId = false
// 	cgBlock.ReadSplitCount = 0
// 	cgBlock.DataBlocksInfoGenerator = []uint64{}

// }
