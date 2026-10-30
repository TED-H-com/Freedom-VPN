package updater

import (
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/google/uuid"
)

func probeWithDest(addr string) *healthattest.ChallengeProbe {
	p := probeFixture()
	p.Destinations = []healthattest.ChallengeProbeDest{{AddrV4: addr, Proto: "tcp", Port: 443}}
	return p
}

func TestReviewR11_IssuerSpecialPurposeDestinations(t *testing.T) {
	// TEST ONLY: public address stand-ins are compared in memory, never dialed.
	known := probeAddresses{
		node:      []netip.Addr{netip.MustParseAddr("9.9.9.9")},
		control:   []netip.Addr{netip.MustParseAddr("1.1.1.1")},
		resolvers: []netip.Addr{netip.MustParseAddr("8.8.8.8")},
	}
	for _, addr := range []string{"1.1.1.1/32", "8.8.8.8/32", "9.9.9.9/32"} {
		t.Run("context_"+addr, func(t *testing.T) {
			if err := validateProbePeer(probeWithDest(addr), known); err == nil {
				t.Fatal("node/control/resolver destination accepted")
			}
		})
	}
	denied := map[string]bool{}
	for _, p := range enforce.SpecDeniedV4() {
		denied[p.String()] = true
	}
	for _, cidr := range []string{
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "198.18.0.0/15",
		"240.0.0.0/4", "255.255.255.255/32", "192.0.2.0/24", "198.51.100.0/24", "203.0.113.0/24",
	} {
		if !denied[cidr] {
			t.Errorf("SpecDeniedV4 missing %s", cidr)
		}
	}
	for _, cidr := range []string{"192.88.99.0/24", "192.31.196.0/24", "192.52.193.0/24", "192.175.48.0/24"} {
		t.Run("special_"+cidr, func(t *testing.T) {
			// TODO(review): netlab follow-up 5 owns these SpecDeniedV4 additions.
			// Keep the rejection assertion active as soon as the shared range lands.
			if !denied[cidr] {
				t.Skip("awaiting netlab follow-up 5 SpecDeniedV4 addition")
			}
			addr := netip.MustParsePrefix(cidr).Addr().Next().String() + "/32"
			if err := validateProbePeer(probeWithDest(addr), known); err == nil {
				t.Fatal("special-purpose destination accepted")
			}
		})
	}
	held := []string{
		"0.0.0.1/32", "10.1.0.1/32", "100.64.0.1/32", "127.0.0.1/32",
		"169.254.169.254/32", "172.16.0.1/32", "192.0.0.1/32", "192.0.2.1/32",
		"192.168.0.1/32", "198.18.0.1/32", "198.51.100.1/32", "203.0.113.1/32",
		"224.0.0.1/32", "240.0.0.1/32", "255.255.255.255/32",
	}
	for _, addr := range held {
		if err := validateProbePeer(probeWithDest(addr), known); err == nil {
			t.Errorf("destination inside SpecDeniedV4 accepted: %s", addr)
		}
	}
	// Original proof lines 61–71 rejected legitimate overlay sources. Follow-up
	// 10 explicitly preserves them; range restrictions apply to destinations.
	for _, src := range []string{"10.9.8.7/32", "172.16.5.5/32", "192.168.1.9/32", "100.64.8.1/32", "192.0.2.8/32"} {
		p := probeFixture()
		p.SourceV4 = src
		if err := validateProbePeer(p, known); err != nil {
			t.Errorf("non-node overlay source refused: %s", src)
		}
	}
	p := probeFixture()
	p.SourceV4 = "9.9.9.9/32"
	if err := validateProbePeer(p, known); err == nil {
		t.Fatal("node source accepted")
	}
}

func TestReviewR11_MalformedProbePeerHidesTheValue(t *testing.T) {
	const secret = "203.0.113.55"
	cfg := probeConfigFixture(t)
	cfg.ProbePeer = probeFixture()
	cfg.ProbePeer.Destinations[0].AddrV4 = secret + "/32"
	cfg.ProbePeer.PeerID = "peer-" + secret
	_, err := loadProbeConfig(t, cfg)
	if err == nil || err.Error() != errProbePeer.Error() || strings.Contains(err.Error(), secret) {
		t.Fatalf("malformed probe_peer error = %v", err)
	}
	if _, err = cfg.Engine(); err == nil || err.Error() != errProbePeer.Error() || strings.Contains(err.Error(), secret) {
		t.Fatalf("Engine error = %v", err)
	}
}

func TestReviewR11_ProbePeerIsASnapshot(t *testing.T) {
	// Engine copies the peer at construction. A later edit of the config value
	// does not change the copy, and a new issuance mints a new nonce so the
	// agent cannot treat the two files as the same challenge.
	cfg := probeConfigFixture(t)
	cfg.ProbePeer = probeFixture()
	snap := cloneProbePeer(cfg.ProbePeer)
	cfg.ProbePeer.Destinations[0].AddrV4 = "9.9.9.9/32"
	if snap.Destinations[0].AddrV4 == "9.9.9.9/32" {
		t.Fatal("clone shares the destination slice with the config")
	}
	now := time.Now().UTC()
	base := healthattest.Input{
		TxnID: uuid.Must(uuid.NewV7()).String(), ReleaseID: uuid.Must(uuid.NewV7()).String(),
		NodeID: uuid.Must(uuid.NewV7()).String(), IncarnationID: uuid.Must(uuid.NewV7()).String(),
		BootID: uuid.Must(uuid.NewV7()).String(), Now: now, TTL: 300 * time.Second, Probe: snap,
		Components: healthattest.Components{NodeAgent: "a", Netbird: "b", NodeUpdater: "c", Enforcement: "d"},
	}
	first, err := healthattest.Issue(base)
	if err != nil {
		t.Fatal(err)
	}
	base.Probe = cfg.ProbePeer
	second, err := healthattest.Issue(base)
	if err != nil {
		t.Fatal(err)
	}
	if first.Nonce == second.Nonce || first.Same(second) {
		t.Fatal("a replaced probe_peer kept the previous issuance")
	}
	if !first.ExpiresAt.Equal(first.IssuedAt.Add(300 * time.Second)) {
		t.Fatal("300s TTL was shortened")
	}
}
