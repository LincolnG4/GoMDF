package records

import (
	"math"
	"testing"

	"github.com/LincolnG4/GoMDF/internal/blocks"
)

func extractAll(t *testing.T, spec ColumnSpec, buf []byte, recSize, count int) *Column {
	t.Helper()
	col, err := NewColumn(spec, count)
	if err != nil {
		t.Fatal(err)
	}
	if err := Extract(col, spec, buf, recSize, count); err != nil {
		t.Fatal(err)
	}
	return col
}

func TestAlignedInts(t *testing.T) {
	// Records: u8 at 0, i16le at 1, u32be at 3.
	buf := []byte{
		0x05, 0xFE, 0xFF, 0x00, 0x00, 0x01, 0x00, // -2 le16, 256 be32
		0xFF, 0x34, 0x12, 0xDE, 0xAD, 0xBE, 0xEF,
	}
	rec := 7
	u8 := extractAll(t, ColumnSpec{ByteOffset: 0, BitCount: 8, DataType: blocks.DTUintLE, InvalBit: -1}, buf, rec, 2)
	if u8.U[0] != 5 || u8.U[1] != 255 {
		t.Errorf("u8 = %v", u8.U)
	}
	i16 := extractAll(t, ColumnSpec{ByteOffset: 1, BitCount: 16, DataType: blocks.DTIntLE, InvalBit: -1}, buf, rec, 2)
	if i16.I[0] != -2 || i16.I[1] != 0x1234 {
		t.Errorf("i16 = %v", i16.I)
	}
	u32 := extractAll(t, ColumnSpec{ByteOffset: 3, BitCount: 32, DataType: blocks.DTUintBE, InvalBit: -1}, buf, rec, 2)
	if u32.U[0] != 256 || u32.U[1] != 0xDEADBEEF {
		t.Errorf("u32 = %v", u32.U)
	}
}

func TestBitFields(t *testing.T) {
	// One byte per record: low nibble and bits 4..6.
	buf := []byte{0xA5, 0x7F}
	lo := extractAll(t, ColumnSpec{BitCount: 4, DataType: blocks.DTUintLE, InvalBit: -1}, buf, 1, 2)
	if lo.U[0] != 0x5 || lo.U[1] != 0xF {
		t.Errorf("low nibble = %v", lo.U)
	}
	mid := extractAll(t, ColumnSpec{BitOffset: 4, BitCount: 3, DataType: blocks.DTUintLE, InvalBit: -1}, buf, 1, 2)
	if mid.U[0] != 0x2 || mid.U[1] != 0x7 {
		t.Errorf("bits 4..6 = %v", mid.U)
	}
	// Signed 4-bit: 0xF -> -1.
	s := extractAll(t, ColumnSpec{BitCount: 4, DataType: blocks.DTIntLE, InvalBit: -1}, []byte{0x0F}, 1, 1)
	if s.I[0] != -1 {
		t.Errorf("signed nibble = %v", s.I)
	}
	// 12-bit field spanning two bytes, little endian: bytes 0x34 0x12 -> 0x1234 & 0xfff = 0x234.
	w := extractAll(t, ColumnSpec{BitCount: 12, DataType: blocks.DTUintLE, InvalBit: -1}, []byte{0x34, 0x12}, 2, 1)
	if w.U[0] != 0x234 {
		t.Errorf("12-bit = %#x", w.U[0])
	}
	// Same field with a 4-bit offset: 0x1234 >> 4 = 0x123.
	w2 := extractAll(t, ColumnSpec{BitOffset: 4, BitCount: 12, DataType: blocks.DTUintLE, InvalBit: -1}, []byte{0x34, 0x12}, 2, 1)
	if w2.U[0] != 0x123 {
		t.Errorf("12-bit offset 4 = %#x", w2.U[0])
	}
}

func TestFloats(t *testing.T) {
	buf := make([]byte, 12)
	le.PutUint64(buf[0:], math.Float64bits(3.25))
	le.PutUint32(buf[8:], math.Float32bits(-1.5))
	f64 := extractAll(t, ColumnSpec{ByteOffset: 0, BitCount: 64, DataType: blocks.DTFloatLE, InvalBit: -1}, buf, 12, 1)
	if f64.F[0] != 3.25 {
		t.Errorf("f64 = %v", f64.F[0])
	}
	f32 := extractAll(t, ColumnSpec{ByteOffset: 8, BitCount: 32, DataType: blocks.DTFloatLE, InvalBit: -1}, buf, 12, 1)
	if f32.F[0] != -1.5 {
		t.Errorf("f32 = %v", f32.F[0])
	}
}

func TestHalfFloat(t *testing.T) {
	cases := map[uint16]float64{
		0x3C00: 1.0, 0xC000: -2.0, 0x7BFF: 65504, 0x0000: 0,
		0x3555: 0.333251953125, // 1/3 rounded to half precision
	}
	for h, want := range cases {
		if got := halfToFloat64(h); got != want {
			t.Errorf("half %#x = %v, want %v", h, got, want)
		}
	}
	if !math.IsInf(halfToFloat64(0x7C00), 1) || !math.IsNaN(halfToFloat64(0x7E00)) {
		t.Error("half inf/nan wrong")
	}
	// subnormal: smallest positive half = 2^-24
	if got := halfToFloat64(0x0001); got != math.Ldexp(1, -24) {
		t.Errorf("half subnormal = %g", got)
	}
}

func TestAsFloat64(t *testing.T) {
	col := extractAll(t, ColumnSpec{BitCount: 8, DataType: blocks.DTIntLE, InvalBit: -1, AsFloat64: true}, []byte{0xFF, 0x02}, 1, 2)
	if col.Kind != KindFloat64 || col.F[0] != -1 || col.F[1] != 2 {
		t.Errorf("AsFloat64 = %+v", col)
	}
}

func TestStringsAndBytes(t *testing.T) {
	buf := append([]byte("AB\x00D"), 0xE9, 0, 0, 0) // "AB", then latin1 é
	s := extractAll(t, ColumnSpec{ByteOffset: 0, BitCount: 32, DataType: blocks.DTStringUTF8, InvalBit: -1}, buf, 8, 1)
	if s.S[0] != "AB" {
		t.Errorf("utf8 = %q", s.S[0])
	}
	l := extractAll(t, ColumnSpec{ByteOffset: 4, BitCount: 32, DataType: blocks.DTStringLatin, InvalBit: -1}, buf, 8, 1)
	if l.S[0] != "é" {
		t.Errorf("latin1 = %q", l.S[0])
	}
	// UTF-16LE "Hi"
	u := extractAll(t, ColumnSpec{ByteOffset: 0, BitCount: 48, DataType: blocks.DTStringUTF16LE, InvalBit: -1}, []byte{'H', 0, 'i', 0, 0, 0}, 6, 1)
	if u.S[0] != "Hi" {
		t.Errorf("utf16 = %q", u.S[0])
	}
	b := extractAll(t, ColumnSpec{ByteOffset: 0, BitCount: 16, DataType: blocks.DTByteArray, InvalBit: -1}, []byte{1, 2}, 2, 1)
	if len(b.B) != 1 || b.B[0][0] != 1 || b.B[0][1] != 2 {
		t.Errorf("bytes = %v", b.B)
	}
}

func TestInvalidation(t *testing.T) {
	// 2 data bytes + 1 invalidation byte; invalidation bit 1.
	spec := ColumnSpec{ByteOffset: 0, BitCount: 8, DataType: blocks.DTUintLE, InvalBit: 1, DataBytes: 2}
	buf := []byte{
		10, 0, 0b10, // invalid
		20, 0, 0b00, // valid
		30, 0, 0b01, // valid (bit 0 set, not bit 1)
	}
	col := extractAll(t, spec, buf, 3, 3)
	if col.Invalid == nil || !col.Invalid.Get(0) || col.Invalid.Get(1) || col.Invalid.Get(2) {
		t.Errorf("invalidation = %+v", col.Invalid)
	}
	if col.Invalid.Count() != 1 {
		t.Errorf("count = %d", col.Invalid.Count())
	}
}

func TestBoundsErrors(t *testing.T) {
	col, _ := NewColumn(ColumnSpec{BitCount: 32, DataType: blocks.DTUintLE, InvalBit: -1}, 1)
	if err := Extract(col, ColumnSpec{ByteOffset: 2, BitCount: 32, DataType: blocks.DTUintLE, InvalBit: -1}, []byte{1, 2, 3, 4}, 4, 1); err == nil {
		t.Error("accepted field beyond record")
	}
	if err := Extract(col, ColumnSpec{BitCount: 32, DataType: blocks.DTUintLE, InvalBit: -1}, []byte{1}, 4, 1); err == nil {
		t.Error("accepted short buffer")
	}
}
