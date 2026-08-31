package source

// Mem is a Source backed by an in-memory byte slice. It is used for
// de-interleaved unsorted data and in tests.
type Mem struct {
	b []byte
}

// NewMem wraps b in a Source. The slice is not copied.
func NewMem(b []byte) *Mem { return &Mem{b: b} }

func (m *Mem) ReadAt(p []byte, off int64) (int, error) {
	if err := checkBounds(off, int64(len(p)), int64(len(m.b))); err != nil {
		return 0, err
	}
	return copy(p, m.b[off:]), nil
}

func (m *Mem) Slice(off, n int64) ([]byte, error) {
	if err := checkBounds(off, n, int64(len(m.b))); err != nil {
		return nil, err
	}
	return m.b[off : off+n : off+n], nil
}

func (m *Mem) Size() int64 { return int64(len(m.b)) }

func (m *Mem) Close() error {
	m.b = nil
	return nil
}
