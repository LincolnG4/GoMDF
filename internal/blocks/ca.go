package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// CA storage types (ca_storage).
const (
	CAStorageCNTemplate = 0
	CAStorageCGTemplate = 1
	CAStorageDGTemplate = 2
)

// CA is a channel array block. The fixed part and the composition link
// are decoded; axis links are not resolved.
type CA struct {
	// Links
	Composition int64 // ca_composition: nested CA (or CN structure)

	Type            uint8    // ca_type
	Storage         uint8    // ca_storage
	NDim            uint16   // ca_ndim
	Flags           uint32   // ca_flags
	ByteOffsetBase  int32    // ca_byte_offset_base
	InvalBitPosBase uint32   // ca_inval_bit_pos_base
	DimSize         []uint64 // ca_dim_size[ca_ndim]
}

// DecodeCA decodes the fixed part of a channel array block at addr.
func DecodeCA(src source.Source, addr int64) (*CA, error) {
	r, err := decodeRaw(src, addr, IDCA, 0, 16)
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
	n := int(b.NDim)
	if len(d) >= 16+8*n {
		b.DimSize = make([]uint64, n)
		for i := 0; i < n; i++ {
			b.DimSize[i] = le.Uint64(d[16+8*i:])
		}
	}
	return b, nil
}
