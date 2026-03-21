package server

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"charm.land/wish/v2/ratelimiter"
	"github.com/charmbracelet/ssh"
	"golang.org/x/time/rate"

	"github.com/paullj/paullj.com/internal/config"
	"github.com/paullj/paullj.com/internal/content"
	"github.com/paullj/paullj.com/internal/images"
	"github.com/paullj/paullj.com/internal/tui"
)

func Run(cfg *config.Config, store *content.PostStore, cache *images.Cache, diskCache *images.DiskCache, imgMode images.ImageMode, aboutRaw string) error {
	handler := func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		m := tui.NewModel(
			store.GetPosts(),
			imgMode,
			cfg,
			aboutRaw,
			cache,
			diskCache,
		)
		return m, []tea.ProgramOption{}
	}

	addr := cfg.SSH.Host + ":" + cfg.SSH.Port
	srv, err := wish.NewServer(
		wish.WithAddress(addr),
		wish.WithHostKeyPath(cfg.SSH.HostKeyPath),
		wish.WithIdleTimeout(cfg.SSH.IdleTimeout.Duration),
		wish.WithMaxTimeout(cfg.SSH.MaxTimeout.Duration),
		wish.WithMiddleware(
			bubbletea.Middleware(handler),
			logging.Middleware(),
			activeterm.Middleware(),
			tui.FilterMiddleware(cfg.SSH.Filter),
			ratelimiter.Middleware(ratelimiter.NewRateLimiter(
				rate.Every(cfg.SSH.RateLimit.Duration), cfg.SSH.RateBurst, 1024,
			)),
		),
	)
	if err != nil {
		return err
	}

	srv.LocalPortForwardingCallback = func(ctx ssh.Context, dhost string, dport uint32) bool {
		return false
	}
	srv.ReversePortForwardingCallback = func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
		return false
	}
	srv.ChannelHandlers = map[string]ssh.ChannelHandler{
		"session": ssh.DefaultSessionHandler,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	log.Printf("starting SSH server on %s", addr)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-done
	log.Println("shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	return srv.Shutdown(shutdownCtx)
}
