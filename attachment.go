package mf4

import (
	"compress/zlib"
	"fmt"
	"io"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

// Attachment is a file attached to the measurement, either embedded in
// the MDF file or referenced by path.
type Attachment struct {
	// Filename is the original file name (or path for external
	// attachments).
	Filename string
	MimeType string
	Comment  string
	// Embedded reports whether the data is stored inside the MDF file;
	// external attachments only carry the Filename reference.
	Embedded   bool
	Compressed bool
	// Size is the original (uncompressed) data size for embedded
	// attachments.
	Size uint64

	file *File
	at   *blocks.AT
}

// Attachments returns the file's attachments.
func (f *File) Attachments() ([]Attachment, error) {
	var out []Attachment
	for addr := f.hd.ATFirst; addr != 0; {
		at, err := blocks.DecodeAT(f.src, addr)
		if err != nil {
			return nil, err
		}
		a := Attachment{
			Embedded:   at.Flags&blocks.ATFlagEmbedded != 0,
			Compressed: at.Flags&blocks.ATFlagCompressed != 0,
			Size:       at.OriginalSize,
			file:       f,
			at:         at,
		}
		if a.Filename, err = blocks.DecodeText(f.src, at.TXFilename); err != nil {
			return nil, err
		}
		if a.MimeType, err = blocks.DecodeText(f.src, at.TXMimetype); err != nil {
			return nil, err
		}
		if a.Comment, err = blocks.CommentText(f.src, at.MDComment); err != nil {
			return nil, err
		}
		out = append(out, a)
		addr = at.ATNext
	}
	return out, nil
}

// Data returns the embedded attachment content, decompressed if needed.
// It errors for external (non-embedded) attachments.
func (a *Attachment) Data() ([]byte, error) {
	if !a.Embedded {
		return nil, fmt.Errorf("attachment %q is external", a.Filename)
	}
	raw, err := a.file.src.Slice(a.at.EmbeddedAddr, int64(a.at.EmbeddedSize))
	if err != nil {
		return nil, err
	}
	if !a.Compressed {
		return append([]byte(nil), raw...), nil
	}
	zr, err := zlib.NewReader(newSliceReader(raw))
	if err != nil {
		return nil, fmt.Errorf("attachment %q: %w", a.Filename, err)
	}
	defer zr.Close()
	buf := make([]byte, a.Size)
	if _, err := io.ReadFull(zr, buf); err != nil {
		return nil, fmt.Errorf("attachment %q: %w", a.Filename, err)
	}
	return buf, nil
}

type sliceReader struct {
	b   []byte
	pos int
}

func newSliceReader(b []byte) *sliceReader { return &sliceReader{b: b} }

func (r *sliceReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.pos:])
	r.pos += n
	return n, nil
}
