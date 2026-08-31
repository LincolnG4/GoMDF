package mf4

import (
	"fmt"
	"sync"

	"github.com/LincolnG4/GoMDF/internal/datasection"
)

// arrayElement is the per-element record source of a "fragmented" array
// (CA storage type "CG template" or "DG template"), where every array
// element is recorded in its own records instead of sharing one record
// with its siblings.
//
// The record layout is the parent group's; only where the records live
// differs: a DG-template element has its own data section (ca_data[k]),
// a CG-template element its own record ID (cg_record_id + k) inside the
// group's unsorted data group.
type arrayElement struct {
	dataAddr    int64  // DG template: ca_data[k] (0 for CG template)
	recordID    uint64 // CG template: cg_record_id + k
	recordCount uint64 // ca_cycle_count[k]

	once   sync.Once
	layout *recordLayout
	err    error
}

// resolve builds (once) the record stream for this element.
func (e *arrayElement) resolve(g *ChannelGroup) (*recordLayout, error) {
	e.once.Do(func() {
		e.layout, e.err = e.build(g)
	})
	return e.layout, e.err
}

func (e *arrayElement) build(g *ChannelGroup) (*recordLayout, error) {
	if e.dataAddr == 0 {
		// CG template: the element's records carry their own record ID
		// in the group's unsorted data group.
		r, err := g.dg.deinterleaved(e.recordID)
		if err != nil {
			return nil, fmt.Errorf("array element record id %d: %w", e.recordID, err)
		}
		return &recordLayout{data: r, recSize: int(g.cg.RecordSize())}, nil
	}
	// DG template: the element has its own data section with the same
	// record layout as the parent group.
	f := g.file
	r, err := datasection.New(f.src, e.dataAddr, f.cfg.cacheBytes)
	if err != nil {
		return nil, fmt.Errorf("array element data section: %w", err)
	}
	return &recordLayout{data: r, recSize: int(g.cg.RecordSize())}, nil
}
