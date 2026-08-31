package mf4_test

import (
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

var vlsdValues = []string{"aa", "bbb", "c"}

// TestSortedVLSD: a sorted group whose string channel stores offsets into
// an SD block.
func TestSortedVLSD(t *testing.T) {
	fx := newFixture()
	fx.hd()

	stream, offsets := vlsdStream(vlsdValues)
	sd := fx.block("##SD", nil, stream)

	// Records: f64 time + u64 VLSD offset.
	var records []byte
	for i, off := range offsets {
		records = append(records, f64le(float64(i))...)
		records = append(records, u64le(off)...)
	}
	dt := fx.block("##DT", nil, records)

	txt := fx.cn(cnSpec{name: fx.tx("text"), data: sd, typ: 1 /*VLSD*/, dtype: 7 /*utf8*/, byteOff: 8, bitCount: 64})
	master := fx.cn(cnSpec{next: txt, name: fx.tx("time"), typ: 2, sync: 1, dtype: 4, byteOff: 0, bitCount: 64})
	cg := fx.cg(0, master, 0, uint64(len(vlsdValues)), 0, 16, 0)
	dg := fx.dg(cg, dt, 0)
	path := fx.finish(t, dg)

	f, err := mf4.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	ch, err := f.Channel("text")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Type != mf4.VLSD {
		t.Errorf("type = %v, want vlsd", ch.Type)
	}
	sig, err := ch.Read()
	if err != nil {
		t.Fatal(err)
	}
	if sig.Type != mf4.SampleString {
		t.Fatalf("sample type = %v", sig.Type)
	}
	for i, want := range vlsdValues {
		if sig.Strings[i] != want {
			t.Errorf("[%d] = %q, want %q", i, sig.Strings[i], want)
		}
	}
	if len(sig.Time) != 3 || sig.Time[2] != 2 {
		t.Errorf("time = %v", sig.Time)
	}
}

// TestUnsorted: two record IDs interleaved in one DT — a fixed group and
// a VLSD service group referenced by the string channel.
func TestUnsorted(t *testing.T) {
	fx := newFixture()
	fx.hd()

	_, offsets := vlsdStream(vlsdValues)
	vals := []byte{10, 20, 30}

	// Interleaved records: id 2 = VLSD entries, id 1 = fixed records
	// (f64 time, u8 value, u64 text offset -> 17 data bytes).
	var raw []byte
	for i := range vlsdValues {
		raw = append(raw, 2)
		raw = append(raw, u32le(uint32(len(vlsdValues[i])))...)
		raw = append(raw, vlsdValues[i]...)
		raw = append(raw, 1)
		raw = append(raw, f64le(float64(i))...)
		raw = append(raw, vals[i])
		raw = append(raw, u64le(offsets[i])...)
	}
	dt := fx.block("##DT", nil, raw)

	// VLSD service group: record ID 2, cg_flags bit 0.
	vlsdCG := fx.cg(0, 0, 2, uint64(len(vlsdValues)), 1, 0, 0)

	txt := fx.cn(cnSpec{name: fx.tx("text"), data: vlsdCG, typ: 1, dtype: 7, byteOff: 9, bitCount: 64})
	val := fx.cn(cnSpec{next: txt, name: fx.tx("value"), typ: 0, dtype: 0, byteOff: 8, bitCount: 8})
	master := fx.cn(cnSpec{next: val, name: fx.tx("time"), typ: 2, sync: 1, dtype: 4, byteOff: 0, bitCount: 64})
	cg1 := fx.cg(vlsdCG, master, 1, 3, 0, 17, 0)
	dg := fx.dg(cg1, dt, 1)
	path := fx.finish(t, dg)

	f, err := mf4.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if len(f.Groups()) != 1 {
		t.Fatalf("got %d public groups, want 1 (VLSD group hidden)", len(f.Groups()))
	}
	g := f.Groups()[0]
	if g.RecordCount != 3 {
		t.Errorf("record count = %d", g.RecordCount)
	}

	value, _ := g.Channel("value")
	sig, err := value.Read()
	if err != nil {
		t.Fatal(err)
	}
	got := sig.Float64s()
	for i, want := range []float64{10, 20, 30} {
		if got[i] != want {
			t.Errorf("value[%d] = %v, want %v", i, got[i], want)
		}
	}
	if len(sig.Time) != 3 || sig.Time[0] != 0 || sig.Time[2] != 2 {
		t.Errorf("time = %v", sig.Time)
	}

	text, _ := g.Channel("text")
	tsig, err := text.Read()
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range vlsdValues {
		if tsig.Strings[i] != want {
			t.Errorf("text[%d] = %q, want %q", i, tsig.Strings[i], want)
		}
	}
}
