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

func benchWriter(b *testing.B, opts ...mf4.WriterOption) {
	b.Helper()
	path := b.TempDir() + "/bench.mf4"
	w, err := mf4.Create(path, opts...)
	if err != nil {
		b.Fatal(err)
	}
	g, _ := w.NewGroup("Bench")
	c1 := g.Float64("Speed", "rpm")
	c2 := g.Float32("Torque", "Nm")
	c3 := g.Int("Gear", "", 8)
	c4 := g.Uint("Flags", "", 32)
	rec := g.Record()
	b.SetBytes(int64(8 + 8 + 4 + 1 + 4)) // record bytes
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec.SetFloat64(0, float64(i)*0.001)
		rec.SetFloat64(c1, float64(i))
		rec.SetFloat64(c2, float64(i)*0.5)
		rec.SetInt(c3, int64(i%8))
		rec.SetUint(c4, uint64(i))
		if err := g.Append(rec); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	if err := w.Close(); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkWriteRecords(b *testing.B)           { benchWriter(b) }
func BenchmarkWriteRecordsCompressed(b *testing.B) { benchWriter(b, mf4.WithCompression()) }
