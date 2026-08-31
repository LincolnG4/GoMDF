package datasection

import "sync"

// lru is a tiny LRU cache of decompressed DZ payloads keyed by the file
// offset of the compressed data, bounded by a byte budget. Safe for
// concurrent use: data-section readers are shared by parallel channel
// reads.
type lru struct {
	mu      sync.Mutex
	budget  int64
	size    int64
	entries map[int64][]byte
	order   []int64 // least recently used first
}

func newLRU(budget int64) *lru {
	if budget <= 0 {
		budget = 128 << 20
	}
	return &lru{budget: budget, entries: make(map[int64][]byte)}
}

func (c *lru) get(key int64) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	buf, ok := c.entries[key]
	if ok {
		c.touch(key)
	}
	return buf, ok
}

func (c *lru) put(key int64, buf []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.entries[key]; ok {
		c.touch(key)
		return
	}
	// Evict least-recently-used entries until the new payload fits.
	// A payload larger than the whole budget is still cached alone.
	for c.size+int64(len(buf)) > c.budget && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		c.size -= int64(len(c.entries[oldest]))
		delete(c.entries, oldest)
	}
	c.entries[key] = buf
	c.size += int64(len(buf))
	c.order = append(c.order, key)
}

func (c *lru) touch(key int64) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	c.order = append(c.order, key)
}
