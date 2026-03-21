package tui

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"

	"charm.land/wish/v2"
	"github.com/charmbracelet/ssh"

	"github.com/paullj/paullj.com/internal/config"
)

func FilterMiddleware(cfg config.FilterConfig) wish.Middleware {
	var conns sync.Map

	return func(next ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			clientVersion := s.Context().ClientVersion()
			for _, prefix := range cfg.BlockedPrefixes {
				if strings.HasPrefix(clientVersion, prefix) {
					log.Printf("blocked client %q from %s", clientVersion, s.RemoteAddr())
					wish.Fatal(s, fmt.Errorf("connection refused"))
					return
				}
			}

			ip, _, err := net.SplitHostPort(s.RemoteAddr().String())
			if err != nil {
				ip = s.RemoteAddr().String()
			}

			val, _ := conns.LoadOrStore(ip, &atomic.Int32{})
			counter := val.(*atomic.Int32)
			n := counter.Add(1)
			defer counter.Add(-1)

			if int(n) > cfg.MaxConcurrentPerIP {
				log.Printf("too many connections from %s (%d)", ip, n)
				wish.Fatal(s, fmt.Errorf("too many connections"))
				return
			}

			next(s)
		}
	}
}
