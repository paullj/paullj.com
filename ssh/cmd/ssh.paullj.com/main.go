package main

import (
	"flag"
	"log"
	"os"

	"github.com/paullj/paullj.com/internal/config"
	"github.com/paullj/paullj.com/internal/content"
	"github.com/paullj/paullj.com/internal/images"
	"github.com/paullj/paullj.com/internal/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	cache := images.NewCache(cfg.SSH.Images.CacheMaxBytes)
	diskCache := images.NewDiskCache(cfg.SSH.Images.CacheDir)

	store, err := content.NewPostStore(
		cfg.Content.PostsDir,
		cache,
		diskCache,
		cfg.SSH.Images.MaxSize,
		cfg.SSH.Images.FetchTimeout.Duration,
		cfg.SSH.Images.MaxAsciiWidth,
	)
	if err != nil {
		log.Fatalf("loading posts: %v", err)
	}

	var aboutRaw string
	if cfg.Content.AboutPath != "" {
		data, err := os.ReadFile(cfg.Content.AboutPath)
		if err != nil {
			log.Printf("warning: could not load about page: %v", err)
		} else {
			aboutRaw = string(data)
		}
	}

	imgMode := images.DetectImageMode()
	log.Printf("image mode: %s", imgMode)

	if err := server.Run(cfg, store, cache, diskCache, imgMode, aboutRaw); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
