package healthattest

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func privateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestD063_IssuedProbeRoundTripAndBinding(t *testing.T) {
	base, _, _, versions := validPair(t)
	probe := &ChallengeProbe{
		PeerID: "probe-test-peer", SourceV4: "100.64.8.1/32",
		// Serialization-only fixture; no packets are sent to these addresses.
		Destinations: []ChallengeProbeDest{
			{AddrV4: "192.0.2.1/32", Proto: "tcp", Port: 443},
			{AddrV4: "198.51.100.1/32", Proto: "udp", Port: 53},
		}, ResolverDNS: true,
	}
	in := Input{TxnID: base.TxnID, ReleaseID: base.ReleaseID, NodeID: base.NodeID,
		IncarnationID: base.IncarnationID, BootID: base.BootID, Components: versions,
		Now: base.IssuedAt, TTL: 45 * time.Second, Probe: probe}
	ch, err := Issue(in)
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal(probe)
	if err != nil {
		t.Fatal(err)
	}
	dir := privateDir(t)
	if err := WriteChallenge(dir, ch); err != nil {
		t.Fatal(err)
	}
	got, _, err := ReadChallenge(dir)
	if err != nil || !probeSame(got.Probe, probe) || !got.Same(ch) {
		t.Fatal("probe changed on disk", err)
	}
	roundTrip, err := json.Marshal(got.Probe)
	if err != nil || !bytes.Equal(roundTrip, want) {
		t.Fatal("probe JSON changed on round trip", err)
	}
	// Configuration changes cannot mutate the issued challenge or its binding.
	probe.PeerID = "different-peer"
	probe.Destinations[0].Port++
	if !ch.Same(got) {
		t.Fatal("issued probe aliases caller storage")
	}
	for _, mutate := range []func(*ChallengeProbe){
		func(p *ChallengeProbe) { p.PeerID += "x" },
		func(p *ChallengeProbe) { p.SourceV4 = "100.64.8.2/32" },
		func(p *ChallengeProbe) { p.Destinations[0].AddrV4 = "203.0.113.1/32" },
		func(p *ChallengeProbe) { p.Destinations[0].Proto = "udp" },
		func(p *ChallengeProbe) { p.Destinations[0].Port++ },
		func(p *ChallengeProbe) { p.Destinations = p.Destinations[:1] },
		func(p *ChallengeProbe) { p.ResolverDNS = false },
	} {
		other, _, err := ReadChallenge(dir)
		if err != nil {
			t.Fatal(err)
		}
		mutate(other.Probe)
		if other.Same(ch) {
			t.Fatal("probe replacement matched issuance")
		}
	}
	_, at, now, _ := validPair(t)
	at.Nonce = ch.Nonce
	if reason := Validate(ch, at, now, versions); reason != "" {
		t.Fatal("matching passed attestation refused", reason)
	}
	next, err := Issue(in)
	if err != nil {
		t.Fatal(err)
	}
	if reason := Validate(next, at, now, versions); reason != ReasonNonceMismatch {
		t.Fatal("passed probe replay accepted", reason)
	}
	if err := MarkConsumed(dir, ch); err != nil {
		t.Fatal(err)
	}
	consumed, _, err := ReadChallenge(dir)
	if err != nil || !probeSame(consumed.Probe, ch.Probe) {
		t.Fatal("consumption changed probe", err)
	}
	if reason := Validate(consumed, at, now, versions); reason != ReasonConsumed {
		t.Fatal("consumed probe accepted", reason)
	}
	in.TTL = 300*time.Second + time.Nanosecond
	if _, err := Issue(in); err == nil {
		t.Fatal("probe issuance exceeded 300s")
	}
}

func TestD053_JournalDirectoryAndLeafModes(t *testing.T) {
	ch, at, _, _ := validPair(t)
	for _, mode := range []os.FileMode{0o701, 0o710, 0o740, 0o770, 0o777} {
		dir := privateDir(t)
		if err := os.Chmod(dir, mode); err != nil {
			t.Fatal(err)
		}
		ops := []func() error{
			func() error { return WriteChallenge(dir, ch) },
			func() error { _, _, err := ReadChallenge(dir); return err },
			func() error { return WriteAttestation(dir, at) },
			func() error { _, err := ObserveAttestation(dir); return err },
			func() error { _, _, err := ReadAttestation(dir); return err },
			func() error { _, _, err := ReadObservedAttestation(dir); return err },
		}
		for i, op := range ops {
			if err := op(); ReasonOf(err) != ReasonBadMode {
				t.Fatalf("mode %o op %d: %v", mode, i, err)
			}
		}
	}
	dir := privateDir(t)
	if err := WriteAttestation(dir, at); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(dir, AttestationFile), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := ObserveAttestation(dir); ReasonOf(err) != ReasonBadMode {
		t.Fatalf("observation accepted shared leaf: %v", err)
	}
	file := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckDir(file); ReasonOf(err) != ReasonMalformed {
		t.Fatal(err)
	}
}

func TestD053_JournalSymlinkWithTrailingSeparator(t *testing.T) {
	dir := privateDir(t)
	link := filepath.Join(privateDir(t), "journal")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if err := CheckDir(link + string(os.PathSeparator)); ReasonOf(err) != ReasonSymlink {
		t.Fatalf("followed directory symlink with trailing separator: %v", err)
	}
}

type ownerInfo struct {
	os.FileInfo
	stat syscall.Stat_t
}

func (i ownerInfo) Sys() any { return &i.stat }

func TestD053_JournalOwnerMustBeEffectiveUser(t *testing.T) {
	info, err := os.Lstat(privateDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := checkOwner(info); err != nil {
		t.Fatal(err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skip("platform does not expose file owner")
	}
	foreign := ownerInfo{FileInfo: info, stat: *st}
	foreign.stat.Uid++
	if err := checkOwner(foreign); ReasonOf(err) != ReasonBadOwner {
		t.Fatalf("foreign owner accepted: %v", err)
	}
}

func TestD053_AttestationObservationHashesDecodedBytes(t *testing.T) {
	dir := privateDir(t)
	_, at, _, _ := validPair(t)
	if err := WriteAttestation(dir, at); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, AttestationFile))
	if err != nil {
		t.Fatal(err)
	}
	got, obs, err := ReadObservedAttestation(dir)
	if err != nil || got != at || obs.Sum != sha256.Sum256(body) || obs.Size != int64(len(body)) {
		t.Fatalf("decoded observation differs: %+v %v", obs, err)
	}
}

func TestChallengeProbeRoundTrip(t *testing.T) {
	ch, _, _, _ := validPair(t)
	if ch.Probe != nil {
		t.Fatal("Issue set a probe")
	}
	dir := privateDir(t)
	if err := WriteChallenge(dir, ch); err != nil {
		t.Fatal(err)
	}
	got, _, err := ReadChallenge(dir)
	if err != nil || !got.Same(ch) || got.Probe != nil {
		t.Fatalf("%v %+v", err, got.Probe)
	}
	ch.Probe = &ChallengeProbe{
		PeerID:   "peer-probe",
		SourceV4: "100.64.8.1/32",
		Destinations: []ChallengeProbeDest{{
			AddrV4: "1.2.3.4/32", Proto: "tcp", Port: 80,
		}},
		ResolverDNS: true,
	}
	if err := WriteChallenge(dir, ch); err != nil {
		t.Fatal(err)
	}
	got, _, err = ReadChallenge(dir)
	if err != nil || !got.Same(ch) {
		t.Fatalf("%v %+v", err, got.Probe)
	}
	other := ch
	other.Probe = nil
	if other.Same(ch) {
		t.Fatal("a missing probe matched a probe")
	}
}

func TestD053_FutureClockBoundAndReasonVocabulary(t *testing.T) {
	ch, at, now, versions := validPair(t)
	at.GeneratedAt = now.Add(MaxClockSkew)
	if reason := Validate(ch, at, now, versions); reason != "" {
		t.Fatal(reason)
	}
	at.GeneratedAt = at.GeneratedAt.Add(time.Nanosecond)
	if reason := Validate(ch, at, now, versions); reason != ReasonGeneratedInFuture {
		t.Fatal(reason)
	}
	at.GeneratedAt = now
	for _, reason := range []string{"unknown_token", "member_123", "secret.example.com", "100.64.0.1"} {
		at.Checks.TunnelProbe.Reason = reason
		if got := Validate(ch, at, now, versions); got != ReasonBadReason {
			t.Fatalf("accepted reason %q: %s", reason, got)
		}
	}
}
