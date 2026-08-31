package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// EV is an event block.
type EV struct {
	// Links
	EVNext      int64   // ev_ev_next
	EVParent    int64   // ev_ev_parent
	EVRange     int64   // ev_ev_range
	TXName      int64   // ev_tx_name
	MDComment   int64   // ev_md_comment
	Scope       []int64 // ev_scope[ev_scope_count]
	ATReference []int64 // ev_at_reference[ev_attachment_count]

	// Data
	Type            uint8   // ev_type
	SyncType        uint8   // ev_sync_type
	RangeType       uint8   // ev_range_type
	Cause           uint8   // ev_cause
	Flags           uint8   // ev_flags
	ScopeCount      uint32  // ev_scope_count
	AttachmentCount uint16  // ev_attachment_count
	CreatorIndex    uint16  // ev_creator_index
	SyncBaseValue   int64   // ev_sync_base_value
	SyncFactor      float64 // ev_sync_factor
}

// SyncValue returns the event's position on its sync axis (e.g. seconds).
func (b *EV) SyncValue() float64 { return float64(b.SyncBaseValue) * b.SyncFactor }

// DecodeEV decodes an event block at addr.
func DecodeEV(src source.Source, addr int64) (*EV, error) {
	r, err := decodeRaw(src, addr, IDEV, 5, 32)
	if err != nil {
		return nil, err
	}
	d := r.data
	b := &EV{
		EVNext:    r.link(0),
		EVParent:  r.link(1),
		EVRange:   r.link(2),
		TXName:    r.link(3),
		MDComment: r.link(4),

		Type:            d[0],
		SyncType:        d[1],
		RangeType:       d[2],
		Cause:           d[3],
		Flags:           d[4],
		ScopeCount:      le.Uint32(d[8:12]),
		AttachmentCount: le.Uint16(d[12:14]),
		CreatorIndex:    le.Uint16(d[14:16]),
		SyncBaseValue:   int64(le.Uint64(d[16:24])),
		SyncFactor:      f64(d[24:32]),
	}
	next := 5
	if n := int(b.ScopeCount); next+n <= len(r.links) {
		b.Scope = r.links[next : next+n]
		next += n
	}
	if n := int(b.AttachmentCount); n > 0 && next+n <= len(r.links) {
		b.ATReference = r.links[next : next+n]
	}
	return b, nil
}
