// Package blocks contains typed representations of the MDF 4.x block types
// and pure decoders for them.
//
// Every struct mirrors the on-disk layout described in the ASAM MDF 4.2
// base standard: a 24-byte header, a link section (kept as raw absolute
// file offsets, never resolved pointers), and a data section. Decoders are
// pure functions over a source.Source at an absolute address — no cursors,
// no shared state — so they are safe for concurrent use.
//
// The structs carry every field of the spec (including ones the reader does
// not use) so that symmetric encoders for write support can be added later
// without remodeling.
package blocks

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/LincolnG4/GoMDF/internal/source"
)

const (
	// HeaderSize is the size of the common block header ("##xx" id,
	// reserved, length, link count).
	HeaderSize = 24
	// LinkSize is the size of one link in the link section.
	LinkSize = 8
)

var le = binary.LittleEndian

// ErrInvalidBlock reports a structurally invalid block (bad ID, impossible
// length, truncated section).
var ErrInvalidBlock = errors.New("invalid block")

// BlockError decorates an error with the block ID and file offset where it
// occurred.
type BlockError struct {
	ID     string // expected or found block ID, e.g. "##CN"
	Offset int64  // absolute file offset of the block
	Err    error
}

func (e *BlockError) Error() string {
	return fmt.Sprintf("block %s at offset 0x%x: %v", e.ID, e.Offset, e.Err)
}

func (e *BlockError) Unwrap() error { return e.Err }

func blockErr(id string, off int64, err error) error {
	return &BlockError{ID: id, Offset: off, Err: err}
}

func blockErrf(id string, off int64, format string, args ...any) error {
	return blockErr(id, off, fmt.Errorf(format, args...))
}

// Header is the common 24-byte block header.
type Header struct {
	ID        string // 4 chars, "##xx"
	Length    uint64 // total block length including header
	LinkCount uint64
}

// DataLen returns the length of the data section.
func (h Header) DataLen() uint64 {
	return h.Length - HeaderSize - h.LinkCount*LinkSize
}

// DecodeHeader reads and validates the common header at addr. If wantID is
// non-empty the block ID must match it.
func DecodeHeader(src source.Source, addr int64, wantID string) (Header, error) {
	buf, err := src.Slice(addr, HeaderSize)
	if err != nil {
		return Header{}, blockErr(wantID, addr, err)
	}
	h := Header{
		ID:        string(buf[0:4]),
		Length:    le.Uint64(buf[8:16]),
		LinkCount: le.Uint64(buf[16:24]),
	}
	if h.ID[0] != '#' || h.ID[1] != '#' {
		return Header{}, blockErrf(wantID, addr, "%w: bad id %q", ErrInvalidBlock, h.ID)
	}
	if wantID != "" && h.ID != wantID {
		return Header{}, blockErrf(wantID, addr, "%w: found %q, want %q", ErrInvalidBlock, h.ID, wantID)
	}
	minLen := uint64(HeaderSize) + h.LinkCount*LinkSize
	if h.Length < minLen || h.LinkCount > (h.Length-HeaderSize)/LinkSize {
		return Header{}, blockErrf(h.ID, addr, "%w: length %d < header+%d links", ErrInvalidBlock, h.Length, h.LinkCount)
	}
	if uint64(src.Size()-addr) < h.Length {
		return Header{}, blockErrf(h.ID, addr, "%w: block length %d exceeds file size", ErrInvalidBlock, h.Length)
	}
	return h, nil
}

// PeekID returns the 4-byte block ID at addr without decoding the block.
func PeekID(src source.Source, addr int64) (string, error) {
	buf, err := src.Slice(addr, 4)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// raw is a fully sliced block: validated header, decoded link section and
// the raw data section bytes.
type raw struct {
	header Header
	links  []int64
	data   []byte
}

// decodeRaw validates the header at addr and slices out links and data.
// minLinks/minData guard against truncated sections before field access.
func decodeRaw(src source.Source, addr int64, wantID string, minLinks int, minData int) (raw, error) {
	h, err := DecodeHeader(src, addr, wantID)
	if err != nil {
		return raw{}, err
	}
	if h.LinkCount < uint64(minLinks) {
		return raw{}, blockErrf(h.ID, addr, "%w: %d links, want >= %d", ErrInvalidBlock, h.LinkCount, minLinks)
	}
	if h.DataLen() < uint64(minData) {
		return raw{}, blockErrf(h.ID, addr, "%w: data section %d bytes, want >= %d", ErrInvalidBlock, h.DataLen(), minData)
	}
	body, err := src.Slice(addr+HeaderSize, int64(h.Length)-HeaderSize)
	if err != nil {
		return raw{}, blockErr(h.ID, addr, err)
	}
	links := make([]int64, h.LinkCount)
	for i := range links {
		links[i] = int64(le.Uint64(body[i*LinkSize:]))
	}
	return raw{header: h, links: links, data: body[h.LinkCount*LinkSize:]}, nil
}

// link returns links[i], or 0 (NIL link) when the section is shorter.
func (r raw) link(i int) int64 {
	if i < 0 || i >= len(r.links) {
		return 0
	}
	return r.links[i]
}

// f64 decodes a little-endian float64.
func f64(b []byte) float64 { return math.Float64frombits(le.Uint64(b)) }
