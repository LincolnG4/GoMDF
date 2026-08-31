package datasection

import "sync"

// lru is a tiny LRU cache of decompressed DZ payloads keyed by the file
// offset of the compressed data. Safe for concurrent use: data-section
// readers are shared by parallel channel reads.
type lru struct {
	mu      sync.Mutex
	cap     int
	entries map[int64][]byte
	order   []int64 // least recently used first
}

func newLRU(capacity int) *lru {
	if capacity <= 0 {
		capacity = 8
	}
	return &lru{cap: capacity, entries: make(map[int64][]byte, capacity)}
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
	if _, ok := c.entries[key]; !ok && len(c.entries) >= c.cap {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}
	c.entries[key] = buf
	c.touch(key)
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
