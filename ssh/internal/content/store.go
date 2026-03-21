package content

import (
	"log"
	"sync"
	"time"

	"github.com/paullj/paullj.com/internal/images"
)

type PostStore struct {
	mu            sync.RWMutex
	posts         []Post
	dir           string
	cache         *images.Cache
	maxSize       int
	fetchTimeout  time.Duration
	maxAsciiWidth int
}

func NewPostStore(dir string, cache *images.Cache, maxSize int, fetchTimeout time.Duration, maxAsciiWidth int) (*PostStore, error) {
	s := &PostStore{
		dir:           dir,
		cache:         cache,
		maxSize:       maxSize,
		fetchTimeout:  fetchTimeout,
		maxAsciiWidth: maxAsciiWidth,
	}
	if err := s.Reload(); err != nil {
		return nil, err
	}
	return s, nil
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
	go PrewarmImageCache(posts, s.cache, s.maxSize, s.fetchTimeout, s.maxAsciiWidth)
	return nil
}

func (s *PostStore) GetPosts() []Post {
	s.mu.RLock()
	out := make([]Post, len(s.posts))
	copy(out, s.posts)
	s.mu.RUnlock()
	return out
}
