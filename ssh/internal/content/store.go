package content

import (
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/paullj/paullj.com/internal/images"
)

type PostStore struct {
	mu            sync.RWMutex
	posts         []Post
	dir           string
	cache         *images.Cache
	diskCache     *images.DiskCache
	maxSize       int
	fetchTimeout  time.Duration
	maxAsciiWidth int
}

func NewPostStore(dir string, cache *images.Cache, diskCache *images.DiskCache, maxSize int, fetchTimeout time.Duration, maxAsciiWidth int) (*PostStore, error) {
	s := &PostStore{
		dir:           dir,
		cache:         cache,
		diskCache:     diskCache,
		maxSize:       maxSize,
		fetchTimeout:  fetchTimeout,
		maxAsciiWidth: maxAsciiWidth,
	}
	if err := s.Reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *PostStore) DiskCache() *images.DiskCache {
	return s.diskCache
}

func (s *PostStore) Reload() error {
	posts, err := LoadPosts(s.dir)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.posts = posts
	s.mu.Unlock()
	log.Printf("loaded %d posts", len(posts))
	if s.diskCache == nil {
		go PrewarmImageCache(posts, s.cache, s.maxSize, s.fetchTimeout, s.maxAsciiWidth, filepath.Dir(s.dir))
	} else {
		log.Println("disk cache present, skipping prewarm")
	}
	return nil
}

func (s *PostStore) GetPosts() []Post {
	s.mu.RLock()
	out := make([]Post, len(s.posts))
	copy(out, s.posts)
	s.mu.RUnlock()
	return out
}
