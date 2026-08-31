package dbc

import (
	"math"
	"strings"
	"testing"
)

const sample = `VERSION "test"

BU_: ECU1 ECU2

BO_ 100 MsgCurve: 4 Vector__XXX
 SG_ State : 23|1@1+ (1,0) [0|1] "" Vector__XXX
 SG_ Curve : 0|16@1+ (0.1,0) [0|6500] "rpm" ECU1,ECU2
 SG_ Counter : 16|7@1+ (1,0) [0|0] "" Vector__XXX

BO_ 2566844926 MsgExt: 8 ECU1
 SG_ BigSig : 7|16@0+ (1,0) [0|0] "deg" Vector__XXX
 SG_ Neg : 32|8@1- (0.5,-10) [0|0] "C" Vector__XXX

BO_ 200 MsgMux: 8 ECU1
 SG_ Mux M : 0|8@1+ (1,0) [0|0] "" Vector__XXX
 SG_ MuxA m0 : 8|8@1+ (1,0) [0|0] "" Vector__XXX
 SG_ MuxB m1 : 8|16@1+ (2,0) [0|0] "" Vector__XXX

BO_ 300 MsgFloat: 8 ECU1
 SG_ F32 : 0|32@1+ (1,0) [0|0] "" Vector__XXX

CM_ BO_ 100 "curve message";
CM_ SG_ 100 Curve "the curve signal";
VAL_ 100 State 0 "off" 1 "on" ;
SIG_VALTYPE_ 300 F32 : 1;
`

func parse(t *testing.T) *Database {
	t.Helper()
	db, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestParseMessages(t *testing.T) {
	db := parse(t)
	if db.Version != "test" {
		t.Errorf("version = %q", db.Version)
	}
	if len(db.Messages) != 4 {
		t.Fatalf("%d messages", len(db.Messages))
	}
	m, ok := db.Message(100, false)
	if !ok {
		t.Fatal("message 100 missing")
	}
	if m.Name != "MsgCurve" || m.Size != 4 || m.Comment != "curve message" {
		t.Errorf("message = %+v", m)
	}
	if len(m.Signals) != 3 {
		t.Fatalf("%d signals", len(m.Signals))
	}
	curve, _ := m.Signal("Curve")
	if curve.StartBit != 0 || curve.Length != 16 || curve.Factor != 0.1 ||
		curve.Unit != "rpm" || curve.BigEndian || curve.Signed {
		t.Errorf("Curve = %+v", curve)
	}
	if curve.Comment != "the curve signal" {
		t.Errorf("comment = %q", curve.Comment)
	}
	if len(curve.Receivers) != 2 || curve.Receivers[1] != "ECU2" {
		t.Errorf("receivers = %v", curve.Receivers)
	}
	state, _ := m.Signal("State")
	if state.ValueTable[0] != "off" || state.ValueTable[1] != "on" {
		t.Errorf("value table = %v", state.ValueTable)
	}
	// Extended id: flag stripped, Extended set.
	ext, ok := db.Message(2566844926&^ExtendedFlag, true)
	if !ok || !ext.Extended || ext.Name != "MsgExt" {
		t.Errorf("extended message = %+v ok=%v", ext, ok)
	}
}

func TestDecodeLittleEndian(t *testing.T) {
	db := parse(t)
	m, _ := db.Message(100, false)
	// Curve = 0x0064 = 100 -> 10.0; Counter = byte2 & 0x7f; State = bit 23.
	payload := []byte{0x64, 0x00, 0x85, 0x00}
	curve, _ := m.Signal("Curve")
	v, ok := curve.Physical(payload)
	if !ok || v != 10.0 {
		t.Errorf("Curve = %v ok=%v", v, ok)
	}
	counter, _ := m.Signal("Counter")
	if v, _ := counter.Physical(payload); v != 5 {
		t.Errorf("Counter = %v, want 5", v)
	}
	state, _ := m.Signal("State")
	if v, _ := state.Physical(payload); v != 1 {
		t.Errorf("State = %v, want 1", v)
	}
	// Too-short payload is reported, not guessed.
	if _, ok := curve.Physical([]byte{0x64}); ok {
		t.Error("short payload accepted")
	}
}

func TestDecodeBigEndianAndSigned(t *testing.T) {
	db := parse(t)
	m, _ := db.Message(2566844926&^ExtendedFlag, true)
	big, _ := m.Signal("BigSig")
	payload := []byte{0x12, 0x34, 0, 0, 0xF6, 0, 0, 0}
	if v, ok := big.Physical(payload); !ok || v != float64(0x1234) {
		t.Errorf("BigSig = %v ok=%v, want %d", v, ok, 0x1234)
	}
	neg, _ := m.Signal("Neg")
	// 0xF6 as int8 = -10 -> -10*0.5 - 10 = -15
	if v, _ := neg.Physical(payload); v != -15 {
		t.Errorf("Neg = %v, want -15", v)
	}
}

func TestBigEndianBitPositions(t *testing.T) {
	// Hand-checked Motorola layouts over a 2-byte payload 0x12 0x34.
	p := []byte{0x12, 0x34}
	cases := []struct {
		start, length int
		want          uint64
	}{
		{7, 16, 0x1234}, // whole payload
		{7, 8, 0x12},    // first byte
		{15, 8, 0x34},   // second byte
		{7, 4, 0x1},     // high nibble of byte 0
		{3, 4, 0x2},     // low nibble of byte 0
	}
	for _, c := range cases {
		s := &Signal{StartBit: c.start, Length: c.length, BigEndian: true, Factor: 1}
		got, ok := s.Raw(p)
		if !ok || got != c.want {
			t.Errorf("start %d len %d = %#x (ok=%v), want %#x", c.start, c.length, got, ok, c.want)
		}
	}
}

func TestMultiplexing(t *testing.T) {
	db := parse(t)
	m, _ := db.Message(200, false)
	if mux := m.multiplexer(); mux == nil || mux.Name != "Mux" {
		t.Fatalf("multiplexer = %v", mux)
	}
	a, _ := m.Signal("MuxA")
	b, _ := m.Signal("MuxB")
	if !a.Multiplexed() || a.MuxValue != 0 || b.MuxValue != 1 {
		t.Errorf("mux values: A=%d B=%d", a.MuxValue, b.MuxValue)
	}
	payload := []byte{1, 0x0A, 0x00, 0, 0, 0, 0, 0}
	if v, ok := m.MuxValueOf(payload); !ok || v != 1 {
		t.Errorf("mux value = %v ok=%v", v, ok)
	}
	if v, _ := b.Physical(payload); v != 20 {
		t.Errorf("MuxB = %v, want 20", v)
	}
}

func TestFloatSignal(t *testing.T) {
	db := parse(t)
	m, _ := db.Message(300, false)
	f, _ := m.Signal("F32")
	if !f.Float {
		t.Fatal("F32 not marked as float")
	}
	bits := math.Float32bits(-2.5)
	payload := []byte{byte(bits), byte(bits >> 8), byte(bits >> 16), byte(bits >> 24), 0, 0, 0, 0}
	if v, ok := f.Physical(payload); !ok || v != -2.5 {
		t.Errorf("F32 = %v ok=%v", v, ok)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse(strings.NewReader("not a dbc file")); err == nil {
		t.Error("accepted non-DBC input")
	}
}

func FuzzParse(f *testing.F) {
	f.Add(sample)
	f.Add("BO_ 1 X: 8 A\n SG_ S : 0|8@1+ (1,0) [0|0] \"\" X\n")
	f.Fuzz(func(t *testing.T, s string) {
		db, err := Parse(strings.NewReader(s))
		if err != nil {
			return
		}
		for _, m := range db.Messages {
			m.MuxValueOf([]byte{1, 2, 3, 4, 5, 6, 7, 8})
			for _, sig := range m.Signals {
				sig.Physical([]byte{1, 2, 3, 4, 5, 6, 7, 8}) // must not panic
			}
		}
	})
}
