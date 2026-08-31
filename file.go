package mf4

import (
	"fmt"
	"io"
	"time"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/source"
)

// File is an open MDF 4.x file. Metadata (groups, channels, conversions)
// is decoded eagerly by Open; sample data is decoded on demand.
//
// A File is safe for concurrent reads.
type File struct {
	src source.Source
	cfg config

	id *blocks.ID
	hd *blocks.HD
	// finalize marks an unfinalized file whose finalization steps are
	// applied in memory while reading.
	finalize bool

	groups   []*ChannelGroup
	channels []*Channel // flattened, in file order
}

type config struct {
	noMmap     bool
	cacheBytes int64
}

// OpenOption configures Open.
type OpenOption func(*config)

// WithoutMmap forces the plain read path instead of memory-mapping the
// file. Use it for files on network shares or files that may be truncated
// while open.
func WithoutMmap() OpenOption { return func(c *config) { c.noMmap = true } }

// WithDecompressCacheSize bounds the per-data-section cache of
// decompressed blocks to n bytes (default 128 MiB). A cache large
// enough for the compressed groups being worked on makes repeated and
// windowed reads of compressed files as fast as uncompressed ones.
func WithDecompressCacheSize(n int64) OpenOption {
	return func(c *config) {
		if n > 0 {
			c.cacheBytes = n
		}
	}
}

// Open opens an MDF file. The file is memory-mapped when possible; see
// WithoutMmap.
func Open(path string, opts ...OpenOption) (*File, error) {
	cfg := defaultConfig(opts)
	var src source.Source
	if cfg.noMmap {
		f, err := osOpenSized(path)
		if err != nil {
			return nil, err
		}
		src = f
	} else {
		m, err := source.OpenMmap(path)
		if err != nil {
			// Fall back to plain reads (empty file, exotic FS, ...).
			f, ferr := osOpenSized(path)
			if ferr != nil {
				return nil, err
			}
			src = f
		} else {
			src = m
		}
	}
	f, err := newFile(src, cfg)
	if err != nil {
		src.Close()
		return nil, err
	}
	return f, nil
}

// OpenReader reads an MDF file through any io.ReaderAt (e.g. a section of
// a larger stream). size is the total file size. If r implements
// io.Closer, File.Close closes it.
func OpenReader(r io.ReaderAt, size int64, opts ...OpenOption) (*File, error) {
	return newFile(source.NewReaderAt(r, size), defaultConfig(opts))
}

func defaultConfig(opts []OpenOption) config {
	cfg := config{cacheBytes: 128 << 20}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

func newFile(src source.Source, cfg config) (*File, error) {
	id, err := blocks.DecodeID(src)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotMDF4, err)
	}
	if id.Version < 400 {
		return nil, fmt.Errorf("%w: version %d", ErrNotMDF4, id.Version)
	}
	finalize := false
	if id.Unfinalized() {
		// Standard finalization steps this reader can apply in memory:
		//   bit 0: update cycle counters for CG/CA (recomputed from data)
		//   bit 1: update cycle counters for SR (SR data not exposed)
		//   bit 2: update length of last DT block (clamped to file end)
		//   bit 3: update length of last RD block (RD data not exposed)
		//   bit 5: update VLSD CG data bytes (not used by this reader)
		// Files needing DL updates (bit 4) or VLSD offset rewrites
		// (bit 6), or with custom flags, cannot be read safely.
		const fixable = 1<<0 | 1<<1 | 1<<2 | 1<<3 | 1<<5
		if id.CustomUnfinFlags != 0 || id.UnfinalizedFlags&^uint16(fixable) != 0 {
			return nil, fmt.Errorf("%w (flags 0x%x, custom 0x%x)", ErrUnfinalized, id.UnfinalizedFlags, id.CustomUnfinFlags)
		}
		finalize = true
	}
	hd, err := blocks.DecodeHD(src, blocks.IDSize)
	if err != nil {
		return nil, err
	}
	f := &File{src: src, cfg: cfg, id: id, hd: hd, finalize: finalize}
	if err := f.buildTree(); err != nil {
		return nil, err
	}
	if finalize && id.UnfinalizedFlags&1 != 0 {
		if err := f.fixCycleCounts(); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// Close releases the underlying file or mapping. Signals already read
// remain valid; further reads fail.
func (f *File) Close() error { return f.src.Close() }

// Version returns the MDF version number (400, 410, 420, ...).
func (f *File) Version() uint16 { return f.id.Version }

// Program returns the identification of the tool that wrote the file.
func (f *File) Program() string { return f.id.Program }

// StartTime returns the absolute start time of the measurement.
func (f *File) StartTime() time.Time {
	return time.Unix(0, int64(f.hd.StartTimeNS)).UTC()
}

// Groups returns the channel groups in file order. VLSD service groups
// are not included.
func (f *File) Groups() []*ChannelGroup { return f.groups }

// Channels returns every channel of every group, in file order.
func (f *File) Channels() []*Channel { return f.channels }

// Channel returns the first channel with the given name across all
// groups. Use ChannelGroup.Channel to disambiguate duplicated names.
func (f *File) Channel(name string) (*Channel, error) {
	for _, c := range f.channels {
		if c.Name == name {
			return c, nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrChannelNotFound, name)
}

func osOpenSized(path string) (source.Source, error) {
	fh, err := osOpen(path)
	if err != nil {
		return nil, err
	}
	st, err := fh.Stat()
	if err != nil {
		fh.Close()
		return nil, err
	}
	return source.NewReaderAt(fh, st.Size()), nil
}
