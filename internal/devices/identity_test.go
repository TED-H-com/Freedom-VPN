package devices

import (
	"io"
	"log/slog"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestID01_D052DiscoveryRequiresConfiguredIssuer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, issuer := range []string{"", "issuer.example.invalid", "https://issuer.example.invalid?secret=TEST", "https://issuer.example.invalid#fragment", " https://issuer.example.invalid"} {
		if _, err := NewIdentityDiscovery(issuer, nil, logger); err == nil {
			t.Error("invalid issuer accepted")
		}
	}
	registry := prometheus.NewRegistry()
	const issuer = "https://issuer.example.invalid/realm"
	discovery, err := NewIdentityDiscovery(issuer, registry, logger)
	if err != nil || discovery.proof() != "shared_issuer_public_subject:"+issuer {
		t.Fatal("issuer was not preserved exactly", err)
	}
	if _, err = NewIdentityDiscovery(issuer, registry, logger); err == nil {
		t.Fatal("duplicate discovery collector not reported")
	}
}
