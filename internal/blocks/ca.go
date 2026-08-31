package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// CA storage types (ca_storage).
const (
	CAStorageCNTemplate = 0
	CAStorageCGTemplate = 1
	CAStorageDGTemplate = 2
)

// CA flag bits (ca_flags).
const (
	CAFlagDynamicSize        = 1 << 0
	CAFlagInputQuantity      = 1 << 1
	CAFlagOutputQuantity     = 1 << 2
	CAFlagComparisonQuantity = 1 << 3
	CAFlagAxis               = 1 << 4
	CAFlagFixedAxes          = 1 << 5
	CAFlagInverseLayout      = 1 << 6
	CAFlagLeftOpenInterval   = 1 << 7
	CAFlagStandardAxis       = 1 << 8
)

// CA is a channel array block.
type CA struct {
	// Links
	Composition int64 // ca_composition: nested CA (or CN structure)
	// DataLinks are the per-element data sections for DG-template
	// storage (ca_data[k]; index 0 equals the parent dg_data). Entries
	// may be NIL for elements that were not recorded.
	DataLinks []int64

	Type            uint8    // ca_type
	Storage         uint8    // ca_storage
	NDim            uint16   // ca_ndim
	Flags           uint32   // ca_flags
	ByteOffsetBase  int32    // ca_byte_offset_base
	InvalBitPosBase uint32   // ca_inval_bit_pos_base
	DimSize         []uint64 // ca_dim_size[ca_ndim]
	// AxisValues holds the fixed axis values (sum of dim sizes entries)
	// when CAFlagFixedAxes is set.
	AxisValues []float64
	// CycleCounts holds ca_cycle_count[k] per element for CG/DG-template
	// storage.
	CycleCounts []uint64
}

// ElementCount returns the product of all dimension sizes.
func (b *CA) ElementCount() uint64 {
	n := uint64(1)
	for _, d := range b.DimSize {
		n *= d
	}
	return n
}

// DecodeCA decodes a channel array block at addr.
func DecodeCA(src source.Source, addr int64) (*CA, error) {
	r, err := decodeRaw(src, addr, IDCA, 1, 16)
	if err != nil {
		return nil, err
	}
	d := r.data
	b := &CA{
		Composition:     r.link(0),
		Type:            d[0],
		Storage:         d[1],
		NDim:            le.Uint16(d[2:4]),
		Flags:           le.Uint32(d[4:8]),
		ByteOffsetBase:  int32(le.Uint32(d[8:12])),
		InvalBitPosBase: le.Uint32(d[12:16]),
	}
	nd := int(b.NDim)
	pos := 16
	if len(d) < pos+8*nd {
		return nil, blockErrf(IDCA, addr, "%w: %d dims exceed data section", ErrInvalidBlock, nd)
	}
	b.DimSize = make([]uint64, nd)
	sum, prod := 0, uint64(1)
	for i := 0; i < nd; i++ {
		b.DimSize[i] = le.Uint64(d[pos+8*i:])
		sum += int(b.DimSize[i])
		prod *= b.DimSize[i]
	}
	pos += 8 * nd

	if b.Flags&CAFlagFixedAxes != 0 && len(d) >= pos+8*sum {
		b.AxisValues = make([]float64, sum)
		for i := range b.AxisValues {
			b.AxisValues[i] = f64(d[pos+8*i:])
		}
		pos += 8 * sum
	}
	if b.Storage != CAStorageCNTemplate && prod <= 1<<20 && len(d) >= pos+8*int(prod) {
		b.CycleCounts = make([]uint64, prod)
		for i := range b.CycleCounts {
			b.CycleCounts[i] = le.Uint64(d[pos+8*i:])
		}
	}
	// DG template: ca_data[k] links follow ca_composition.
	if b.Storage == CAStorageDGTemplate {
		if uint64(len(r.links)) < 1+prod {
			return nil, blockErrf(IDCA, addr, "%w: DG-template array needs %d ca_data links", ErrInvalidBlock, prod)
		}
		b.DataLinks = r.links[1 : 1+prod]
	}
	return b, nil
}
