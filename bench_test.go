package mf4_test

import (
	"bytes"
	"os"
	"testing"

	mf4 "github.com/LincolnG4/GoMDF"
)

func BenchmarkOpen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		f, err := mf4.Open("samples/ASAP2_Demo_V171.mf4")
		if err != nil {
			b.Fatal(err)
		}
		f.Close()
	}
}

func BenchmarkReadChannel(b *testing.B) {
	f, err := mf4.Open("samples/sample1.mf4")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	ch := f.Channels()[1]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ch.Read(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadAll(b *testing.B) {
	f, err := mf4.Open("samples/ASAP2_Demo_V171.mf4")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := f.ReadAll(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadCompressed(b *testing.B) {
	f, err := mf4.Open("samples/sample_compressed.mf4")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	ch := f.Channels()[1]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ch.Read(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChunks(b *testing.B) {
	f, err := mf4.Open("samples/sample1.mf4")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	g := f.Groups()[0]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		it := g.Chunks(g.Channels(), mf4.WithChunkSamples(1000))
		for it.Next() {
		}
		if it.Err() != nil {
			b.Fatal(it.Err())
		}
	}
}

// FuzzOpen feeds mutated MDF bytes through the whole metadata pipeline;
// it must never panic.
func FuzzOpen(f *testing.F) {
	for _, name := range []string{"samples/sample2.mf4", "samples/sample3.mf4"} {
		if b, err := os.ReadFile(name); err == nil {
			f.Add(b)
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		file, err := mf4.OpenReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return
		}
		for _, c := range file.Channels() {
			c.Read() //nolint:errcheck // errors are fine, panics are not
		}
		file.Attachments() //nolint:errcheck
		file.Events()      //nolint:errcheck
		file.History()     //nolint:errcheck
		file.Close()
	})
}
