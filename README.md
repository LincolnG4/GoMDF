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
- Supports sorted and unsorted files, compressed data (DZ deflate +
  transposition), data lists (DL/HL), variable-length data (VLSD/SD),
  invalidation bits, non-byte-aligned integer channels, and the full set
  of conversion rules (linear, rational, algebraic formulas, value/range
  tables, text tables).

Values are cross-validated against [asammdf](https://github.com/danielhrisca/asammdf).

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
plot(sig.Time, sig.Floats)

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
```

See `examples/mdf-reader` for a complete program.

## Options

| Option | Effect |
|---|---|
| `mf4.WithoutMmap()` | plain reads instead of mmap (network shares, etc.) |
| `mf4.WithDecompressCacheSize(n)` | decompressed-block cache per data section |
| `mf4.Raw()` | skip conversions, native recorded values |
| `mf4.WithRange(from, n)` | sample window |
| `mf4.WithChunkSamples(n)` | chunk size for `Chunks` |

## Not supported yet

- MDF 4.2 column-oriented storage (`##LD`/`##DV`) — returns `ErrUnsupported`
- Channel array (`##CA`) composition — exposed as raw byte columns
- Writing files (the block layer is structured to support it later)

## Testing

`go test ./...` runs unit, sample-file and golden tests. Golden files
under `testdata/golden` are generated with
`testdata/scripts/gen_golden.py` (requires Python + asammdf) and
committed, so CI needs no Python.
