package records

// bitField extracts a bitCount-wide unsigned field starting bitOffset
// bits into the bytes at p.
//
// Little-endian: the field's least significant bit is bit bitOffset of
// p[0]; bytes contribute in little-endian order.
// Big-endian: the containing bytes hold a big-endian integer whose
// bitOffset least significant bits precede the field.
func bitField(p []byte, bitOffset uint8, bitCount uint32, bigE bool) uint64 {
	nBytes := (int(bitOffset) + int(bitCount) + 7) / 8
	var v uint64
	if bigE {
		for i := 0; i < nBytes; i++ {
			v = v<<8 | uint64(p[i])
		}
	} else {
		for i := nBytes - 1; i >= 0; i-- {
			v = v<<8 | uint64(p[i])
		}
	}
	v >>= bitOffset
	if bitCount < 64 {
		v &= (1 << bitCount) - 1
	}
	return v
}
