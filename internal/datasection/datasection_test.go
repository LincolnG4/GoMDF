package datasection

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/source"
)

func TestUntranspose(t *testing.T) {
	// 3 cols, 2 rows + 1 remainder byte. Original rows: {1,2,3},{4,5,6}, rem {9}.
	// Transposed (column-major): 1,4, 2,5, 3,6, then remainder 9.
	in := []byte{1, 4, 2, 5, 3, 6, 9}
	want := []byte{1, 2, 3, 4, 5, 6, 9}
	if got := untranspose(in, 3); !bytes.Equal(got, want) {
		t.Errorf("untranspose = %v, want %v", got, want)
	}
}

// TestReadAtMatchesFullRead opens every sample's data groups and checks
// that windowed ReadAt calls reproduce the full section content.
func TestReadAtMatchesFullRead(t *testing.T) {
	names, _ := filepath.Glob("../../samples/*.mf4")
	for _, path := range names {
		t.Run(filepath.Base(path), func(t *testing.T) {
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			src := source.NewMem(b)
			hd, err := blocks.DecodeHD(src, blocks.IDSize)
			if err != nil {
				t.Fatal(err)
			}
			for dga := hd.DGFirst; dga != 0; {
				dg, err := blocks.DecodeDG(src, dga)
				if err != nil {
					t.Fatal(err)
				}
				r, err := New(src, dg.Data, 2)
				if err != nil {
					t.Fatalf("New(0x%x): %v", dg.Data, err)
				}
				full := make([]byte, r.Size())
				if _, err := r.ReadAt(full, 0); err != nil {
					t.Fatalf("full ReadAt: %v", err)
				}
				// Re-read in awkward window sizes and compare.
				r2, err := New(src, dg.Data, 1)
				if err != nil {
					t.Fatal(err)
				}
				for _, win := range []int64{1, 7, 1024, r2.Size()} {
					if win == 0 {
						continue
					}
					got := make([]byte, r2.Size())
					for off := int64(0); off < r2.Size(); off += win {
						n := win
						if off+n > r2.Size() {
							n = r2.Size() - off
						}
						if _, err := r2.ReadAt(got[off:off+n], off); err != nil {
							t.Fatalf("windowed ReadAt(%d @%d): %v", n, off, err)
						}
					}
					if !bytes.Equal(got, full) {
						t.Fatalf("window %d: content mismatch", win)
					}
				}
				t.Logf("dg at 0x%x: %d bytes, %d segments", dga, r.Size(), len(r.segs))
				dga = dg.DGNext
			}
		})
	}
}
