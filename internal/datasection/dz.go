package datasection

import (
	"compress/zlib"
	"fmt"
	"io"
)

// decompressed returns the full uncompressed content of a DZ segment,
// via the cache.
func (r *Reader) decompressed(seg *segment) ([]byte, error) {
	if buf, ok := r.cache.get(seg.addr); ok {
		return buf, nil
	}
	comp, err := r.src.Slice(seg.addr, seg.compLen)
	if err != nil {
		return nil, err
	}
	buf, err := inflate(comp, seg.length)
	if err != nil {
		return nil, fmt.Errorf("DZ block at 0x%x: %w", seg.addr, err)
	}
	if seg.kind == segTransposeDeflate {
		buf = untranspose(buf, int(seg.zipParam))
	}
	r.cache.put(seg.addr, buf)
	return buf, nil
}

func inflate(comp []byte, orgLen int64) ([]byte, error) {
	zr, err := zlib.NewReader(newByteReader(comp))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	buf := make([]byte, orgLen)
	if _, err := io.ReadFull(zr, buf); err != nil {
		return nil, fmt.Errorf("inflate: %w", err)
	}
	return buf, nil
}

// untranspose reverses the byte transposition applied before deflate:
// the writer stored the data matrix (rows of `cols` bytes) column-major.
func untranspose(in []byte, cols int) []byte {
	if cols <= 1 || cols >= len(in) {
		return in
	}
	rows := len(in) / cols
	body := rows * cols
	out := make([]byte, len(in))
	for j := 0; j < cols; j++ {
		col := in[j*rows : (j+1)*rows]
		for i, b := range col {
			out[i*cols+j] = b
		}
	}
	copy(out[body:], in[body:]) // trailing remainder is stored as-is
	return out
}

// byteReader avoids the bufio wrapper zlib adds for plain io.Readers.
type byteReader struct {
	b   []byte
	pos int
}

func newByteReader(b []byte) *byteReader { return &byteReader{b: b} }

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.pos:])
	r.pos += n
	return n, nil
}

func (r *byteReader) ReadByte() (byte, error) {
	if r.pos >= len(r.b) {
		return 0, io.EOF
	}
	b := r.b[r.pos]
	r.pos++
	return b, nil
}
