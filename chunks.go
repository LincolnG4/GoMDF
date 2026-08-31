package mf4

import (
	"fmt"
	"iter"
)

// Chunks reads a set of channels of one group incrementally, window by
// window — the streaming interface for very large files and progressive
// visualization.
//
//	it := group.Chunks(group.Channels())
//	for it.Next() {
//	    for _, sig := range it.Signals() { ... }
//	}
//	if err := it.Err() { ... }
type Chunks struct {
	group *ChannelGroup
	chans []*Channel
	cfg   readConfig

	pos, end int
	window   int
	sigs     []*Signal
	err      error
}

// Chunks returns an iterator over windows of samples for the given
// channels (all belonging to this group). Options: WithChunkSamples sets
// the window size, WithRange restricts the overall sample range, Raw
// skips conversions.
func (g *ChannelGroup) Chunks(channels []*Channel, opts ...ReadOption) *Chunks {
	it := &Chunks{group: g, chans: channels, cfg: newReadConfig(opts)}
	for _, c := range channels {
		if c.group != g {
			it.err = fmt.Errorf("channel %q does not belong to group %q", c.Name, g.Name)
			return it
		}
	}
	from, count, err := it.cfg.resolveRange(int(g.RecordCount))
	if err != nil {
		it.err = err
		return it
	}
	it.pos, it.end = from, from+count
	it.window = it.cfg.chunkSamples
	if it.window <= 0 {
		recSize := int(g.cg.RecordSize())
		if recSize == 0 {
			recSize = 1
		}
		it.window = (1 << 20) / recSize
		if it.window < 1 {
			it.window = 1
		}
	}
	return it
}

// Next decodes the next window. It returns false when the range is
// exhausted or an error occurred (check Err).
func (it *Chunks) Next() bool {
	if it.err != nil || it.pos >= it.end {
		return false
	}
	n := it.window
	if it.pos+n > it.end {
		n = it.end - it.pos
	}
	cfg := it.cfg
	cfg.from, cfg.count = it.pos, n

	// Master values are shared across the window's signals.
	var master []float64
	g := it.group
	if g.master != nil {
		vals, err := g.readMasterFloats(&cfg)
		if err != nil {
			it.err = fmt.Errorf("group %q master: %w", g.Name, err)
			return false
		}
		master = vals
	}
	sigs := make([]*Signal, len(it.chans))
	for i, c := range it.chans {
		sig, err := c.read(&cfg)
		if err != nil {
			it.err = fmt.Errorf("channel %q: %w", c.Name, err)
			return false
		}
		if c == g.master && sig.Type == SampleFloat64 {
			sig.Time = sig.Floats
		} else {
			sig.Time = master
		}
		sigs[i] = sig
	}
	it.sigs = sigs
	it.pos += n
	return true
}

// Signals returns the current window, one aligned Signal per requested
// channel (Offset marks the window start).
func (it *Chunks) Signals() []*Signal { return it.sigs }

// Err returns the first error encountered.
func (it *Chunks) Err() error { return it.err }

// Close releases the iterator's buffers. It is not required for
// correctness but friendly to long-lived programs.
func (it *Chunks) Close() error {
	it.sigs = nil
	it.pos = it.end
	return nil
}

// All adapts the iterator to a range-over-func sequence:
//
//	for sigs := range it.All() { ... }
//
// Check it.Err() after the loop.
func (it *Chunks) All() iter.Seq[[]*Signal] {
	return func(yield func([]*Signal) bool) {
		for it.Next() {
			if !yield(it.sigs) {
				return
			}
		}
	}
}
