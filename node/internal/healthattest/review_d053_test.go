package healthattest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestD053_Review_SymlinkedJournalDirRejected(t *testing.T) {
	// Regression proof from the D-053 adversarial review.
	base := t.TempDir()
	realDir := filepath.Join(base, "real-journal")
	if err := os.MkdirAll(realDir, 0o700); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(base, "journal-symlink")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}

	ch, at, _, _ := validPair(t)
	if err := WriteChallenge(linkDir, ch); ReasonOf(err) != ReasonSymlink {
		t.Fatalf("WriteChallenge through symlinked journal dir succeeded or returned wrong reason: err=%v (reason=%q), want %q", err, ReasonOf(err), ReasonSymlink)
	}
	if _, _, err := ReadChallenge(linkDir); ReasonOf(err) != ReasonSymlink {
		t.Fatalf("ReadChallenge through symlinked journal dir succeeded or returned wrong reason: err=%v (reason=%q), want %q", err, ReasonOf(err), ReasonSymlink)
	}
	if err := WriteAttestation(linkDir, at); ReasonOf(err) != ReasonSymlink {
		t.Fatalf("WriteAttestation through symlinked journal dir succeeded or returned wrong reason: err=%v (reason=%q), want %q", err, ReasonOf(err), ReasonSymlink)
	}
	if _, err := ObserveAttestation(linkDir); ReasonOf(err) != ReasonSymlink {
		t.Fatalf("ObserveAttestation through symlinked journal dir succeeded or returned wrong reason: err=%v (reason=%q), want %q", err, ReasonOf(err), ReasonSymlink)
	}
	if _, _, err := ReadAttestation(linkDir); ReasonOf(err) != ReasonSymlink {
		t.Fatalf("ReadAttestation through symlinked journal dir succeeded or returned wrong reason: err=%v (reason=%q), want %q", err, ReasonOf(err), ReasonSymlink)
	}
}

func TestD053_Review_SharedWritableJournalDirRejected(t *testing.T) {
	// Regression proof from the D-053 adversarial review.
	dir := filepath.Join(t.TempDir(), "shared-journal")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o777 {
		t.Fatalf("expected dir perm 0777, got %o", info.Mode().Perm())
	}

	ch, at, _, _ := validPair(t)
	if err := WriteChallenge(dir, ch); ReasonOf(err) != ReasonBadMode {
		t.Fatalf("WriteChallenge in 0777 journal dir succeeded or returned wrong reason: err=%v (reason=%q), want %q", err, ReasonOf(err), ReasonBadMode)
	}
	if err := WriteAttestation(dir, at); ReasonOf(err) != ReasonBadMode {
		t.Fatalf("WriteAttestation in 0777 journal dir succeeded or returned wrong reason: err=%v (reason=%q), want %q", err, ReasonOf(err), ReasonBadMode)
	}
	if _, err := ObserveAttestation(dir); ReasonOf(err) != ReasonBadMode {
		t.Fatalf("ObserveAttestation in 0777 journal dir succeeded or returned wrong reason: err=%v (reason=%q), want %q", err, ReasonOf(err), ReasonBadMode)
	}
	if _, _, err := ReadAttestation(dir); ReasonOf(err) != ReasonBadMode {
		t.Fatalf("ReadAttestation in 0777 journal dir succeeded or returned wrong reason: err=%v (reason=%q), want %q", err, ReasonOf(err), ReasonBadMode)
	}
}

func TestD053_Review_ValidateRejectsFutureGeneratedAtClockSkew(t *testing.T) {
	// Regression proof from the D-053 adversarial review.
	ch, at, now, versions := validPair(t)
	// Updater evaluates at t = IssuedAt + 1s, but attestation claims GeneratedAt = IssuedAt + 40s.
	at.GeneratedAt = now.Add(39 * time.Second)
	if got := Validate(ch, at, now, versions); got == "" {
		t.Fatalf("Validate accepted attestation with GeneratedAt (%s) 39s in the future of updater now (%s)", at.GeneratedAt, now)
	}
}

func TestD053_Review_ValidateRejectsFreeFormReasonAndURLs(t *testing.T) {
	// Regression proof from the D-053 adversarial review.
	ch, at, now, versions := validPair(t)
	at.Checks.TunnelProbe.Reason = "https://member.example.com/query?dns=secret.internal&peer=100.64.0.99"
	if got := Validate(ch, at, now, versions); got != ReasonBadReason {
		t.Fatalf("Validate accepted URL/member data in check reason: got %q, want %q", got, ReasonBadReason)
	}
}

func TestD053_Review_Held_ValidateAndFileDefenses(t *testing.T) {
	ch, at, now, versions := validPair(t)
	if got := Validate(ch, at, now, versions); got != "" {
		t.Fatalf("valid pair rejected: %q", got)
	}

	// 1. Every check status of not_run or unknown across all three checks fails closed.
	for _, tc := range []struct {
		name   string
		mutate func(*Attestation)
		want   Reason
	}{
		{
			name:   "internet_only not_run",
			mutate: func(a *Attestation) { a.Checks.InternetOnly.Status = api.HealthCheckStatusNotRun },
			want:   ReasonNotRun,
		},
		{
			name:   "dns_policy not_run",
			mutate: func(a *Attestation) { a.Checks.DNSPolicy.Status = api.HealthCheckStatusNotRun },
			want:   ReasonNotRun,
		},
		{
			name:   "tunnel_probe not_run",
			mutate: func(a *Attestation) { a.Checks.TunnelProbe.Status = api.HealthCheckStatusNotRun },
			want:   ReasonNotRun,
		},
		{
			name:   "internet_only unknown status",
			mutate: func(a *Attestation) { a.Checks.InternetOnly.Status = "bypass" },
			want:   ReasonUnknownStatus,
		},
		{
			name:   "empty netbird version in manifest",
			mutate: func(a *Attestation) { a.NetbirdVersion = "" },
			want:   ReasonNetbirdVersion,
		},
		{
			name:   "netbird version suffix mismatch",
			mutate: func(a *Attestation) { a.NetbirdVersion = versions.Netbird + "-dirty" },
			want:   ReasonNetbirdVersion,
		},
		{
			name:   "invalid UTF-8 in reason",
			mutate: func(a *Attestation) { a.Checks.DNSPolicy.Reason = string([]byte{0xff, 0xfe}) },
			want:   ReasonBadReason,
		},
		{
			name:   "DEL control character in reason",
			mutate: func(a *Attestation) { a.Checks.DNSPolicy.Reason = "bad\x7freason" },
			want:   ReasonBadReason,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cp := at
			tc.mutate(&cp)
			if got := Validate(ch, cp, now, versions); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}

	// 2. File size bounds (> MaxFileBytes = 64 KiB), partial writes, and trailing JSON tokens are rejected.
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	attPath := filepath.Join(dir, AttestationFile)
	oversized := `{"txn_id":"` + strings.Repeat("a", MaxFileBytes) + `"}`
	if err := os.WriteFile(attPath, []byte(oversized), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ObserveAttestation(dir); ReasonOf(err) != ReasonTooLarge {
		t.Fatalf("ObserveAttestation on >64KiB file: got %v (%q), want %q", err, ReasonOf(err), ReasonTooLarge)
	}
	if _, _, err := ReadAttestation(dir); ReasonOf(err) != ReasonTooLarge {
		t.Fatalf("ReadAttestation on >64KiB file: got %v (%q), want %q", err, ReasonOf(err), ReasonTooLarge)
	}

	// Partial JSON write fails ReadAttestation.
	if err := os.WriteFile(attPath, []byte(`{"txn_id":"018f5b2a`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadAttestation(dir); err == nil {
		t.Fatal("partial JSON write was accepted")
	}

	// Trailing JSON token fails with ReasonMalformed.
	if err := WriteAttestation(dir, at); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(attPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(attPath, append(raw, []byte(`{"extra":1}`)...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadAttestation(dir); ReasonOf(err) != ReasonMalformed {
		t.Fatalf("ReadAttestation with trailing JSON token: got %v (%q), want %q", err, ReasonOf(err), ReasonMalformed)
	}
}
