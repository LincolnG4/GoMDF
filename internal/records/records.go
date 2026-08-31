// Package records extracts channel sample columns from MDF record
// buffers.
//
// A record buffer holds N contiguous records of a fixed size (the sorted
// layout; unsorted data is de-interleaved into this shape first). Each
// channel occupies a bit field at a fixed position in every record;
// extraction strides over the buffer once per channel with a typed loop,
// appending to a typed destination slice — no per-sample boxing.
package records

import (
	"fmt"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

// Kind is the native storage class of an extracted column.
type Kind uint8

const (
	KindFloat64 Kind = iota
	KindInt64
	KindUint64
	KindString
	KindBytes
)

// ColumnSpec describes where a channel lives inside a record and how to
// decode it.
type ColumnSpec struct {
	ByteOffset uint32
	BitOffset  uint8
	BitCount   uint32
	DataType   uint8 // blocks.DT* constant
	// InvalBit is the invalidation bit position within the invalidation
	// bytes, or -1 when the channel has none.
	InvalBit int64
	// DataBytes is cg_data_bytes: where invalidation bytes start within
	// the record.
	DataBytes uint32
	// AsFloat64 forces numeric values to be widened to float64 during
	// extraction (used when a numeric conversion follows anyway).
	AsFloat64 bool
}

// NativeKind returns the storage class the spec extracts to.
func NativeKind(dataType uint8, asFloat bool) (Kind, error) {
	switch dataType {
	case blocks.DTUintLE, blocks.DTUintBE:
		if asFloat {
			return KindFloat64, nil
		}
		return KindUint64, nil
	case blocks.DTIntLE, blocks.DTIntBE:
		if asFloat {
			return KindFloat64, nil
		}
		return KindInt64, nil
	case blocks.DTFloatLE, blocks.DTFloatBE:
		return KindFloat64, nil
	case blocks.DTStringLatin, blocks.DTStringUTF8, blocks.DTStringUTF16LE, blocks.DTStringUTF16BE:
		return KindString, nil
	case blocks.DTByteArray, blocks.DTMIMESample, blocks.DTMIMEStream, blocks.DTCANopenDate, blocks.DTCANopenTime:
		return KindBytes, nil
	default:
		return 0, fmt.Errorf("unsupported channel data type %d", dataType)
	}
}

// Column is the typed destination of an extraction. Exactly one of the
// slices (matching Kind) is appended to.
type Column struct {
	Kind Kind
	F    []float64
	I    []int64
	U    []uint64
	S    []string
	B    [][]byte
	// Invalid is a bitset of per-sample invalidation flags; nil when the
	// spec has no invalidation bit.
	Invalid *Bitset
}

// NewColumn prepares a column for capacity samples.
func NewColumn(spec ColumnSpec, capacity int) (*Column, error) {
	kind, err := NativeKind(spec.DataType, spec.AsFloat64)
	if err != nil {
		return nil, err
	}
	c := &Column{Kind: kind}
	switch kind {
	case KindFloat64:
		c.F = make([]float64, 0, capacity)
	case KindInt64:
		c.I = make([]int64, 0, capacity)
	case KindUint64:
		c.U = make([]uint64, 0, capacity)
	case KindString:
		c.S = make([]string, 0, capacity)
	case KindBytes:
		c.B = make([][]byte, 0, capacity)
	}
	if spec.InvalBit >= 0 {
		c.Invalid = NewBitset(capacity)
	}
	return c, nil
}

// Len returns the number of extracted samples.
func (c *Column) Len() int {
	switch c.Kind {
	case KindFloat64:
		return len(c.F)
	case KindInt64:
		return len(c.I)
	case KindUint64:
		return len(c.U)
	case KindString:
		return len(c.S)
	default:
		return len(c.B)
	}
}
