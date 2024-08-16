package mf4

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"

	"github.com/LincolnG4/GoMDF/blocks"
	"github.com/LincolnG4/GoMDF/blocks/CC"
	"github.com/LincolnG4/GoMDF/blocks/CG"
	"github.com/LincolnG4/GoMDF/blocks/CN"
	"github.com/LincolnG4/GoMDF/blocks/DG"
	"github.com/LincolnG4/GoMDF/blocks/DL"
	"github.com/LincolnG4/GoMDF/blocks/SD"
	"github.com/LincolnG4/GoMDF/blocks/SI"
)

type DataGroup struct {
	block        *DG.Block
	ChannelGroup []*ChannelGroup
}

func NewDataGroup(f *os.File, address int64) DataGroup {
	dataGroupBlock := DG.New(f, address)
	return DataGroup{
		block:        dataGroupBlock,
		ChannelGroup: []*ChannelGroup{},
	}
}

type ChannelGroup struct {
	Block       *CG.Block
	Channels    map[string]*Channel
	DataGroup   *DG.Block
	SourceInfo  SI.SourceInfo
	Comment     string
	IsVLSDBlock bool
}

type Channel struct {
	// channel's name
	Name string

	// conversion formula to convert the raw values to physical values with a
	// physical unit
	Conversion CC.Conversion

	// channel type
	Type string

	// pointer to the master channel of the channel group.
	// A 'nil' value indicates that this channel itself is the master.
	Master *Channel

	// pointer to data group
	DataGroup *DG.Block

	// data group's index
	DataGroupIndex int

	// pointer to channel group
	ChannelGroup *CG.Block

	// channel group's index
	ChannelGroupIndex int

	// unsorted channels mapped
	isUnsorted bool

	// describes the source of an acquisition mode or of a signal
	SourceInfo SI.SourceInfo

	// additional information about the channel. Can be 'nil'
	Comment string

	// Samples are cached in memory if file was set with MemoryOptimized is true
	CachedSamples []interface{}

	// Conversion applied
	isConverted bool

	// pointer to mf4 file
	mf4 *MF4

	// pointer to the CNBLOCK
	block *CN.Block
}

// parseSignalMeasure decode signal sample based on data type
func parseSignalMeasure(buf *bytes.Buffer, byteOrder binary.ByteOrder, dataType interface{}) (interface{}, error) {
	switch dataType.(type) {
	case string:
		return buf.String(), nil
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

func parseSignalMeasure2(data []byte, byteOrder binary.ByteOrder, dataType interface{}) (interface{}, error) {
	switch v := dataType.(type) {
	case string:
		return string(data), nil
	case []uint8:
		return hex.EncodeToString(data), nil
	case int8:
		return int8(data[0]), nil
	case uint8:
		return data[0], nil
	case int16:
		if len(data) < 2 {
			return nil, fmt.Errorf("not enough data to read int16")
		}
		return int16(byteOrder.Uint16(data)), nil
	case uint16:
		if len(data) < 2 {
			return nil, fmt.Errorf("not enough data to read uint16")
		}
		return byteOrder.Uint16(data), nil
	case int32:
		if len(data) < 4 {
			return nil, fmt.Errorf("not enough data to read int32")
		}
		return int32(byteOrder.Uint32(data)), nil
	case uint32:
		if len(data) < 4 {
			return nil, fmt.Errorf("not enough data to read uint32")
		}
		return byteOrder.Uint32(data), nil
	case int64:
		if len(data) < 8 {
			return nil, fmt.Errorf("not enough data to read int64")
		}
		return int64(byteOrder.Uint64(data)), nil
	case uint64:
		if len(data) < 8 {
			return nil, fmt.Errorf("not enough data to read uint64")
		}
		return byteOrder.Uint64(data), nil
	case float32:
		if len(data) < 4 {
			return nil, fmt.Errorf("not enough data to read float32")
		}
		return math.Float32frombits(byteOrder.Uint32(data)), nil
	case float64:
		if len(data) < 8 {
			return nil, fmt.Errorf("not enough data to read float64")
		}
		return math.Float64frombits(byteOrder.Uint64(data)), nil
	default:
		return nil, fmt.Errorf("unsupported data type: %T", v)
	}
}

// readMeasure return extract sample measure from DTBlock//DLBlock with fixed
// lenght
func (c *Channel) readSingleChannel(isDataList bool) ([]interface{}, error) {
	var dtl *DL.Block
	var err error
	var readAddr int64
	var dataBlockSize uint64

	// Byte slice order conversion
	byteOrder := c.block.ByteOrder()

	// Get configuration
	size := c.block.SignalBytesRange()
	rowSize := int64(c.ChannelGroup.Data.DataBytes)
	dataType := c.block.LoadDataType(int(size))

	readAddr = c.datablockAddress(c.DataGroup.Link.Data)
	length, err := blocks.GetLength(c.mf4.File, readAddr)
	if err != nil {
		return nil, err
	}
	buffer := make([]byte, length)

	// Create a large buffer to read the entire data at once
	if isDataList {
		dtl, err = DL.New(c.mf4.File, c.mf4.MdfVersion(), c.DataGroup.Link.Data)
		if err != nil {
			return nil, err
		}
	} else {
		_, err = io.ReadFull(c.mf4.File, buffer)
		if err != nil {
			return nil, err
		}
	}

	measure := make([]interface{}, 0, c.ChannelGroup.Data.CycleCount)

	var offset, target uint64
	k := 0
	pos := int64(c.block.Data.ByteOffset)
	for i := uint64(0); i < c.ChannelGroup.Data.CycleCount; i++ {
		if i == target && isDataList {
			// Next list
			if k == len(dtl.Link.Data) && dtl.Next() != 0 {
				dtl, err = DL.New(c.mf4.File, c.mf4.MdfVersion(), dtl.Next())
				if err != nil {

					return nil, err
				}
				k = 0
			}
			//Next Data
			offset = dtl.DataSectionLength(k)
			target += offset

			readAddr = c.datablockAddress(dtl.Link.Data[k])

			if _, err = c.mf4.File.Seek(readAddr, io.SeekStart); err != nil {
				return nil, err
			}
			length, err := blocks.GetLength(c.mf4.File, readAddr)
			if err != nil {
				return nil, err
			}
			if length != dataBlockSize {
				buffer = make([]byte, length)
				dataBlockSize = length
			}

			_, err = io.ReadFull(c.mf4.File, buffer)
			if err != nil {
				return nil, err
			}
			pos = int64(c.block.Data.ByteOffset)
			k += 1
		}
		// Safely slice the buffer
		data := buffer[pos : pos+int64(size)]

		value, err := parseSignalMeasure2(data, byteOrder, dataType)
		if err != nil {
			return nil, err
		}
		measure = append(measure, value)

		pos += rowSize

	}

	return measure, nil
}

func (c *Channel) readMeasureRow(bufValue []byte) (interface{}, error) {
	size := c.block.SignalBytesRange()
	data := make([]byte, size)
	byteOrder := c.block.ByteOrder()
	dataType := c.block.LoadDataType(len(data))
	buf := bytes.NewBuffer(bufValue)
	return parseSignalMeasure(buf, byteOrder, dataType)
}

// readMeasureFromSDBlock return extract sample measure from SDBlock or a list of SDBlocks
func (c *Channel) readMeasureFromSDBlock(isDataList bool) ([]interface{}, error) {
	var measure []interface{}
	var length uint32
	var dtl *DL.Block
	var err error
	var sdb *SD.Block

	address := c.block.Link.Data

	headerLen := blocks.HeaderSize
	next := int64(headerLen)

	// byte slice order convert
	byteOrder := c.block.ByteOrder()

	if isDataList {
		dtl, err = DL.New(c.mf4.File, c.mf4.MdfVersion(), address)
		if err != nil {
			return nil, err
		}
		sdb = SD.New(c.mf4.File, dtl.Link.Data[0])
	} else {
		sdb = SD.New(c.mf4.File, address)
	}

	size := c.block.SignalBytesRange()
	data := make([]byte, size)
	dataType := c.block.LoadDataType(len(data))

	buflenght := make([]byte, 4)

	target := int64(sdb.Header.Length)
	k := 0
	for {
		if target >= next && isDataList {
			// Next list
			if k == len(dtl.Link.Data) && dtl.Next() != 0 {
				dtl, err = DL.New(c.mf4.File, c.mf4.MdfVersion(), dtl.Next())
				if err != nil {
					return nil, err
				}
				k = 0
			}
			if k+1 > len(dtl.Link.Data) {
				break
			}
			sdb := SD.New(c.mf4.File, dtl.Link.Data[k])
			target = int64(sdb.Header.Length)
			next = int64(headerLen)
			k += 1
		}
		if !isDataList && next >= target {
			break
		}

		// Read the Link section from the binary file
		if err := binary.Read(c.mf4.File, binary.LittleEndian, &buflenght); err != nil {
			return nil, fmt.Errorf("error reading buflenght section sdblock: %s", err)
		}

		length = binary.LittleEndian.Uint32(buflenght)

		bufValue := make([]byte, length)
		if err := binary.Read(c.mf4.File, binary.LittleEndian, &bufValue); err != nil {
			return nil, fmt.Errorf("error reading bufValue section sdblock: %s", err)
		}

		buf := bytes.NewBuffer(bufValue)
		value, err := parseSignalMeasure(buf, byteOrder, dataType)
		if err != nil {
			return nil, err
		}

		measure = append(measure, value)

		next += int64(len(buflenght)) + int64(length)
	}

	return measure, nil
}

// extractSample returns a array with sample extracted from datablock based on
// header id
func (c *Channel) extractSample() ([]interface{}, error) {
	if c.block.IsVLSD() {
		return c.readVLSDSample()
	}
	return c.readFixedLenghtSample()
}

// readFixedLenghtSample extracts samples from channel type Fixed Length Signal
// Data
func (c Channel) readFixedLenghtSample() ([]interface{}, error) {
	blockHeader, err := blocks.GetHeaderID(c.mf4.File, c.DataGroup.Link.Data)
	if err != nil {
		return nil, err
	}

	switch blockHeader {
	case blocks.DtID, blocks.DvID:
		return c.readSingleDataBlock()
	case blocks.DlID:
		return c.readDataList()
	default:
		fmt.Println(blockHeader)
		return nil, fmt.Errorf("package not ready to read this file")
	}
}

// readVLSDSample extracts samples from channel type Variable Length Signal Data
// (VLSD)
func (c *Channel) readVLSDSample() ([]interface{}, error) {
	var sample []interface{}
	var blockHeader string
	var err error

	if c.DataGroup.Data.RecIDSize != 0 {
		blockHeader, err = blocks.GetHeaderID(c.mf4.File, c.DataGroup.Link.Data)
	} else {
		blockHeader, err = blocks.GetHeaderID(c.mf4.File, c.block.Link.Data)
	}

	if err != nil {
		return nil, err
	}

	switch blockHeader {
	case blocks.DtID:
		return c.readDTBlockUnsorted()
	case blocks.SdID:
		return c.readSDBlock()
	case blocks.DlID:
		return c.readListOfSDBlock()
	case blocks.DzID:
		fmt.Println(blockHeader)
		return nil, fmt.Errorf("package not ready to read this file")
	case blocks.CgID:
		return c.readSingleDataBlockVLSD()
	default:
		fmt.Println(blockHeader)
		return sample, fmt.Errorf("package not ready to read this file")
	}

}

// Sample returns a array with the measures of the channel applying conversion
// block on it
func (c *Channel) Sample() ([]interface{}, error) {
	var sample []interface{}
	var err error

	if c.CachedSamples != nil {
		if !c.isConverted {
			c.applyConversion(&c.CachedSamples)
			c.isConverted = true
		}
		return c.CachedSamples, nil
	}

	sample, err = c.extractSample()
	if err != nil {
		return nil, err
	}

	c.applyConversion(&sample)
	if !c.mf4.ReadOptions.MemoryOptimized {
		c.CachedSamples = sample
	}
	return sample, nil
}

// RawSample returns a array with the measures of the channel not applying
// conversion block on it
func (c *Channel) RawSample() ([]interface{}, error) {
	sample, err := c.extractSample()
	if err != nil {
		return nil, err
	}
	return sample, nil
}

// readSDBlock returns measure from SDBlock
func (c *Channel) readSDBlock() ([]interface{}, error) {
	return c.readMeasureFromSDBlock(false)
}

func (c *Channel) readDTBlockUnsorted() ([]interface{}, error) {
	return nil, nil
}

// readListOfSDBlock returns measures from a DLBlock of SDBlock
func (c *Channel) readListOfSDBlock() ([]interface{}, error) {
	return c.readMeasureFromSDBlock(true)
}

// readSingleDataBlock returns measure from DTBlock
func (c *Channel) readSingleDataBlock() ([]interface{}, error) {
	return c.readSingleChannel(false)
}

// readSingleDataBlock returns measure from DTBlock
func (c *Channel) readSingleDataBlockVLSD() ([]interface{}, error) {
	return nil, nil
}

// readDataList returns measure from DLBlock
func (c *Channel) readDataList() ([]interface{}, error) {
	return c.readSingleChannel(true)
}

// signalValueAddress returns the offset from the signal in the DTBlock
func (c *Channel) signalValueAddress(dataAddress int64) int64 {
	return int64(blocks.HeaderSize) + dataAddress
}

// signalValueAddress returns the offset from the signal in the DTBlock
func (c *Channel) datablockAddress(dataAddress int64) int64 {
	return dataAddress
}

func (c *Channel) applyConversion(sample *[]interface{}) {
	if c.Conversion == nil {
		return
	}

	c.Conversion.Apply(sample)
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
	return int64(c.getRecordIDSize()) + int64(c.getDataBytes())
}

func (c *Channel) getRecordIDSize() uint8 {
	return c.DataGroup.RecordIDSize()
}

func (c *Channel) readRecordID(recordArray []byte) int64 {
	switch c.getRecordIDSize() {
	case 1:
		return int64(recordArray[0])
	case 2:
		return int64(binary.LittleEndian.Uint16(recordArray))
	case 4:
		return int64(binary.LittleEndian.Uint32(recordArray))
	case 8:
		return int64(binary.LittleEndian.Uint64(recordArray))
	default:
		return 0
	}
}

func (c *Channel) getDataBytes() uint32 {
	return c.ChannelGroup.GetDataBytes()
}

func (c *Channel) getInvalidationBitPos() uint32 {
	return c.block.InvalBitPos()
}
