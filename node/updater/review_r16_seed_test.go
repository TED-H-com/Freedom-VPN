package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestReviewR16_DownloadRedirectCannotBlessDifferentBytes follows an allowed
// redirect and still requires the artifact bytes to match the signed hash.
func TestReviewR16_DownloadRedirectCannotBlessDifferentBytes(t *testing.T) {
	good := []byte("signed-release-bytes")
	sum := sha256.Sum256(good)
	want := hex.EncodeToString(sum[:])
	mux := http.NewServeMux()
	mux.HandleFunc("/bytes", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(good) })
	mux.HandleFunc("/evil", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("not-the-release")) })
	mux.HandleFunc("/via", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/bytes", http.StatusFound)
	})
	mux.HandleFunc("/swap", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/evil", http.StatusFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	dir := t.TempDir()
	eng := &Engine{
		AllowLoopback: true,
		HTTP:          srv.Client(),
		Download:      DownloadConfig{MaxAttempts: 1, ContentRetries: 0, Initial: time.Millisecond, MaxWait: time.Millisecond, AttemptTimeout: time.Second},
		Jitter:        func() float64 { return 0 },
	}
	if err := eng.download(context.Background(), srv.URL+"/swap", filepath.Join(dir, "bad"), want, hashRaw, 0, 1<<20, nil); err == nil {
		t.Fatal("TODO(review): a redirect blessed bytes that do not match the signed hash")
	}
	dest := filepath.Join(dir, "good")
	if err := eng.download(context.Background(), srv.URL+"/via", dest, want, hashRaw, int64(len(good)), 1<<20, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || !bytes.Equal(got, good) {
		t.Fatalf("redirected body = %q %v", got, err)
	}
}
