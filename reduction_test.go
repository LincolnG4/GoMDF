package mf4_test

import (
	"os"
	"path/filepath"
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

// TestSampleReduction reads the corpus sample-reduction example and
// checks the aggregates are self-consistent. Skipped without corpus.
func TestSampleReduction(t *testing.T) {
	root := os.Getenv("GOMDF_CORPUS")
	if root == "" {
		t.Skip("GOMDF_CORPUS not set")
	}
	f, err := mf4.Open(filepath.Join(root, "SampleReduction/Simple/Vector_SampleReduction.mf4"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	found := 0
	for _, g := range f.Groups() {
		reds, err := g.Reductions()
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range reds {
			if r.CycleCount == 0 {
				continue
			}
			found++
			for _, ch := range g.Channels() {
				rs, err := r.Read(ch)
				if err != nil {
					t.Fatalf("%s reduction: %v", ch.Name, err)
				}
				if uint64(rs.Mean.Len()) != r.CycleCount {
					t.Fatalf("%s: %d reduced samples, want %d", ch.Name, rs.Mean.Len(), r.CycleCount)
				}
				mean, min, max := rs.Mean.Float64s(), rs.Min.Float64s(), rs.Max.Float64s()
				if mean == nil {
					continue
				}
				if ch == g.Master() {
					// Master sub-records are interval start / raster
					// min / raster max: check monotonic starts instead.
					for i := 1; i < len(mean); i++ {
						if mean[i] <= mean[i-1] {
							t.Fatalf("master interval starts not increasing at %d: %v %v", i, mean[i-1], mean[i])
						}
					}
					continue
				}
				for i := range mean {
					if !(min[i] <= mean[i] && mean[i] <= max[i]) {
						t.Fatalf("%s[%d]: min %v mean %v max %v", ch.Name, i, min[i], mean[i], max[i])
					}
				}
				if ch != g.Master() && rs.Mean.Time == nil {
					t.Errorf("%s: reduced signal without time", ch.Name)
				}
			}
			t.Logf("group %q: interval %v sync %d, %d intervals", g.Name, r.Interval, r.Sync, r.CycleCount)
		}
	}
	if found == 0 {
		t.Fatal("no reductions found in sample file")
	}
}
