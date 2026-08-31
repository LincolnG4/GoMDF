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

	groups   []*ChannelGroup
	channels []*Channel // flattened, in file order
}

type config struct {
	noMmap    bool
	cacheSize int
}

// OpenOption configures Open.
type OpenOption func(*config)

// WithoutMmap forces the plain read path instead of memory-mapping the
// file. Use it for files on network shares or files that may be truncated
// while open.
func WithoutMmap() OpenOption { return func(c *config) { c.noMmap = true } }

// WithDecompressCacheSize sets how many decompressed data blocks (up to
// 4 MiB each) are cached per data section. The default is 8.
func WithDecompressCacheSize(n int) OpenOption {
	return func(c *config) {
		if n > 0 {
			c.cacheSize = n
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
	cfg := config{cacheSize: 8}
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
	if id.Unfinalized() {
		return nil, fmt.Errorf("%w (flags 0x%x)", ErrUnfinalized, id.UnfinalizedFlags)
	}
	hd, err := blocks.DecodeHD(src, blocks.IDSize)
	if err != nil {
		return nil, err
	}
	f := &File{src: src, cfg: cfg, id: id, hd: hd}
	if err := f.buildTree(); err != nil {
		return nil, err
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
