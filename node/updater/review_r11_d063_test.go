package updater

import (
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
	// TODO(review): SpecDeniedV4 already contains the ranges named in the
	// question except 192.88.99.0/24. validateProbePeer also accepts three
	// other RFC 6890 special-purpose ranges, and it has no input for the
	// node's public address, the control URL, or the configured resolvers.
	denied := map[string]bool{}
	for _, p := range enforce.SpecDeniedV4() {
		denied[p.String()] = true
	}
	asked := []string{
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.88.99.0/24",
		"198.18.0.0/15", "240.0.0.0/4", "255.255.255.255/32",
		"192.0.2.0/24", "198.51.100.0/24", "203.0.113.0/24",
	}
	var missing []string
	for _, cidr := range asked {
		if !denied[cidr] {
			missing = append(missing, cidr)
		}
	}
	gaps := []string{"192.88.99.1/32", "192.31.196.1/32", "192.52.193.1/32", "192.175.48.1/32", "1.1.1.1/32", "8.8.8.8/32", "9.9.9.9/32"}
	var accepted []string
	for _, addr := range gaps {
		if err := validateProbePeer(probeWithDest(addr)); err == nil {
			accepted = append(accepted, addr)
		}
	}
	held := []string{
		"0.0.0.1/32", "10.1.0.1/32", "100.64.0.1/32", "127.0.0.1/32",
		"169.254.169.254/32", "172.16.0.1/32", "192.0.0.1/32", "192.0.2.1/32",
		"192.168.0.1/32", "198.18.0.1/32", "198.51.100.1/32", "203.0.113.1/32",
		"224.0.0.1/32", "240.0.0.1/32", "255.255.255.255/32",
	}
	var leaked []string
	for _, addr := range held {
		if err := validateProbePeer(probeWithDest(addr)); err == nil {
			leaked = append(leaked, addr)
		}
	}
	if len(leaked) != 0 {
		t.Fatalf("destinations inside SpecDeniedV4 were accepted: %v", leaked)
	}
	sources := []string{"10.9.8.7/32", "172.16.5.5/32", "192.168.1.9/32", "100.64.8.1/32", "192.0.2.8/32"}
	var sourceOK []string
	for _, src := range sources {
		p := probeFixture()
		p.SourceV4 = src
		if err := validateProbePeer(p); err == nil {
			sourceOK = append(sourceOK, src)
		}
	}
	if len(accepted) > 0 || len(sourceOK) > 0 {
		t.Fatalf("SpecDeniedV4 is missing %v. validateProbePeer accepted destinations %v (1.1.1.1 stands in for a control address, 8.8.8.8 for a configured resolver, 9.9.9.9 for a node public address that is not in the updater config) and sources %v", missing, accepted, sourceOK)
	}
	t.Fatalf("proof inconclusive: missing deny ranges %v but none of the sample destinations were accepted", missing)
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
