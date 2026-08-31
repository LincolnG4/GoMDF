package mf4

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

// ChannelIndex identifies a channel within its GroupWriter.
type ChannelIndex int

// wchan is one channel definition of a group being written.
type wchan struct {
	name, unit string
	comment    string
	dataType   uint8 // blocks.DT*
	bitCount   uint32
	byteOffset uint32
	cnType     uint8
	syncType   uint8
	vlsd       bool
	hasConv    bool
	convB      float64 // phys = convB + convA*raw
	convA      float64

	// VLSD signal-data accumulation
	sd *chunkStream

	cnAddr int64 // block address, for cn_data patching
}

// GroupWriter defines one channel group and receives its records.
type GroupWriter struct {
	w        *Writer
	name     string
	comment  string
	noMaster bool

	chans   []*wchan
	recSize int
	count   uint64 // records appended

	data *chunkStream // record bytes

	cgAddr int64
	dgAddr int64
}

// chunkStream accumulates bytes and flushes them as DT (or DZ) blocks,
// remembering the written segments for the finalize-time data list.
type chunkStream struct {
	blockID string // "##DT" or "##SD"
	recSize int    // record size, for DZ byte transposition; 0 for linear streams
	buf     []byte
	segs    []wseg
	total   uint64
}

type wseg struct {
	addr   int64
	orgLen uint64
}

// ---- channel definition ----

func (g *GroupWriter) add(c *wchan) ChannelIndex {
	c.byteOffset = uint32(g.recSize)
	g.recSize += int(c.bitCount / 8)
	g.chans = append(g.chans, c)
	return ChannelIndex(len(g.chans) - 1)
}

func (g *GroupWriter) master(name, unit string) ChannelIndex {
	return g.add(&wchan{name: name, unit: unit, dataType: blocks.DTFloatLE,
		bitCount: 64, cnType: blocks.CNMaster, syncType: blocks.SyncTime})
}

// MasterTime defines a float64 time master channel (seconds). Only
// useful with WithoutMaster; at most one master per group.
func (g *GroupWriter) MasterTime(name, unit string) ChannelIndex {
	return g.master(name, unit)
}

// Float64 adds a 64-bit float channel.
func (g *GroupWriter) Float64(name, unit string) ChannelIndex {
	return g.add(&wchan{name: name, unit: unit, dataType: blocks.DTFloatLE, bitCount: 64})
}

// Float32 adds a 32-bit float channel.
func (g *GroupWriter) Float32(name, unit string) ChannelIndex {
	return g.add(&wchan{name: name, unit: unit, dataType: blocks.DTFloatLE, bitCount: 32})
}

// Int adds a signed integer channel of 8, 16, 32 or 64 bits.
func (g *GroupWriter) Int(name, unit string, bits int) ChannelIndex {
	return g.add(&wchan{name: name, unit: unit, dataType: blocks.DTIntLE, bitCount: uint32(bits)})
}

// Uint adds an unsigned integer channel of 8, 16, 32 or 64 bits.
func (g *GroupWriter) Uint(name, unit string, bits int) ChannelIndex {
	return g.add(&wchan{name: name, unit: unit, dataType: blocks.DTUintLE, bitCount: uint32(bits)})
}

// String adds a variable-length string channel (VLSD).
func (g *GroupWriter) String(name string) ChannelIndex {
	return g.add(&wchan{name: name, dataType: blocks.DTStringUTF8, bitCount: 64,
		cnType: blocks.CNVLSD, vlsd: true, sd: &chunkStream{blockID: blocks.IDSD}})
}

// SetLinearConversion attaches a linear conversion to a channel:
// physical = offset + factor*raw. Readers apply it automatically.
func (g *GroupWriter) SetLinearConversion(i ChannelIndex, offset, factor float64) {
	c := g.chans[i]
	c.hasConv, c.convB, c.convA = true, offset, factor
}

// SetComment attaches a comment to a channel.
func (g *GroupWriter) SetComment(i ChannelIndex, comment string) {
	g.chans[i].comment = comment
}

// ---- record building ----

// Record is a reusable buffer for one row of samples. Not safe for
// concurrent use; use one Record per writing goroutine.
type Record struct {
	g       *GroupWriter
	buf     []byte
	strings []string
	err     error
}

// Record returns a new reusable record for this group.
func (g *GroupWriter) Record() *Record {
	return &Record{g: g, buf: make([]byte, g.recSize), strings: make([]string, len(g.chans))}
}

func (r *Record) fail(i ChannelIndex, want string) {
	if r.err == nil {
		r.err = fmt.Errorf("channel %q is not a %s channel", r.g.chans[i].name, want)
	}
}

// SetFloat64 sets a float64 (or float32) channel value. Also sets the
// master time for master channels.
func (r *Record) SetFloat64(i ChannelIndex, v float64) {
	c := r.g.chans[i]
	switch {
	case c.dataType == blocks.DTFloatLE && c.bitCount == 64:
		binary.LittleEndian.PutUint64(r.buf[c.byteOffset:], math.Float64bits(v))
	case c.dataType == blocks.DTFloatLE && c.bitCount == 32:
		binary.LittleEndian.PutUint32(r.buf[c.byteOffset:], math.Float32bits(float32(v)))
	default:
		r.fail(i, "float")
	}
}

// SetInt sets a signed integer channel value.
func (r *Record) SetInt(i ChannelIndex, v int64) {
	c := r.g.chans[i]
	if c.dataType != blocks.DTIntLE {
		r.fail(i, "signed integer")
		return
	}
	putUint(r.buf[c.byteOffset:], uint64(v), c.bitCount)
}

// SetUint sets an unsigned integer channel value.
func (r *Record) SetUint(i ChannelIndex, v uint64) {
	c := r.g.chans[i]
	if c.dataType != blocks.DTUintLE {
		r.fail(i, "unsigned integer")
		return
	}
	putUint(r.buf[c.byteOffset:], v, c.bitCount)
}

// SetString sets a string (VLSD) channel value.
func (r *Record) SetString(i ChannelIndex, s string) {
	if !r.g.chans[i].vlsd {
		r.fail(i, "string")
		return
	}
	r.strings[i] = s
}

func putUint(dst []byte, v uint64, bits uint32) {
	switch bits {
	case 8:
		dst[0] = byte(v)
	case 16:
		binary.LittleEndian.PutUint16(dst, uint16(v))
	case 32:
		binary.LittleEndian.PutUint32(dst, uint32(v))
	default:
		binary.LittleEndian.PutUint64(dst, v)
	}
}

// ---- appending ----

// Append writes one record. The Record can be reused immediately.
func (g *GroupWriter) Append(r *Record) error {
	if r.err != nil {
		return r.err
	}
	if !g.w.started {
		if err := g.w.begin(); err != nil {
			return err
		}
	}
	if g.w.closed {
		return fmt.Errorf("writer is closed")
	}
	// Resolve VLSD values first: the record stores the byte offset of
	// each value within the channel's signal data stream.
	for i, c := range g.chans {
		if !c.vlsd {
			continue
		}
		s := r.strings[i]
		binary.LittleEndian.PutUint64(r.buf[c.byteOffset:], c.sd.total)
		var len4 [4]byte
		binary.LittleEndian.PutUint32(len4[:], uint32(len(s)))
		if err := g.w.appendStream(c.sd, len4[:]); err != nil {
			return err
		}
		if err := g.w.appendStream(c.sd, []byte(s)); err != nil {
			return err
		}
	}
	g.count++
	if g.w.growing {
		g.w.growBytes += int64(len(r.buf))
		_, err := g.w.write(r.buf)
		return err
	}
	return g.w.appendStream(g.data, r.buf)
}

// Column is one channel's values for a batch append. Exactly one of the
// slices matching the channel type must be set.
type Column struct {
	Ch ChannelIndex
	F  []float64
	I  []int64
	U  []uint64
	S  []string
}

// AppendColumns writes n records built from whole columns — the batch
// path for data that already exists in arrays. Columns not provided
// keep zero values.
func (g *GroupWriter) AppendColumns(n int, cols ...Column) error {
	for _, col := range cols {
		l := len(col.F) + len(col.I) + len(col.U) + len(col.S)
		if l < n {
			return fmt.Errorf("column %q has %d values, need %d", g.chans[col.Ch].name, l, n)
		}
	}
	rec := g.Record()
	for row := 0; row < n; row++ {
		for _, col := range cols {
			switch {
			case col.F != nil:
				rec.SetFloat64(col.Ch, col.F[row])
			case col.I != nil:
				rec.SetInt(col.Ch, col.I[row])
			case col.U != nil:
				rec.SetUint(col.Ch, col.U[row])
			case col.S != nil:
				rec.SetString(col.Ch, col.S[row])
			}
		}
		if err := g.Append(rec); err != nil {
			return err
		}
	}
	return nil
}
