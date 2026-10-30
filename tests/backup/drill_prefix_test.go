package backup

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestAT30_DeriveDrillPrefix(t *testing.T) {
	t.Parallel()

	t.Run("random suffix is 8 lowercase hex", func(t *testing.T) {
		t.Parallel()
		got, err := deriveDrillPrefix("", bytes.NewReader([]byte{0x0a, 0x1b, 0x2c, 0x3d}))
		if err != nil {
			t.Fatal(err)
		}
		const want = "fvpn-w4-ops-backups-0a1b2c3d"
		if got != want {
			t.Fatalf("prefix = %q, want %q", got, want)
		}
		names, err := drillNamesForPrefix(got)
		if err != nil {
			t.Fatal(err)
		}
		if names.Image == "fvpn-w4-ops-backups-postgres:17" {
			t.Fatal("image tag must be per-run, not the legacy shared tag")
		}
	})

	t.Run("env prefix is kept and entropy is not read", func(t *testing.T) {
		t.Parallel()
		const want = "fvpn-w4-ops-backups-gha17000000000-1"
		got, err := deriveDrillPrefix(want, errEntropy{})
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("prefix = %q, want %q", got, want)
		}
	})

	t.Run("short entropy fails", func(t *testing.T) {
		t.Parallel()
		if _, err := deriveDrillPrefix("", bytes.NewReader([]byte{0x01})); err == nil {
			t.Fatal("expected error when fewer than 4 random bytes are available")
		}
	})

	t.Run("nil entropy fails when unset", func(t *testing.T) {
		t.Parallel()
		if _, err := deriveDrillPrefix("", nil); err == nil {
			t.Fatal("expected error when entropy source is nil")
		}
	})

	rejected := []string{
		"fvpn-w4-ops-backups",
		"fvpn-w4-ops-backups-",
		"fvpn-pg",
		"fvpn-pg-primary",
		"fvpn-w4-ops-backups-ABC",
		"fvpn-w4-ops-backups-abc_def",
		"fvpn-w4-ops-backups-abc--def",
		" fvpn-w4-ops-backups-abc",
		"../fvpn-w4-ops-backups-abc",
		"fvpn-w4-ops-backups-" + strings.Repeat("a", 80),
	}
	for _, prefix := range rejected {
		t.Run("reject "+prefix, func(t *testing.T) {
			t.Parallel()
			if _, err := deriveDrillPrefix(prefix, errEntropy{}); err == nil {
				t.Fatalf("accepted prefix %q", prefix)
			}
		})
	}
}

func TestAT30_DrillNamesStayInsidePrefix(t *testing.T) {
	t.Parallel()

	names, err := drillNamesFromEnv("fvpn-w4-ops-backups-01abcdef", errEntropy{})
	if err != nil {
		t.Fatal(err)
	}
	want := drillNames{
		Prefix:           "fvpn-w4-ops-backups-01abcdef",
		Image:            "fvpn-w4-ops-backups-01abcdef-postgres:17",
		PrimaryContainer: "fvpn-w4-ops-backups-01abcdef-primary",
		PrimaryVolume:    "fvpn-w4-ops-backups-01abcdef-primary-pgdata",
		RepoVolume:       "fvpn-w4-ops-backups-01abcdef-repo",
		RestoreContainer: "fvpn-w4-ops-backups-01abcdef-restore",
		RestoreVolume:    "fvpn-w4-ops-backups-01abcdef-restore-pgdata",
		VerifyPrefix:     "fvpn-w4-ops-backups-01abcdef-verify",
		Network:          "fvpn-w4-ops-backups-01abcdef-net",
	}
	if names != want {
		t.Fatalf("names = %+v, want %+v", names, want)
	}

	owned := []string{
		names.Prefix,
		names.Image,
		names.PrimaryContainer,
		names.PrimaryVolume,
		names.RepoVolume,
		names.RestoreContainer,
		names.RestoreVolume,
		names.Network,
		names.VerifyPrefix + "-pg",
		names.VerifyPrefix + "-pgdata",
		names.RestoreContainer + "-init",
		"/" + names.PrimaryContainer,
	}
	for _, name := range owned {
		if !ownedByDrillPrefix(name, names.Prefix) {
			t.Errorf("expected %q to be owned by %s", name, names.Prefix)
		}
	}

	// Neighbors, the legacy shared image, the bare family, and other tasks.
	outsiders := []string{
		"fvpn-pg",
		"fvpn-pg-primary",
		"fvpn-w4-ops-backups",
		"fvpn-w4-ops-backups-primary",
		"fvpn-w4-ops-backups-postgres:17",
		"fvpn-w4-ops-backups-01abcdee-primary",
		"fvpn-w4-ops-backups-01abcdefextra-primary",
		"xfvpn-w4-ops-backups-01abcdef-primary",
		"bridge",
		"none",
		"",
		"-fvpn-w4-ops-backups-01abcdef",
	}
	for _, name := range outsiders {
		if ownedByDrillPrefix(name, names.Prefix) {
			t.Errorf("prefix %s must not own %q", names.Prefix, name)
		}
	}
	if ownedByDrillPrefix(names.PrimaryContainer, "fvpn-w4-ops-backups") {
		t.Fatal("bare family prefix must not own drill objects")
	}
	if ownedByDrillPrefix(names.PrimaryContainer, "fvpn-pg") {
		t.Fatal("fvpn-pg must not own drill objects")
	}
}

type errEntropy struct{}

func (errEntropy) Read([]byte) (int, error) {
	return 0, errors.New("entropy must not be read")
}
