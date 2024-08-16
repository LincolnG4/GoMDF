package CN

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/LincolnG4/GoMDF/blocks"
	"github.com/LincolnG4/GoMDF/blocks/CC"
	"github.com/LincolnG4/GoMDF/blocks/TX"
)

type Block struct {
	Header blocks.Header
	Link   Link
	Data   Data
}

type Link struct {
	Next         int64
	Composition  int64
	TxName       int64
	SiSource     int64
	CcConvertion int64
	Data         int64
	MdUnit       int64
	MdComment    int64
	//Version 4.1
	AtReference int64
	//Version 4.1
	DefaultX [3]int64
}

type Data struct {
	Type        uint8
	SyncType    uint8
	DataType    uint8
	BitOffset   uint8
	ByteOffset  uint32
	BitCount    uint32
	Flags       uint32
	InvalBitPos uint32
	Precision   uint8
	// Use [1]byte for versions >= 4.1
	Reserved uint8
	//Version 4.1
	AttachmentCount uint16
	ValRangeMin     float64
	ValRangeMax     float64
	LimitMin        float64
	LimitMax        float64
	LimitExtMin     float64
	LimitExtMax     float64
}

const (
	UnsignedIntegerLE uint8 = iota // 0
	UnsignedIntegerBE              // 1
	SignedIntegerLE                // 2
	SignedIntegerBE                // 3
	IEEE754FloatLE                 // 4
	IEEE754FloatBE                 // 5
	StringSBC                      // 6
	StringUTF8                     // 7
	StringUTF16LE                  // 8
	StringUTF16BE                  // 9
	ByteArrayUnknown               // 10
	MIMESample                     // 11
	MIMEStream                     // 12
	CANopenDate                    // 13
	CANopenTime                    // 14
	// Version 4.2 only
	ComplexNumberLE // 15
	ComplexNumberBE // 16
)

// Type
const (
	FixedLenght uint8 = iota
	//variable length signal data
	VLSD
	Master
	VirtualMaster
	Synchronization
	// Version 4.1 only
	MaximumLengthData
	// Version 4.1 only
	VirtualData
)

func New(file *os.File, version uint16, startAddress int64) (*Block, error) {
	var b Block
	var err error

	// Initialize the header
	b.Header = blocks.Header{}

	// Read the header directly
	b.Header, err = blocks.GetHeader(file, startAddress, blocks.CnID)
	if err != nil {
		return b.BlankBlock(), err
	}

	// Calculate the size of the Link Block and read the binary data directly into the buffer
	linkBlockSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	linkBuffer := make([]byte, linkBlockSize)
	if _, err := io.ReadFull(file, linkBuffer); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading link section chblock: %s", err)
	}

	// Parse Link fields from the buffer
	linkFields := make([]int64, linkBlockSize/8)
	for i := 0; i < len(linkFields); i++ {
		linkFields[i] = int64(binary.LittleEndian.Uint64(linkBuffer[i*8 : (i+1)*8]))
	}

	// Handle version-specific Link fields
	if version >= 420 {
		linkFields = append(linkFields, blocks.ReadInt64FromBinary(file))
	}

	// Populate Link struct fields
	b.Link = Link{
		Next:         linkFields[0],
		Composition:  linkFields[1],
		TxName:       linkFields[2],
		SiSource:     linkFields[3],
		CcConvertion: linkFields[4],
		Data:         linkFields[5],
		MdUnit:       linkFields[6],
		MdComment:    linkFields[7],
	}

	// Calculate the size of the Data Block and read directly
	dataBlockSize := blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	dataBuffer := make([]byte, dataBlockSize)
	if _, err := io.ReadFull(file, dataBuffer); err != nil {
		return b.BlankBlock(), fmt.Errorf("error reading data chblock: %s", err)
	}

	// Populate the Data struct from the binary data
	if err := binary.Read(bytes.NewReader(dataBuffer), binary.LittleEndian, &b.Data); err != nil {
		return b.BlankBlock(), fmt.Errorf("error parsing data block: %s", err)
	}

	// Handle version-specific fields if version >= 4.10
	if version >= 410 {
		for i := 0; i < int(b.Data.AttachmentCount); i++ {
			b.Link.AtReference = linkFields[8]
		}

		if b.Data.Flags == 12 {
			b.Link.DefaultX = [3]int64{linkFields[9], linkFields[10], linkFields[11]}
		}
	}

	return &b, nil
}

// Conversion return Conversion structs that hold the formula to convert
// raw sample to desired value.
func (b *Block) Conversion(file *os.File, channelDataType uint8) (CC.Conversion, error) {
	cc, err := b.NewConversion(file)
	if err != nil {
		return nil, err
	}
	if cc == nil {
		return nil, nil
	}
	return cc.Get(file, channelDataType)
}

// NewConversion create a new CCBlock according to the Link.CcConvertion field.
func (b *Block) NewConversion(file *os.File) (*CC.Block, error) {
	if b.Link.CcConvertion == 0 {
		return nil, nil
	}
	return CC.New(file, b.Link.CcConvertion)
}

// Master returns a pointer to the master. It returns 'nil' if the channel is
// the master.
func MasterPointer() *Block {
	return &Block{}
}

func (b *Block) LoadDataType(lenSize int) interface{} {
	var dtype interface{} = 0
	switch b.DataType() {
	case UnsignedIntegerLE, UnsignedIntegerBE:
		switch lenSize {
		case 1:
			dtype = uint8(0)
		case 2:
			dtype = uint16(0)
		case 4:
			dtype = uint32(0)
		case 8:
			dtype = uint64(0)
		default:
			dtype = uint64(0)
		}
	case SignedIntegerLE, SignedIntegerBE:
		switch lenSize {
		case 1:
			dtype = int8(0)
		case 2:
			dtype = int16(0)
		case 4:
			dtype = int32(0)
		case 8:
			dtype = int64(0)
		default:
			dtype = int64(0)
		}

	case IEEE754FloatLE, IEEE754FloatBE:
		switch lenSize {
		case 4:
			dtype = float32(0)
		case 8:
			dtype = float64(0)
		default:
			dtype = float64(0)
		}
	case StringSBC, StringUTF8, StringUTF16LE, StringUTF16BE:
		dtype = ""
	case ByteArrayUnknown:
		dtype = []byte{}
	}
	return dtype
}

func LittleEndianArray() []int {
	return []int{0, 2, 4, 8, 15}
}

func (b *Block) ByteOrder() binary.ByteOrder {
	//Data Types that are Little Endian: 0, 2, 4, 8, 15
	if slices.Contains(LittleEndianArray(), int(b.DataType())) {
		return binary.LittleEndian
	}
	return binary.BigEndian
}

// SignalBytesRange is number of Bytes required to store (cn_bit_count + cn_bit_offset) bits
func (b *Block) SignalBytesRange() uint32 {
	return (b.Data.BitCount + uint32(b.Data.BitOffset)) / 8
}

func (b *Block) IsComposed() bool {
	return b.Link.Composition != 0
}

func (b *Block) IsAllValuesInvalid() bool {
	// Bit 0 corresponds to the "all values invalid" flag
	// Check if bit 0 is set
	return blocks.IsBitSet(int(b.Data.Flags), 0)
}

func (b *Block) IsAllValuesValid() bool {
	if !blocks.IsBitSet(int(b.Data.Flags), 0) && !blocks.IsBitSet(int(b.Data.Flags), 1) {
		return true
	}
	return false
}

func (b *Block) InvalBitPos() uint32 {
	return b.Data.InvalBitPos
}

// IsVLSD returns `true` if channel is variable length signal data. Otherwise it
// returns `false`
func (b *Block) IsVLSD() bool {
	return b.IotaType() == VLSD
}

// IsVLSD returns `true` if channel is the master. Otherwise it returns `false`
func (b *Block) IsMaster() bool {
	return b.IotaType() == Master
}

func (b *Block) Type() string {
	switch b.IotaType() {
	case FixedLenght:
		return "FixedLenght"
	case VLSD:
		return "VLSD"
	case Master:
		return "Master"
	case VirtualMaster:
		return "VirtualMaster"
	case Synchronization:
		return "Synchronization"
	case MaximumLengthData:
		return "MaximumLengthData"
	case VirtualData:
		return "VirtualData"
	default:
		return ""
	}
}

func (b *Block) ChannelName(f *os.File) string {
	t, err := TX.GetText(f, b.TxName())
	if err != nil {
		return ""
	}

	return t
}

func (b *Block) TxName() int64 {
	return b.Link.TxName
}

func (b *Block) CommentMd() int64 {
	return b.Link.MdComment
}

func (b *Block) IotaType() uint8 {
	return b.Data.Type
}

func (b *Block) SyncType() uint8 {
	return b.Data.SyncType
}

func (b *Block) DataType() uint8 {
	return b.Data.DataType
}

func (b *Block) Next() int64 {
	return b.Link.Next
}

func (b *Block) BlankBlock() *Block {
	return &Block{
		Header: blocks.Header{
			ID:        blocks.SplitIdToArray(blocks.CnID),
			Reserved:  [4]byte{},
			Length:    blocks.CnblockSize,
			LinkCount: 0,
		},
		Link: Link{},
		Data: Data{},
	}
}
