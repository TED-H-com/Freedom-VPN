package healthattest

import (
	"crypto/sha256"
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
