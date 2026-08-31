// Package source provides random-access data sources for MDF files.
//
// All block decoding in this module reads through the Source interface at
// absolute file offsets. Implementations are safe for concurrent use: none
// of them keeps a read cursor.
package source

import (
	"errors"
	"fmt"
	"io"
)

// ErrOutOfBounds is returned when a read extends past the end of the source.
var ErrOutOfBounds = errors.New("read out of bounds")

// Source is a random-access view of an MDF file.
type Source interface {
	io.ReaderAt

	// Slice returns n bytes starting at off. For memory-mapped sources the
	// returned slice aliases the mapping (zero-copy) and must not be
	// modified or retained past Close. Other implementations allocate.
	Slice(off int64, n int64) ([]byte, error)

	// Size returns the total size of the source in bytes.
	Size() int64

	// Close releases the underlying resources.
	Close() error
}

func checkBounds(off, n, size int64) error {
	if off < 0 || n < 0 || off > size || n > size-off {
		return fmt.Errorf("%w: offset %d, length %d, size %d", ErrOutOfBounds, off, n, size)
	}
	return nil
}
