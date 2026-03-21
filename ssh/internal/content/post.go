package content

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"charm.land/glamour/v2"
	"github.com/adrg/frontmatter"

	"github.com/paullj/paullj.com/internal/images"
)

type Post struct {
	Title       string    `yaml:"title"`
	Date        time.Time `yaml:"date"`
	Description string    `yaml:"description"`
	Slug        string
	Body        string
}

func LoadPosts(dir string) ([]Post, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("glob posts: %w", err)
	}

	var posts []Post
	for _, f := range files {
		p, err := loadPost(f)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", f, err)
		}
		posts = append(posts, p)
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})
	return posts, nil
}

func loadPost(path string) (Post, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Post{}, err
	}

	var p Post
	body, err := frontmatter.Parse(strings.NewReader(string(data)), &p)
	if err != nil {
		return Post{}, fmt.Errorf("parse frontmatter: %w", err)
	}

	p.Body = string(body)
	p.Slug = strings.TrimSuffix(filepath.Base(path), ".md")

	if p.Title == "" {
		p.Title = p.Slug
	}
	return p, nil
}

// MdImageRe extracts ![alt](url) from raw markdown before glamour processing.
var MdImageRe = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

func RenderMarkdown(content string, width int, style string) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return r.Render(content)
}

// RenderMarkdownWithImages renders markdown and replaces image placeholders
// with terminal-encoded images.
func RenderMarkdownWithImages(content string, width int, style string, mode images.ImageMode, darkTheme bool, cache *images.Cache, maxSize int, fetchTimeout time.Duration, maxAsciiWidth int) (string, error) {
	type imageRef struct {
		marker string
		alt    string
		url    string
	}
	var refs []imageRef
	replaced := MdImageRe.ReplaceAllStringFunc(content, func(match string) string {
		sub := MdImageRe.FindStringSubmatch(match)
		marker := fmt.Sprintf("IMGPLACEHOLDER%d", len(refs))
		refs = append(refs, imageRef{marker: marker, alt: sub[1], url: sub[2]})
		return marker
	})

	rendered, err := RenderMarkdown(replaced, width, style)
	if err != nil {
		return "", err
	}

	for _, ref := range refs {
		encoded := resolveImage(ref.url, ref.alt, width, mode, darkTheme, cache, maxSize, fetchTimeout, maxAsciiWidth)
		padded := regexp.MustCompile(`[^\n]*` + regexp.QuoteMeta(ref.marker) + `[^\n]*`)
		rendered = padded.ReplaceAllLiteralString(rendered, encoded)
	}

	return rendered, nil
}

func imageCacheKey(url string, width int, mode images.ImageMode, darkTheme bool, maxAsciiWidth int) string {
	w := width
	if w > maxAsciiWidth {
		w = maxAsciiWidth
	}
	themeKey := 0
	if darkTheme {
		themeKey = 1
	}
	return fmt.Sprintf("%s:%d:%d:%d", url, w, mode, themeKey)
}

func resolveImage(url, alt string, width int, mode images.ImageMode, darkTheme bool, cache *images.Cache, maxSize int, fetchTimeout time.Duration, maxAsciiWidth int) string {
	cacheKey := imageCacheKey(url, width, mode, darkTheme, maxAsciiWidth)
	if cached, ok := cache.Get(cacheKey); ok {
		return cached
	}

	data, err := images.FetchImage(url, maxSize, fetchTimeout)
	if err != nil {
		log.Printf("image fetch %s: %v", url, err)
		return fmt.Sprintf("[Image: %s]", alt)
	}

	var encoded string
	switch mode {
	case images.ImageModeAscii:
		encoded, err = images.EncodeAscii(data, width, maxAsciiWidth, darkTheme)
	case images.ImageModeChafa:
		encoded, err = images.EncodeChafa(data, width, maxAsciiWidth, darkTheme)
	}
	if err != nil {
		log.Printf("image encode %s: %v", url, err)
		return fmt.Sprintf("[Image: %s]", alt)
	}

	cache.Put(cacheKey, encoded)
	return encoded
}

// PrewarmImageCache pre-renders all images from posts for both themes and modes.
func PrewarmImageCache(posts []Post, cache *images.Cache, maxSize int, fetchTimeout time.Duration, maxAsciiWidth int) {
	var urls []string
	seen := make(map[string]bool)
	for _, p := range posts {
		matches := MdImageRe.FindAllStringSubmatch(p.Body, -1)
		for _, m := range matches {
			url := m[2]
			if !seen[url] {
				seen[url] = true
				urls = append(urls, url)
			}
		}
	}
	if len(urls) == 0 {
		return
	}

	log.Printf("prewarming image cache for %d images...", len(urls))
	for _, url := range urls {
		data, err := images.FetchImage(url, maxSize, fetchTimeout)
		if err != nil {
			log.Printf("prewarm fetch %s: %v", url, err)
			continue
		}
		for _, dark := range []bool{true, false} {
			for _, mode := range []images.ImageMode{images.ImageModeChafa, images.ImageModeAscii} {
				key := imageCacheKey(url, maxAsciiWidth, mode, dark, maxAsciiWidth)
				if _, ok := cache.Get(key); ok {
					continue
				}
				var encoded string
				switch mode {
				case images.ImageModeAscii:
					encoded, err = images.EncodeAscii(data, maxAsciiWidth, maxAsciiWidth, dark)
				case images.ImageModeChafa:
					encoded, err = images.EncodeChafa(data, maxAsciiWidth, maxAsciiWidth, dark)
				}
				if err != nil {
					log.Printf("prewarm encode %s: %v", url, err)
					continue
				}
				cache.Put(key, encoded)
			}
		}
	}
	log.Println("image cache prewarmed")
}
