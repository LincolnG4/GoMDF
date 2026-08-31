package datasection

import (
	"fmt"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/source"
)

// NewColumnStorage resolves an MDF 4.2 column-storage section (an LD
// chain of DV blocks, plus optional DI invalidation blocks) into two
// logical streams: the sample records (recSize bytes each, without
// invalidation bytes) and the invalidation bytes (invalSize bytes per
// record). inval is nil when the section carries no invalidation data.
//
// LD offsets are counted in samples; recSize/invalSize convert them to
// byte offsets. Omitted (NIL) invalidation blocks read as zero bytes —
// all samples valid — as the spec prescribes.
func NewColumnStorage(src source.Source, addr int64, cacheBytes int64, recSize, invalSize int, clamp bool) (data *Reader, inval *Reader, err error) {
	if recSize <= 0 {
		return nil, nil, fmt.Errorf("column storage needs a record size")
	}
	data = &Reader{src: src, cache: newLRU(cacheBytes), clamp: clamp}
	var iv *Reader
	sampleIdx := uint64(0) // running block index for equal-sample-count lists
	for addr != 0 {
		ld, err := blocks.DecodeLD(src, addr)
		if err != nil {
			return nil, nil, err
		}
		if ld.Flags&blocks.LDFlagInvalidationData != 0 && invalSize > 0 && iv == nil {
			iv = &Reader{src: src, cache: newLRU(cacheBytes), clamp: clamp}
		}
		for i, dataAddr := range ld.Data {
			var samples uint64
			if ld.Flags&blocks.LDFlagEqualSampleCount != 0 {
				samples = ld.EqualSampleCount * sampleIdx
			} else {
				samples = ld.SampleOffsets[i]
			}
			if err := addColumnBlock(data, dataAddr, int64(samples)*int64(recSize), blocks.IDDV); err != nil {
				return nil, nil, err
			}
			if iv != nil {
				logical := int64(samples) * int64(invalSize)
				ia := int64(0)
				if i < len(ld.InvalData) {
					ia = ld.InvalData[i]
				}
				if ia == 0 {
					// Omitted DI block: that range is all-valid. The
					// length is settled after the loop when the total
					// sample count is known; use the data block's span.
					iv.segs = append(iv.segs, segment{logical: logical, length: -1, kind: segZero})
				} else if err := addColumnBlock(iv, ia, logical, blocks.IDDI); err != nil {
					return nil, nil, err
				}
			}
			sampleIdx++
		}
		addr = ld.LDNext
	}
	finishReader(data)
	if iv != nil {
		// Zero segments got a placeholder length: each spans invalSize
		// bytes per sample of the matching data block.
		for i := range iv.segs {
			if iv.segs[i].kind == segZero && iv.segs[i].length < 0 {
				iv.segs[i].length = zeroSegLen(data, iv.segs[i].logical, recSize, invalSize)
			}
		}
		finishReader(iv)
	}
	return data, iv, nil
}

// addColumnBlock adds one DV/DI (or DZ) block as a segment.
func addColumnBlock(r *Reader, addr, logical int64, wantID string) error {
	id, err := blocks.PeekID(r.src, addr)
	if err != nil {
		return err
	}
	switch id {
	case wantID, blocks.IDDT, blocks.IDSD: // be lenient about the concrete stored type
		return r.addStored(addr, logical)
	case blocks.IDDZ:
		return r.addZipped(addr, logical)
	default:
		return fmt.Errorf("unexpected block %q in column-storage list at offset 0x%x", id, addr)
	}
}

// zeroSegLen computes the byte length of an omitted invalidation block
// from the corresponding data block's sample span.
func zeroSegLen(data *Reader, invalLogical int64, recSize, invalSize int) int64 {
	startSample := invalLogical / int64(invalSize)
	for _, s := range data.segs {
		if s.logical == startSample*int64(recSize) {
			samples := s.length / int64(recSize)
			return samples * int64(invalSize)
		}
	}
	return 0
}

// finishReader computes the size and sorts segments (shared with New).
func finishReader(r *Reader) {
	for _, s := range r.segs {
		if end := s.logical + s.length; end > r.size {
			r.size = end
		}
	}
	sortSegs(r.segs)
}
