package source

import "io"

// ReaderAt adapts any io.ReaderAt with a known size to the Source
// interface. It is the fallback used when memory-mapping is disabled or
// unavailable.
type ReaderAt struct {
	r    io.ReaderAt
	size int64
}

// NewReaderAt wraps r. If r is also an io.Closer, Close is forwarded.
func NewReaderAt(r io.ReaderAt, size int64) *ReaderAt {
	return &ReaderAt{r: r, size: size}
}

func (r *ReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if err := checkBounds(off, int64(len(p)), r.size); err != nil {
		return 0, err
	}
	return r.r.ReadAt(p, off)
}

func (r *ReaderAt) Slice(off, n int64) ([]byte, error) {
	if err := checkBounds(off, n, r.size); err != nil {
		return nil, err
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(io.NewSectionReader(r.r, off, n), buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (r *ReaderAt) Size() int64 { return r.size }

func (r *ReaderAt) Close() error {
	if c, ok := r.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
