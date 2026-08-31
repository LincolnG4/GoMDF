package mf4

import (
	"github.com/LincolnG4/GoMDF/internal/records"
)

// SampleType tells which Signal slice is populated.
type SampleType uint8

const (
	SampleFloat64 SampleType = iota
	SampleInt64
	SampleUint64
	SampleString
	SampleBytes
)

func (t SampleType) String() string {
	switch t {
	case SampleFloat64:
		return "float64"
	case SampleInt64:
		return "int64"
	case SampleUint64:
		return "uint64"
	case SampleString:
		return "string"
	default:
		return "bytes"
	}
}

// Bitset is a packed per-sample flag set (invalidation bits).
type Bitset = records.Bitset

// Signal holds the samples of one channel as a typed column. Exactly one
// of the sample slices — the one matching Type — is populated.
type Signal struct {
	Channel *Channel

	// Time holds the master channel values aligned 1:1 with the samples;
	// nil when the group has no master channel. For the master channel
	// itself Time aliases the sample slice.
	Time []float64

	Type    SampleType
	Floats  []float64
	Ints    []int64
	Uints   []uint64
	Strings []string
	Bytes   [][]byte

	// Invalid flags samples recorded as invalid; nil when the channel has
	// no invalidation bit.
	Invalid *Bitset

	// Offset is the absolute index of the first sample (non-zero for
	// window reads).
	Offset int
}

// Len returns the number of samples.
func (s *Signal) Len() int {
	switch s.Type {
	case SampleFloat64:
		return len(s.Floats)
	case SampleInt64:
		return len(s.Ints)
	case SampleUint64:
		return len(s.Uints)
	case SampleString:
		return len(s.Strings)
	default:
		return len(s.Bytes)
	}
}

// Float64s returns the samples widened to float64: Floats as-is, Ints and
// Uints converted (lossy above 2^53), nil for string and byte signals.
func (s *Signal) Float64s() []float64 {
	switch s.Type {
	case SampleFloat64:
		return s.Floats
	case SampleInt64:
		out := make([]float64, len(s.Ints))
		for i, v := range s.Ints {
			out[i] = float64(v)
		}
		return out
	case SampleUint64:
		out := make([]float64, len(s.Uints))
		for i, v := range s.Uints {
			out[i] = float64(v)
		}
		return out
	}
	return nil
}

func fromColumn(c *Channel, col *records.Column, offset int) *Signal {
	s := &Signal{Channel: c, Invalid: col.Invalid, Offset: offset}
	switch col.Kind {
	case records.KindFloat64:
		s.Type, s.Floats = SampleFloat64, col.F
	case records.KindInt64:
		s.Type, s.Ints = SampleInt64, col.I
	case records.KindUint64:
		s.Type, s.Uints = SampleUint64, col.U
	case records.KindString:
		s.Type, s.Strings = SampleString, col.S
	case records.KindBytes:
		s.Type, s.Bytes = SampleBytes, col.B
	}
	return s
}
