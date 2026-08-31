package mf4_test

import (
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

func TestOpenMetadata(t *testing.T) {
	f, err := mf4.Open("samples/ASAP2_Demo_V171.mf4")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if v := f.Version(); v != 410 {
		t.Errorf("Version = %d, want 410", v)
	}
	if len(f.Groups()) == 0 || len(f.Channels()) == 0 {
		t.Fatalf("empty tree: %d groups, %d channels", len(f.Groups()), len(f.Channels()))
	}
	for _, g := range f.Groups() {
		t.Logf("group %q: %d records, %d channels, master=%v",
			g.Name, g.RecordCount, len(g.Channels()), g.Master())
		for _, c := range g.Channels() {
			t.Logf("  %v conv=%v", c, c.Conversion.Kind)
		}
	}
}

func TestOpenAllSamples(t *testing.T) {
	for _, name := range []string{
		"sample1.mf4", "sample2.mf4", "sample3.mf4",
		"sample_compressed.mf4", "Discrete_deflate.mf4",
	} {
		f, err := mf4.Open("samples/" + name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(f.Channels()) == 0 {
			t.Errorf("%s: no channels", name)
		}
		f.Close()
	}
}

func TestOpenGarbage(t *testing.T) {
	if _, err := mf4.Open("go.mod"); err == nil {
		t.Error("Open accepted a non-MDF file")
	}
	if _, err := mf4.Open("does-not-exist.mf4"); err == nil {
		t.Error("Open accepted a missing file")
	}
}
