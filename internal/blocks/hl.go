package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// HL is a header list block: precedes a DL chain whose data blocks are all
// DZ blocks with the given zip type.
type HL struct {
	DLFirst int64 // hl_dl_first

	Flags   uint16 // hl_flags
	ZipType uint8  // hl_zip_type
}

// DecodeHL decodes a header list block at addr.
func DecodeHL(src source.Source, addr int64) (*HL, error) {
	r, err := decodeRaw(src, addr, IDHL, 1, 3)
	if err != nil {
		return nil, err
	}
	return &HL{
		DLFirst: r.link(0),
		Flags:   le.Uint16(r.data[0:2]),
		ZipType: r.data[2],
	}, nil
}
