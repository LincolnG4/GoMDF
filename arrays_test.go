package mf4_test

import (
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

// Records are (f64 time, u8 value) = 9 bytes for all array fixtures.
func arrRec(ti float64, v byte) []byte { return append(f64le(ti), v) }

// TestDGTemplateArray: each array element has its own data section
// (ca_data[k]) and its own cycle count and timestamps.
func TestDGTemplateArray(t *testing.T) {
	fx := newFixture()
	fx.hd()

	// Element 0: 3 samples; element 1: 2 samples; element 2: not recorded.
	dt0 := fx.block("##DT", nil, concat(arrRec(0, 10), arrRec(1, 11), arrRec(2, 12)))
	dt1 := fx.block("##DT", nil, concat(arrRec(0.5, 20), arrRec(1.5, 21)))

	ca := fx.ca(2, []uint64{3}, 0, []int64{dt0, dt1, 0}, []uint64{3, 2, 0})
	arr := fx.cnComp(cnSpec{name: fx.tx("arr"), dtype: 0, byteOff: 8, bitCount: 8}, ca)
	master := fx.cn(cnSpec{next: arr, name: fx.tx("t"), typ: 2, sync: 1, dtype: 4, bitCount: 64})
	cg := fx.cg(0, master, 0, 3, 0, 9, 0)
	dg := fx.dg(cg, dt0, 0) // ca_data[0] == dg_data per spec
	path := fx.finish(t, dg)

	f, err := mf4.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g := f.Groups()[0]

	e0, ok := g.Channel("arr[0]")
	if !ok {
		t.Fatal("arr[0] missing")
	}
	sig, err := e0.Read()
	if err != nil {
		t.Fatal(err)
	}
	if sig.Len() != 3 || sig.Uints[0] != 10 || sig.Uints[2] != 12 {
		t.Fatalf("arr[0] = %v", sig.Uints)
	}
	if len(sig.Time) != 3 || sig.Time[2] != 2 {
		t.Fatalf("arr[0] time = %v", sig.Time)
	}

	e1, _ := g.Channel("arr[1]")
	sig1, err := e1.Read()
	if err != nil {
		t.Fatal(err)
	}
	if sig1.Len() != 2 || sig1.Uints[0] != 20 || sig1.Uints[1] != 21 {
		t.Fatalf("arr[1] = %v", sig1.Uints)
	}
	// Element 1 has its own timestamps, not the group's.
	if sig1.Time[0] != 0.5 || sig1.Time[1] != 1.5 {
		t.Fatalf("arr[1] time = %v", sig1.Time)
	}
	// Element 2 was not recorded (NIL link) and must not appear.
	if _, ok := g.Channel("arr[2]"); ok {
		t.Error("unrecorded element arr[2] exposed")
	}
	// Windowed read still works per element.
	win, err := e0.Read(mf4.WithRange(1, 2), mf4.Raw())
	if err != nil {
		t.Fatal(err)
	}
	if win.Uints[0] != 11 || win.Uints[1] != 12 {
		t.Fatalf("window = %v", win.Uints)
	}
}

// TestCGTemplateArray: elements share one unsorted data group, each with
// record ID cg_record_id + k.
func TestCGTemplateArray(t *testing.T) {
	fx := newFixture()
	fx.hd()

	// Interleaved records: IDs 5 (element 0), 6 (element 1), 7 (element 2).
	var raw []byte
	add := func(id byte, ti float64, v byte) {
		raw = append(raw, id)
		raw = append(raw, arrRec(ti, v)...)
	}
	add(5, 0, 100)
	add(6, 0.1, 200)
	add(5, 1, 101)
	add(7, 0.2, 30)
	add(6, 1.1, 201)
	add(5, 2, 102)
	dt := fx.block("##DT", nil, raw)

	ca := fx.ca(1, []uint64{3}, 0, nil, []uint64{3, 2, 1})
	arr := fx.cnComp(cnSpec{name: fx.tx("arr"), dtype: 0, byteOff: 8, bitCount: 8}, ca)
	master := fx.cn(cnSpec{next: arr, name: fx.tx("t"), typ: 2, sync: 1, dtype: 4, bitCount: 64})
	cg := fx.cg(0, master, 5 /*record id*/, 3, 0, 9, 0)
	dg := fx.dg(cg, dt, 1 /*rec id size*/)
	path := fx.finish(t, dg)

	f, err := mf4.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g := f.Groups()[0]

	want := []struct {
		name  string
		vals  []uint64
		times []float64
	}{
		{"arr[0]", []uint64{100, 101, 102}, []float64{0, 1, 2}},
		{"arr[1]", []uint64{200, 201}, []float64{0.1, 1.1}},
		{"arr[2]", []uint64{30}, []float64{0.2}},
	}
	for _, w := range want {
		ch, ok := g.Channel(w.name)
		if !ok {
			t.Fatalf("%s missing", w.name)
		}
		sig, err := ch.Read()
		if err != nil {
			t.Fatalf("%s: %v", w.name, err)
		}
		if sig.Len() != len(w.vals) {
			t.Fatalf("%s: len %d, want %d", w.name, sig.Len(), len(w.vals))
		}
		for i, v := range w.vals {
			if sig.Uints[i] != v {
				t.Errorf("%s[%d] = %d, want %d", w.name, i, sig.Uints[i], v)
			}
			if sig.Time[i] != w.times[i] {
				t.Errorf("%s time[%d] = %v, want %v", w.name, i, sig.Time[i], w.times[i])
			}
		}
	}
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
