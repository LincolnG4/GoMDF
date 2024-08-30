package mf4

import (
	"os"

	"github.com/LincolnG4/GoMDF/blocks/DG"
)

type DataGroup struct {
	block           *DG.Block
	ChannelGroup    []*ChannelGroup
	CachedDataGroup []byte
}

func NewDataGroup(f *os.File, address int64) DataGroup {
	dataGroupBlock := DG.New(f, address)
	return DataGroup{
		block:        dataGroupBlock,
		ChannelGroup: []*ChannelGroup{},
	}
}

func (d *DataGroup) DataAddress() int64 {
	return d.block.Link.Data
}
