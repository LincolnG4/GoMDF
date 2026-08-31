package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// LD flag bits.
const (
	LDFlagEqualSampleCount = 1 << 0
	LDFlagInvalidationData = 1 << 31
)

// LD is an MDF 4.2 column-storage data list block. Decoded for detection;
// column-oriented reading is not yet supported by this module.
type LD struct {
	LDNext    int64
	Data      []int64 // ld_data[ld_count] (DV/DZ blocks)
	InvalData []int64 // ld_inval_data[ld_count] if LDFlagInvalidationData

	Flags uint32
	Count uint32
}

// DecodeLD decodes a column-storage list block header at addr.
func DecodeLD(src source.Source, addr int64) (*LD, error) {
	r, err := decodeRaw(src, addr, IDLD, 1, 8)
	if err != nil {
		return nil, err
	}
	b := &LD{
		LDNext: r.link(0),
		Flags:  le.Uint32(r.data[0:4]),
		Count:  le.Uint32(r.data[4:8]),
	}
	n := int(b.Count)
	if 1+n <= len(r.links) {
		b.Data = r.links[1 : 1+n]
	}
	if b.Flags&LDFlagInvalidationData != 0 && 1+2*n <= len(r.links) {
		b.InvalData = r.links[1+n : 1+2*n]
	}
	return b, nil
}
