package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// CC flag bits (cc_flags).
const (
	CCFlagPrecision = 1 << 0
	CCFlagPhysRange = 1 << 1
	CCFlagStatus    = 1 << 2
)

// CC is a channel conversion block.
type CC struct {
	// Links
	TXName    int64   // cc_tx_name
	MDUnit    int64   // cc_md_unit
	MDComment int64   // cc_md_comment
	CCInverse int64   // cc_cc_inverse
	Refs      []int64 // cc_ref[cc_ref_count]: TX or nested CC blocks

	// Data
	Type        uint8     // cc_type
	Precision   uint8     // cc_precision
	Flags       uint16    // cc_flags
	RefCount    uint16    // cc_ref_count
	ValCount    uint16    // cc_val_count
	PhyRangeMin float64   // cc_phy_range_min
	PhyRangeMax float64   // cc_phy_range_max
	Vals        []float64 // cc_val[cc_val_count]
	// For CCBitfieldToText the cc_val entries are bit masks; RawVals keeps
	// the undecoded uint64 bit patterns for that case.
	RawVals []uint64
}

// DecodeCC decodes a channel conversion block at addr. addr == 0 returns
// (nil, nil): no conversion (1:1).
func DecodeCC(src source.Source, addr int64) (*CC, error) {
	if addr == 0 {
		return nil, nil
	}
	r, err := decodeRaw(src, addr, IDCC, 4, 24)
	if err != nil {
		return nil, err
	}
	d := r.data
	b := &CC{
		TXName:    r.link(0),
		MDUnit:    r.link(1),
		MDComment: r.link(2),
		CCInverse: r.link(3),

		Type:        d[0],
		Precision:   d[1],
		Flags:       le.Uint16(d[2:4]),
		RefCount:    le.Uint16(d[4:6]),
		ValCount:    le.Uint16(d[6:8]),
		PhyRangeMin: f64(d[8:16]),
		PhyRangeMax: f64(d[16:24]),
	}
	if n := int(b.RefCount); n > 0 {
		if 4+n > len(r.links) {
			return nil, blockErrf(IDCC, addr, "%w: ref count %d exceeds %d links", ErrInvalidBlock, n, len(r.links))
		}
		b.Refs = r.links[4 : 4+n]
	}
	if n := int(b.ValCount); n > 0 {
		if len(d) < 24+8*n {
			return nil, blockErrf(IDCC, addr, "%w: val count %d exceeds data section", ErrInvalidBlock, n)
		}
		b.Vals = make([]float64, n)
		b.RawVals = make([]uint64, n)
		for i := 0; i < n; i++ {
			b.RawVals[i] = le.Uint64(d[24+8*i:])
			b.Vals[i] = f64(d[24+8*i:])
		}
	}
	return b, nil
}
