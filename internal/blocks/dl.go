package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// DL is a data list block: an ordered list of data blocks (DT/DV/SD/RD or
// DZ) forming one logical data section.
type DL struct {
	// Links
	DLNext int64   // dl_dl_next
	Data   []int64 // dl_data[dl_count]

	// Data
	Flags uint8  // dl_flags
	Count uint32 // dl_count
	// EqualLength is set when DLFlagEqualLength: every referenced block
	// (except possibly the last of the whole list) holds this many bytes.
	EqualLength uint64
	// Offsets[i] is the byte offset of dl_data[i]'s content within the
	// logical data section. Present when DLFlagEqualLength is not set.
	Offsets []uint64
}

// DecodeDL decodes a data list block at addr.
func DecodeDL(src source.Source, addr int64) (*DL, error) {
	r, err := decodeRaw(src, addr, IDDL, 1, 8)
	if err != nil {
		return nil, err
	}
	d := r.data
	b := &DL{
		DLNext: r.link(0),
		Flags:  d[0],
		Count:  le.Uint32(d[4:8]),
	}
	n := int(b.Count)
	if 1+n > len(r.links) {
		return nil, blockErrf(IDDL, addr, "%w: dl_count %d exceeds %d links", ErrInvalidBlock, n, len(r.links))
	}
	b.Data = r.links[1 : 1+n]
	if b.Flags&DLFlagEqualLength != 0 {
		if len(d) < 16 {
			return nil, blockErrf(IDDL, addr, "%w: missing dl_equal_length", ErrInvalidBlock)
		}
		b.EqualLength = le.Uint64(d[8:16])
	} else {
		if len(d) < 8+8*n {
			return nil, blockErrf(IDDL, addr, "%w: missing dl_offset array", ErrInvalidBlock)
		}
		b.Offsets = make([]uint64, n)
		for i := 0; i < n; i++ {
			b.Offsets[i] = le.Uint64(d[8+8*i:])
		}
	}
	return b, nil
}
