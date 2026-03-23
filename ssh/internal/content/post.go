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
	Draft       bool      `yaml:"draft"`
	Slug        string
	Body        string
}

func LoadPosts(dir string, includeDrafts bool) ([]Post, error) {
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
		if p.Draft && !includeDrafts {
			continue
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
func RenderMarkdownWithImages(content string, width int, style string, mode images.ImageMode, darkTheme bool, cache *images.Cache, diskCache *images.DiskCache, maxSize int, fetchTimeout time.Duration, maxAsciiWidth int, contentDir string) (string, []MermaidDiagram, error) {
	// Step 1: Extract alerts before Glamour mangles blockquote syntax
	content, alertRefs := extractAlerts(content)

	// Step 2: Extract footnote definitions and references
	content, footnoteDefs := extractFootnoteDefs(content)
	content, footnoteOrder := extractFootnoteRefs(content, footnoteDefs)

	// Step 3: Extract mermaid diagrams
	content, mermaidRefs := extractMermaid(content)

	// Step 4: Extract images
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
		return "", nil, err
	}

	for _, ref := range refs {
		encoded := resolveImage(ref.url, ref.alt, width, mode, darkTheme, cache, diskCache, maxSize, fetchTimeout, maxAsciiWidth, contentDir)
		padded := regexp.MustCompile(`[^\n]*` + regexp.QuoteMeta(ref.marker) + `[^\n]*`)
		rendered = padded.ReplaceAllLiteralString(rendered, encoded)
	}

	// Replace mermaid placeholders with rendered diagrams
	var mermaidOverflows []MermaidDiagram
	rendered, mermaidOverflows = replaceMermaid(rendered, mermaidRefs, width, darkTheme)

	// Replace alert placeholders with styled output
	rendered = replaceAlerts(rendered, alertRefs, width, style, darkTheme)

	// Replace footnote ref placeholders with styled [N]
	rendered = replaceFootnoteRefs(rendered, footnoteOrder, darkTheme)

	// Append footnote section at bottom
	rendered += renderFootnoteSection(footnoteDefs, footnoteOrder, width, style, darkTheme)

	return rendered, mermaidOverflows, nil
}

func ImageCacheKey(url string, width int, mode images.ImageMode, darkTheme bool, maxAsciiWidth int) string {
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

func resolveImage(url, alt string, width int, mode images.ImageMode, darkTheme bool, cache *images.Cache, diskCache *images.DiskCache, maxSize int, fetchTimeout time.Duration, maxAsciiWidth int, contentDir string) string {
	cacheKey := ImageCacheKey(url, width, mode, darkTheme, maxAsciiWidth)
	if cached, ok := cache.Get(cacheKey); ok {
		log.Printf("image cache hit (memory) %s", url)
		return cached
	}

	if diskCache != nil {
		if cached, err := diskCache.Get(cacheKey); err == nil {
			log.Printf("image cache hit (disk) %s", url)
			cache.Put(cacheKey, cached)
			return cached
		}
	}

	log.Printf("image cache miss %s, rendering on-the-fly", url)
	data, err := images.FetchImage(url, maxSize, fetchTimeout, contentDir)
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
func PrewarmImageCache(posts []Post, cache *images.Cache, maxSize int, fetchTimeout time.Duration, maxAsciiWidth int, contentDir string) {
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
		data, err := images.FetchImage(url, maxSize, fetchTimeout, contentDir)
		if err != nil {
			log.Printf("prewarm fetch %s: %v", url, err)
			continue
		}
		for _, dark := range []bool{true, false} {
			for _, mode := range []images.ImageMode{images.ImageModeChafa, images.ImageModeAscii} {
				key := ImageCacheKey(url, maxAsciiWidth, mode, dark, maxAsciiWidth)
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
