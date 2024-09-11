package mf4

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/LincolnG4/GoMDF/blocks"
	"github.com/LincolnG4/GoMDF/blocks/CC"
	"github.com/LincolnG4/GoMDF/blocks/CG"
	"github.com/LincolnG4/GoMDF/blocks/CN"
	"github.com/LincolnG4/GoMDF/blocks/DG"
	"github.com/LincolnG4/GoMDF/blocks/DL"
	"github.com/LincolnG4/GoMDF/blocks/DZ"
	"github.com/LincolnG4/GoMDF/blocks/HL"
	"github.com/LincolnG4/GoMDF/blocks/SI"
)

type ChannelGroup struct {
	Block       *CG.Block
	Channels    map[string]*Channel
	DataGroup   *DG.Block
	SourceInfo  SI.SourceInfo
	Comment     string
	IsVLSDBlock bool
}

type Channel struct {
	//channel's name
	Name string

	//conversion formula to convert the raw values to physical values with a
	//physical unit
	Conversion CC.Conversion

	//channel type
	Type string

	//pointer to the master channel of the channel group.
	//A 'nil' value indicates that this channel itself is the master.
	Master *Channel

	//pointer to data group
	DataGroup *DataGroup

	//data group's index
	DataGroupIndex int

	//pointer to channel group
	ChannelGroup *CG.Block

	//channel group's index
	ChannelGroupIndex int

	//unsorted channels mapped
	isUnsorted bool

	//describes the source of an acquisition mode or of a signal
	SourceInfo SI.SourceInfo

	//additional information about the channel. Can be 'nil'
	Comment string

	//Samples are cached in memory if file was set with MemoryOptimized is true
	CachedSamples []interface{}

	//Conversion applied
	isConverted bool

	//pointer to mf4 file
	mf4 *MF4

	//pointer to the CNBLOCK
	block *CN.Block

	channelReader *ChannelReader

	startAddress int64
}

type ChannelReader struct {
	//Byte order conversion (LittleEndian/BigEndian)
	ByteOrder binary.ByteOrder

	//Number of bits per row
	SizeMeasureRow int64

	DataType interface{}

	DataAddress int64

	MeasureBuffer []byte

	//Length of the row in block
	RowSize int64

	//Offset in the row
	StartOffset int64
}

func (cn *ChannelReader) loadBuffer(f *os.File) error {
	length, err := blocks.GetLength(f, cn.DataAddress)
	if err != nil {
		return err
	}

	if length != uint64(len(cn.MeasureBuffer)) {
		cn.MeasureBuffer = make([]byte, length)
	}

	return nil
}

func readBlockFromFile(f *os.File, dataAddress int64, buf []byte) error {
	var err error
	if _, err = f.Seek(dataAddress+int64(blocks.HeaderSize), io.SeekStart); err != nil {
		return err
	}

	_, err = io.ReadFull(f, buf)
	if err != nil {
		return err
	}
	return nil
}

func (cn *ChannelReader) readBlockToMemory(f *os.File) error {
	err := cn.loadBuffer(f)
	if err != nil {
		return err
	}

	return readBlockFromFile(f, cn.DataAddress, cn.MeasureBuffer)
}

func (cn *ChannelReader) readDatablock(f *os.File, pos int64) (interface{}, error) {
	//check if end of datablock
	if pos+int64(cn.SizeMeasureRow) > int64(len(cn.MeasureBuffer)) {
		length, err := blocks.GetLength(f, cn.DataAddress)
		if err != nil {
			return nil, err
		}

		buf := make([]byte, length)
		err = readBlockFromFile(f, cn.DataAddress, buf)
		if err != nil {
			return nil, err
		}
		cn.MeasureBuffer = append(cn.MeasureBuffer, buf...)
	}
	return parseSignalMeasure(cn.MeasureBuffer[pos:pos+int64(cn.SizeMeasureRow)], cn.ByteOrder, cn.DataType)
}

func (c *Channel) loadChannelReader(addr int64) *ChannelReader {
	size := c.block.SignalBytesRange()

	return &ChannelReader{
		ByteOrder:      c.block.ByteOrder(),
		SizeMeasureRow: int64(size),
		DataType:       c.block.LoadDataType(int(size)),
		DataAddress:    addr,
		StartOffset:    int64(c.block.Data.ByteOffset),
		RowSize:        int64(c.ChannelGroup.Data.DataBytes),
		MeasureBuffer:  c.DataGroup.CachedDataGroup,
	}
}

func (c *Channel) readDataList(measure *[]interface{}) error {
	dtl, err := DL.New(c.mf4.File, c.mf4.MdfVersion(), c.channelReader.DataAddress)
	if err != nil {
		return err
	}

	id, err := blocks.GetHeaderID(c.mf4.File, dtl.Link.Data[0])
	if err != nil {
		return err
	}

	target := len(dtl.Link.Data)
	i := 0
	for i < target {
		c.channelReader = c.newChannelReader(dtl.Link.Data[i])
		if err != nil {
			return err
		}

		err = c.extractSample(id, measure)
		if err != nil {
			return err
		}
		i++

		if i == target && dtl.Next() != 0 {
			dtl, err = DL.New(c.mf4.File, c.mf4.MdfVersion(), dtl.Next())
			if err != nil {
				return err
			}
			target = len(dtl.Link.Data)
			i = 0
		}
	}

	return nil
}

func (c *Channel) newChannelReader(addr int64) *ChannelReader {
	return c.loadChannelReader(addr)
}

// readMeasure return extract sample measure from DTBlock//DLBlock with fixed
// lenght
func (c *Channel) readDT(measure *[]interface{}) error {
	var err error

	if c.DataGroup.CachedDataGroup == nil {
		err = c.channelReader.readBlockToMemory(c.mf4.File)
		if err != nil {
			return err
		}
	}

	pos := c.channelReader.StartOffset
	for i := uint64(0); i < c.ChannelGroup.Data.CycleCount; i++ {
		if pos >= int64(len(c.channelReader.MeasureBuffer)) {
			return nil
		}

		value, err := c.channelReader.readDatablock(c.mf4.File, pos)
		if err != nil {
			return err
		}

		*measure = append(*measure, value)
		pos += c.channelReader.RowSize
	}

	return nil
}

// readMeasureFromSDBlock return extract sample measure from SDBlock or a list of SDBlocks
func (c *Channel) readSdBlock(measure *[]interface{}) error {
	var err error
	if c.DataGroup.CachedDataGroup == nil {
		err = c.channelReader.readBlockToMemory(c.mf4.File)
		if err != nil {
			return err
		}
	}

	var (
		pos    int64 = 0
		length uint32
		value  interface{}
	)

	length = binary.LittleEndian.Uint32(c.channelReader.MeasureBuffer[pos : pos+4])
	pos += 4

	value, err = parseSignalMeasure(c.channelReader.MeasureBuffer[pos:pos+int64(length)], c.channelReader.ByteOrder, c.channelReader.DataType)
	if err != nil {
		return err
	}

	*measure = append(*measure, value)
	return nil
}

// extractSample returns a array with sample extracted from datablock based on
// header id
func (c *Channel) extractSample(id string, measure *[]interface{}) error {
	if c.block.IsVLSD() {
		return c.readVLSDSample(id, measure)
	}
	return c.readFixedLenghtSample(id, measure)
}

func (c *Channel) readDataZipped(measure *[]interface{}) error {
	var (
		dz  *DZ.Block
		err error
	)

	dz, err = DZ.New(c.mf4.File, c.channelReader.DataAddress)
	if err != nil {
		return err
	}

	c.DataGroup.CachedDataGroup, err = dz.Read()
	if err != nil {
		return err
	}

	c.channelReader.MeasureBuffer = c.DataGroup.CachedDataGroup

	return c.extractSample(dz.BlockTypeModified(), measure)
}

func (c *Channel) readHeaderList(measure *[]interface{}) error {
	var (
		hl  *HL.Block
		err error
	)

	hl, err = HL.New(c.mf4.File, c.channelReader.DataAddress)
	if err != nil {
		return err
	}

	id := blocks.DlID
	c.channelReader = c.newChannelReader(hl.Link.DlFirst)
	if err != nil {
		return err
	}

	return c.extractSample(id, measure)
}

// readVLSDSample extracts samples from channel type Variable Length Signal Data
// (VLSD)
func (c *Channel) readVLSDSample(id string, measure *[]interface{}) error {
	switch id {
	case blocks.DtID:
		fmt.Println(id)
		return nil
	case blocks.SdID:
		return c.readSdBlock(measure)
	case blocks.DlID:
		return c.readDataList(measure)
	case blocks.DzID:
		fmt.Println(id)
		return fmt.Errorf("package not ready to read this file")
	case blocks.CgID:
		return c.readSingleDataBlockVLSD()
	default:
		fmt.Println(id)
		return fmt.Errorf("package not ready to read this file")
	}

}

// readFixedLenghtSample extracts samples from channel type Fixed Length Signal
// Data
func (c Channel) readFixedLenghtSample(blockID string, measure *[]interface{}) error {
	switch blockID {
	case blocks.DtID, blocks.DvID:
		return c.readDT(measure)
	case blocks.DlID:
		return c.readDataList(measure)
	case blocks.DzID:
		return c.readDataZipped(measure)
	case blocks.HlID:
		return c.readHeaderList(measure)
	default:
		fmt.Println(blockID)
		return fmt.Errorf("package not ready to read this file")
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
		}
		return c.CachedSamples, nil
	}

	sample, err = c.RawSample()
	if err != nil {
		return nil, err
	}

	c.applyConversion(&sample)
	if !c.mf4.IsMemoryOptimized() {
		c.CachedSamples = sample
	}
	return sample, nil
}

func parseSignalMeasure(data []byte, byteOrder binary.ByteOrder, dataType interface{}) (interface{}, error) {
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

func (c *Channel) LoadDataAdress() {
	if c.block.Link.Data != 0 {
		c.startAddress = c.block.Link.Data
	} else {
		c.startAddress = c.DataGroup.DataAddress()
	}
}

// RawSample returns a array with the measures of the channel not applying
// conversion block on it
func (c *Channel) RawSample() ([]interface{}, error) {
	c.LoadDataAdress()

	if c.block.Link.Data != 0 {
		c.startAddress = c.block.Link.Data
	} else {
		c.startAddress = c.DataGroup.DataAddress()
	}

	id, err := blocks.GetHeaderID(c.mf4.File, c.startAddress)
	if err != nil {
		return nil, err
	}

	c.channelReader = c.newChannelReader(c.startAddress)
	measure := make([]interface{}, 0, c.ChannelGroup.Data.CycleCount)
	err = c.extractSample(id, &measure)
	if err != nil {
		return nil, err
	}

	return measure, err
}

func (c *Channel) loadDataBlockAddressDataList(cnReader *ChannelReader, i int) (int64, error) {
	dtl, err := DL.New(c.mf4.File, c.mf4.MdfVersion(), cnReader.DataAddress)
	if err != nil {
		return -1, err
	}
	return dtl.Link.Data[i], nil
}

// readSingleDataBlock returns measure from DTBlock
// func (c *Channel) readSingleDataBlock() ([]interface{}, error) {
// 	return c.readSingleDT()
// }

// readSingleDataBlock returns measure from DTBlock
func (c *Channel) readSingleDataBlockVLSD() error {
	return nil
}

// readDataList returns measure from DLBlock
func (c *Channel) readDL(addr int64) ([]interface{}, error) {
	return c.readDL(addr)
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
	c.isConverted = true
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
	return c.DataGroup.block.RecordIDSize()
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
