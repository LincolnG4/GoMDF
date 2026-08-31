package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// CG is a channel group block.
type CG struct {
	// Links
	CGNext      int64 // cg_cg_next
	CNFirst     int64 // cg_cn_first
	TXAcqName   int64 // cg_tx_acq_name
	SIAcqSource int64 // cg_si_acq_source
	SRFirst     int64 // cg_sr_first
	MDComment   int64 // cg_md_comment
	CGMaster    int64 // cg_cg_master (4.2, only if CGFlagRemoteMaster)

	// Data
	RecordID      uint64 // cg_record_id
	CycleCount    uint64 // cg_cycle_count
	Flags         uint16 // cg_flags
	PathSeparator uint16 // cg_path_separator (>= 4.1)
	DataBytes     uint32 // cg_data_bytes: record size without invalidation bytes
	InvalBytes    uint32 // cg_inval_bytes: invalidation bytes per record
}

// RecordSize returns the full record size including invalidation bytes.
func (b *CG) RecordSize() uint64 { return uint64(b.DataBytes) + uint64(b.InvalBytes) }

// IsVLSD reports whether this group holds variable-length signal data
// records rather than fixed channel records.
func (b *CG) IsVLSD() bool { return b.Flags&CGFlagVLSD != 0 }

// DecodeCG decodes a channel group block at addr.
func DecodeCG(src source.Source, addr int64) (*CG, error) {
	r, err := decodeRaw(src, addr, IDCG, 6, 32)
	if err != nil {
		return nil, err
	}
	d := r.data
	b := &CG{
		CGNext:      r.link(0),
		CNFirst:     r.link(1),
		TXAcqName:   r.link(2),
		SIAcqSource: r.link(3),
		SRFirst:     r.link(4),
		MDComment:   r.link(5),

		RecordID:      le.Uint64(d[0:8]),
		CycleCount:    le.Uint64(d[8:16]),
		Flags:         le.Uint16(d[16:18]),
		PathSeparator: le.Uint16(d[18:20]),
		DataBytes:     le.Uint32(d[24:28]),
		InvalBytes:    le.Uint32(d[28:32]),
	}
	// MDF 4.2: an extra link to the remote master channel group.
	if b.Flags&CGFlagRemoteMaster != 0 {
		b.CGMaster = r.link(6)
	}
	return b, nil
}
