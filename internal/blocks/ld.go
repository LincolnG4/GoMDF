package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// LD flag bits.
const (
	LDFlagEqualSampleCount = 1 << 0
	LDFlagTimeValues       = 1 << 1
	LDFlagAngleValues      = 1 << 2
	LDFlagDistanceValues   = 1 << 3
	LDFlagInvalidationData = 1 << 31
)

// LD is an MDF 4.2 column-storage list block: like a DL, but offsets are
// counted in samples (records), values live in DV blocks and
// invalidation bytes in separate DI blocks.
type LD struct {
	LDNext    int64
	Data      []int64 // ld_data[ld_count]: DV (or DZ) blocks
	InvalData []int64 // ld_inval_data[ld_count]: DI (or DZ) blocks; entries may be NIL (all valid)

	Flags uint32
	Count uint32
	// EqualSampleCount is the per-block sample count when
	// LDFlagEqualSampleCount is set.
	EqualSampleCount uint64
	// SampleOffsets[i] is the sample index of block i's first record
	// within the section (present when the equal flag is not set).
	SampleOffsets []uint64
}

// DecodeLD decodes a column-storage list block at addr.
func DecodeLD(src source.Source, addr int64) (*LD, error) {
	r, err := decodeRaw(src, addr, IDLD, 1, 8)
	if err != nil {
		return nil, err
	}
	d := r.data
	b := &LD{
		LDNext: r.link(0),
		Flags:  le.Uint32(d[0:4]),
		Count:  le.Uint32(d[4:8]),
	}
	n := int(b.Count)
	if 1+n > len(r.links) {
		return nil, blockErrf(IDLD, addr, "%w: ld_count %d exceeds %d links", ErrInvalidBlock, n, len(r.links))
	}
	b.Data = r.links[1 : 1+n]
	if b.Flags&LDFlagInvalidationData != 0 {
		if 1+2*n > len(r.links) {
			return nil, blockErrf(IDLD, addr, "%w: missing ld_inval_data links", ErrInvalidBlock)
		}
		b.InvalData = r.links[1+n : 1+2*n]
	}
	if b.Flags&LDFlagEqualSampleCount != 0 {
		if len(d) < 16 {
			return nil, blockErrf(IDLD, addr, "%w: missing ld_equal_sample_count", ErrInvalidBlock)
		}
		b.EqualSampleCount = le.Uint64(d[8:16])
	} else {
		if len(d) < 8+8*n {
			return nil, blockErrf(IDLD, addr, "%w: missing ld_sample_offset array", ErrInvalidBlock)
		}
		b.SampleOffsets = make([]uint64, n)
		for i := 0; i < n; i++ {
			b.SampleOffsets[i] = le.Uint64(d[8+8*i:])
		}
	}
	// ld_time/angle/distance_values arrays may follow; nothing in the
	// read path needs them (they are binary-search accelerators).
	return b, nil
}
