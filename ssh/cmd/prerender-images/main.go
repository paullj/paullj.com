package main

import (
	"flag"
	"log"
	"path/filepath"
	"strings"

	"github.com/paullj/paullj.com/internal/config"
	"github.com/paullj/paullj.com/internal/content"
	"github.com/paullj/paullj.com/internal/images"
)

func main() {
	outDir := flag.String("out", "/app/image-cache", "output directory for pre-rendered images")
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	posts, err := content.LoadPosts(cfg.Content.PostsDir, true)
	if err != nil {
		log.Fatalf("loading posts: %v", err)
	}

	dc := images.NewDiskCache(*outDir)

	var urls []string
	seen := make(map[string]bool)
	for _, p := range posts {
		matches := content.MdImageRe.FindAllStringSubmatch(p.Body, -1)
		for _, m := range matches {
			url := m[2]
			if !seen[url] && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				seen[url] = true
				urls = append(urls, url)
			}
		}
	}

	if len(urls) == 0 {
		log.Println("no local images found, nothing to prerender")
		return
	}

	maxW := cfg.SSH.Images.MaxAsciiWidth
	log.Printf("prerendering %d local images at width %d...", len(urls), maxW)

	for _, url := range urls {
		data, err := images.FetchImage(url, cfg.SSH.Images.MaxSize, cfg.SSH.Images.FetchTimeout.Duration, filepath.Dir(cfg.Content.PostsDir))
		if err != nil {
			log.Printf("fetch %s: %v", url, err)
			continue
		}

		for _, dark := range []bool{true, false} {
			for _, mode := range []images.ImageMode{images.ImageModeChafa, images.ImageModeAscii} {
				key := content.ImageCacheKey(url, maxW, mode, dark, maxW)
				var encoded string
				switch mode {
				case images.ImageModeAscii:
					encoded, err = images.EncodeAscii(data, maxW, maxW, dark)
				case images.ImageModeChafa:
					encoded, err = images.EncodeChafa(data, maxW, maxW, dark)
				}
				if err != nil {
					log.Printf("encode %s (mode=%s dark=%v): %v", url, mode, dark, err)
					continue
				}
				if err := dc.Put(key, encoded); err != nil {
					log.Printf("write cache %s: %v", url, err)
				}
			}
		}
	}
	log.Println("prerender complete")
}
