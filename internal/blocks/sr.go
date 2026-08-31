package blocks

import "github.com/LincolnG4/GoMDF/internal/source"

// SR is a sample reduction block. Decoded for completeness; reduced data
// is not exposed by the public API yet.
type SR struct {
	// Links
	SRNext int64 // sr_sr_next
	Data   int64 // sr_data

	// Data
	CycleCount uint64  // sr_cycle_count
	Interval   float64 // sr_interval
	SyncType   uint8   // sr_sync_type
	Flags      uint8   // sr_flags
}

// DecodeSR decodes a sample reduction block at addr.
func DecodeSR(src source.Source, addr int64) (*SR, error) {
	r, err := decodeRaw(src, addr, IDSR, 2, 18)
	if err != nil {
		return nil, err
	}
	d := r.data
	return &SR{
		SRNext:     r.link(0),
		Data:       r.link(1),
		CycleCount: le.Uint64(d[0:8]),
		Interval:   f64(d[8:16]),
		SyncType:   d[16],
		Flags:      d[17],
	}, nil
}
