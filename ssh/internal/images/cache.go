package images

import "sync"

// Cache is a thread-safe LRU cache for encoded images.
type Cache struct {
	mu       sync.Mutex
	entries  map[string]*cacheEntry
	order    []string // oldest first
	size     int
	maxBytes int
}

type cacheEntry struct {
	data string
	size int
}

func NewCache(maxBytes int) *Cache {
	return &Cache{
		entries:  make(map[string]*cacheEntry),
		maxBytes: maxBytes,
	}
}

func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		return "", false
	}
	c.remove(key)
	c.order = append(c.order, key)
	return e.data, true
}

func (c *Cache) Put(key, data string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	sz := len(data)
	for c.size+sz > c.maxBytes && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		if e, ok := c.entries[oldest]; ok {
			c.size -= e.size
			delete(c.entries, oldest)
		}
	}

	c.entries[key] = &cacheEntry{data: data, size: sz}
	c.order = append(c.order, key)
	c.size += sz
}

func (c *Cache) remove(key string) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			return
		}
	}
}
