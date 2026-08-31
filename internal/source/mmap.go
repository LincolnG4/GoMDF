package source

import (
	"fmt"
	"os"

	"github.com/edsrzf/mmap-go"
)

// MmapFile is a Source backed by a read-only memory mapping of a file.
// Slice is zero-copy. Note that if the underlying file is truncated by
// another process while mapped, reads may fault (SIGBUS); callers that
// cannot rule this out should use NewReaderAt instead.
type MmapFile struct {
	f *os.File
	m mmap.MMap
}

// OpenMmap memory-maps path read-only. It falls back with an error (not a
// panic) when the platform or file does not support mapping, e.g. empty
// files; callers should then fall back to NewReaderAt.
func OpenMmap(path string) (*MmapFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	m, err := mmap.Map(f, mmap.RDONLY, 0)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("mmap %s: %w", path, err)
	}
	return &MmapFile{f: f, m: m}, nil
}

func (s *MmapFile) ReadAt(p []byte, off int64) (int, error) {
	if err := checkBounds(off, int64(len(p)), s.Size()); err != nil {
		return 0, err
	}
	return copy(p, s.m[off:]), nil
}

func (s *MmapFile) Slice(off, n int64) ([]byte, error) {
	if err := checkBounds(off, n, s.Size()); err != nil {
		return nil, err
	}
	return s.m[off : off+n : off+n], nil
}

func (s *MmapFile) Size() int64 { return int64(len(s.m)) }

func (s *MmapFile) Close() error {
	err := s.m.Unmap()
	if cerr := s.f.Close(); err == nil {
		err = cerr
	}
	return err
}
