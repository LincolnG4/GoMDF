package mf4

import (
	"fmt"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/datasection"
	"github.com/LincolnG4/GoMDF/internal/records"
)

// SyncKind is the synchronization domain of a reduction interval
// (sr_sync_type).
type SyncKind uint8

const (
	SyncKindTime     SyncKind = 1 // interval in seconds
	SyncKindAngle    SyncKind = 2 // interval in radians
	SyncKindDistance SyncKind = 3 // interval in meters
	SyncKindIndex    SyncKind = 4 // interval in record indexes
)

// Reduction is one precomputed sample reduction of a channel group: for
// every interval of Interval length, the mean, minimum and maximum of
// each channel. Writers store several reductions at different interval
// lengths so viewers can pick the best one for the current zoom level —
// reading a reduction is much cheaper than reducing the full data.
type Reduction struct {
	// Interval is the reduction interval length (unit per Sync).
	Interval float64
	Sync     SyncKind
	// CycleCount is the number of reduction records (intervals).
	CycleCount uint64

	group *ChannelGroup
	sr    *blocks.SR
}

// ReducedSignal holds one channel's samples aggregated per reduction
// interval. Mean, Min and Max are aligned 1:1; their Time is the start
// value of each interval (taken from the master channel's first
// sub-record).
//
// For the master channel itself the spec assigns different meanings:
// Mean holds the interval start value, Min/Max the minimum and maximum
// raster (sample spacing) within the interval.
type ReducedSignal struct {
	Mean, Min, Max *Signal
}

// Reductions returns the group's precomputed sample reductions (largest
// resolution first, typically), or nil when the file stores none.
func (g *ChannelGroup) Reductions() ([]*Reduction, error) {
	var out []*Reduction
	for addr := g.cg.SRFirst; addr != 0; {
		sr, err := blocks.DecodeSR(g.file.src, addr)
		if err != nil {
			return nil, err
		}
		out = append(out, &Reduction{
			Interval:   sr.Interval,
			Sync:       SyncKind(sr.SyncType),
			CycleCount: sr.CycleCount,
			group:      g,
			sr:         sr,
		})
		addr = sr.SRNext
	}
	return out, nil
}

// Read extracts a channel's reduced samples. The channel must belong to
// the reduction's group. Raw() and WithRange apply as for Channel.Read
// (the range is counted in reduction intervals).
func (r *Reduction) Read(c *Channel, opts ...ReadOption) (*ReducedSignal, error) {
	if c.group != r.group {
		return nil, fmt.Errorf("channel %q does not belong to group %q", c.Name, r.group.Name)
	}
	cfg := newReadConfig(opts)
	out := &ReducedSignal{}
	var err error
	// The three sub-records lay the normal record layout side by side.
	for i, dst := range []**Signal{&out.Mean, &out.Min, &out.Max} {
		if *dst, err = r.readSub(c, &cfg, i); err != nil {
			return nil, err
		}
	}
	// Master values: the master channel's mean sub-record.
	if m := r.group.master; m != nil && m != c {
		mSig, err := r.readSub(m, &readConfig{from: cfg.from, count: cfg.count}, 0)
		if err != nil {
			return nil, fmt.Errorf("reduction master: %w", err)
		}
		t := mSig.Float64s()
		out.Mean.Time, out.Min.Time, out.Max.Time = t, t, t
	} else if m == c && out.Mean.Type == SampleFloat64 {
		out.Mean.Time, out.Min.Time, out.Max.Time = out.Mean.Floats, out.Mean.Floats, out.Mean.Floats
	}
	return out, nil
}

// readSub extracts one sub-record column (0 mean, 1 min, 2 max).
func (r *Reduction) readSub(c *Channel, cfg *readConfig, sub int) (*Signal, error) {
	g := r.group
	from, count, err := cfg.resolveRange(int(r.CycleCount))
	if err != nil {
		return nil, err
	}
	conv, err := c.compiledConversion()
	if err != nil {
		return nil, err
	}
	spec, err := c.columnSpec(cfg, conv)
	if err != nil {
		return nil, err
	}
	dataBytes := int(g.cg.DataBytes)
	spec.ByteOffset += uint32(sub * dataBytes)
	// Invalidation bytes (when present) follow the three sub-records.
	recSize := 3 * dataBytes
	if r.sr.Flags&1 != 0 {
		spec.DataBytes = uint32(recSize)
		recSize += int(g.cg.InvalBytes)
	} else {
		spec.InvalBit = -1
	}

	col, err := records.NewColumn(spec, count)
	if err != nil {
		return nil, err
	}
	rd, err := r.dataReader()
	if err != nil {
		return nil, err
	}
	total := int(rd.Size() / int64(recSize))
	if from+count > total {
		return nil, fmt.Errorf("reduction data holds %d records, need %d", total, from+count)
	}
	chunk := (1 << 20) / recSize
	if chunk < 1 {
		chunk = 1
	}
	buf := make([]byte, chunk*recSize)
	for done := 0; done < count; {
		n := chunk
		if done+n > count {
			n = count - done
		}
		if _, err := rd.ReadAt(buf[:n*recSize], int64(from+done)*int64(recSize)); err != nil {
			return nil, err
		}
		if err := records.Extract(col, spec, buf[:n*recSize], recSize, n); err != nil {
			return nil, err
		}
		done += n
	}
	if !cfg.raw {
		if err := applyConversion(col, conv); err != nil {
			return nil, err
		}
	}
	return fromColumn(c, col, from), nil
}

func (r *Reduction) dataReader() (*datasection.Reader, error) {
	f := r.group.file
	if f.finalize {
		return datasection.NewFinalizing(f.src, r.sr.Data, f.cfg.cacheBytes)
	}
	return datasection.New(f.src, r.sr.Data, f.cfg.cacheBytes)
}
