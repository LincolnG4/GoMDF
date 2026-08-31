# GoMDF User Manual

GoMDF reads and writes ASAM MDF 4.x (`.mf4`) files — the standard format
for automotive measurement data. This manual covers the full API; the
[README](../README.md) has the short version.

- [1. Installation](#1-installation)
- [2. Concepts](#2-concepts)
- [3. Opening a file](#3-opening-a-file)
- [4. Exploring metadata](#4-exploring-metadata)
- [5. Reading samples](#5-reading-samples)
- [6. Streaming large files](#6-streaming-large-files)
- [7. Sample reductions (fast zoomed-out plots)](#7-sample-reductions-fast-zoomed-out-plots)
- [8. Bus logging (CAN/DBC)](#8-bus-logging-candbc)
- [9. Writing files](#9-writing-files)
- [10. Error handling](#10-error-handling)
- [11. Performance guide](#11-performance-guide)
- [12. Compatibility and validation](#12-compatibility-and-validation)
- [13. Limitations](#13-limitations)

## 1. Installation

```
go get github.com/LincolnG4/GoMDF
```

Requires Go 1.23+. The only dependency is `github.com/edsrzf/mmap-go`.

```go
import mf4 "github.com/LincolnG4/GoMDF"
```

## 2. Concepts

An MDF file is a tree of *data groups*, each holding one or more
*channel groups*. A **channel group** (`mf4.ChannelGroup`) is a set of
signals recorded together: every sample cycle appends one fixed-size
*record* containing one value per channel. Groups typically map to
acquisition rates or bus message types (a 10 ms ECU task, a CAN frame).

Each group usually has a **master channel** — the time (or angle,
distance) axis all other channels of the group share.

A **channel** (`mf4.Channel`) describes one signal: name, unit, data
type, bit position inside the record, and an optional **conversion
rule** that maps raw recorded values to physical values (linear scaling,
lookup tables, algebraic formulas, enum texts).

Reading a channel yields a **Signal**: a typed column of samples plus
the aligned master values.

## 3. Opening a file

```go
f, err := mf4.Open("measurement.mf4")
if err != nil { ... }
defer f.Close()
```

`Open` memory-maps the file and decodes only the metadata tree; sample
data is decoded on demand. Options:

| Option | Meaning |
|---|---|
| `mf4.WithoutMmap()` | plain `pread` I/O instead of mmap — use for files on network shares or files that may be truncated while open |
| `mf4.WithDecompressCacheSize(n)` | byte budget for cached decompressed blocks of compressed files (default 128 MiB) |

To read from something other than a file path:

```go
f, err := mf4.OpenReader(readerAt, size)   // any io.ReaderAt
```

**Unfinalized files** (a logger crashed or lost power mid-recording)
open transparently when the standard finalization steps can be applied
in memory — cycle counters are recomputed and the last data block is
resized. Files needing unsupported repair steps return
`mf4.ErrUnfinalized`.

Basic file information:

```go
f.Version()    // 400, 410, 420
f.Program()    // writer tool id
f.StartTime()  // absolute measurement start (time.Time)
```

## 4. Exploring metadata

```go
for _, g := range f.Groups() {
    fmt.Println(g.Name, g.RecordCount, g.Comment)
    for _, ch := range g.Channels() {
        fmt.Println(" ", ch.Name, ch.Unit, ch.DataType, ch.Conversion.Kind)
    }
}

ch, err := f.Channel("EngineSpeed")   // first match across groups
ch, ok := g.Channel("EngineSpeed")    // within one group
g.Master()                            // the group's time channel (or nil)
f.Channels()                          // every channel, file order
```

`Channel` fields: `Name`, `Unit`, `Comment`, `Source` (ECU/bus/tool
provenance), `Type` (fixed-length, VLSD, master, virtual…), `DataType`
(raw MDF type), `BitCount`, `Conversion` (introspection: kind, params,
formula), `IsArray`.

**Array channels** (CA composition, e.g. maps, matrices and
classification results) are expanded into one scalar channel per
element, named `base[i][j]` (row-major); the base channel remains
available as a raw byte column. All three MDF array storage types work:

- *CN template* — all elements share one record (the common case).
- *CG template* — each element has its own record ID in an unsorted
  group.
- *DG template* — each element has its own data section, so elements
  have individual cycle counts and timestamps. Elements that were never
  recorded (NIL link) are not exposed.

For CG/DG-template arrays each element channel reads from its own record
stream, so `Read` returns that element's own samples and `Time`.

Other metadata:

```go
atts, _ := f.Attachments()      // embedded or referenced files
data, _ := atts[0].Data()       // embedded content (decompressed)
evs, _ := f.Events()            // triggers, markers; SyncValue in seconds
hist, _ := f.History()          // who/what wrote and modified the file
tree, _ := f.Hierarchy()        // logical channel tree (CH blocks)
```

## 5. Reading samples

```go
sig, err := ch.Read()
```

`Read` decodes the whole channel with its conversion applied and
returns a `*Signal`. A Signal is **typed columns** — exactly one slice
is populated, indicated by `sig.Type`:

| `sig.Type` | populated slice | produced by |
|---|---|---|
| `SampleFloat64` | `sig.Floats` | float channels, any numeric conversion |
| `SampleInt64` | `sig.Ints` | signed integers read raw |
| `SampleUint64` | `sig.Uints` | unsigned integers read raw |
| `SampleString` | `sig.Strings` | string channels, text conversions |
| `SampleBytes` | `sig.Bytes` | byte arrays, MIME data, unexpanded arrays |

Always aligned 1:1 with the samples:

- `sig.Time` — the master channel values (nil if the group has none).
- `sig.Invalid` — a `*Bitset` of per-sample invalidation flags
  (`sig.Invalid.Get(i)`); nil when the channel has no invalidation bit.
- `sig.Offset` — absolute index of the first sample (for window reads).

Convenience: `sig.Float64s()` returns the samples widened to
`[]float64` whatever the numeric type — handy for plotting.

Read options:

```go
sig, _ := ch.Read(mf4.Raw())                  // native values, no conversion
sig, _ := ch.Read(mf4.WithRange(1_000_000, 50_000)) // 50k samples from index 1M
```

Read everything in parallel (one goroutine per channel internally):

```go
signals, err := f.ReadAll()     // map[channelName]*Signal
```

Conversion behavior worth knowing:

- Numeric conversions output `float64`; enum-style conversions output
  strings.
- A value/range→text table whose *default* is a numeric conversion (the
  common "numeric signal with coded error values" pattern) is treated as
  numeric: matched text codes become `NaN`.
- Algebraic formulas are compiled once per channel, not per sample.

## 6. Streaming large files

For files larger than RAM, progressive loading, or a visualization
tool's windowed access:

```go
g := ch.Group()
it := g.Chunks(g.Channels(), mf4.WithChunkSamples(100_000))
for sigs := range it.All() {          // or: for it.Next() { sigs := it.Signals() }
    // sigs[i] corresponds to the i-th requested channel;
    // all share the same window and Time slice; Offset marks the start
}
if err := it.Err(); err != nil { ... }
```

Chunks decode only the records of each window; on compressed files only
the data blocks a window touches are decompressed (and cached). Any
subset of the group's channels can be requested, and `Raw()` /
`WithRange` apply.

## 7. Sample reductions (fast zoomed-out plots)

Writers may store precomputed aggregates of a channel group: for every
interval of a fixed length, the mean, minimum and maximum of each
channel. A viewer can draw a multi-million-sample signal from a few
hundred reduction records instead of reading the raw data.

```go
reds, err := g.Reductions()          // nil when the file stores none
for _, r := range reds {
    fmt.Println(r.Interval, r.Sync, r.CycleCount) // e.g. 0.01 s, 9602 intervals
}

r := reds[0]                          // pick by interval vs. pixels on screen
rs, err := r.Read(ch)                 // ReadOption values apply (Raw, WithRange)
plotBand(rs.Mean.Time, rs.Min.Float64s(), rs.Max.Float64s())
plotLine(rs.Mean.Time, rs.Mean.Float64s())
```

`Read` returns a `*ReducedSignal` with three aligned `*Signal` values —
`Mean`, `Min`, `Max` — sharing the interval start times in `Time`.
`WithRange` counts in intervals, so a zoom window maps directly.

For the group's own master channel the spec assigns different meanings:
`Mean` is the interval start value and `Min`/`Max` are the smallest and
largest sample spacing (raster) inside the interval.

## 8. Bus logging (CAN/DBC)

MDF bus logs store raw CAN/LIN frames (identifier, payload bytes,
timestamp). With a database describing the frames, GoMDF decodes the
payloads into physical signals.

```go
f, _ := mf4.Open("canlog.mf4")

// Bus logs usually embed the .dbc they were recorded with.
dbs, _ := f.EmbeddedDatabases()
sigs, err := f.DecodeBus()            // uses the embedded databases
// ... or supply your own:
db, _ := mf4.LoadDBC("vehicle.dbc")   // also mf4.ParseDBC(reader)
sigs, err = f.DecodeBus(db)

for _, s := range sigs {
    fmt.Println(s.QualifiedName(), s.Unit, s.Len())  // "MsgSine.Sine" "Volts" 265
    plot(s.Time, s.Floats)
}
```

A `*BusSignal` carries `Name`, `Unit`, `Comment`, its parent `Message`
and `MessageID`, the `BusChannel` it was seen on, and aligned `Time` /
`Floats`. Signals with a DBC value table (`VAL_`) also get `Strings`
with the text per sample. Signal names are not unique across messages —
use `QualifiedName()` or `MessageID`.

Supported DBC features: little- and big-endian (Intel/Motorola) layouts,
signed and unsigned values, IEEE float signals (`SIG_VALTYPE_`), scaling
and offset, value tables (`VAL_`), comments (`CM_`), extended (29-bit)
identifiers and multiplexed signals (`M` / `m<n>`). Frames whose id is
in no database, and frames too short for a signal, are skipped.

To show raw traffic instead, the frame groups stay available as ordinary
channel groups:

```go
for _, fg := range f.BusFrameGroups() {
    fmt.Println(fg.Kind, fg.Group.Name, fg.Group.RecordCount) // "CAN" ...
    id, _ := fg.Group.Channel("CAN_DataFrame.ID")
    data, _ := fg.Group.Channel("CAN_DataFrame.DataBytes")
    // id.Read(), data.Read() -> identifiers and payload bytes
}
```

## 9. Writing files

### Defining the file

```go
w, err := mf4.Create("drive.mf4",
    mf4.WithStartTime(time.Now()),
    mf4.WithFileComment("test drive 42"),
    // mf4.WithCompression(),          // DZ transposed-deflate chunks
    // mf4.WithChunkSize(1<<20),
    // mf4.WithProgramID("MyLogger"),
)

eng, _ := w.NewGroup("Engine")   // index 0 = implicit float64 master "t" (s)
spd  := eng.Float64("EngineSpeed", "rpm")
tq   := eng.Float32("Torque", "Nm")
gear := eng.Int("Gear", "", 8)          // 8/16/32/64-bit signed
flg  := eng.Uint("StatusFlags", "", 32) // 8/16/32/64-bit unsigned
msg  := eng.String("Message")           // variable-length text (VLSD)

adc := eng.Uint("Battery", "V", 16)
eng.SetLinearConversion(adc, 0, 0.001)  // physical = 0 + 0.001*raw
eng.SetComment(spd, "crankshaft speed")
```

`NewGroup(name, mf4.WithoutMaster())` skips the implicit master;
`g.MasterTime(name, unit)` defines a custom one. All groups and
channels must be defined before the first append; the layout freezes
then.

### Streaming records (the data-logger path)

```go
rec := eng.Record()          // reusable buffer, one per writing goroutine
for running {
    rec.SetFloat64(0, tSeconds)    // master
    rec.SetFloat64(spd, v1)
    rec.SetFloat64(tq, v2)         // also fills Float32 channels
    rec.SetInt(gear, g)
    rec.SetUint(flg, bits)
    rec.SetString(msg, note)       // "" for no text this cycle
    if err := eng.Append(rec); err != nil { ... }
}
err := w.Close()             // finalizes the file — do not skip
```

Setters are allocation-free; a type mismatch is reported by the next
`Append`. Throughput is tens of millions of records per second.

**Crash safety.** With a single group and no compression, records
stream into one growing data block and the file on disk is a valid
*unfinalized* MDF at all times — the same convention vehicle loggers
use. Call `w.Flush()` at checkpoints to push data to stable storage; if
the process dies before `Close`, everything flushed is recoverable (and
GoMDF itself opens such files transparently, see §3).

With several groups or compression, data is written as per-group chunks
and the linking data lists are only written by `Close` — a crashed file
keeps its metadata but loses the data section.

### Batch columns (the export path)

```go
slow, _ := w.NewGroup("Ambient")
temp := slow.Float64("Temp", "degC")
err := slow.AppendColumns(len(ts),
    mf4.Column{Ch: 0, F: ts},        // master
    mf4.Column{Ch: temp, F: values},
)
```

`Column` carries exactly one slice (`F`, `I`, `U`, or `S`) matching the
channel type. Batch and record appends can be mixed freely; groups may
be filled in any order and at different rates.

### Layout summary

| Situation | On-disk layout |
|---|---|
| 1 group, no compression | single growing DT block, crash-safe, sorted |
| several groups | sorted multi-DG; per-group chunk lists (DL) |
| compression on | DZ blocks (deflate + byte transposition), per-group DL |

All layouts are sorted (no record IDs), which is the fastest layout for
readers, and verified to open in asammdf.

## 10. Error handling

The package never panics on malformed input (fuzz-tested); everything
returns errors.

```go
errors.Is(err, mf4.ErrNotMDF4)         // not an MDF >= 4.00 file
errors.Is(err, mf4.ErrUnfinalized)     // unfinalized and not repairable
errors.Is(err, mf4.ErrChannelNotFound)
errors.Is(err, mf4.ErrUnsupported)     // valid MDF, unimplemented feature

var be *mf4.BlockError                 // structural errors carry
errors.As(err, &be)                    // block ID + file offset
```

## 11. Performance guide

Measured on a 2M-record, 8-channel file (90 MB raw / 32 MB compressed),
i5-11600K, vs asammdf 8.8.25 (numpy backend), identical results:

| Operation | GoMDF | asammdf | ratio |
|---|---|---|---|
| open (metadata only) | 0.03 ms | 0.3 ms | ~10x |
| read 1 channel, uncompressed | 11 ms | 56 ms | 5x |
| read all 8 channels, uncompressed | 99 ms | 450 ms | 4.5x |
| read 1 channel, compressed (cold, parallel inflate) | 114 ms | 114 ms | 1x |
| read 1 channel, compressed (warm cache) | 12 ms | 114 ms | 9.5x |
| read all 8, compressed (warm cache) | 85 ms | 894 ms | 10x |
| write 2M records, record-by-record | 124 ms | 357 ms (numpy batch) | 3x |
| write compressed | 413 ms | 436 ms | ~1x |

Tips:

- Reading a single channel does not load the file: with mmap only the
  touched pages are read. Metadata-only operations are effectively free.
- `ReadAll` parallelizes across channels; prefer it over a loop of
  `Read` when you need many channels.
- For compressed files read repeatedly (viz tools), leave the
  decompress cache at its default or size it to the compressed groups
  you work with; every block is then decompressed exactly once. The
  first pass inflates blocks in parallel across cores, so even a cold
  read keeps up with asammdf; every read after it is ~10x faster.
- `WithRange`/`Chunks` on compressed files touch only the blocks in the
  window.
- When writing, `Record` reuse is what makes the append path
  allocation-free — create it once per goroutine, not per sample.
- Compression costs ~3x write throughput and buys ~3x smaller files
  (more for slowly-changing signals, thanks to byte transposition).

## 12. Compatibility and validation

- Reads MDF 4.00 / 4.10 / 4.20 (except the 4.2 column-storage layout,
  see below). Writes MDF 4.10.
- The reader passes the complete official ASAM MDF 4.2 example suite:
  97/97 files, including unsorted, compressed, VLSD, arrays, bus
  logging, events, sample reduction and unfinalized logger files.
- 15,000+ channels are cross-checked value-by-value against asammdf;
  written files are verified to read back identically in asammdf.
- Documented divergences where GoMDF follows the spec and asammdf does
  not: UTF-16 string channels are decoded (not passed through byte-wise),
  range→scale partial-conversion matching uses `min <= v < max` for all
  array sizes, and CA array element offsets follow the spec formula
  (verified against raw record bytes).

## 13. Limitations

Reading:
- Bus databases: DBC only (no ARXML/LDF); LIN and FlexRay frame groups
  are detected and readable but decoding rules are DBC-shaped. Extended
  multiplexing (`SG_MUL_VAL_`) is not applied.
- CA arrays: axis channels are not attached to their array (the axis
  channels themselves read normally); dynamic-size arrays use their
  declared maximum size.
- MDF 4.2 `RV`/`RI` (column-oriented reduction data) is not exposed;
  `LD`/`DV`/`DI` column storage for normal data is supported.

Writing:
- Invalidation bits, attachments, events: not written.
- Conversions: linear only (readers apply anything, so raw + linear
  covers most logging).
- Unsorted/VLSD-CG layouts are read but never written (sorted layouts
  are strictly better for consumers).
