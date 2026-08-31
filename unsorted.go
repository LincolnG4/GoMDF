package mf4

import (
	"fmt"
	"io"

	"github.com/LincolnG4/GoMDF/internal/datasection"
	"github.com/LincolnG4/GoMDF/internal/source"
)

// deinterleaved returns the record stream for one record ID of an
// unsorted data group, de-interleaving the whole DG once on first use.
//
// Fixed-length channel groups yield a buffer of contiguous records
// (identical to the sorted layout). VLSD channel groups yield a stream of
// [u32 length][payload] entries — exactly the SD layout, with each
// value's offset equal to the offset the pointing channel recorded.
func (dg *dataGroup) deinterleaved(recordID uint64) (*datasection.Reader, error) {
	dg.dinOnce.Do(func() { dg.dinErr = dg.deinterleave() })
	if dg.dinErr != nil {
		return nil, dg.dinErr
	}
	r, ok := dg.dinBufs[recordID]
	if !ok {
		return nil, fmt.Errorf("record ID %d not present in data group %d", recordID, dg.index)
	}
	return r, nil
}

func (dg *dataGroup) deinterleave() error {
	section, err := dg.sectionReader()
	if err != nil {
		return err
	}
	idSize := int(dg.block.RecIDSize)
	switch idSize {
	case 1, 2, 4, 8:
	default:
		return fmt.Errorf("invalid record ID size %d", idSize)
	}

	type groupBuf struct {
		buf     []byte
		recSize int // fixed record size, 0 for VLSD groups
		vlsd    bool
	}
	bufs := make(map[uint64]*groupBuf, len(dg.groups)+len(dg.derivedIDs))
	for _, cg := range dg.groups {
		gb := &groupBuf{vlsd: cg.IsVLSD()}
		if !gb.vlsd {
			gb.recSize = int(cg.RecordSize())
			gb.buf = make([]byte, 0, int(cg.CycleCount)*gb.recSize)
		}
		bufs[cg.RecordID] = gb
	}
	// CG-template array elements share the parent group's record layout
	// but use their own record IDs, without a CGBLOCK of their own.
	for id, recSize := range dg.derivedIDs {
		if _, ok := bufs[id]; !ok {
			bufs[id] = &groupBuf{recSize: recSize}
		}
	}

	// One sequential pass over the data section.
	size := section.Size()
	rd := io.NewSectionReader(section, 0, size)
	br := newBufferedReader(rd)
	var idBuf [8]byte
	var lenBuf [4]byte
	for pos := int64(0); pos < size; {
		if _, err := io.ReadFull(br, idBuf[:idSize]); err != nil {
			return fmt.Errorf("record ID at offset %d: %w", pos, err)
		}
		pos += int64(idSize)
		var id uint64
		for i := idSize - 1; i >= 0; i-- {
			id = id<<8 | uint64(idBuf[i])
		}
		gb, ok := bufs[id]
		if !ok {
			return fmt.Errorf("unknown record ID %d at offset %d", id, pos-int64(idSize))
		}
		if gb.vlsd {
			if _, err := io.ReadFull(br, lenBuf[:]); err != nil {
				return fmt.Errorf("VLSD length at offset %d: %w", pos, err)
			}
			n := int64(uint32(lenBuf[0]) | uint32(lenBuf[1])<<8 | uint32(lenBuf[2])<<16 | uint32(lenBuf[3])<<24)
			gb.buf = append(gb.buf, lenBuf[:]...)
			start := len(gb.buf)
			gb.buf = append(gb.buf, make([]byte, n)...)
			if _, err := io.ReadFull(br, gb.buf[start:]); err != nil {
				return fmt.Errorf("VLSD payload at offset %d: %w", pos, err)
			}
			pos += 4 + n
		} else {
			start := len(gb.buf)
			gb.buf = append(gb.buf, make([]byte, gb.recSize)...)
			if _, err := io.ReadFull(br, gb.buf[start:]); err != nil {
				return fmt.Errorf("record at offset %d: %w", pos, err)
			}
			pos += int64(gb.recSize)
		}
	}

	dg.dinBufs = make(map[uint64]*datasection.Reader, len(bufs))
	for id, gb := range bufs {
		r, err := datasection.NewFromBytes(source.NewMem(gb.buf))
		if err != nil {
			return err
		}
		dg.dinBufs[id] = r
	}
	return nil
}

// newBufferedReader wraps r with a modest read buffer for the sequential
// de-interleave pass.
func newBufferedReader(r io.Reader) io.Reader {
	return &bufferedReader{r: r, buf: make([]byte, 0, 1<<20)}
}

type bufferedReader struct {
	r   io.Reader
	buf []byte
	pos int
}

func (b *bufferedReader) Read(p []byte) (int, error) {
	if b.pos >= len(b.buf) {
		b.buf = b.buf[:cap(b.buf)]
		n, err := b.r.Read(b.buf)
		if n == 0 {
			return 0, err
		}
		b.buf = b.buf[:n]
		b.pos = 0
	}
	n := copy(p, b.buf[b.pos:])
	b.pos += n
	return n, nil
}
