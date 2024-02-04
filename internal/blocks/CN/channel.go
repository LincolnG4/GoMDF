package CN

import (
	"encoding/binary"
	"fmt"
	"os"
	"slices"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/blocks/CC"
	"github.com/LincolnG4/GoMDF/internal/blocks/TX"
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
	// Version 4.2
	ComplexNumberLE // 15
	ComplexNumberBE // 16
)

func New(file *os.File, version uint16, startAdress int64) *Block {
	var b Block
	var err error

	b.Header = blocks.Header{}

	b.Header, err = blocks.GetHeader(file, startAdress, blocks.CnID)
	if err != nil {
		return b.BlankBlock()
	}

	//Calculates size of Link Block
	blockSize := blocks.CalculateLinkSize(b.Header.LinkCount)
	buffEach := make([]byte, blockSize)
	// Read the Link section from the binary file
	if err := binary.Read(file, binary.LittleEndian, &buffEach); err != nil {
		fmt.Println("error reading link section chblock:", err)
	}

	// Populate the Link fields dynamically based on version
	linkFields := []int64{}
	for i := 0; i < len(buffEach)/8; i++ {
		linkFields = append(linkFields, int64(binary.LittleEndian.Uint64(buffEach[i*8:(i+1)*8])))
	}

	// Handle version-specific fields in Link based on the version
	if version >= 420 {
		linkFields = append(linkFields, blocks.ReadInt64FromBinary(file))
	}

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
	//Calculates size of Data Block
	blockSize = blocks.CalculateDataSize(b.Header.Length, b.Header.LinkCount)
	buf := blocks.LoadBuffer(file, blockSize)

	// Create a buffer based on block size
	if err := binary.Read(buf, binary.LittleEndian, &b.Data); err != nil {
		fmt.Println("error reading data chblock:", err)
	}

	if version < 410 {
		return &b
	}

	//Handling versions >= 4.10
	for i := 0; i < int(b.Data.AttachmentCount); i++ {
		b.Link.AtReference = linkFields[8]
	}

	if b.Data.Flags == 12 {
		b.Link.DefaultX = [3]int64{linkFields[9], linkFields[10], linkFields[11]}
	}
	return &b
}

// GetConversion return Conversion structs that hold the formula to convert
// raw sample to desired value.
func (b *Block) GetConversion(file *os.File, channelDataType uint8) CC.Conversion {
	cc := b.NewConversion(file)
	if cc == nil {
		return nil
	}
	return cc.Get(file, channelDataType)
}

// NewConversion create a new CCBlock according to the Link.CcConvertion field.
func (b *Block) NewConversion(file *os.File) *CC.Block {
	if b.Link.CcConvertion == 0 {
		return nil
	}
	return CC.New(file, b.Link.CcConvertion)
}

func (b *Block) LoadDataType(lenSize int) interface{} {
	var dtype interface{}

	switch b.GetDataType() {
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

func (b *Block) IsLittleEndian() bool {
	//Data Types that are Little Endian: 0, 2, 4, 8, 15
	littleEndianFormats := []int{0, 2, 4, 8, 15}

	return slices.Contains(littleEndianFormats, int(b.GetDataType()))
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

func (b *Block) GetChannelName(f *os.File) string {
	t, err := TX.GetText(f, b.getTxName())
	if err != nil {
		return ""
	}

	return t
}

func (b *Block) getTxName() int64 {
	return b.Link.TxName
}

func (b *Block) GetCommentMd() int64 {
	return b.Link.MdComment
}

func (b *Block) GetType() uint8 {
	return b.Data.Type
}

func (b *Block) GetSyncType() uint8 {
	return b.Data.SyncType
}

func (b *Block) GetDataType() uint8 {
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
