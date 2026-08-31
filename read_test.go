package mf4_test

import (
	"math"
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

// sawtooth is the known first-124-sample pattern of the int8 channel
// ASAM.M.SCALAR.SBYTE.IDENTICAL.DISCRETE in the ASAM demo files.
var sawtooth = []int64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, -126, -116, -106, -96, -86, -76, -66, -56, -46, -36, -26, -16, -6, 4, 14, 24, 34, 44, 54, 64, 74, 84, 94, 104, 114, 124, -122, -112, -102, -92, -82, -72, -62, -52, -42, -32, -22, -12, -2, 8, 18, 28, 38, 48, 58, 68, 78, 88, 98, 108, 118, -128, -118, -108, -98, -88, -78, -68, -58, -48, -38, -28, -18, -8, 2, 12, 22, 32, 42, 52, 62, 72, 82, 92, 102, 112, 122, -124, -114, -104, -94, -84, -74, -64, -54, -44, -34, 0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, -126, -116, -106, -96, -86, -76, -66, -56, -46, -36, -26}

func openFile(t *testing.T, name string) *mf4.File {
	t.Helper()
	f, err := mf4.Open("samples/" + name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func readInts(t *testing.T, f *mf4.File, name string, opts ...mf4.ReadOption) *mf4.Signal {
	t.Helper()
	ch, err := f.Channel(name)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := ch.Read(opts...)
	if err != nil {
		t.Fatal(err)
	}
	return sig
}

func TestReadSimpleDT(t *testing.T) {
	f := openFile(t, "sample2.mf4")
	sig := readInts(t, f, "channel_b")
	want := []uint64{5, 10, 0, 10, 5, 10, 10, 5, 0, 0, 10, 5, 10, 5, 10, 0, 0, 5, 5, 0}
	if sig.Type != mf4.SampleUint64 && sig.Type != mf4.SampleInt64 {
		t.Fatalf("type = %v", sig.Type)
	}
	if sig.Len() != len(want) {
		t.Fatalf("len = %d, want %d", sig.Len(), len(want))
	}
	got := sig.Float64s()
	for i, w := range want {
		if got[i] != float64(w) {
			t.Errorf("[%d] = %v, want %d", i, got[i], w)
		}
	}
	if sig.Time == nil || len(sig.Time) != sig.Len() {
		t.Errorf("no aligned master values (len %d)", len(sig.Time))
	}
}

func TestReadDeflate(t *testing.T) {
	f := openFile(t, "Discrete_deflate.mf4")
	sig := readInts(t, f, "ASAM.M.SCALAR.SBYTE.IDENTICAL.DISCRETE", mf4.Raw())
	if sig.Type != mf4.SampleInt64 {
		t.Fatalf("type = %v, want int64", sig.Type)
	}
	for i, w := range sawtooth {
		if sig.Ints[i] != w {
			t.Fatalf("[%d] = %d, want %d", i, sig.Ints[i], w)
		}
	}
}

func TestReadDataList(t *testing.T) {
	f := openFile(t, "ASAP2_Demo_V171.mf4")
	sig := readInts(t, f, "ASAM.M.SCALAR.SBYTE.IDENTICAL.DISCRETE", mf4.Raw())
	if sig.Type != mf4.SampleInt64 {
		t.Fatalf("type = %v, want int64", sig.Type)
	}
	for i, w := range sawtooth {
		if sig.Ints[i] != w {
			t.Fatalf("[%d] = %d, want %d", i, sig.Ints[i], w)
		}
	}
}

func TestNestedConversion(t *testing.T) {
	f := openFile(t, "sample3.mf4")
	sig := readInts(t, f, "VehSpd_Cval_CPC")
	want := []float64{99.74609375, 99.7578125, 99.69140625}
	if sig.Type != mf4.SampleFloat64 {
		t.Fatalf("type = %v, want float64", sig.Type)
	}
	if sig.Len() != len(want) {
		t.Fatalf("len = %d, want %d", sig.Len(), len(want))
	}
	for i, w := range want {
		if math.Abs(sig.Floats[i]-w) > 1e-12 {
			t.Errorf("[%d] = %v, want %v", i, sig.Floats[i], w)
		}
	}
}

func TestRangeRead(t *testing.T) {
	f := openFile(t, "ASAP2_Demo_V171.mf4")
	full := readInts(t, f, "ASAM.M.SCALAR.SBYTE.IDENTICAL.DISCRETE", mf4.Raw())
	win := readInts(t, f, "ASAM.M.SCALAR.SBYTE.IDENTICAL.DISCRETE", mf4.Raw(), mf4.WithRange(50, 30))
	if win.Len() != 30 || win.Offset != 50 {
		t.Fatalf("window len=%d offset=%d", win.Len(), win.Offset)
	}
	for i := 0; i < 30; i++ {
		if win.Ints[i] != full.Ints[50+i] {
			t.Errorf("window[%d] = %d, want %d", i, win.Ints[i], full.Ints[50+i])
		}
	}
	// Tiny chunk size must give identical results.
	tiny := readInts(t, f, "ASAM.M.SCALAR.SBYTE.IDENTICAL.DISCRETE", mf4.Raw(), mf4.WithChunkSamples(7))
	for i := range full.Ints {
		if tiny.Ints[i] != full.Ints[i] {
			t.Fatalf("chunked[%d] = %d, want %d", i, tiny.Ints[i], full.Ints[i])
		}
	}
}

func TestReadAll(t *testing.T) {
	f := openFile(t, "ASAP2_Demo_V171.mf4")
	sigs, err := f.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(sigs) == 0 {
		t.Fatal("no signals")
	}
	for name, sig := range sigs {
		if sig.Len() == 0 && sig.Channel.Group().RecordCount > 0 {
			t.Errorf("%s: empty signal for %d records", name, sig.Channel.Group().RecordCount)
		}
	}
}

func TestReadAllSamples(t *testing.T) {
	for _, name := range []string{
		"sample1.mf4", "sample2.mf4", "sample3.mf4",
		"sample_compressed.mf4", "Discrete_deflate.mf4",
	} {
		f := openFile(t, name)
		if _, err := f.ReadAll(); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
