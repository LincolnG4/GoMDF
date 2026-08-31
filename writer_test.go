package mf4_test

import (
	"math"
	"path/filepath"
	"testing"
	"time"

	mf4 "github.com/LincolnG4/GoMDF"
)

// roundTrip writes a file with fn and reopens it with the reader.
func roundTrip(t *testing.T, fn func(w *mf4.Writer), opts ...mf4.WriterOption) *mf4.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "out.mf4")
	w, err := mf4.Create(path, opts...)
	if err != nil {
		t.Fatal(err)
	}
	fn(w)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := mf4.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestWriteStreamingSingleGroup(t *testing.T) {
	start := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	f := roundTrip(t, func(w *mf4.Writer) {
		g, err := w.NewGroup("Engine")
		if err != nil {
			t.Fatal(err)
		}
		spd := g.Float64("EngineSpeed", "rpm")
		gear := g.Int("Gear", "", 8)
		flags := g.Uint("Status", "", 16)
		rec := g.Record()
		for i := 0; i < 1000; i++ {
			rec.SetFloat64(0, float64(i)*0.01)
			rec.SetFloat64(spd, 800+float64(i))
			rec.SetInt(gear, int64(i%7-1))
			rec.SetUint(flags, uint64(i*3))
			if err := g.Append(rec); err != nil {
				t.Fatal(err)
			}
		}
	}, mf4.WithStartTime(start), mf4.WithFileComment("test drive"))

	if got := f.StartTime(); !got.Equal(start) {
		t.Errorf("start time %v, want %v", got, start)
	}
	g := f.Groups()[0]
	if g.Name != "Engine" || g.RecordCount != 1000 {
		t.Fatalf("group %q count %d", g.Name, g.RecordCount)
	}
	if g.Master() == nil || g.Master().Name != "t" {
		t.Fatal("missing master channel")
	}
	spd, _ := g.Channel("EngineSpeed")
	sig, err := spd.Read()
	if err != nil {
		t.Fatal(err)
	}
	if sig.Len() != 1000 || sig.Floats[0] != 800 || sig.Floats[999] != 1799 {
		t.Fatalf("speed = len %d, first %v, last %v", sig.Len(), sig.Floats[0], sig.Floats[999])
	}
	if sig.Time[500] != 5.0 {
		t.Errorf("time[500] = %v", sig.Time[500])
	}
	gear, _ := g.Channel("Gear")
	gs, err := gear.Read(mf4.Raw())
	if err != nil {
		t.Fatal(err)
	}
	if gs.Type != mf4.SampleInt64 || gs.Ints[0] != -1 {
		t.Fatalf("gear[0] = %v (%v)", gs.Ints[0], gs.Type)
	}
	if spd.Unit != "rpm" {
		t.Errorf("unit = %q", spd.Unit)
	}
}

func TestWriteMultiGroupAndColumns(t *testing.T) {
	n1, n2 := 5000, 300
	f := roundTrip(t, func(w *mf4.Writer) {
		fast, _ := w.NewGroup("Fast")
		sine := fast.Float32("Sine", "V")
		slow, _ := w.NewGroup("Slow")
		temp := slow.Float64("Temp", "degC")
		count := slow.Uint("Count", "", 32)

		tf := make([]float64, n1)
		sf := make([]float64, n1)
		for i := range tf {
			tf[i] = float64(i) * 0.001
			sf[i] = math.Sin(tf[i])
		}
		if err := fast.AppendColumns(n1,
			mf4.Column{Ch: 0, F: tf},
			mf4.Column{Ch: sine, F: sf}); err != nil {
			t.Fatal(err)
		}
		ts := make([]float64, n2)
		tv := make([]float64, n2)
		cv := make([]uint64, n2)
		for i := range ts {
			ts[i] = float64(i) * 0.1
			tv[i] = 20 + float64(i)*0.01
			cv[i] = uint64(i)
		}
		if err := slow.AppendColumns(n2,
			mf4.Column{Ch: 0, F: ts},
			mf4.Column{Ch: temp, F: tv},
			mf4.Column{Ch: count, U: cv}); err != nil {
			t.Fatal(err)
		}
	})

	if len(f.Groups()) != 2 {
		t.Fatalf("%d groups", len(f.Groups()))
	}
	sine, _ := f.Channel("Sine")
	sig, err := sine.Read()
	if err != nil {
		t.Fatal(err)
	}
	if sig.Len() != n1 {
		t.Fatalf("sine len %d", sig.Len())
	}
	if math.Abs(sig.Floats[1000]-math.Sin(1.0)) > 1e-6 {
		t.Errorf("sine[1000] = %v", sig.Floats[1000])
	}
	temp, _ := f.Channel("Temp")
	ts, err := temp.Read()
	if err != nil {
		t.Fatal(err)
	}
	if ts.Len() != n2 || ts.Floats[10] != 20.1 {
		t.Fatalf("temp len %d [10]=%v", ts.Len(), ts.Floats[10])
	}
}

func TestWriteCompressed(t *testing.T) {
	f := roundTrip(t, func(w *mf4.Writer) {
		g, _ := w.NewGroup("Data")
		v := g.Float64("Ramp", "")
		rec := g.Record()
		for i := 0; i < 200000; i++ {
			rec.SetFloat64(0, float64(i)*0.001)
			rec.SetFloat64(v, float64(i%100))
			if err := g.Append(rec); err != nil {
				t.Fatal(err)
			}
		}
	}, mf4.WithCompression(), mf4.WithChunkSize(64<<10))

	ch, _ := f.Channel("Ramp")
	sig, err := ch.Read()
	if err != nil {
		t.Fatal(err)
	}
	if sig.Len() != 200000 || sig.Floats[123456] != float64(123456%100) {
		t.Fatalf("len %d [123456]=%v", sig.Len(), sig.Floats[123456])
	}
}

func TestWriteVLSDStrings(t *testing.T) {
	msgs := []string{"engine start", "", "fault code P0301 detected", "done"}
	f := roundTrip(t, func(w *mf4.Writer) {
		g, _ := w.NewGroup("Events")
		msg := g.String("Message")
		rec := g.Record()
		for i, m := range msgs {
			rec.SetFloat64(0, float64(i))
			rec.SetString(msg, m)
			if err := g.Append(rec); err != nil {
				t.Fatal(err)
			}
		}
	})
	ch, _ := f.Channel("Message")
	sig, err := ch.Read()
	if err != nil {
		t.Fatal(err)
	}
	if sig.Type != mf4.SampleString {
		t.Fatalf("type %v", sig.Type)
	}
	for i, m := range msgs {
		if sig.Strings[i] != m {
			t.Errorf("[%d] = %q, want %q", i, sig.Strings[i], m)
		}
	}
}

func TestWriteLinearConversion(t *testing.T) {
	f := roundTrip(t, func(w *mf4.Writer) {
		g, _ := w.NewGroup("ADC")
		raw := g.Uint("Voltage", "V", 16)
		g.SetLinearConversion(raw, 0, 0.001) // mV counts -> V
		rec := g.Record()
		for i := 0; i < 10; i++ {
			rec.SetFloat64(0, float64(i))
			rec.SetUint(raw, uint64(i*500))
			if err := g.Append(rec); err != nil {
				t.Fatal(err)
			}
		}
	})
	ch, _ := f.Channel("Voltage")
	sig, err := ch.Read()
	if err != nil {
		t.Fatal(err)
	}
	if sig.Type != mf4.SampleFloat64 || sig.Floats[4] != 2.0 {
		t.Fatalf("converted[4] = %v (%v)", sig.Floats[4], sig.Type)
	}
	raw, _ := ch.Read(mf4.Raw())
	if raw.Type != mf4.SampleUint64 || raw.Uints[4] != 2000 {
		t.Fatalf("raw[4] = %v (%v)", raw.Uints[4], raw.Type)
	}
}

// TestWriteCrashRecovery simulates a power loss: records are flushed
// but Close never runs. The unfinalized file must still open and hold
// every flushed record.
func TestWriteCrashRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crash.mf4")
	w, err := mf4.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := w.NewGroup("Logger")
	v := g.Float64("Signal", "")
	rec := g.Record()
	for i := 0; i < 5000; i++ {
		rec.SetFloat64(0, float64(i)*0.01)
		rec.SetFloat64(v, float64(i))
		if err := g.Append(rec); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	// No Close: the process "dies" here.

	f, err := mf4.Open(path)
	if err != nil {
		t.Fatalf("open crashed file: %v", err)
	}
	defer f.Close()
	ch, err := f.Channel("Signal")
	if err != nil {
		t.Fatal(err)
	}
	sig, err := ch.Read()
	if err != nil {
		t.Fatal(err)
	}
	if sig.Len() != 5000 || sig.Floats[4999] != 4999 {
		t.Fatalf("recovered %d records, last %v", sig.Len(), sig.Floats[sig.Len()-1])
	}
}

func TestWriteErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "e.mf4")
	w, _ := mf4.Create(path)
	g, _ := w.NewGroup("G")
	v := g.Float64("V", "")
	rec := g.Record()
	rec.SetInt(v, 1) // type mismatch
	if err := g.Append(rec); err == nil {
		t.Error("type mismatch not reported")
	}
	rec2 := g.Record()
	rec2.SetFloat64(v, 1)
	if err := g.Append(rec2); err != nil {
		t.Fatal(err)
	}
	if _, err := w.NewGroup("late"); err == nil {
		t.Error("NewGroup after append not rejected")
	}
	w.Close()
	if err := g.Append(rec2); err == nil {
		t.Error("append after close not rejected")
	}
}
