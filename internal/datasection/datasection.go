// Package datasection presents an MDF data section — whatever a dg_data or
// cn_data link points to (a plain DT/DV/SD/RD block, a DZ compressed
// block, a DL/HL data list chaining many of either) — as one logical,
// randomly addressable byte stream.
//
// Compressed (DZ) segments are decompressed lazily on first access and
// kept in a small per-reader LRU cache, so windowed reads touch only the
// blocks they need.
package datasection

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/source"
)

// ErrUnsupported marks valid MDF layouts this package cannot read yet.
var ErrUnsupported = errors.New("unsupported data section layout")

// ErrColumnStorage reports that the section uses MDF 4.2 column-oriented
// storage (LD/DV blocks); callers that know the record sizes should
// retry with NewColumnStorage.
var ErrColumnStorage = errors.New("column-oriented storage")

type segKind uint8

const (
	segStored segKind = iota // plain bytes in the file
	segDeflate
	segTransposeDeflate
	segZero // reads as zero bytes (omitted invalidation blocks)
)

// segment maps a logical byte range of the data section onto the file.
type segment struct {
	logical  int64 // logical offset of the first byte
	length   int64 // uncompressed content length
	addr     int64 // file offset of the content (stored) or compressed payload (DZ)
	compLen  int64 // compressed payload length (DZ only)
	zipParam uint32
	kind     segKind
}

// Reader is a random-access view of one data section.
type Reader struct {
	src   source.Source
	segs  []segment
	size  int64
	cache *lru
	clamp bool
}

// New resolves the data section starting at addr. addr == 0 yields an
// empty reader. cacheBytes bounds the decompressed-block cache
// (<= 0: the 128 MiB default).
func New(src source.Source, addr int64, cacheBytes int64) (*Reader, error) {
	return newReader(src, addr, cacheBytes, false)
}

// NewFinalizing is New for unfinalized files: a last data block whose
// stored length overruns the end of the file (finalization flag "update
// of length for last DT block required") is clamped to the file end.
func NewFinalizing(src source.Source, addr int64, cacheBytes int64) (*Reader, error) {
	return newReader(src, addr, cacheBytes, true)
}

func newReader(src source.Source, addr int64, cacheBytes int64, clamp bool) (*Reader, error) {
	r := &Reader{src: src, cache: newLRU(cacheBytes), clamp: clamp}
	if addr == 0 {
		return r, nil
	}
	if err := r.resolve(addr); err != nil {
		return nil, err
	}
	for _, s := range r.segs {
		if end := s.logical + s.length; end > r.size {
			r.size = end
		}
	}
	sortSegs(r.segs)
	return r, nil
}

func sortSegs(segs []segment) {
	sort.Slice(segs, func(i, j int) bool { return segs[i].logical < segs[j].logical })
}

// Size returns the logical (uncompressed) size of the data section.
func (r *Reader) Size() int64 { return r.size }

func (r *Reader) resolve(addr int64) error {
	id, err := blocks.PeekID(r.src, addr)
	if err != nil {
		return err
	}
	switch id {
	case blocks.IDDT, blocks.IDDV, blocks.IDSD, blocks.IDRD:
		return r.addStored(addr, 0)
	case blocks.IDDZ:
		return r.addZipped(addr, 0)
	case blocks.IDHL:
		hl, err := blocks.DecodeHL(r.src, addr)
		if err != nil {
			return err
		}
		return r.resolveDLChain(hl.DLFirst)
	case blocks.IDDL:
		return r.resolveDLChain(addr)
	case blocks.IDLD:
		return fmt.Errorf("%w at offset 0x%x", ErrColumnStorage, addr)
	case blocks.IDDI, blocks.IDRV, blocks.IDRI:
		return fmt.Errorf("%w: unexpected %s block", ErrUnsupported, id)
	default:
		return fmt.Errorf("unexpected data block %q at offset 0x%x", id, addr)
	}
}

// resolveDLChain walks a DL chain, adding one segment per referenced data
// block. Each leaf's own block type is inspected — DT and DZ entries can
// be mixed.
func (r *Reader) resolveDLChain(addr int64) error {
	globalIdx := 0 // block index across the whole chain, for equal-length lists
	for addr != 0 {
		dl, err := blocks.DecodeDL(r.src, addr)
		if err != nil {
			return err
		}
		for i, dataAddr := range dl.Data {
			if dataAddr == 0 {
				continue
			}
			var logical int64
			if dl.Flags&blocks.DLFlagEqualLength != 0 {
				logical = int64(dl.EqualLength) * int64(globalIdx)
			} else {
				logical = int64(dl.Offsets[i])
			}
			id, err := blocks.PeekID(r.src, dataAddr)
			if err != nil {
				return err
			}
			switch id {
			case blocks.IDDT, blocks.IDDV, blocks.IDSD, blocks.IDRD:
				err = r.addStored(dataAddr, logical)
			case blocks.IDDZ:
				err = r.addZipped(dataAddr, logical)
			default:
				err = fmt.Errorf("unexpected block %q in data list at offset 0x%x", id, dataAddr)
			}
			if err != nil {
				return err
			}
			globalIdx++
		}
		addr = dl.DLNext
	}
	return nil
}

func (r *Reader) addStored(addr, logical int64) error {
	h, err := blocks.DecodeHeader(r.src, addr, "")
	if err != nil && r.clamp {
		if h, err = clampedHeader(r.src, addr); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	length := int64(h.DataLen())
	if r.clamp {
		if max := r.src.Size() - addr - blocks.HeaderSize; length > max {
			// Block overruns the file: truncated while recording.
			length = max
		} else if length == 0 && max > 0 {
			// Unfinalized logger: the last DT block's length was never
			// updated and its records run to the end of the file.
			length = max
		}
	}
	r.segs = append(r.segs, segment{
		logical: logical,
		length:  length,
		addr:    addr + blocks.HeaderSize,
		kind:    segStored,
	})
	return nil
}

// clampedHeader re-reads a block header tolerating a stored length that
// overruns the file (the unfinalized-file case) by clamping it.
func clampedHeader(src source.Source, addr int64) (blocks.Header, error) {
	buf, err := src.Slice(addr, blocks.HeaderSize)
	if err != nil {
		return blocks.Header{}, err
	}
	h := blocks.Header{
		ID:        string(buf[0:4]),
		Length:    le64(buf[8:16]),
		LinkCount: le64(buf[16:24]),
	}
	if h.ID[0] != '#' || h.ID[1] != '#' || h.LinkCount != 0 {
		return h, fmt.Errorf("invalid data block %q at 0x%x", h.ID, addr)
	}
	if max := uint64(src.Size() - addr); h.Length > max {
		h.Length = max
	}
	if h.Length < blocks.HeaderSize {
		h.Length = blocks.HeaderSize
	}
	return h, nil
}

func le64(b []byte) uint64 {
	_ = b[7]
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

func (r *Reader) addZipped(addr, logical int64) error {
	dz, err := blocks.DecodeDZ(r.src, addr)
	if err != nil {
		return err
	}
	kind := segDeflate
	switch dz.ZipType {
	case blocks.ZipDeflate:
	case blocks.ZipTransposeDeflate:
		kind = segTransposeDeflate
	default:
		return fmt.Errorf("%w: DZ zip type %d", ErrUnsupported, dz.ZipType)
	}
	r.segs = append(r.segs, segment{
		logical:  logical,
		length:   int64(dz.OrgDataLength),
		addr:     dz.DataAddr,
		compLen:  int64(dz.DataLength),
		zipParam: dz.ZipParameter,
		kind:     kind,
	})
	return nil
}

// ReadAt implements io.ReaderAt over the logical data section.
func (r *Reader) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off > r.size {
		return 0, fmt.Errorf("%w: offset %d, size %d", source.ErrOutOfBounds, off, r.size)
	}
	r.prefetch(off, int64(len(p)))
	n := 0
	for n < len(p) && off < r.size {
		// Find the last segment starting at or before off.
		i := sort.Search(len(r.segs), func(i int) bool { return r.segs[i].logical > off }) - 1
		if i < 0 || off >= r.segs[i].logical+r.segs[i].length {
			return n, fmt.Errorf("data section gap at logical offset %d", off)
		}
		seg := &r.segs[i]
		delta := off - seg.logical
		want := int64(len(p) - n)
		if avail := seg.length - delta; want > avail {
			want = avail
		}
		switch seg.kind {
		case segStored:
			b, err := r.src.Slice(seg.addr+delta, want)
			if err != nil {
				return n, err
			}
			copy(p[n:], b)
		case segZero:
			for i := int64(0); i < want; i++ {
				p[n+int(i)] = 0
			}
		default:
			buf, err := r.decompressed(seg)
			if err != nil {
				return n, err
			}
			copy(p[n:], buf[delta:delta+want])
		}
		n += int(want)
		off += want
	}
	if n < len(p) {
		return n, fmt.Errorf("%w: read %d of %d bytes", source.ErrOutOfBounds, n, len(p))
	}
	return n, nil
}

// NewFromBytes wraps an in-memory buffer (already de-interleaved records
// or an SD-layout stream) as a data section.
func NewFromBytes(src source.Source) (*Reader, error) {
	n := src.Size()
	r := &Reader{src: src, size: n, cache: newLRU(1)}
	if n > 0 {
		r.segs = []segment{{logical: 0, length: n, addr: 0, kind: segStored}}
	}
	return r, nil
}

// prefetch decompresses, in parallel, all compressed segments of the
// range [off, off+n) that are not in the cache yet. Sequential reads of
// a freshly opened compressed file then use every core instead of
// inflating block by block.
func (r *Reader) prefetch(off, n int64) {
	first := sort.Search(len(r.segs), func(i int) bool { return r.segs[i].logical > off }) - 1
	if first < 0 {
		first = 0
	}
	// Look ahead beyond the requested range: sequential readers then
	// find the next blocks already inflated.
	lookahead := runtime.NumCPU() * 2
	var missing []*segment
	for i := first; i < len(r.segs) && len(missing) < lookahead; i++ {
		seg := &r.segs[i]
		if seg.kind == segStored {
			continue
		}
		if seg.logical >= off+n && len(missing) == 0 {
			break // range itself needs nothing
		}
		if _, ok := r.cache.get(seg.addr); !ok {
			missing = append(missing, seg)
		}
	}
	if len(missing) < 2 {
		return // nothing to parallelize
	}
	workers := runtime.NumCPU()
	if workers > len(missing) {
		workers = len(missing)
	}
	var wg sync.WaitGroup
	next := make(chan *segment, len(missing))
	for _, seg := range missing {
		next <- seg
	}
	close(next)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for seg := range next {
				// Errors surface on the sequential path right after.
				r.decompressed(seg) //nolint:errcheck
			}
		}()
	}
	wg.Wait()
}
