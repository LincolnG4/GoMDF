package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// HD is the header block, root of the block tree (always at offset 64).
type HD struct {
	// Links
	DGFirst   int64 // hd_dg_first
	FHFirst   int64 // hd_fh_first
	CHFirst   int64 // hd_ch_first
	ATFirst   int64 // hd_at_first
	EVFirst   int64 // hd_ev_first
	MDComment int64 // hd_md_comment

	// Data
	StartTimeNS    uint64  // hd_start_time_ns: ns since 1970-01-01 UTC
	TZOffsetMin    int16   // hd_tz_offset_min
	DSTOffsetMin   int16   // hd_dst_offset_min
	TimeFlags      uint8   // hd_time_flags
	TimeClass      uint8   // hd_time_class
	Flags          uint8   // hd_flags
	StartAngleRad  float64 // hd_start_angle_rad
	StartDistanceM float64 // hd_start_distance_m
}

// DecodeHD decodes the header block at addr (normally IDSize).
func DecodeHD(src source.Source, addr int64) (*HD, error) {
	r, err := decodeRaw(src, addr, IDHD, 6, 32)
	if err != nil {
		return nil, err
	}
	d := r.data
	return &HD{
		DGFirst:   r.link(0),
		FHFirst:   r.link(1),
		CHFirst:   r.link(2),
		ATFirst:   r.link(3),
		EVFirst:   r.link(4),
		MDComment: r.link(5),

		StartTimeNS:    le.Uint64(d[0:8]),
		TZOffsetMin:    int16(le.Uint16(d[8:10])),
		DSTOffsetMin:   int16(le.Uint16(d[10:12])),
		TimeFlags:      d[12],
		TimeClass:      d[13],
		Flags:          d[14],
		StartAngleRad:  f64(d[16:24]),
		StartDistanceM: f64(d[24:32]),
	}, nil
}
