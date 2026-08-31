package mf4_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

// TestGolden compares converted sample values against asammdf-generated
// JSON files (testdata/golden, produced by testdata/scripts/gen_golden.py).
// It is skipped when no golden files have been generated.
func TestGolden(t *testing.T) {
	goldens, _ := filepath.Glob("testdata/golden/*.json")
	if len(goldens) == 0 {
		t.Skip("no golden files; run testdata/scripts/gen_golden.py")
	}
	for _, gp := range goldens {
		t.Run(filepath.Base(gp), func(t *testing.T) {
			raw, err := os.ReadFile(gp)
			if err != nil {
				t.Fatal(err)
			}
			var golden struct {
				File     string                     `json:"file"`
				Channels map[string]json.RawMessage `json:"channels"`
			}
			if err := json.Unmarshal(raw, &golden); err != nil {
				t.Fatal(err)
			}
			f := openFile(t, golden.File)
			groups := f.Groups()
			for key, entryRaw := range golden.Channels {
				var entry struct {
					Group int      `json:"group"`
					Len   int      `json:"len"`
					First []any    `json:"first"`
					Sum   *float64 `json:"sum"`
					Error string   `json:"error"`
				}
				if err := json.Unmarshal(entryRaw, &entry); err != nil {
					t.Fatal(err)
				}
				if entry.Error != "" {
					continue // asammdf could not read it either
				}
				name := key[strings.Index(key, ":")+1:]
				if entry.Group >= len(groups) {
					t.Errorf("%s: group %d missing", key, entry.Group)
					continue
				}
				ch, ok := groups[entry.Group].Channel(name)
				if !ok {
					t.Errorf("%s: channel missing", key)
					continue
				}
				sig, err := ch.Read()
				if err != nil {
					t.Errorf("%s: %v", key, err)
					continue
				}
				if sig.Len() != entry.Len {
					t.Errorf("%s: len %d, want %d", key, sig.Len(), entry.Len)
					continue
				}
				checkHead(t, key, sig, entry.First)
				if entry.Sum != nil && sig.Type != mf4.SampleString && sig.Type != mf4.SampleBytes {
					sum := 0.0
					for _, v := range sig.Float64s() {
						if !math.IsNaN(v) {
							sum += v
						}
					}
					if !closeEnough(sum, *entry.Sum) {
						t.Errorf("%s: sum %v, want %v", key, sum, *entry.Sum)
					}
				}
			}
		})
	}
}

func checkHead(t *testing.T, key string, sig *mf4.Signal, want []any) {
	t.Helper()
	for i, w := range want {
		if i >= sig.Len() {
			return
		}
		switch sig.Type {
		case mf4.SampleString:
			if ws, ok := w.(string); ok && sig.Strings[i] != ws {
				t.Errorf("%s[%d] = %q, want %q", key, i, sig.Strings[i], ws)
			}
		case mf4.SampleBytes:
			// golden stores a lossy decode; skip byte-array comparison
		default:
			got := sig.Float64s()[i]
			switch wv := w.(type) {
			case nil: // NaN in golden
				if !math.IsNaN(got) {
					t.Errorf("%s[%d] = %v, want NaN", key, i, got)
				}
			case float64:
				if !closeEnough(got, wv) {
					t.Errorf("%s[%d] = %v, want %v", key, i, got, wv)
				}
			case string:
				if wf, err := strconv.ParseFloat(wv, 64); err == nil && !closeEnough(got, wf) {
					t.Errorf("%s[%d] = %v, want %v", key, i, got, wf)
				}
			default:
				t.Errorf("%s[%d]: unhandled golden type %T", key, i, w)
			}
		}
	}
}

func closeEnough(a, b float64) bool {
	if a == b {
		return true
	}
	diff := math.Abs(a - b)
	scale := math.Max(math.Abs(a), math.Abs(b))
	return diff <= 1e-9*math.Max(scale, 1)
}

var _ = fmt.Sprintf // keep fmt while the golden schema evolves
