package records

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

var (
	le = binary.LittleEndian
	be = binary.BigEndian
)

// Extract appends the samples of count records from buf (count contiguous
// records of recSize bytes) to col.
func Extract(col *Column, spec ColumnSpec, buf []byte, recSize, count int) error {
	if len(buf) < recSize*count {
		return fmt.Errorf("record buffer too short: %d bytes for %d records of %d", len(buf), count, recSize)
	}
	off := int(spec.ByteOffset)
	nBytes := int(spec.BitCount) / 8
	end := off + (int(spec.BitOffset)+int(spec.BitCount)+7)/8
	if end > recSize {
		return fmt.Errorf("channel field [%d..%d) exceeds record size %d", off, end, recSize)
	}
	aligned := spec.BitOffset == 0 && spec.BitCount%8 == 0

	switch spec.DataType {
	case blocks.DTUintLE, blocks.DTUintBE, blocks.DTIntLE, blocks.DTIntBE:
		extractInts(col, spec, buf, recSize, count, aligned)
	case blocks.DTFloatLE, blocks.DTFloatBE:
		if err := extractFloats(col, spec, buf, recSize, count, aligned); err != nil {
			return err
		}
	case blocks.DTStringLatin, blocks.DTStringUTF8, blocks.DTStringUTF16LE, blocks.DTStringUTF16BE:
		if !aligned {
			return fmt.Errorf("string channel with bit offset %d/count %d", spec.BitOffset, spec.BitCount)
		}
		extractStrings(col, spec, buf, recSize, count, nBytes)
	case blocks.DTByteArray, blocks.DTMIMESample, blocks.DTMIMEStream,
		blocks.DTCANopenDate, blocks.DTCANopenTime:
		for r := 0; r < count; r++ {
			p := buf[r*recSize+off:]
			col.B = append(col.B, append([]byte(nil), p[:nBytes]...))
		}
	default:
		return fmt.Errorf("unsupported channel data type %d", spec.DataType)
	}

	if col.Invalid != nil {
		extractInvalid(col.Invalid, spec, buf, recSize, count)
	}
	return nil
}

// extractInts covers the four integer data types, fast paths first.
func extractInts(col *Column, spec ColumnSpec, buf []byte, recSize, count int, aligned bool) {
	off := int(spec.ByteOffset)
	bigE := spec.DataType == blocks.DTUintBE || spec.DataType == blocks.DTIntBE
	signed := spec.DataType == blocks.DTIntLE || spec.DataType == blocks.DTIntBE

	// Fast paths: byte-aligned power-of-two widths.
	if aligned {
		var get func(p []byte) uint64
		switch {
		case spec.BitCount == 8:
			get = func(p []byte) uint64 { return uint64(p[0]) }
		case spec.BitCount == 16 && !bigE:
			get = func(p []byte) uint64 { return uint64(le.Uint16(p)) }
		case spec.BitCount == 16 && bigE:
			get = func(p []byte) uint64 { return uint64(be.Uint16(p)) }
		case spec.BitCount == 32 && !bigE:
			get = func(p []byte) uint64 { return uint64(le.Uint32(p)) }
		case spec.BitCount == 32 && bigE:
			get = func(p []byte) uint64 { return uint64(be.Uint32(p)) }
		case spec.BitCount == 64 && !bigE:
			get = le.Uint64
		case spec.BitCount == 64 && bigE:
			get = be.Uint64
		}
		if get != nil {
			appendInts(col, spec, buf, recSize, count, off, signed, get)
			return
		}
	}
	// Generic bit-field path.
	get := func(p []byte) uint64 { return bitField(p, spec.BitOffset, spec.BitCount, bigE) }
	appendInts(col, spec, buf, recSize, count, off, signed, get)
}

func appendInts(col *Column, spec ColumnSpec, buf []byte, recSize, count, off int, signed bool, get func([]byte) uint64) {
	switch {
	case signed:
		shift := 64 - spec.BitCount
		if col.Kind == KindFloat64 {
			for r := 0; r < count; r++ {
				v := int64(get(buf[r*recSize+off:])<<shift) >> shift
				col.F = append(col.F, float64(v))
			}
		} else {
			for r := 0; r < count; r++ {
				v := int64(get(buf[r*recSize+off:])<<shift) >> shift
				col.I = append(col.I, v)
			}
		}
	case col.Kind == KindFloat64:
		for r := 0; r < count; r++ {
			col.F = append(col.F, float64(get(buf[r*recSize+off:])))
		}
	default:
		for r := 0; r < count; r++ {
			col.U = append(col.U, get(buf[r*recSize+off:]))
		}
	}
}

func extractFloats(col *Column, spec ColumnSpec, buf []byte, recSize, count int, aligned bool) error {
	off := int(spec.ByteOffset)
	bigE := spec.DataType == blocks.DTFloatBE
	if !aligned {
		return fmt.Errorf("float channel with bit offset %d/count %d", spec.BitOffset, spec.BitCount)
	}
	switch spec.BitCount {
	case 64:
		for r := 0; r < count; r++ {
			u := ordered64(buf[r*recSize+off:], bigE)
			col.F = append(col.F, math.Float64frombits(u))
		}
	case 32:
		for r := 0; r < count; r++ {
			var u uint32
			if bigE {
				u = be.Uint32(buf[r*recSize+off:])
			} else {
				u = le.Uint32(buf[r*recSize+off:])
			}
			col.F = append(col.F, float64(math.Float32frombits(u)))
		}
	case 16:
		for r := 0; r < count; r++ {
			var u uint16
			if bigE {
				u = be.Uint16(buf[r*recSize+off:])
			} else {
				u = le.Uint16(buf[r*recSize+off:])
			}
			col.F = append(col.F, halfToFloat64(u))
		}
	default:
		return fmt.Errorf("unsupported float bit count %d", spec.BitCount)
	}
	return nil
}

func ordered64(p []byte, bigE bool) uint64 {
	if bigE {
		return be.Uint64(p)
	}
	return le.Uint64(p)
}

// halfToFloat64 converts an IEEE 754 half-precision value.
func halfToFloat64(h uint16) float64 {
	sign := uint64(h>>15) << 63
	exp := uint64(h>>10) & 0x1f
	frac := uint64(h & 0x3ff)
	var bits64 uint64
	switch exp {
	case 0:
		if frac == 0 {
			bits64 = sign // signed zero
		} else {
			// subnormal: normalize
			e := uint64(1023 - 15 + 1)
			for frac&0x400 == 0 {
				frac <<= 1
				e--
			}
			bits64 = sign | e<<52 | (frac&0x3ff)<<42
		}
	case 0x1f:
		bits64 = sign | 0x7ff<<52 | frac<<42 // inf / nan
	default:
		bits64 = sign | (exp-15+1023)<<52 | frac<<42
	}
	return math.Float64frombits(bits64)
}

func extractInvalid(dst *Bitset, spec ColumnSpec, buf []byte, recSize, count int) {
	byteIdx := int(spec.DataBytes) + int(spec.InvalBit)/8
	mask := byte(1) << (spec.InvalBit % 8)
	if byteIdx >= recSize {
		return
	}
	for r := 0; r < count; r++ {
		dst.Append(buf[r*recSize+byteIdx]&mask != 0)
	}
}
