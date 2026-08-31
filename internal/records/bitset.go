package records

import "math/bits"

// Bitset is a packed per-sample flag set (used for invalidation bits).
type Bitset struct {
	words []uint64
	n     int
}

// NewBitset returns an empty bitset with capacity for n bits.
func NewBitset(n int) *Bitset {
	return &Bitset{words: make([]uint64, 0, (n+63)/64)}
}

// Append adds one bit.
func (b *Bitset) Append(v bool) {
	if b.n%64 == 0 {
		b.words = append(b.words, 0)
	}
	if v {
		b.words[b.n/64] |= 1 << (b.n % 64)
	}
	b.n++
}

// Get reports bit i.
func (b *Bitset) Get(i int) bool {
	if i < 0 || i >= b.n {
		return false
	}
	return b.words[i/64]&(1<<(i%64)) != 0
}

// Len returns the number of bits.
func (b *Bitset) Len() int { return b.n }

// Count returns the number of set bits.
func (b *Bitset) Count() int {
	c := 0
	for _, w := range b.words {
		c += bits.OnesCount64(w)
	}
	return c
}
