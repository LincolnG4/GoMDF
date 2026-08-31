package dbc

import "math"

// Raw extracts the signal's raw (unscaled) bits from a payload. It
// reports false when the payload is too short for the signal's bits.
//
// Little-endian (Intel, @1) signals count StartBit as the least
// significant bit, with bit i living in byte i/8 at bit i%8.
// Big-endian (Motorola, @0) signals count StartBit as the most
// significant bit in the same numbering, and continue towards lower
// bit positions.
func (s *Signal) Raw(payload []byte) (uint64, bool) {
	if s.Length <= 0 || s.Length > 64 {
		return 0, false
	}
	if s.BigEndian {
		return s.rawBig(payload)
	}
	return s.rawLittle(payload)
}

func (s *Signal) rawLittle(p []byte) (uint64, bool) {
	startByte := s.StartBit / 8
	bitOff := s.StartBit % 8
	need := (bitOff + s.Length + 7) / 8
	if startByte < 0 || startByte+need > len(p) {
		return 0, false
	}
	v := leWord(p[startByte:])
	v >>= uint(bitOff)
	if bitOff+s.Length > 64 {
		// The field spans a ninth byte.
		v |= uint64(p[startByte+8]) << uint(64-bitOff)
	}
	return v & mask(s.Length), true
}

func (s *Signal) rawBig(p []byte) (uint64, bool) {
	// Position of the signal's most significant bit counted MSB-first
	// from the start of the payload.
	msb := 8*(s.StartBit/8) + (7 - s.StartBit%8)
	firstByte := msb / 8
	bitOff := msb % 8
	need := (bitOff + s.Length + 7) / 8
	if firstByte < 0 || firstByte+need > len(p) {
		return 0, false
	}
	v := beWord(p[firstByte:])
	shift := 64 - bitOff - s.Length
	if shift >= 0 {
		v >>= uint(shift)
	} else {
		// Spans a ninth byte: shift left and pull the missing low bits
		// out of it.
		v <<= uint(-shift)
		v |= uint64(p[firstByte+8]) >> uint(8+shift)
	}
	return v & mask(s.Length), true
}

// leWord reads up to 8 bytes little-endian, zero-padding a short slice.
func leWord(p []byte) uint64 {
	var v uint64
	n := len(p)
	if n > 8 {
		n = 8
	}
	for i := n - 1; i >= 0; i-- {
		v = v<<8 | uint64(p[i])
	}
	return v
}

// beWord reads up to 8 bytes big-endian, zero-padding a short slice.
func beWord(p []byte) uint64 {
	var v uint64
	n := len(p)
	if n > 8 {
		n = 8
	}
	for i := 0; i < n; i++ {
		v = v<<8 | uint64(p[i])
	}
	return v << uint(8*(8-n))
}

func mask(bits int) uint64 {
	if bits >= 64 {
		return ^uint64(0)
	}
	return 1<<uint(bits) - 1
}

// Physical decodes the signal's physical value from a payload:
// raw * Factor + Offset, with sign extension or IEEE float
// reinterpretation as declared. It reports false when the payload is
// too short.
func (s *Signal) Physical(payload []byte) (float64, bool) {
	raw, ok := s.Raw(payload)
	if !ok {
		return 0, false
	}
	return s.PhysicalFromRaw(raw), true
}

// PhysicalFromRaw applies sign, float reinterpretation and scaling to an
// already extracted raw value.
func (s *Signal) PhysicalFromRaw(raw uint64) float64 {
	var v float64
	switch {
	case s.Float && s.Length == 32:
		v = float64(math.Float32frombits(uint32(raw)))
	case s.Float && s.Length == 64:
		v = math.Float64frombits(raw)
	case s.Signed:
		shift := uint(64 - s.Length)
		v = float64(int64(raw<<shift) >> shift)
	default:
		v = float64(raw)
	}
	return v*s.Factor + s.Offset
}

// SignedRaw returns the raw value sign-extended to int64 (for value
// table lookups).
func (s *Signal) SignedRaw(raw uint64) int64 {
	if !s.Signed || s.Length >= 64 {
		return int64(raw)
	}
	shift := uint(64 - s.Length)
	return int64(raw<<shift) >> shift
}

// MuxValueOf returns the multiplexer switch value carried by a payload,
// or false when the message is not multiplexed or the payload is short.
func (m *Message) MuxValueOf(payload []byte) (int64, bool) {
	mux := m.multiplexer()
	if mux == nil {
		return 0, false
	}
	raw, ok := mux.Raw(payload)
	if !ok {
		return 0, false
	}
	return mux.SignedRaw(raw), true
}
