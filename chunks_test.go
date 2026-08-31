package mf4_test

import (
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

func TestChunks(t *testing.T) {
	f := openFile(t, "ASAP2_Demo_V171.mf4")
	ch, err := f.Channel("ASAM.M.SCALAR.SBYTE.IDENTICAL.DISCRETE")
	if err != nil {
		t.Fatal(err)
	}
	g := ch.Group()
	full, err := ch.Read(mf4.Raw())
	if err != nil {
		t.Fatal(err)
	}

	it := g.Chunks([]*mf4.Channel{ch}, mf4.Raw(), mf4.WithChunkSamples(100))
	var got []int64
	windows := 0
	for it.Next() {
		sigs := it.Signals()
		if len(sigs) != 1 {
			t.Fatalf("got %d signals", len(sigs))
		}
		if sigs[0].Offset != len(got) {
			t.Errorf("window offset %d, want %d", sigs[0].Offset, len(got))
		}
		got = append(got, sigs[0].Ints...)
		windows++
	}
	if it.Err() != nil {
		t.Fatal(it.Err())
	}
	if windows < 2 {
		t.Fatalf("expected multiple windows, got %d", windows)
	}
	if len(got) != full.Len() {
		t.Fatalf("chunked total %d, want %d", len(got), full.Len())
	}
	for i := range got {
		if got[i] != full.Ints[i] {
			t.Fatalf("[%d] = %d, want %d", i, got[i], full.Ints[i])
		}
	}
}

func TestChunksRangeOverFunc(t *testing.T) {
	f := openFile(t, "sample2.mf4")
	g := f.Groups()[0]
	n := 0
	it := g.Chunks(g.Channels(), mf4.WithChunkSamples(8))
	for sigs := range it.All() {
		for _, s := range sigs {
			n += s.Len()
		}
	}
	if it.Err() != nil {
		t.Fatal(it.Err())
	}
	want := int(g.RecordCount) * len(g.Channels())
	if n != want {
		t.Fatalf("total samples %d, want %d", n, want)
	}
}

func TestChunksWrongGroup(t *testing.T) {
	f := openFile(t, "ASAP2_Demo_V171.mf4")
	g0 := f.Groups()[0]
	g1 := f.Groups()[1]
	it := g0.Chunks(g1.Channels())
	if it.Next() || it.Err() == nil {
		t.Error("accepted channels from another group")
	}
}
