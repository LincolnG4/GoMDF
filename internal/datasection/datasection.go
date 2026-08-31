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
	"sort"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/source"
)

// ErrUnsupported marks valid MDF layouts this package cannot read yet
// (MDF 4.2 column-oriented storage).
var ErrUnsupported = errors.New("unsupported data section layout")

type segKind uint8

const (
	segStored segKind = iota // plain bytes in the file
	segDeflate
	segTransposeDeflate
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
}

// New resolves the data section starting at addr. addr == 0 yields an
// empty reader. cacheSize is the number of decompressed DZ payloads kept.
func New(src source.Source, addr int64, cacheSize int) (*Reader, error) {
	r := &Reader{src: src, cache: newLRU(cacheSize)}
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
	sort.Slice(r.segs, func(i, j int) bool { return r.segs[i].logical < r.segs[j].logical })
	return r, nil
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
	case blocks.IDLD, blocks.IDDI, blocks.IDRV, blocks.IDRI:
		return fmt.Errorf("%w: %s column-oriented storage", ErrUnsupported, id)
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
	if err != nil {
		return err
	}
	r.segs = append(r.segs, segment{
		logical: logical,
		length:  int64(h.DataLen()),
		addr:    addr + blocks.HeaderSize,
		kind:    segStored,
	})
	return nil
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
