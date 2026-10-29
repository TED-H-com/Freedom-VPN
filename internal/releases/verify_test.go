package releases

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"io"
	"net/netip"
	"testing"
)

func TestSEC01_FetcherRejectsNonpublicAddresses(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "169.254.169.254", "10.0.0.1", "100.64.0.1", "192.0.2.1", "::1", "::ffff:127.0.0.1", "fe80::1", "fc00::1", "2001:db8::1", "64:ff9b::a00:1"} {
		if publicAddress(netip.MustParseAddr(ip)) {
			t.Fatalf("accepted %s", ip)
		}
	}
}

type testFetcher struct{ data []byte }

func (f testFetcher) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(f.data)), nil
}
func TestUPD02_ArtifactDigestAndSize(t *testing.T) {
	// TEST ONLY: validator origin policy suffices for this isolated stream check.
	v := &Validator{origins: map[string]bool{"releases.example.invalid": true}}
	s := &Service{Validator: v}
	good := []byte("TEST ONLY artifact")
	sum := sha256.Sum256(good)
	a := api.ManifestArtifact{Url: "https://releases.example.invalid/amd64.tar.zst", SizeBytes: decstr.U64(len(good)), Sha256: hex.EncodeToString(sum[:])}
	if e := s.verifyArtifact(context.Background(), testFetcher{good}, a); e != nil {
		t.Fatal(e)
	}
	altered := bytes.Clone(good)
	altered[0] = 'X'
	if e := s.verifyArtifact(context.Background(), testFetcher{altered}, a); !errors.Is(e, apierr.New(apierr.HashMismatch, "")) {
		t.Fatal(e)
	}
	for _, data := range [][]byte{good[:len(good)-1], append(bytes.Clone(good), '!')} {
		if e := s.verifyArtifact(context.Background(), testFetcher{data}, a); !errors.Is(e, apierr.New(apierr.SizeMismatch, "")) {
			t.Fatal(e)
		}
	}
}
