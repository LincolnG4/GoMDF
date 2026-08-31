package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// CN is a channel block.
type CN struct {
	// Links
	CNNext       int64    // cn_cn_next
	Composition  int64    // cn_composition: CA or CN block
	TXName       int64    // cn_tx_name
	SISource     int64    // cn_si_source
	CCConversion int64    // cn_cc_conversion
	Data         int64    // cn_data: SD/DZ/DL/CG/AT... for VLSD & friends
	MDUnit       int64    // cn_md_unit
	MDComment    int64    // cn_md_comment
	ATReference  []int64  // cn_at_reference[attachment_count] (>= 4.1)
	DefaultX     [3]int64 // cn_default_x: DG, CG, CN (only if CNFlagDefaultX)

	// Data
	Type            uint8   // cn_type
	SyncType        uint8   // cn_sync_type
	DataType        uint8   // cn_data_type
	BitOffset       uint8   // cn_bit_offset
	ByteOffset      uint32  // cn_byte_offset
	BitCount        uint32  // cn_bit_count
	Flags           uint32  // cn_flags
	InvalBitPos     uint32  // cn_inval_bit_pos
	Precision       uint8   // cn_precision
	AttachmentCount uint16  // cn_attachment_count
	ValRangeMin     float64 // cn_val_range_min
	ValRangeMax     float64 // cn_val_range_max
	LimitMin        float64 // cn_limit_min
	LimitMax        float64 // cn_limit_max
	LimitExtMin     float64 // cn_limit_ext_min
	LimitExtMax     float64 // cn_limit_ext_max
}

// DecodeCN decodes a channel block at addr.
func DecodeCN(src source.Source, addr int64) (*CN, error) {
	r, err := decodeRaw(src, addr, IDCN, 8, 72)
	if err != nil {
		return nil, err
	}
	d := r.data
	b := &CN{
		CNNext:       r.link(0),
		Composition:  r.link(1),
		TXName:       r.link(2),
		SISource:     r.link(3),
		CCConversion: r.link(4),
		Data:         r.link(5),
		MDUnit:       r.link(6),
		MDComment:    r.link(7),

		Type:            d[0],
		SyncType:        d[1],
		DataType:        d[2],
		BitOffset:       d[3],
		ByteOffset:      le.Uint32(d[4:8]),
		BitCount:        le.Uint32(d[8:12]),
		Flags:           le.Uint32(d[12:16]),
		InvalBitPos:     le.Uint32(d[16:20]),
		Precision:       d[20],
		AttachmentCount: le.Uint16(d[22:24]),
		ValRangeMin:     f64(d[24:32]),
		ValRangeMax:     f64(d[32:40]),
		LimitMin:        f64(d[40:48]),
		LimitMax:        f64(d[48:56]),
		LimitExtMin:     f64(d[56:64]),
		LimitExtMax:     f64(d[64:72]),
	}
	// Optional links after the 8 fixed ones (>= 4.1): attachment_count
	// attachment references, then 3 default-X links if the flag is set.
	next := 8
	if n := int(b.AttachmentCount); n > 0 && next+n <= len(r.links) {
		b.ATReference = r.links[next : next+n]
		next += n
	}
	if b.Flags&CNFlagDefaultX != 0 && next+3 <= len(r.links) {
		copy(b.DefaultX[:], r.links[next:next+3])
	}
	return b, nil
}
