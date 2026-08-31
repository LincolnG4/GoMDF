package mf4

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

// Writer creates MDF 4.1 files. Define groups and channels first, then
// append records (streaming) or columns (batch); Close finalizes the
// file.
//
// Layout is chosen automatically:
//   - one group, no compression: records stream into a single growing
//     data block. The file on disk is marked unfinalized until Close, so
//     a recording interrupted by a crash or power loss stays readable
//     (the standard finalization steps recover it) — the mode for data
//     loggers in a running vehicle.
//   - several groups or compression: per-group chunks are written as the
//     data arrives and linked into per-group data lists at Close,
//     producing a sorted file (no record IDs, fastest to read).
//
// A Writer buffers about one chunk per group; memory use is bounded and
// independent of recording length.
type Writer struct {
	f   *os.File
	buf *bufio.Writer
	off int64 // current end-of-file offset

	cfg       writerConfig
	groups    []*GroupWriter
	started   bool
	closed    bool
	growing   bool // single growing DT block (crash-safe streaming)
	growAddr  int64
	growBytes int64

	patches []patch

	hdAddr int64
	fhAddr int64
}

type patch struct {
	off int64
	val uint64
}

type writerConfig struct {
	startTime time.Time
	program   string
	comment   string
	compress  bool
	chunkSize int
}

// WriterOption configures Create/NewWriter.
type WriterOption func(*writerConfig)

// WithStartTime sets the absolute measurement start time (default: now).
func WithStartTime(t time.Time) WriterOption {
	return func(c *writerConfig) { c.startTime = t }
}

// WithCompression deflates each data chunk (DZ blocks with byte
// transposition). Reduces size typically 3-10x for numeric data;
// disables the crash-safe growing-block layout.
func WithCompression() WriterOption {
	return func(c *writerConfig) { c.compress = true }
}

// WithChunkSize sets the per-group data chunk size in bytes (default
// 1 MiB, max 4 MiB when compression is on — the DZ block limit).
func WithChunkSize(n int) WriterOption {
	return func(c *writerConfig) {
		if n > 0 {
			c.chunkSize = n
		}
	}
}

// WithProgramID sets the 8-character writer program identification.
func WithProgramID(id string) WriterOption {
	return func(c *writerConfig) { c.program = id }
}

// WithFileComment attaches a comment to the file header.
func WithFileComment(s string) WriterOption {
	return func(c *writerConfig) { c.comment = s }
}

// Create creates an MDF file at path.
func Create(path string, opts ...WriterOption) (*Writer, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	return NewWriter(f, opts...)
}

// NewWriter writes an MDF file to f, which must support WriteAt (a
// regular file). The Writer takes ownership of f; Close closes it.
func NewWriter(f *os.File, opts ...WriterOption) (*Writer, error) {
	cfg := writerConfig{
		startTime: time.Now(),
		program:   "GoMDF   ",
		chunkSize: 1 << 20,
	}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.compress && cfg.chunkSize > 4<<20 {
		cfg.chunkSize = 4 << 20 // DZ uncompressed-size limit
	}
	return &Writer{f: f, buf: bufio.NewWriterSize(f, 1<<20), cfg: cfg}, nil
}

// write appends raw bytes at the current end of file.
func (w *Writer) write(b []byte) (addr int64, err error) {
	addr = w.off
	n, err := w.buf.Write(b)
	w.off += int64(n)
	return addr, err
}

// writeBlock appends a block at the next 8-aligned offset.
func (w *Writer) writeBlock(b []byte) (int64, error) {
	if err := w.align8(); err != nil {
		return 0, err
	}
	return w.write(b)
}

var zeros [8]byte

func (w *Writer) align8() error {
	if pad := int(w.off % 8); pad != 0 {
		_, err := w.write(zeros[:8-pad])
		return err
	}
	return nil
}

func (w *Writer) patchLink(off int64, val int64) {
	w.patches = append(w.patches, patch{off: off, val: uint64(val)})
}

// NewGroup adds a channel group (one record layout + master channel).
// All groups and channels must be defined before the first append.
// Unless WithoutMaster is given, a float64 time master channel "t" in
// seconds is created at index 0.
func (w *Writer) NewGroup(name string, opts ...GroupOption) (*GroupWriter, error) {
	if w.started {
		return nil, fmt.Errorf("NewGroup after data was appended")
	}
	g := &GroupWriter{w: w, name: name}
	for _, o := range opts {
		o(g)
	}
	if !g.noMaster {
		g.master("t", "s")
	}
	w.groups = append(w.groups, g)
	return g, nil
}

// GroupOption configures NewGroup.
type GroupOption func(*GroupWriter)

// WithoutMaster creates the group without an implicit time master
// channel (for value-only groups, or to define a custom master with
// MasterTime).
func WithoutMaster() GroupOption { return func(g *GroupWriter) { g.noMaster = true } }

// Flush writes all buffered data to the OS and syncs it to stable
// storage. In the single-group streaming layout the file is readable
// (as an unfinalized MDF) up to this point even if the process dies.
func (w *Writer) Flush() error {
	if err := w.buf.Flush(); err != nil {
		return err
	}
	return w.f.Sync()
}

// Close flushes remaining data, writes the data lists, patches all
// links and counters, marks the file finalized and closes it.
func (w *Writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	err := w.finalize()
	if cerr := w.f.Close(); err == nil {
		err = cerr
	}
	return err
}
