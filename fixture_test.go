package mf4_test

// Minimal MDF 4.1 file builder used to create fixtures for layouts the
// checked-in samples do not cover (VLSD channels, unsorted data groups).

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

type fixture struct{ buf []byte }

func newFixture() *fixture {
	fx := &fixture{}
	id := make([]byte, 64)
	copy(id[0:8], "MDF     ")
	copy(id[8:16], "4.10    ")
	copy(id[16:24], "GoMDF   ")
	binary.LittleEndian.PutUint16(id[28:30], 410)
	fx.buf = id
	return fx
}

// block appends one block (8-aligned) and returns its address.
func (fx *fixture) block(id string, links []int64, data []byte) int64 {
	for len(fx.buf)%8 != 0 {
		fx.buf = append(fx.buf, 0)
	}
	addr := int64(len(fx.buf))
	length := uint64(24 + 8*len(links) + len(data))
	head := make([]byte, 24)
	copy(head[0:4], id)
	binary.LittleEndian.PutUint64(head[8:16], length)
	binary.LittleEndian.PutUint64(head[16:24], uint64(len(links)))
	fx.buf = append(fx.buf, head...)
	for _, l := range links {
		fx.buf = binary.LittleEndian.AppendUint64(fx.buf, uint64(l))
	}
	fx.buf = append(fx.buf, data...)
	return addr
}

// patchLink overwrites link i of the block at addr.
func (fx *fixture) patchLink(addr int64, i int, target int64) {
	binary.LittleEndian.PutUint64(fx.buf[addr+24+int64(i)*8:], uint64(target))
}

func (fx *fixture) tx(s string) int64 {
	return fx.block("##TX", nil, append([]byte(s), 0))
}

// cn builds a channel block. links: next, composition, txname, si, cc,
// data, mdunit, mdcomment.
type cnSpec struct {
	next, name, data         int64
	typ, sync, dtype, bitOff uint8
	byteOff, bitCount, flags uint32
	invalPos                 uint32
}

func (fx *fixture) cn(s cnSpec) int64 {
	data := make([]byte, 72)
	data[0], data[1], data[2], data[3] = s.typ, s.sync, s.dtype, s.bitOff
	binary.LittleEndian.PutUint32(data[4:8], s.byteOff)
	binary.LittleEndian.PutUint32(data[8:12], s.bitCount)
	binary.LittleEndian.PutUint32(data[12:16], s.flags)
	binary.LittleEndian.PutUint32(data[16:20], s.invalPos)
	return fx.block("##CN", []int64{s.next, 0, s.name, 0, 0, s.data, 0, 0}, data)
}

// setVersion overrides the file version written by newFixture (4.10).
func (fx *fixture) setVersion(v uint16, s string) {
	copy(fx.buf[8:16], "        ")
	copy(fx.buf[8:16], s)
	binary.LittleEndian.PutUint16(fx.buf[28:30], v)
}

// ld builds a column-storage list block. sampleOffsets nil means the
// equal-sample-count variant with the given equal count.
func (fx *fixture) ld(dv, di []int64, sampleOffsets []uint64, equal uint64) int64 {
	flags := uint32(0)
	links := append([]int64{0}, dv...)
	if di != nil {
		flags |= 1 << 31
		links = append(links, di...)
	}
	var data []byte
	if sampleOffsets == nil {
		flags |= 1 << 0
		data = make([]byte, 16)
		binary.LittleEndian.PutUint64(data[8:16], equal)
	} else {
		data = make([]byte, 8+8*len(sampleOffsets))
		for i, o := range sampleOffsets {
			binary.LittleEndian.PutUint64(data[8+8*i:], o)
		}
	}
	binary.LittleEndian.PutUint32(data[0:4], flags)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(dv)))
	return fx.block("##LD", links, data)
}

// cg builds a channel group block. links: next, cn_first, acqname, si, sr, md.
func (fx *fixture) cg(next, cnFirst int64, recordID, cycles uint64, flags uint16, dataBytes, invalBytes uint32) int64 {
	data := make([]byte, 32)
	binary.LittleEndian.PutUint64(data[0:8], recordID)
	binary.LittleEndian.PutUint64(data[8:16], cycles)
	binary.LittleEndian.PutUint16(data[16:18], flags)
	binary.LittleEndian.PutUint32(data[24:28], dataBytes)
	binary.LittleEndian.PutUint32(data[28:32], invalBytes)
	return fx.block("##CG", []int64{next, cnFirst, 0, 0, 0, 0}, data)
}

func (fx *fixture) dg(cgFirst, data int64, recIDSize uint8) int64 {
	return fx.block("##DG", []int64{0, cgFirst, data, 0}, []byte{recIDSize, 0, 0, 0, 0, 0, 0, 0})
}

func (fx *fixture) finish(t *testing.T, dgFirst int64) string {
	t.Helper()
	// HD is always right after the ID block; it was reserved first.
	fx.patchLink(64, 0, dgFirst)
	path := filepath.Join(t.TempDir(), "fixture.mf4")
	if err := os.WriteFile(path, fx.buf, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func (fx *fixture) hd() int64 {
	return fx.block("##HD", make([]int64, 6), make([]byte, 32))
}

func f64le(v float64) []byte {
	return binary.LittleEndian.AppendUint64(nil, math.Float64bits(v))
}

func u64le(v uint64) []byte { return binary.LittleEndian.AppendUint64(nil, v) }

func u32le(v uint32) []byte { return binary.LittleEndian.AppendUint32(nil, v) }

// vlsdStream builds an SD-layout stream and the per-value offsets.
func vlsdStream(values []string) (stream []byte, offsets []uint64) {
	for _, v := range values {
		offsets = append(offsets, uint64(len(stream)))
		stream = append(stream, u32le(uint32(len(v)))...)
		stream = append(stream, v...)
	}
	return stream, offsets
}

// ca builds a channel array block.
//
//	storage 0 = CN template, 1 = CG template, 2 = DG template
//	dataLinks: ca_data[k] (DG template only)
//	cycles:    ca_cycle_count[k] (CG/DG template)
func (fx *fixture) ca(storage uint8, dims []uint64, byteOffBase int32, dataLinks []int64, cycles []uint64) int64 {
	data := make([]byte, 16+8*len(dims))
	data[0] = 0 // ca_type: array
	data[1] = storage
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(dims)))
	binary.LittleEndian.PutUint32(data[8:12], uint32(byteOffBase))
	for i, d := range dims {
		binary.LittleEndian.PutUint64(data[16+8*i:], d)
	}
	for _, c := range cycles {
		data = binary.LittleEndian.AppendUint64(data, c)
	}
	links := append([]int64{0}, dataLinks...) // ca_composition + ca_data
	return fx.block("##CA", links, data)
}

// cnComp is cn() with an explicit composition link.
func (fx *fixture) cnComp(s cnSpec, composition int64) int64 {
	addr := fx.cn(s)
	fx.patchLink(addr, 1, composition)
	return addr
}
