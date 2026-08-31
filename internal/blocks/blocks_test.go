package blocks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LincolnG4/GoMDF/internal/source"
)

func openSample(t *testing.T, name string) source.Source {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("../../samples", name))
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	return source.NewMem(b)
}

// TestWalkSamples decodes the full metadata tree of every sample file.
func TestWalkSamples(t *testing.T) {
	names, err := filepath.Glob("../../samples/*.mf4")
	if err != nil || len(names) == 0 {
		t.Fatalf("no samples found: %v", err)
	}
	for _, path := range names {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			src := openSample(t, name)
			defer src.Close()

			id, err := DecodeID(src)
			if err != nil {
				t.Fatalf("ID: %v", err)
			}
			if id.Version < 400 || id.Version >= 500 {
				t.Fatalf("unexpected version %d", id.Version)
			}
			hd, err := DecodeHD(src, IDSize)
			if err != nil {
				t.Fatalf("HD: %v", err)
			}

			var dgN, cgN, cnN, ccN int
			for dga := hd.DGFirst; dga != 0; {
				dg, err := DecodeDG(src, dga)
				if err != nil {
					t.Fatalf("DG: %v", err)
				}
				dgN++
				for cga := dg.CGFirst; cga != 0; {
					cg, err := DecodeCG(src, cga)
					if err != nil {
						t.Fatalf("CG: %v", err)
					}
					cgN++
					for cna := cg.CNFirst; cna != 0; {
						cn, err := DecodeCN(src, cna)
						if err != nil {
							t.Fatalf("CN: %v", err)
						}
						cnN++
						if name, err := DecodeText(src, cn.TXName); err != nil {
							t.Errorf("CN name: %v", err)
						} else if name == "" && cn.TXName != 0 {
							t.Errorf("empty channel name with non-nil link")
						}
						if cc, err := DecodeCC(src, cn.CCConversion); err != nil {
							t.Errorf("CC: %v", err)
						} else if cc != nil {
							ccN++
						}
						if _, err := DecodeSI(src, cn.SISource); err != nil {
							t.Errorf("SI: %v", err)
						}
						cna = cn.CNNext
					}
					cga = cg.CGNext
				}
				dga = dg.DGNext
			}

			// File history chain.
			fhN := 0
			for fha := hd.FHFirst; fha != 0; {
				fh, err := DecodeFH(src, fha)
				if err != nil {
					t.Fatalf("FH: %v", err)
				}
				fhN++
				fha = fh.FHNext
			}
			// Attachments.
			atN := 0
			for ata := hd.ATFirst; ata != 0; {
				at, err := DecodeAT(src, ata)
				if err != nil {
					t.Fatalf("AT: %v", err)
				}
				atN++
				ata = at.ATNext
			}
			// Events.
			evN := 0
			for eva := hd.EVFirst; eva != 0; {
				ev, err := DecodeEV(src, eva)
				if err != nil {
					t.Fatalf("EV: %v", err)
				}
				evN++
				eva = ev.EVNext
			}

			if dgN == 0 || cgN == 0 || cnN == 0 {
				t.Fatalf("empty tree: %d DG, %d CG, %d CN", dgN, cgN, cnN)
			}
			if fhN == 0 {
				t.Errorf("no file history (required by spec)")
			}
			t.Logf("v%d %d DG, %d CG, %d CN, %d CC, %d FH, %d AT, %d EV",
				id.Version, dgN, cgN, cnN, ccN, fhN, atN, evN)
		})
	}
}

// TestDecodeErrors exercises the no-panic guarantee on malformed input.
func TestDecodeErrors(t *testing.T) {
	if _, err := DecodeID(source.NewMem([]byte("not an mdf file, but 64 bytes long padding padding padding pad!"))); err == nil {
		t.Error("DecodeID accepted garbage")
	}
	if _, err := DecodeID(source.NewMem([]byte("short"))); err == nil {
		t.Error("DecodeID accepted short file")
	}
	if _, err := DecodeHD(source.NewMem(make([]byte, 128)), 64); err == nil {
		t.Error("DecodeHD accepted zero bytes")
	}
	// Header claiming a huge link count.
	bad := make([]byte, 64)
	copy(bad, "##CN")
	le.PutUint64(bad[8:], 64)     // length
	le.PutUint64(bad[16:], 1<<60) // link count
	if _, err := DecodeCN(source.NewMem(bad), 0); err == nil {
		t.Error("DecodeCN accepted absurd link count")
	}
}
