package mf4_test

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

// TestBusDecoding decodes the corpus CAN logs with their embedded DBC
// and checks the signals against hand-verified values.
func TestBusDecoding(t *testing.T) {
	root := os.Getenv("GOMDF_CORPUS")
	if root == "" {
		t.Skip("GOMDF_CORPUS not set")
	}
	// Expected first samples per (message id, signal), verified against
	// asammdf's extract_bus_logging output.
	want := map[string]struct {
		n     int
		first []float64
		unit  string
	}{
		"16.Sine":       {79, []float64{0, 0.0483, 0.0965}, "Volts"},
		"17.Sine":       {265, []float64{0.0314, 0.0628, 0.0941}, "Volts"},
		"17.SineJitter": {265, []float64{-0.7962, -0.7191, -0.5443}, "Volts"},
		"18.Peak1":      {159, []float64{0, 0, 0}, ""},
		"18.Peak2":      {159, []float64{0, 0, 0}, ""},
		"100.Curve":     {795, []float64{10, 10.8, 11.6}, ""},
		"100.Counter":   {795, []float64{0, 0, 0}, ""},
		"100.State":     {795, []float64{0, 0, 0}, ""},
		"101.Line":      {79, []float64{1, 25, 50}, ""},
		"102.Discrete":  {80, []float64{2, 4, 2}, ""},
	}

	for _, name := range []string{
		"BusLogging/CAN/Vector_CAN_DataFrame_Sort_ID.MF4",
		"BusLogging/CAN/Vector_CAN_DataFrame_Sort_Bus.MF4",
	} {
		t.Run(filepath.Base(name), func(t *testing.T) {
			f, err := mf4.Open(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			if len(f.BusFrameGroups()) == 0 {
				t.Fatal("no bus frame groups detected")
			}
			dbs, err := f.EmbeddedDatabases()
			if err != nil {
				t.Fatal(err)
			}
			if len(dbs) != 1 || !strings.HasSuffix(dbs[0].Name, ".dbc") {
				t.Fatalf("embedded databases = %v", dbs)
			}

			sigs, err := f.DecodeBus()
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]*mf4.BusSignal{}
			for _, s := range sigs {
				got[strings.TrimPrefix(s.QualifiedName(), "Msg")] = s
				got[key(s)] = s
			}
			if len(sigs) != len(want) {
				t.Errorf("%d signals, want %d", len(sigs), len(want))
			}
			for k, w := range want {
				s, ok := got[k]
				if !ok {
					t.Errorf("%s missing", k)
					continue
				}
				if s.Len() != w.n {
					t.Errorf("%s: %d samples, want %d", k, s.Len(), w.n)
					continue
				}
				for i, v := range w.first {
					if math.Abs(s.Floats[i]-v) > 1e-4 {
						t.Errorf("%s[%d] = %v, want %v", k, i, s.Floats[i], v)
					}
				}
				if s.Unit != w.unit {
					t.Errorf("%s: unit %q, want %q", k, s.Unit, w.unit)
				}
				if len(s.Time) != s.Len() {
					t.Errorf("%s: %d timestamps for %d samples", k, len(s.Time), s.Len())
				}
				for i := 1; i < len(s.Time); i++ {
					if s.Time[i] < s.Time[i-1] {
						t.Errorf("%s: time goes backwards at %d", k, i)
						break
					}
				}
			}
		})
	}
}

func key(s *mf4.BusSignal) string {
	return itoa(int(s.MessageID)) + "." + s.Name
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}

// TestBusDecodingExplicitDatabase feeds a DBC from disk instead of the
// embedded one.
func TestBusDecodingExplicitDatabase(t *testing.T) {
	root := os.Getenv("GOMDF_CORPUS")
	if root == "" {
		t.Skip("GOMDF_CORPUS not set")
	}
	src := filepath.Join(root, "BusLogging/CAN/Vector_CAN_DataFrame_Sort_ID.MF4")
	f, err := mf4.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Write the embedded database out and load it back from disk.
	atts, _ := f.Attachments()
	if len(atts) == 0 {
		t.Skip("no attachment")
	}
	data, err := atts[0].Data()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "easy.dbc")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := mf4.LoadDBC(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(db.MessageNames()) != 6 {
		t.Errorf("messages = %v", db.MessageNames())
	}
	sigs, err := f.DecodeBus(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(sigs) != 10 {
		t.Fatalf("%d signals, want 10", len(sigs))
	}
	// Signals are sorted by bus, id, name.
	if sigs[0].MessageID != 16 || sigs[0].Name != "Sine" {
		t.Errorf("first signal = %+v", sigs[0])
	}
}

// TestBusNoDatabase reports a clear error when nothing describes the bus.
func TestBusNoDatabase(t *testing.T) {
	f := openFile(t, "sample2.mf4")
	if groups := f.BusFrameGroups(); len(groups) != 0 {
		t.Errorf("non-bus file reported %d frame groups", len(groups))
	}
	if _, err := f.DecodeBus(); err == nil {
		t.Error("DecodeBus without any database succeeded")
	}
}
