package mf4

import (
	"fmt"
	"sync"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/conversion"
)

// ChannelType classifies a channel (cn_type).
type ChannelType uint8

const (
	FixedLength   ChannelType = blocks.CNFixedLength
	VLSD          ChannelType = blocks.CNVLSD          // variable-length signal data
	Master        ChannelType = blocks.CNMaster        // master (e.g. time) channel
	VirtualMaster ChannelType = blocks.CNVirtualMaster // master derived from the record index
	Sync          ChannelType = blocks.CNSync
	MaxLength     ChannelType = blocks.CNMaxLength
	VirtualData   ChannelType = blocks.CNVirtualData
)

func (t ChannelType) String() string {
	switch t {
	case FixedLength:
		return "fixed-length"
	case VLSD:
		return "vlsd"
	case Master:
		return "master"
	case VirtualMaster:
		return "virtual-master"
	case Sync:
		return "sync"
	case MaxLength:
		return "max-length"
	case VirtualData:
		return "virtual-data"
	}
	return fmt.Sprintf("channel-type(%d)", uint8(t))
}

// DataType is the raw MDF channel data type (cn_data_type).
type DataType uint8

const (
	UintLE        DataType = blocks.DTUintLE
	UintBE        DataType = blocks.DTUintBE
	IntLE         DataType = blocks.DTIntLE
	IntBE         DataType = blocks.DTIntBE
	FloatLE       DataType = blocks.DTFloatLE
	FloatBE       DataType = blocks.DTFloatBE
	StringLatin1  DataType = blocks.DTStringLatin
	StringUTF8    DataType = blocks.DTStringUTF8
	StringUTF16LE DataType = blocks.DTStringUTF16LE
	StringUTF16BE DataType = blocks.DTStringUTF16BE
	ByteArray     DataType = blocks.DTByteArray
	MIMESample    DataType = blocks.DTMIMESample
	MIMEStream    DataType = blocks.DTMIMEStream
	CANopenDate   DataType = blocks.DTCANopenDate
	CANopenTime   DataType = blocks.DTCANopenTime
	ComplexLE     DataType = blocks.DTComplexLE
	ComplexBE     DataType = blocks.DTComplexBE
)

// SourceType classifies a signal or acquisition source (si_type).
type SourceType uint8

const (
	SourceOther SourceType = iota
	SourceECU
	SourceBus
	SourceIO
	SourceTool
	SourceUser
)

// BusType is the bus kind of a source (si_bus_type).
type BusType uint8

const (
	BusNone BusType = iota
	BusOther
	BusCAN
	BusLIN
	BusMOST
	BusFlexRay
	BusKLine
	BusEthernet
	BusUSB
)

// SourceInfo describes the source of a channel or channel group.
type SourceInfo struct {
	Name    string
	Path    string
	Comment string
	Type    SourceType
	Bus     BusType
}

// ChannelGroup is a set of channels sharing one record layout and master
// channel.
type ChannelGroup struct {
	Name        string
	Comment     string
	Source      SourceInfo
	RecordCount uint64

	file     *File
	dg       *dataGroup
	cg       *blocks.CG
	channels []*Channel
	master   *Channel

	masterOnce sync.Once
	masterVals []float64
	masterErr  error
}

// Channels returns the group's channels in file order (master included).
func (g *ChannelGroup) Channels() []*Channel { return g.channels }

// Master returns the group's master channel (typically time), or nil if
// the group has none.
func (g *ChannelGroup) Master() *Channel { return g.master }

// Channel returns the named channel of this group.
func (g *ChannelGroup) Channel(name string) (*Channel, bool) {
	for _, c := range g.channels {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}

// Channel describes one signal.
type Channel struct {
	Name    string
	Unit    string
	Comment string
	Source  SourceInfo

	Type       ChannelType
	DataType   DataType
	BitCount   uint32
	Conversion ConversionInfo
	// IsArray marks channels whose composition is a channel array (CA);
	// they are exposed as raw byte columns.
	IsArray bool

	group  *ChannelGroup
	parent *Channel // enclosing structure channel, if a component
	cn     *blocks.CN
	cnAddr int64
	cc     *blocks.CC

	convOnce sync.Once
	conv     *conversion.Conversion
	convErr  error
}

// Group returns the channel group this channel belongs to.
func (c *Channel) Group() *ChannelGroup { return c.group }

func (c *Channel) String() string {
	return fmt.Sprintf("%s [%s] (%s, %d bits)", c.Name, c.Unit, c.Type, c.BitCount)
}
