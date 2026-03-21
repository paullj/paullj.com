package images

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// DiskCache persists encoded images to disk, keyed by SHA-256 of the cache key.
type DiskCache struct {
	dir string
}

// NewDiskCache creates a DiskCache backed by dir. Returns nil if dir is empty.
func NewDiskCache(dir string) *DiskCache {
	if dir == "" {
		return nil
	}
	return &DiskCache{dir: dir}
}

func (d *DiskCache) keyPath(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(d.dir, fmt.Sprintf("%x", h))
}

// Get reads a cached entry from disk. Returns ("", error) on miss.
func (d *DiskCache) Get(key string) (string, error) {
	data, err := os.ReadFile(d.keyPath(key))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Put writes a cached entry to disk, creating the directory if needed.
func (d *DiskCache) Put(key, data string) error {
	if err := os.MkdirAll(d.dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(d.keyPath(key), []byte(data), 0o644)
}
