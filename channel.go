package mf4

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"reflect"

	"github.com/LincolnG4/GoMDF/blocks"
	"github.com/LincolnG4/GoMDF/blocks/CC"
	"github.com/LincolnG4/GoMDF/blocks/CG"
	"github.com/LincolnG4/GoMDF/blocks/CN"
	"github.com/LincolnG4/GoMDF/blocks/DG"
	"github.com/LincolnG4/GoMDF/blocks/DL"
	"github.com/LincolnG4/GoMDF/blocks/SI"
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

// parseSignalMeasure decode signal sample based on data type
func parseSignalMeasure(buf *bytes.Buffer, byteOrder binary.ByteOrder, dataType interface{}) (interface{}, error) {
	switch dataType.(type) {
	case string:
		strBytes, err := buf.ReadBytes(0) // Assuming strings are NULL-terminated
		if err != nil {
			return nil, err
		}
		return string(strBytes[:len(strBytes)-1]), nil
	case []uint8:
		byteArray := make([]byte, buf.Len())
		_, err := io.ReadFull(buf, byteArray) // Read all bytes into the array
		if err != nil {
			return nil, err
		}
		return hex.EncodeToString(byteArray), nil
	default:
		value := reflect.New(reflect.TypeOf(dataType)).Interface()
		if err := binary.Read(buf, byteOrder, value); err != nil {
			return nil, err
		}
		return reflect.ValueOf(value).Elem().Interface(), nil
	}
}

// readMeasure return extract sample measure from DTBlock//DLBlock
func (c *Channel) readMeasure(file *os.File, version uint16, isDataList bool) ([]interface{}, error) {
	// init
	cn := c.Block
	cg := c.ChannelGroup

	var dtl *DL.Block
	var err error
	var readAddr int64

	if isDataList {
		dtl, err = DL.New(file, version, c.DataGroup.Link.Data)
		if err != nil {
			return nil, err
		}
	} else {
		readAddr = c.signalValueAddress(c.DataGroup.Link.Data)
	}

	// byte slice order convert
	byteOrder := cn.ByteOrder()

	// get config
	size := cn.SignalBytesRange()
	rowSize := int64(cg.Data.DataBytes)

	data := make([]byte, size)
	measure := make([]interface{}, 0)

	dataType := cn.LoadDataType(len(data))

	var offset, target uint64
	k := 0
	for i := uint64(0); i < c.ChannelGroup.Data.CycleCount; i++ {
		if i == target && isDataList {
			// Next list
			if k == len(dtl.Link.Data) && dtl.Next() != 0 {
				dtl, err = DL.New(file, version, dtl.Next())
				if err != nil {
					return nil, err
				}
				k = 0
			}
			//Next Data
			offset = dtl.DataSectionLength(k)
			target += offset
			readAddr = c.signalValueAddress(dtl.Link.Data[k])
			k += 1
		}

		seekRead(file, readAddr, data)
		buf := bytes.NewBuffer(data)
		value, err := parseSignalMeasure(buf, byteOrder, dataType)
		if err != nil {
			return nil, err
		}
		measure = append(measure, value)
		readAddr += rowSize
	}
	return measure, nil
}

// readSingleDataBlock returns measure from DTBlock
func (c *Channel) readSingleDataBlock(file *os.File) ([]interface{}, error) {
	return c.readMeasure(file, 0, false)
}

// readDataList returns measure from DLBlock
func (c *Channel) readDataList(file *os.File, version uint16) ([]interface{}, error) {
	return c.readMeasure(file, version, true)
}

// signalValueAddress returns the offset from the signal in the DTBlock
func (c *Channel) signalValueAddress(dataAddress int64) int64 {
	return int64(blocks.HeaderSize) + dataAddress + int64(c.DataGroup.GetRecordIDSize()) + int64(c.Block.Data.ByteOffset)
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
