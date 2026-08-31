package mf4_test

import (
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

// buildColumnFixture writes an MDF 4.2 column-storage file: 4 records of
// (f64 time, u8 value) in two DV blocks, invalidation bytes in DI blocks
// (the second one omitted = all valid).
func buildColumnFixture(t *testing.T, equalCount bool) *mf4.File {
	t.Helper()
	fx := newFixture()
	fx.setVersion(420, "4.20")
	fx.hd()

	// DV blocks: records are time f64 + value u8 (9 bytes), no
	// invalidation bytes inside.
	rec := func(ti float64, v byte) []byte {
		return append(f64le(ti), v)
	}
	dv1 := fx.block("##DV", nil, append(rec(0, 10), rec(1, 20)...))
	dv2 := fx.block("##DV", nil, append(rec(2, 30), rec(3, 40)...))
	// DI blocks: 1 invalidation byte per record; sample 1 invalid.
	di1 := fx.block("##DI", nil, []byte{0b0, 0b1})

	var ld int64
	if equalCount {
		ld = fx.ld([]int64{dv1, dv2}, []int64{di1, 0}, nil, 2)
	} else {
		ld = fx.ld([]int64{dv1, dv2}, []int64{di1, 0}, []uint64{0, 2}, 0)
	}

	val := fx.cn(cnSpec{name: fx.tx("value"), dtype: 0, byteOff: 8, bitCount: 8,
		flags: 1 << 1 /* invalidation bit valid */, invalPos: 0})
	master := fx.cn(cnSpec{next: val, name: fx.tx("time"), typ: 2, sync: 1, dtype: 4, bitCount: 64})
	cg := fx.cg(0, master, 0, 4, 0, 9 /*data bytes*/, 1 /*inval bytes*/)
	dg := fx.dg(cg, ld, 0)
	path := fx.finish(t, dg)

	f, err := mf4.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestColumnStorage(t *testing.T) {
	for _, equal := range []bool{false, true} {
		name := "sampleOffsets"
		if equal {
			name = "equalCount"
		}
		t.Run(name, func(t *testing.T) {
			f := buildColumnFixture(t, equal)
			g := f.Groups()[0]
			if g.RecordCount != 4 {
				t.Fatalf("record count %d", g.RecordCount)
			}
			ch, _ := g.Channel("value")
			sig, err := ch.Read()
			if err != nil {
				t.Fatal(err)
			}
			want := []uint64{10, 20, 30, 40}
			if sig.Len() != 4 {
				t.Fatalf("len %d", sig.Len())
			}
			for i, wv := range want {
				if sig.Uints[i] != wv {
					t.Errorf("[%d] = %d, want %d", i, sig.Uints[i], wv)
				}
				if sig.Time[i] != float64(i) {
					t.Errorf("time[%d] = %v", i, sig.Time[i])
				}
			}
			// Invalidation from the DI stream: only sample 1 invalid;
			// samples 2-3 come from an omitted (all-valid) DI block.
			if sig.Invalid == nil {
				t.Fatal("no invalidation bits")
			}
			for i, wantInv := range []bool{false, true, false, false} {
				if sig.Invalid.Get(i) != wantInv {
					t.Errorf("invalid[%d] = %v, want %v", i, sig.Invalid.Get(i), wantInv)
				}
			}
			// Window read across the block boundary.
			win, err := ch.Read(mf4.WithRange(1, 2), mf4.Raw())
			if err != nil {
				t.Fatal(err)
			}
			if win.Uints[0] != 20 || win.Uints[1] != 30 {
				t.Errorf("window = %v", win.Uints)
			}
		})
	}
}
