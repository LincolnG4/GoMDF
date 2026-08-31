# GoMDF — fast ASAM MDF 4.x reader for Go

GoMDF reads ASAM MDF 4.x (`.mf4`) measurement files — the standard format
for automotive measurement data — with an API designed for large files and
for building tools (viewers, exporters) on top of it.

- **Fast on large files**: memory-mapped I/O, lazy block-level
  decompression with caching, and typed columnar decoding (no per-sample
  boxing). Channels decode in parallel.
- **Typed signals**: samples come back as `[]float64`, `[]int64`,
  `[]uint64`, `[]string` or `[][]byte` — never `[]interface{}`.
- **Streaming**: read whole channels, sample windows, or iterate chunk by
  chunk for progressive loading.
- **Robust**: no panics on malformed files (fuzz-tested); errors carry the
  block type and file offset.
- **Writes MDF files too**: a streaming writer for data loggers (bounded
  memory, crash-safe unfinalized layout, `Flush()` durability points) and
  a batch column API for exports, with optional DZ compression.
- **Decodes CAN bus logs**: point it at a `.dbc` (or use the one embedded
  in the log) and get physical signals instead of raw frame bytes.
- **Sample reductions**: reads precomputed min/mean/max aggregates, so a
  viewer can draw a multi-million-sample signal from a few hundred
  records.
- Supports sorted and unsorted files, compressed data (DZ deflate +
  transposition, inflated in parallel), data lists (DL/HL), MDF 4.2
  column storage (LD/DV/DI), variable-length data (VLSD/SD),
  invalidation bits, non-byte-aligned integer channels, all three
  channel-array (CA) storage types, the channel hierarchy (CH) tree,
  unfinalized logger files (standard finalization steps applied in
  memory), and the full set of conversion rules (linear, rational,
  algebraic formulas, value/range tables, text tables).

The reader is validated against the official ASAM MDF 4.2 example suite
(97/97 files) and cross-checked value-by-value against
[asammdf](https://github.com/danielhrisca/asammdf); written files are
verified to read back identically in asammdf.

**Performance** (2M records x 8 channels, identical results, vs asammdf
8.8.25): metadata open ~10x faster, uncompressed reads 4-5x, compressed
reads up to 10x, record-by-record writing 3x faster than asammdf's numpy
batch path at 16M records/s. Full table in the
[manual](docs/MANUAL.md#9-performance-guide).

Full usage documentation: **[docs/MANUAL.md](docs/MANUAL.md)**.

## Install

```
go get github.com/LincolnG4/GoMDF
```

## Usage

```go
f, err := mf4.Open("measurement.mf4")
if err != nil {
    log.Fatal(err)
}
defer f.Close()

// Metadata
fmt.Println(f.Version(), f.StartTime())
for _, g := range f.Groups() {
    fmt.Println(g.Name, g.RecordCount, len(g.Channels()))
}

// Read one channel (conversion applied; Time holds the master values)
ch, _ := f.Channel("EngineSpeed")
sig, err := ch.Read()
fmt.Println(sig.Time, sig.Floats)

// Raw values, or a window of samples
raw, _ := ch.Read(mf4.Raw())
win, _ := ch.Read(mf4.WithRange(1_000_000, 10_000))

// Everything, in parallel
signals, err := f.ReadAll()

// Stream a big group chunk by chunk
it := ch.Group().Chunks(ch.Group().Channels(), mf4.WithChunkSamples(100_000))
for sigs := range it.All() {
    // sigs are aligned; sigs[i].Offset marks the window start
}
if err := it.Err(); err != nil {
    log.Fatal(err)
}

// Attachments
atts, _ := f.Attachments()
data, _ := atts[0].Data()

// CAN logs: decode frames into physical signals with the embedded DBC
sigs, _ := f.DecodeBus()
for _, s := range sigs {
    plot(s.Time, s.Floats)   // s.QualifiedName() e.g. "MsgSine.Sine"
}

// Precomputed min/mean/max aggregates for fast zoomed-out plots
reds, _ := g.Reductions()
rs, _ := reds[0].Read(ch)
plotBand(rs.Mean.Time, rs.Min.Float64s(), rs.Max.Float64s())
```

See `examples/mdf-reader` for a complete program.

## Options

| Option | Effect |
|---|---|
| `mf4.WithoutMmap()` | plain reads instead of mmap (network shares, etc.) |
| `mf4.WithDecompressCacheSize(n)` | byte budget for cached decompressed blocks (default 128 MiB) |
| `mf4.Raw()` | skip conversions, native recorded values |
| `mf4.WithRange(from, n)` | sample window |
| `mf4.WithChunkSamples(n)` | chunk size for `Chunks` |

## Writing

```go
w, _ := mf4.Create("drive.mf4")            // add mf4.WithCompression() for DZ
eng, _ := w.NewGroup("Engine")             // gets a float64 "t" master at index 0
spd  := eng.Float64("EngineSpeed", "rpm")
gear := eng.Int("Gear", "", 8)
adc  := eng.Uint("RawADC", "V", 16)
eng.SetLinearConversion(adc, 0, 0.001)     // phys = 0.001 * raw
msg  := eng.String("Message")              // VLSD text channel

rec := eng.Record()                        // reusable, allocation-free
for running {
    rec.SetFloat64(0, tSeconds)
    rec.SetFloat64(spd, speed)
    rec.SetInt(gear, g)
    rec.SetUint(adc, raw)
    rec.SetString(msg, note)
    eng.Append(rec)
    // w.Flush() at checkpoints: the file on disk is readable even if
    // power is lost before Close (unfinalized-MDF logger layout).
}
w.Close()                                  // finalizes the file
```

Batch data goes through `AppendColumns`:

```go
g.AppendColumns(n, mf4.Column{Ch: 0, F: timestamps},
                   mf4.Column{Ch: temp, F: values})
```

Layout is picked automatically: one group without compression streams
into a single growing data block (crash-safe); several groups or
compression produce a sorted multi-group file with per-group chunked
data lists. Streaming throughput is on the order of tens of millions of
records per second.

## Not supported yet

- Bus databases other than DBC (ARXML, LDF); extended multiplexing
  (`SG_MUL_VAL_`)
- CA axis channels are not attached to their array (they read fine as
  ordinary channels)
- MDF 4.2 `RV`/`RI` column-oriented *reduction* data
- Writer: invalidation bits, attachments, events, non-linear conversions

Known divergences from asammdf (GoMDF follows the spec): UTF-16 string
channels are actually decoded; range-to-scale partial conversions use
the spec's `min <= v < max` matching for any sample count; CA element
byte offsets follow the spec formula (verified against raw records).

## Testing

`go test ./...` runs unit, sample-file and golden tests. Golden files
under `testdata/golden` are generated with
`testdata/scripts/gen_golden.py` (requires Python + asammdf) and
committed, so CI needs no Python.
