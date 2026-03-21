package images

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip      string
		private bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true},
		{"::1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if got := isPrivateIP(ip); got != tt.private {
			t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.private)
		}
	}
}

func TestFetchImage_BlocksPrivateIPs(t *testing.T) {
	urls := []string{
		"http://127.0.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::1]/",
		"http://10.0.0.1/secret",
	}
	for _, u := range urls {
		_, err := FetchImage(u, 1<<20, 2*time.Second)
		if err == nil {
			t.Errorf("FetchImage(%s) should have been blocked", u)
		}
	}
}

func TestReadLocalImage_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	allowed := filepath.Join(dir, "content")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file inside allowed dir
	img := filepath.Join(allowed, "test.png")
	if err := os.WriteFile(img, []byte("fake-image"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a file outside allowed dir
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should succeed: file inside content dir
	data, err := readLocalImage(img, 1<<20, allowed)
	if err != nil {
		t.Fatalf("expected success for file in allowed dir: %v", err)
	}
	if string(data) != "fake-image" {
		t.Fatalf("unexpected data: %s", data)
	}

	// Should fail: path traversal
	_, err = readLocalImage(filepath.Join(allowed, "../secret.txt"), 1<<20, allowed)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}

	// Should fail: absolute path outside
	_, err = readLocalImage(secret, 1<<20, allowed)
	if err == nil {
		t.Fatal("expected error for path outside allowed dir")
	}
}
