package mf4

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

var wle = binary.LittleEndian

// begin writes the identification block and the full metadata tree
// (HD, FH, DG/CG/CN chains, conversions, names). Called on the first
// append; afterwards the definitions are frozen.
func (w *Writer) begin() error {
	if w.started {
		return nil
	}
	if len(w.groups) == 0 {
		return fmt.Errorf("no channel groups defined")
	}
	for _, g := range w.groups {
		if g.recSize == 0 {
			return fmt.Errorf("group %q has no channels", g.name)
		}
		g.data = &chunkStream{blockID: blocks.IDDT, recSize: g.recSize}
	}
	w.started = true
	w.growing = len(w.groups) == 1 && !w.cfg.compress

	// Identification block: unfinalized until Close. Bit 0 = cycle
	// counters not final; bit 2 = last DT length not final (growing
	// layout only).
	unfin := uint16(1 << 0)
	if w.growing {
		unfin |= 1 << 2
	}
	if _, err := w.write(blocks.BuildID(w.cfg.program, 410, "4.10", unfin)); err != nil {
		return err
	}

	// Header block (must sit at offset 64); links patched at the end.
	hd := make([]byte, 32)
	wle.PutUint64(hd[0:8], uint64(w.cfg.startTime.UnixNano()))
	hdAddr, err := w.writeBlock(blocks.Build(blocks.IDHD, make([]int64, 6), hd))
	if err != nil {
		return err
	}
	w.hdAddr = hdAddr

	// File history: one entry describing this tool.
	fhMD, err := w.writeBlock(blocks.BuildMD(
		"<FHcomment><TX>created</TX><tool_id>GoMDF</tool_id>" +
			"<tool_vendor>GoMDF</tool_vendor><tool_version>2</tool_version></FHcomment>"))
	if err != nil {
		return err
	}
	fh := make([]byte, 16)
	wle.PutUint64(fh[0:8], uint64(w.cfg.startTime.UnixNano()))
	if w.fhAddr, err = w.writeBlock(blocks.Build(blocks.IDFH, []int64{0, fhMD}, fh)); err != nil {
		return err
	}
	w.patchLink(w.hdAddr+24+8, w.fhAddr)

	if w.cfg.comment != "" {
		md, err := w.writeBlock(blocks.BuildMD("<HDcomment><TX>" + xmlEscape(w.cfg.comment) + "</TX></HDcomment>"))
		if err != nil {
			return err
		}
		w.patchLink(w.hdAddr+24+40, md)
	}

	// Groups in reverse order so every dg_dg_next / cn_cn_next target is
	// already written.
	var nextDG int64
	for gi := len(w.groups) - 1; gi >= 0; gi-- {
		g := w.groups[gi]
		if err := w.writeGroupMeta(g, nextDG); err != nil {
			return err
		}
		nextDG = g.dgAddr
	}
	w.patchLink(w.hdAddr+24, nextDG)

	if w.growing {
		// Data records stream straight into one growing DT block.
		if err := w.align8(); err != nil {
			return err
		}
		if w.growAddr, err = w.write(blocks.Build(blocks.IDDT, nil, nil)); err != nil {
			return err
		}
		w.patchLink(w.groups[0].dgAddr+24+16, w.growAddr)
	}
	// Apply the metadata link patches now, before any record is
	// written: a crashed recording then still has a complete, valid
	// metadata tree on disk (only counters and the last block length
	// stay open — exactly what the unfinalized flags announce).
	return w.applyPatches()
}

// applyPatches flushes the write buffer and applies (then clears) the
// pending in-place patches.
func (w *Writer) applyPatches() error {
	if err := w.buf.Flush(); err != nil {
		return err
	}
	var buf8 [8]byte
	for _, p := range w.patches {
		wle.PutUint64(buf8[:], p.val)
		if _, err := w.f.WriteAt(buf8[:], p.off); err != nil {
			return err
		}
	}
	w.patches = w.patches[:0]
	return nil
}

func (w *Writer) writeGroupMeta(g *GroupWriter, nextDG int64) error {
	var nextCN int64
	for ci := len(g.chans) - 1; ci >= 0; ci-- {
		c := g.chans[ci]
		txName, err := w.writeBlock(blocks.BuildTX(c.name))
		if err != nil {
			return err
		}
		var unit, comment, cc int64
		if c.unit != "" {
			if unit, err = w.writeBlock(blocks.BuildTX(c.unit)); err != nil {
				return err
			}
		}
		if c.comment != "" {
			if comment, err = w.writeBlock(blocks.BuildTX(c.comment)); err != nil {
				return err
			}
		}
		if c.hasConv {
			ccData := make([]byte, 24+16)
			ccData[0] = blocks.CCLinear
			wle.PutUint16(ccData[6:8], 2) // cc_val_count
			wle.PutUint64(ccData[24:32], f64bits(c.convB))
			wle.PutUint64(ccData[32:40], f64bits(c.convA))
			if cc, err = w.writeBlock(blocks.Build(blocks.IDCC, make([]int64, 4), ccData)); err != nil {
				return err
			}
		}
		cn := make([]byte, 72)
		cn[0] = c.cnType
		cn[1] = c.syncType
		cn[2] = c.dataType
		wle.PutUint32(cn[4:8], c.byteOffset)
		wle.PutUint32(cn[8:12], c.bitCount)
		links := []int64{nextCN, 0, txName, 0, cc, 0, unit, comment}
		if c.cnAddr, err = w.writeBlock(blocks.Build(blocks.IDCN, links, cn)); err != nil {
			return err
		}
		nextCN = c.cnAddr
	}

	txAcq, err := w.writeBlock(blocks.BuildTX(g.name))
	if err != nil {
		return err
	}
	cg := make([]byte, 32)
	wle.PutUint32(cg[24:28], uint32(g.recSize)) // cg_data_bytes
	if g.cgAddr, err = w.writeBlock(blocks.Build(blocks.IDCG, []int64{0, nextCN, txAcq, 0, 0, 0}, cg)); err != nil {
		return err
	}
	// DG: record ID size 0 (sorted); dg_data patched at finalize.
	if g.dgAddr, err = w.writeBlock(blocks.Build(blocks.IDDG, []int64{nextDG, g.cgAddr, 0, 0}, make([]byte, 8))); err != nil {
		return err
	}
	return nil
}

// appendStream adds bytes to a chunk stream, flushing full chunks as
// data blocks.
func (w *Writer) appendStream(cs *chunkStream, b []byte) error {
	cs.total += uint64(len(b))
	cs.buf = append(cs.buf, b...)
	if len(cs.buf) >= w.cfg.chunkSize {
		return w.flushChunk(cs)
	}
	return nil
}

// flushChunk writes the buffered bytes as one DT/SD block — or a DZ
// block when compression is on and actually smaller.
func (w *Writer) flushChunk(cs *chunkStream) error {
	if len(cs.buf) == 0 {
		return nil
	}
	block := blocks.Build(cs.blockID, nil, cs.buf)
	if w.cfg.compress {
		if dz := buildDZ(cs.blockID, cs.buf, cs.recSize); dz != nil && len(dz) < len(block) {
			block = dz
		}
	}
	addr, err := w.writeBlock(block)
	if err != nil {
		return err
	}
	cs.segs = append(cs.segs, wseg{addr: addr, orgLen: uint64(len(cs.buf))})
	cs.buf = cs.buf[:0]
	return nil
}

// buildDZ deflates content into a ##DZ block (nil when compression
// fails). Record streams are byte-transposed first — grouping each
// record column's bytes together compresses slowly-changing signals far
// better.
func buildDZ(orgID string, content []byte, recSize int) []byte {
	zipType := uint8(blocks.ZipDeflate)
	zipParam := uint32(0)
	src := content
	if recSize > 1 && len(content) > recSize {
		src = transpose(content, recSize)
		zipType = blocks.ZipTransposeDeflate
		zipParam = uint32(recSize)
	}
	var comp bytes.Buffer
	zw, err := zlib.NewWriterLevel(&comp, zlib.BestSpeed)
	if err != nil {
		return nil
	}
	if _, err := zw.Write(src); err != nil {
		return nil
	}
	if err := zw.Close(); err != nil {
		return nil
	}
	data := make([]byte, 24+comp.Len())
	copy(data[0:2], orgID[2:4]) // "DT"/"SD" without "##"
	data[2] = zipType
	wle.PutUint32(data[4:8], zipParam)
	wle.PutUint64(data[8:16], uint64(len(content)))
	wle.PutUint64(data[16:24], uint64(comp.Len()))
	copy(data[24:], comp.Bytes())
	return blocks.Build(blocks.IDDZ, nil, data)
}

// linkStream turns a stream's written segments into the address the
// owning link should point at: nothing, the single data block, or a DL
// list block.
func (w *Writer) linkStream(cs *chunkStream) (int64, error) {
	if err := w.flushChunk(cs); err != nil {
		return 0, err
	}
	switch len(cs.segs) {
	case 0:
		return 0, nil
	case 1:
		return cs.segs[0].addr, nil
	}
	links := make([]int64, 1+len(cs.segs))
	data := make([]byte, 8+8*len(cs.segs))
	wle.PutUint32(data[4:8], uint32(len(cs.segs)))
	var off uint64
	for i, s := range cs.segs {
		links[1+i] = s.addr
		wle.PutUint64(data[8+8*i:], off)
		off += s.orgLen
	}
	return w.writeBlock(blocks.Build(blocks.IDDL, links, data))
}

// finalize completes the file: remaining chunks, data lists, link and
// counter patches, finalized ID block.
func (w *Writer) finalize() error {
	if !w.started {
		if err := w.begin(); err != nil {
			return err
		}
	}
	for _, g := range w.groups {
		if w.growing {
			// Patch the growing DT block's length.
			w.patches = append(w.patches, patch{off: w.growAddr + 8, val: uint64(blocks.HeaderSize + w.growBytes)})
		} else {
			addr, err := w.linkStream(g.data)
			if err != nil {
				return err
			}
			w.patchLink(g.dgAddr+24+16, addr)
		}
		for _, c := range g.chans {
			if c.vlsd {
				addr, err := w.linkStream(c.sd)
				if err != nil {
					return err
				}
				w.patchLink(c.cnAddr+24+40, addr)
			}
		}
		// cg_cycle_count sits 8 bytes into the CG data section.
		w.patches = append(w.patches, patch{off: g.cgAddr + 24 + 6*8 + 8, val: g.count})
	}
	if err := w.applyPatches(); err != nil {
		return err
	}
	// Finalized identification block.
	if _, err := w.f.WriteAt(blocks.BuildID(w.cfg.program, 410, "4.10", 0), 0); err != nil {
		return err
	}
	return w.f.Sync()
}

// transpose stores the record matrix column-major (the inverse of the
// reader's untranspose); trailing remainder bytes stay in place.
func transpose(in []byte, cols int) []byte {
	rows := len(in) / cols
	out := make([]byte, len(in))
	for i := 0; i < rows; i++ {
		row := in[i*cols : (i+1)*cols]
		for j, b := range row {
			out[j*rows+i] = b
		}
	}
	copy(out[rows*cols:], in[rows*cols:])
	return out
}

func f64bits(v float64) uint64 { return math.Float64bits(v) }

func xmlEscape(s string) string {
	var out bytes.Buffer
	for _, r := range s {
		switch r {
		case '<':
			out.WriteString("&lt;")
		case '>':
			out.WriteString("&gt;")
		case '&':
			out.WriteString("&amp;")
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}
