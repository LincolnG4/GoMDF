package mf4

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
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

type ChannelGroup struct {
	Block      *CG.Block
	Channels   map[string]*Channel
	DataGroup  *DG.Block
	SourceInfo SI.SourceInfo
	Comment    string
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

	// describes the source of an acquisition mode or of a signal
	SourceInfo SI.SourceInfo

	// additional information about the channel. Can be 'nil'
	Comment string

	//pointer to mf4 file
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

// readMeasure return extract sample measure from DTBlock//DLBlock
func (c *Channel) readMeasure(file *os.File, version uint16, isDataList bool) ([]interface{}, error) {
	// init
	cn := c.block
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

// readMeasure return extract sample measure from DTBlock//DLBlock
func (c *Channel) readMeasureFromSDBlock(file *os.File) ([]interface{}, error) {
	var measure []interface{}
	var length uint32

	address := c.block.Link.Data

	b := SD.New(file, address)
	a := blocks.HeaderSize

	size := c.block.SignalBytesRange()
	data := make([]byte, size)
	dataType := c.block.LoadDataType(len(data))
	byteOrder := c.block.ByteOrder()

	buflenght := make([]byte, 4)
	for a < b.Header.Length {
		// Read the Link section from the binary file
		if err := binary.Read(file, binary.LittleEndian, &buflenght); err != nil {
			return nil, fmt.Errorf("error reading link section sdblock: %s", err)
		}

		length = binary.LittleEndian.Uint32(buflenght)
		bufValue := make([]byte, length)
		if err := binary.Read(file, binary.LittleEndian, &bufValue); err != nil {
			return nil, fmt.Errorf("error reading link section sdblock: %s", err)
		}

		buf := bytes.NewBuffer(bufValue)
		value, err := parseSignalMeasure(buf, byteOrder, dataType)
		if err != nil {
			return nil, err
		}

		measure = append(measure, value)
		a += uint64(len(buflenght)) + uint64(length)
	}

	return measure, nil
}

// readMeasure return extract sample measure from DTBlock//DLBlock
func (c *Channel) readMeasureFromListOfSDBlock(file *os.File) ([]interface{}, error) {
	var measure []interface{}
	var length uint32
	//var offset, target uint64

	headerLen := blocks.HeaderSize
	next := int64(headerLen)
	address := c.block.Link.Data

	// byte slice order convert
	byteOrder := c.block.ByteOrder()

	dtl, err := DL.New(file, c.mf4.MdfVersion(), address)
	if err != nil {
		return nil, err
	}

	size := c.block.SignalBytesRange()
	data := make([]byte, size)
	dataType := c.block.LoadDataType(len(data))

	buflenght := make([]byte, 4)
	sdb := SD.New(file, dtl.Link.Data[0])
	target := int64(sdb.Header.Length)
	k := 0
	for {
		if target >= next {
			// Next list
			if k == len(dtl.Link.Data) && dtl.Next() != 0 {
				dtl, err = DL.New(file, c.mf4.MdfVersion(), dtl.Next())
				if err != nil {
					return nil, err
				}
				k = 0
			}
			if k+1 > len(dtl.Link.Data) {
				break
			}
			sdb := SD.New(file, dtl.Link.Data[k])
			target = int64(sdb.Header.Length)
			next = int64(headerLen)
			k += 1
		}

		// Read the Link section from the binary file
		if err := binary.Read(file, binary.LittleEndian, &buflenght); err != nil {
			return nil, fmt.Errorf("error reading buflenght section sdblock: %s", err)
		}

		length = binary.LittleEndian.Uint32(buflenght)

		bufValue := make([]byte, length)
		if err := binary.Read(file, binary.LittleEndian, &bufValue); err != nil {
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
	var sample []interface{}
	blockHeader, err := blocks.GetHeaderID(c.mf4.File, c.DataGroup.Link.Data)
	if err != nil {
		return nil, err
	}

	switch blockHeader {
	case blocks.DtID, blocks.DvID:
		sample, err = c.readSingleDataBlock(c.mf4.File)
	case blocks.DlID:
		sample, err = c.readDataList(c.mf4.File, c.mf4.MdfVersion())
	default:
		return nil, fmt.Errorf("package not ready to read this file")
	}
	return sample, err
}

// readVLSDSample extracts samples from channel type Variable Length Signal Data
// (VLSD)
func (c *Channel) readVLSDSample() ([]interface{}, error) {
	var sample []interface{}
	var err error

	blockHeader, err := blocks.GetHeaderID(c.mf4.File, c.block.Link.Data)
	if err != nil {
		return nil, err
	}

	switch blockHeader {
	case blocks.SdID:
		return c.readSDBlock(c.mf4.File)
	case blocks.DlID:
		return c.readListOfSDBlock(c.mf4.File)
	case blocks.DzID:
		return nil, fmt.Errorf("package not ready to read this file")
	default:
		return sample, fmt.Errorf("package not ready to read this file")
	}

}

// Sample returns a array with the measures of the channel applying conversion
// block on it
func (c *Channel) Sample() ([]interface{}, error) {
	sample, err := c.RawSample()
	if err != nil {
		return nil, err
	}

	c.applyConversion(&sample)
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
func (c *Channel) readSDBlock(file *os.File) ([]interface{}, error) {
	return c.readMeasureFromSDBlock(file)
}

// readListOfSDBlock returns measures from a DLBlock of SDBlock
func (c *Channel) readListOfSDBlock(file *os.File) ([]interface{}, error) {
	return c.readMeasureFromListOfSDBlock(file)
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
	return int64(blocks.HeaderSize) + dataAddress + int64(c.DataGroup.GetRecordIDSize()) + int64(c.block.Data.ByteOffset)
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
	return int64(c.getRecordID()) + int64(c.getDataBytes())
}

func (c *Channel) getRecordID() uint8 {
	return c.DataGroup.GetRecordIDSize()
}

func (c *Channel) getDataBytes() uint32 {
	return c.ChannelGroup.GetDataBytes()
}

func (c *Channel) getInvalidationBitPos() uint32 {
	return c.block.InvalBitPos()
}
