package mf4

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/blocks/CC"
	"github.com/LincolnG4/GoMDF/internal/blocks/CG"
	"github.com/LincolnG4/GoMDF/internal/blocks/CN"
	"github.com/LincolnG4/GoMDF/internal/blocks/DG"
	"github.com/LincolnG4/GoMDF/internal/blocks/SI"
)

type Channel struct {
	Name         string
	Block        *CN.Block
	Convertion   CC.Conversion
	DataGroup    *DG.Block
	ChannelGroup *CG.Block
	SourceInfo   SI.SourceInfo
	Comment      string
	MdComment    string
}

func (c *Channel) readSingleDataBlock(file *os.File) ([]interface{}, error) {
	var byteOrder binary.ByteOrder

	cn := c.Block
	cg := c.ChannelGroup
	dg := c.DataGroup

	readAddr := int64(blocks.HeaderSize) + dg.Link.Data + int64(dg.GetRecordIDSize()) + int64(cn.Data.ByteOffset)
	size := (cn.Data.BitCount + uint32(cn.Data.BitOffset)) / 8
	data := make([]byte, size)
	sample := make([]interface{}, 0)

	rowSize := int64(cg.Data.DataBytes)

	if cn.IsLittleEndian() {
		byteOrder = binary.LittleEndian
	} else {
		byteOrder = binary.BigEndian
	}

	dtype := cn.LoadDataType(len(data))

	// Create a new instance of the data type using reflection
	sliceElemType := reflect.TypeOf(dtype)
	sliceElem := reflect.New(sliceElemType).Interface()

	for i := uint64(0); i < cg.Data.CycleCount; i += 1 {
		seekRead(file, readAddr, data)
		buf := bytes.NewBuffer(data)
		err := binary.Read(buf, byteOrder, sliceElem)
		if err != nil {
			return nil, fmt.Errorf("error during parsing channel: %s ", err)
		}
		sample = append(sample, reflect.ValueOf(sliceElem).Elem().Interface())
		readAddr += rowSize
	}

	return sample, nil
}

func (c *Channel) applyConvertion(sample *[]interface{}) {
	if c.Convertion == nil {
		return
	}
	c.Convertion.Apply(sample)
}

func (c *Channel) readInvalidationBit(file *os.File) (bool, error) {
	address := c.getInvalidationBitStart()

	if _, err := file.Seek(address, io.SeekCurrent); err != nil {
		return false, err
	}

	var invalByte uint8
	if err := binary.Read(file, binary.LittleEndian, &invalByte); err != nil {
		return false, err
	}

	// Within this Byte read the bit specified by (cn_inval_bit_pos & 0x07)
	invalBitPos := uint(c.getInvalidationBitPos() & 0x07)
	isBitSet := blocks.IsBitSet(int(invalByte), int(invalBitPos))

	return isBitSet, nil
}

func (c *Channel) getInvalidationBitStart() int64 {
	return int64(c.getRecordID()) + int64(c.getDataBytes())
}

func (c *Channel) getRecordID() uint8 {
	return c.DataGroup.GetRecordIDSize()
}

func (c *Channel) getDataBytes() uint32 {
	return c.ChannelGroup.GetDataBytes()
}

func (c *Channel) getInvalidationBitPos() uint32 {
	return c.Block.InvalBitPos()
}
