package images

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/draw"
)

type ImageMode int

const (
	ImageModeChafa ImageMode = iota
	ImageModeAscii
)

// DetectImageMode checks if chafa is available, returning ImageModeChafa if so.
func DetectImageMode() ImageMode {
	if _, err := exec.LookPath("chafa"); err == nil {
		return ImageModeChafa
	}
	return ImageModeAscii
}

func (m ImageMode) String() string {
	if m == ImageModeChafa {
		return "chafa"
	}
	return "halfblock"
}

// isPrivateIP returns true if the IP is in a private/loopback/link-local range.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{mustParseCIDR("127.0.0.0/8")},
		{mustParseCIDR("10.0.0.0/8")},
		{mustParseCIDR("172.16.0.0/12")},
		{mustParseCIDR("192.168.0.0/16")},
		{mustParseCIDR("169.254.0.0/16")},
		{mustParseCIDR("::1/128")},
		{mustParseCIDR("fc00::/7")},
		{mustParseCIDR("fe80::/10")},
	}
	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

func mustParseCIDR(s string) *net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return network
}

// ssrfSafeDialer returns a dialer that blocks connections to private/loopback IPs.
func ssrfSafeDialer(timeout time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address: %w", err)
		}

		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("DNS lookup failed: %w", err)
		}

		for _, ip := range ips {
			if isPrivateIP(ip.IP) {
				return nil, fmt.Errorf("blocked: %s resolves to private IP %s", host, ip.IP)
			}
		}

		// Connect only to the first validated IP to prevent TOCTOU with round-robin DNS.
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
}

// FetchImage loads an image from a URL (http/https) or local file path.
// For local paths, contentDir restricts reads to that directory (pass "" to skip).
func FetchImage(url string, maxSize int, fetchTimeout time.Duration, contentDir ...string) ([]byte, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		var base string
		if len(contentDir) > 0 {
			base = contentDir[0]
		}
		return readLocalImage(url, maxSize, base)
	}

	transport := &http.Transport{
		DialContext: ssrfSafeDialer(fetchTimeout),
	}
	client := &http.Client{
		Timeout:   fetchTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxSize)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxSize {
		return nil, fmt.Errorf("image exceeds %dMB limit", maxSize>>20)
	}
	return data, nil
}

func readLocalImage(path string, maxSize int, allowedBase string) ([]byte, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	if allowedBase != "" {
		base, err := filepath.Abs(allowedBase)
		if err != nil {
			return nil, fmt.Errorf("resolve base: %w", err)
		}
		if !strings.HasPrefix(abs, base+string(filepath.Separator)) && abs != base {
			return nil, fmt.Errorf("path %s is outside allowed directory", path)
		}
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	if len(data) > maxSize {
		return nil, fmt.Errorf("image exceeds %dMB limit", maxSize>>20)
	}
	return data, nil
}

// EncodeAscii renders an image as half-block ANSI art.
func EncodeAscii(imgData []byte, widthCells int, maxAsciiWidth int, darkTheme bool) (string, error) {
	src, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return "", err
	}

	w := widthCells
	if w > maxAsciiWidth {
		w = maxAsciiWidth
	}

	bounds := src.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	h := srcH * w / srcW
	if h%2 != 0 {
		h++
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var bg color.RGBA
	if darkTheme {
		bg = color.RGBA{0, 0, 0, 255}
	} else {
		bg = color.RGBA{255, 255, 255, 255}
	}

	blend := func(x, y int) (uint8, uint8, uint8) {
		r, g, b, a := dst.At(x, y).RGBA()
		alpha := float64(a) / 0xffff
		br, bgg, bb := float64(bg.R), float64(bg.G), float64(bg.B)
		return uint8(float64(r>>8)*alpha + br*(1-alpha)),
			uint8(float64(g>>8)*alpha + bgg*(1-alpha)),
			uint8(float64(b>>8)*alpha + bb*(1-alpha))
	}

	var sb strings.Builder
	for y := 0; y < h; y += 2 {
		for x := 0; x < w; x++ {
			tr, tg, tb := blend(x, y)
			var br, bgg, bb uint8
			if y+1 < h {
				br, bgg, bb = blend(x, y+1)
			} else {
				br, bgg, bb = bg.R, bg.G, bg.B
			}
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", tr, tg, tb, br, bgg, bb)
		}
		sb.WriteString("\x1b[0m\n")
	}
	return sb.String(), nil
}

// EncodeChafa renders an image using the external chafa tool.
// Falls back to EncodeAscii if chafa is not installed.
func EncodeChafa(imgData []byte, widthCells int, maxAsciiWidth int, darkTheme bool) (string, error) {
	chafaPath, err := exec.LookPath("chafa")
	if err != nil {
		return EncodeAscii(imgData, widthCells, maxAsciiWidth, darkTheme)
	}

	w := widthCells
	if w > maxAsciiWidth {
		w = maxAsciiWidth
	}

	cmd := exec.Command(chafaPath,
		"--format=symbols",
		"--colors=full",
		"--color-space=din99d",
		"--dither=diffusion",
		"--color-extractor=median",
		"--work=9",
		fmt.Sprintf("--size=%dx", w),
		"-",
	)
	cmd.Stdin = bytes.NewReader(imgData)

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return EncodeAscii(imgData, widthCells, maxAsciiWidth, darkTheme)
	}
	return out.String(), nil
}
