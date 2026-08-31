package mf4

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/LincolnG4/GoMDF/internal/blocks"
	"github.com/LincolnG4/GoMDF/internal/conversion"
	"github.com/LincolnG4/GoMDF/internal/datasection"
	"github.com/LincolnG4/GoMDF/internal/records"
)

// ReadOption configures Channel.Read, File.ReadAll and ChannelGroup.Chunks.
type ReadOption func(*readConfig)

type readConfig struct {
	raw          bool
	from         int
	count        int // -1: to the end
	chunkSamples int // 0: auto (~1 MiB of records)
}

// Raw skips the channel conversion: samples keep their native recorded
// type and values.
func Raw() ReadOption { return func(c *readConfig) { c.raw = true } }

// WithRange limits the read to n samples starting at sample index from.
func WithRange(from, n int) ReadOption {
	return func(c *readConfig) { c.from, c.count = from, n }
}

// WithChunkSamples sets how many samples Chunks yields per iteration (and
// the internal decode granularity). Default is about 1 MiB of records.
func WithChunkSamples(n int) ReadOption {
	return func(c *readConfig) {
		if n > 0 {
			c.chunkSamples = n
		}
	}
}

func newReadConfig(opts []ReadOption) readConfig {
	cfg := readConfig{count: -1}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

// resolveRange clamps [from, from+count) to total records.
func (cfg *readConfig) resolveRange(total int) (from, count int, err error) {
	from, count = cfg.from, cfg.count
	if from < 0 || from > total {
		return 0, 0, fmt.Errorf("sample range start %d out of [0, %d]", from, total)
	}
	if count < 0 || from+count > total {
		count = total - from
	}
	return from, count, nil
}

// Read decodes the channel's samples. By default the conversion rule is
// applied (numeric results become float64, textual ones strings); pass
// Raw() for the native recorded values, or WithRange for a window.
func (c *Channel) Read(opts ...ReadOption) (*Signal, error) {
	cfg := newReadConfig(opts)
	sig, err := c.read(&cfg)
	if err != nil {
		return nil, fmt.Errorf("channel %q: %w", c.Name, err)
	}
	if c.group.master != nil && c.group.master != c {
		master, err := c.group.masterValues(&cfg)
		if err != nil {
			return nil, fmt.Errorf("channel %q: master: %w", c.Name, err)
		}
		sig.Time = master
	} else if c.group.master == c && sig.Type == SampleFloat64 {
		sig.Time = sig.Floats
	}
	return sig, nil
}

// read decodes the samples without attaching master values.
func (c *Channel) read(cfg *readConfig) (*Signal, error) {
	g := c.group
	from, count, err := cfg.resolveRange(int(g.RecordCount))
	if err != nil {
		return nil, err
	}

	// Virtual channels have no record data: the value is the record index.
	if c.Type == VirtualMaster || c.Type == VirtualData {
		return c.readVirtual(cfg, from, count)
	}

	conv, err := c.compiledConversion()
	if err != nil {
		return nil, err
	}
	col, err := c.extractColumn(cfg, conv, from, count)
	if err != nil {
		return nil, err
	}

	// VLSD: the extracted column holds byte offsets into the channel's
	// signal-data section; resolve them to the actual values.
	if c.Type == VLSD {
		if col, err = c.resolveVLSD(col); err != nil {
			return nil, err
		}
	}

	if !cfg.raw {
		if err := applyConversion(col, conv); err != nil {
			return nil, err
		}
	}
	return fromColumn(c, col, from), nil
}

// extractColumn runs the chunked record extraction for this channel.
func (c *Channel) extractColumn(cfg *readConfig, conv *conversion.Conversion, from, count int) (*records.Column, error) {
	g := c.group
	spec, err := c.columnSpec(cfg, conv)
	if err != nil {
		return nil, err
	}
	col, err := records.NewColumn(spec, count)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return col, nil
	}
	r, err := g.recordSource()
	if err != nil {
		return nil, err
	}
	recSize := int(g.cg.RecordSize())
	if recSize == 0 {
		return nil, fmt.Errorf("record size 0")
	}
	total := int(r.Size() / int64(recSize))
	if from+count > total {
		return nil, fmt.Errorf("data section holds %d records, need %d", total, from+count)
	}

	chunk := cfg.chunkSamples
	if chunk <= 0 {
		chunk = (1 << 20) / recSize
		if chunk < 1 {
			chunk = 1
		}
	}
	buf := make([]byte, chunk*recSize)
	for done := 0; done < count; {
		n := chunk
		if done+n > count {
			n = count - done
		}
		off := int64(from+done) * int64(recSize)
		if _, err := r.ReadAt(buf[:n*recSize], off); err != nil {
			return nil, err
		}
		if err := records.Extract(col, spec, buf[:n*recSize], recSize, n); err != nil {
			return nil, err
		}
		done += n
	}
	return col, nil
}

// columnSpec builds the extraction spec, deciding the storage class from
// the conversion that will follow.
func (c *Channel) columnSpec(cfg *readConfig, conv *conversion.Conversion) (records.ColumnSpec, error) {
	cn := c.cn
	spec := records.ColumnSpec{
		ByteOffset: cn.ByteOffset,
		BitOffset:  cn.BitOffset,
		BitCount:   cn.BitCount,
		DataType:   cn.DataType,
		InvalBit:   -1,
		DataBytes:  c.group.cg.DataBytes,
	}
	if cn.Flags&blocks.CNFlagInvalBit != 0 {
		spec.InvalBit = int64(cn.InvalBitPos)
	}
	if c.IsArray {
		// Array composition: expose the raw bytes of the whole element
		// range until array composition is fully supported.
		spec.DataType = blocks.DTByteArray
	}
	if c.Type == VLSD {
		// The in-record value is a uint64 offset into the signal data.
		spec.DataType = blocks.DTUintLE
		spec.AsFloat64 = false
		return spec, nil
	}
	numericIn := cn.DataType <= blocks.DTFloatBE
	if !cfg.raw && numericIn && conv != nil && !conv.TextInput() && !conv.Textual() {
		// A numeric conversion follows: extract straight to float64.
		spec.AsFloat64 = true
	}
	if !cfg.raw && numericIn && conv.Textual() {
		spec.AsFloat64 = true
	}
	return spec, nil
}

// applyConversion converts col in place according to the channel rule.
func applyConversion(col *records.Column, conv *conversion.Conversion) error {
	if conv == nil {
		return nil
	}
	switch {
	case conv.Textual() && col.Kind == records.KindFloat64:
		col.S = conv.Strings(col.F)
		col.Kind, col.F = records.KindString, nil
	case conv.Textual() && col.Kind == records.KindString:
		col.S = conv.TextToText(col.S)
	case conv.TextInput() && col.Kind == records.KindString:
		if conv.Textual() {
			col.S = conv.TextToText(col.S)
		} else {
			col.F = conv.FromText(col.S)
			col.Kind, col.S = records.KindFloat64, nil
		}
	case col.Kind == records.KindFloat64:
		conv.Convert(col.F, col.F)
	default:
		// Numeric conversion attached to a string/bytes channel (some
		// writers emit identity conversions there): nothing to apply.
	}
	return nil
}

// readVirtual synthesizes samples for virtual (record-index) channels.
func (c *Channel) readVirtual(cfg *readConfig, from, count int) (*Signal, error) {
	vals := make([]float64, count)
	for i := range vals {
		vals[i] = float64(from + i)
	}
	if !cfg.raw {
		conv, err := c.compiledConversion()
		if err != nil {
			return nil, err
		}
		conv.Convert(vals, vals)
	}
	sig := &Signal{Channel: c, Type: SampleFloat64, Floats: vals, Offset: from}
	return sig, nil
}

// masterValues returns the group's master samples for the given range,
// cached for full-range reads.
func (g *ChannelGroup) masterValues(cfg *readConfig) ([]float64, error) {
	full := cfg.from == 0 && (cfg.count < 0 || cfg.count >= int(g.RecordCount))
	if full {
		g.masterOnce.Do(func() {
			g.masterVals, g.masterErr = g.readMasterFloats(&readConfig{count: -1})
		})
		return g.masterVals, g.masterErr
	}
	return g.readMasterFloats(cfg)
}

func (g *ChannelGroup) readMasterFloats(cfg *readConfig) ([]float64, error) {
	mCfg := *cfg
	mCfg.raw = false
	sig, err := g.master.read(&mCfg)
	if err != nil {
		return nil, err
	}
	if sig.Type != SampleFloat64 {
		return sig.Float64s(), nil
	}
	return sig.Floats, nil
}

// recordSource returns the group's record bytes as a random-access
// reader: the data section directly for sorted files, the de-interleaved
// buffer for unsorted ones.
func (g *ChannelGroup) recordSource() (*datasection.Reader, error) {
	if g.dg.block.RecIDSize == 0 {
		return g.dg.sectionReader()
	}
	return g.dg.deinterleaved(g.cg.RecordID)
}

// sectionReader lazily builds the shared data-section reader for the DG.
func (dg *dataGroup) sectionReader() (*datasection.Reader, error) {
	dg.sectionOnce.Do(func() {
		if dg.file.finalize {
			dg.section, dg.sectionErr = datasection.NewFinalizing(dg.file.src, dg.block.Data, dg.file.cfg.cacheSize)
		} else {
			dg.section, dg.sectionErr = datasection.New(dg.file.src, dg.block.Data, dg.file.cfg.cacheSize)
		}
	})
	return dg.section, dg.sectionErr
}

// compiledConversion compiles the channel's CC block once.
func (c *Channel) compiledConversion() (*conversion.Conversion, error) {
	c.convOnce.Do(func() {
		c.conv, c.convErr = c.group.file.compileConversion(c.cc, c.cn.DataType, 0)
	})
	return c.conv, c.convErr
}

const maxConversionDepth = 16

// compileConversion resolves cc_ref links (TX texts or nested CC blocks)
// and compiles the conversion.
func (f *File) compileConversion(cc *blocks.CC, dataType uint8, depth int) (*conversion.Conversion, error) {
	if cc == nil {
		return nil, nil
	}
	if depth > maxConversionDepth {
		return nil, fmt.Errorf("conversion nesting deeper than %d", maxConversionDepth)
	}
	refs := make([]conversion.Ref, len(cc.Refs))
	for i, addr := range cc.Refs {
		if addr == 0 {
			continue
		}
		id, err := blocks.PeekID(f.src, addr)
		if err != nil {
			return nil, err
		}
		switch id {
		case blocks.IDTX, blocks.IDMD:
			if refs[i].Text, err = blocks.DecodeText(f.src, addr); err != nil {
				return nil, err
			}
		case blocks.IDCC:
			nestedCC, err := blocks.DecodeCC(f.src, addr)
			if err != nil {
				return nil, err
			}
			if refs[i].Conv, err = f.compileConversion(nestedCC, dataType, depth+1); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("conversion ref to unexpected block %q", id)
		}
	}
	intInput := dataType <= blocks.DTIntBE
	return conversion.Compile(cc, refs, intInput)
}

// ReadAll reads every channel of the file in parallel. The map is keyed
// by channel name; for duplicated names the first channel in file order
// wins.
func (f *File) ReadAll(opts ...ReadOption) (map[string]*Signal, error) {
	type result struct {
		i   int
		sig *Signal
		err error
	}
	chans := f.channels
	results := make([]*Signal, len(chans))
	sem := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup
	var firstErr error
	var mu sync.Mutex
	for i, c := range chans {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, c *Channel) {
			defer func() { <-sem; wg.Done() }()
			sig, err := c.Read(opts...)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			results[i] = sig
		}(i, c)
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	out := make(map[string]*Signal, len(chans))
	for i, c := range chans {
		if _, dup := out[c.Name]; !dup {
			out[c.Name] = results[i]
		}
	}
	return out, nil
}
