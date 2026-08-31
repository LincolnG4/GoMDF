package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// FH is a file history block.
type FH struct {
	// Links
	FHNext    int64 // fh_fh_next
	MDComment int64 // fh_md_comment

	// Data
	TimeNS       uint64 // fh_time_ns
	TZOffsetMin  int16  // fh_tz_offset_min
	DSTOffsetMin int16  // fh_dst_offset_min
	TimeFlags    uint8  // fh_time_flags
}

// DecodeFH decodes a file history block at addr.
func DecodeFH(src source.Source, addr int64) (*FH, error) {
	r, err := decodeRaw(src, addr, IDFH, 2, 13)
	if err != nil {
		return nil, err
	}
	d := r.data
	return &FH{
		FHNext:    r.link(0),
		MDComment: r.link(1),

		TimeNS:       le.Uint64(d[0:8]),
		TZOffsetMin:  int16(le.Uint16(d[8:10])),
		DSTOffsetMin: int16(le.Uint16(d[10:12])),
		TimeFlags:    d[12],
	}, nil
}
