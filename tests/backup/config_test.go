package backup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSECOPS05_RenderArchivingConfig_ProductionAndLabOverrides(t *testing.T) {
	t.Parallel()

	// Production default: archive_mode=on, archive_timeout=300s, stanza=fvpn.
	prodConf, err := RenderArchivingConfig(ArchivingConfig{})
	if err != nil {
		t.Fatalf("RenderArchivingConfig default failed: %v", err)
	}
	for _, want := range []string{
		"wal_level = replica",
		"archive_mode = on",
		"archive_timeout = 300s",
		"archive_command = 'pgbackrest --stanza=fvpn archive-push %p'",
		"max_wal_senders = 5",
	} {
		if !strings.Contains(prodConf, want) {
			t.Errorf("production archiving config missing %q:\n%s", want, prodConf)
		}
	}

	// Lab TEST ONLY override: shorter archive_timeout.
	labConf, err := RenderArchivingConfig(ArchivingConfig{
		Stanza:         "fvpn",
		ArchiveTimeout: "10s",
	})
	if err != nil {
		t.Fatalf("RenderArchivingConfig lab override failed: %v", err)
	}
	if !strings.Contains(labConf, "archive_timeout = 10s") {
		t.Errorf("lab archiving config missing 10s timeout:\n%s", labConf)
	}

	// Isolated restore mode: archive_mode=off so promoted restore timelines never push WAL.
	isoConf, err := RenderArchivingConfig(ArchivingConfig{
		IsolatedRestore: true,
	})
	if err != nil {
		t.Fatalf("RenderArchivingConfig isolated failed: %v", err)
	}
	if !strings.Contains(isoConf, "archive_mode = off") {
		t.Errorf("isolated restore config must set archive_mode = off:\n%s", isoConf)
	}

	// Reject malformed or injected values.
	badCases := []ArchivingConfig{
		{Stanza: "fvpn; rm -rf /"},
		{Stanza: "../etc"},
		{ArchiveTimeout: "0s"},
		{ArchiveTimeout: "300hours"},
		{ArchiveTimeout: "300s\narchive_mode = off"},
	}
	for _, tc := range badCases {
		if _, err := RenderArchivingConfig(tc); err == nil {
			t.Errorf("expected error for invalid ArchivingConfig %+v, got nil", tc)
		}
	}
}

func TestSECOPS05_RenderPgBackRestConfig_POSIXAndProductionS3(t *testing.T) {
	t.Parallel()

	// Production S3 profile with AES-256-CBC encryption and §16.4/§17.2 retention.
	prodOut, err := RenderPgBackRestConfig(PgBackRestConfig{
		Stanza:            "fvpn",
		RepoType:          "s3",
		RepoPath:          "/fvpn-central-pg17",
		S3Bucket:          "fvpn-prod-db-backups.example.com",
		S3Endpoint:        "s3.ap-northeast-1.amazonaws.com",
		S3Region:          "ap-northeast-1",
		CipherType:        "aes-256-cbc",
		RetentionFullType: "time",
		RetentionFull:     35,
		RetentionDiff:     14,
		PGPath:            "/var/lib/postgresql/17/main",
		PGPort:            5432,
		PGUser:            "postgres",
		PGDatabase:        "fvpn",
	})
	if err != nil {
		t.Fatalf("RenderPgBackRestConfig production S3 failed: %v", err)
	}
	for _, want := range []string{
		"repo1-type=s3",
		"repo1-s3-bucket=fvpn-prod-db-backups.example.com",
		"repo1-cipher-type=aes-256-cbc",
		"repo1-retention-full-type=time",
		"repo1-retention-full=35",
		"repo1-retention-diff=14",
		"[fvpn]",
		"pg1-path=/var/lib/postgresql/17/main",
	} {
		if !strings.Contains(prodOut, want) {
			t.Errorf("production S3 config missing %q:\n%s", want, prodOut)
		}
	}

	// Production S3 without encryption must fail closed.
	_, err = RenderPgBackRestConfig(PgBackRestConfig{
		Stanza:        "fvpn",
		RepoType:      "s3",
		RepoPath:      "/fvpn-central-pg17",
		S3Bucket:      "fvpn-prod-db-backups.example.com",
		S3Endpoint:    "s3.ap-northeast-1.amazonaws.com",
		S3Region:      "ap-northeast-1",
		CipherType:    "",
		RetentionFull: 35,
		RetentionDiff: 14,
		PGPath:        "/var/lib/postgresql/17/main",
		PGUser:        "postgres",
		PGDatabase:    "fvpn",
	})
	if err == nil {
		t.Fatal("expected unencrypted S3 repository config to be rejected")
	}

	// Lab POSIX profile.
	labOut, err := RenderPgBackRestConfig(PgBackRestConfig{
		Stanza:            "fvpn",
		RepoType:          "posix",
		RepoPath:          "/var/lib/pgbackrest",
		CipherType:        "aes-256-cbc",
		RetentionFullType: "count",
		RetentionFull:     7,
		RetentionDiff:     7,
		PGPath:            "/var/lib/postgresql/data",
		PGPort:            5432,
		PGUser:            "fvpn",
		PGDatabase:        "fvpn",
	})
	if err != nil {
		t.Fatalf("RenderPgBackRestConfig lab POSIX failed: %v", err)
	}
	if !strings.Contains(labOut, "repo1-type=posix") || !strings.Contains(labOut, "repo1-path=/var/lib/pgbackrest") {
		t.Errorf("lab POSIX config missing expected fields:\n%s", labOut)
	}
}

func TestAT30_ParseTargetTimestamp_UTCAndRejection(t *testing.T) {
	t.Parallel()

	valid := []struct {
		in     string
		wantPG string
		wantTS time.Time
	}{
		{
			in:     "2026-10-29T18:02:00Z",
			wantPG: "2026-10-29 18:02:00+00",
			wantTS: time.Date(2026, 10, 29, 18, 2, 0, 0, time.UTC),
		},
		{
			in:     "2026-10-29T18:02:00.123456Z",
			wantPG: "2026-10-29 18:02:00.123456+00",
			wantTS: time.Date(2026, 10, 29, 18, 2, 0, 123456000, time.UTC),
		},
		{
			in:     "2026-10-29 18:02:00.5+00",
			wantPG: "2026-10-29 18:02:00.5+00",
			wantTS: time.Date(2026, 10, 29, 18, 2, 0, 500000000, time.UTC),
		},
	}
	for _, tc := range valid {
		gotTS, gotPG, err := ParseTargetTimestamp(tc.in)
		if err != nil {
			t.Fatalf("ParseTargetTimestamp(%q) unexpected error: %v", tc.in, err)
		}
		if !gotTS.Equal(tc.wantTS) {
			t.Errorf("ParseTargetTimestamp(%q) time = %v, want %v", tc.in, gotTS, tc.wantTS)
		}
		if gotPG != tc.wantPG {
			t.Errorf("ParseTargetTimestamp(%q) pg = %q, want %q", tc.in, gotPG, tc.wantPG)
		}
	}

	invalid := []string{
		"",
		"2026-10-29T18:02:00-04:00", // non-UTC offset prohibited (§4)
		"2026-10-29T18:02:00+08:00",
		"2026-02-30T18:02:00Z", // invalid calendar date
		"2026-10-29",
		"2026-10-29T18:02:00Z; rm -rf /",
		"2026-10-29T18:02:00Z' OR '1'='1",
	}
	for _, bad := range invalid {
		if _, _, err := ParseTargetTimestamp(bad); err == nil {
			t.Errorf("ParseTargetTimestamp(%q) succeeded, expected error", bad)
		}
	}
}

func TestAT30_ParseTargetLSN_ValidAndInvalid(t *testing.T) {
	t.Parallel()

	lsn1, canon1, err := ParseTargetLSN("0/204d2e8")
	if err != nil {
		t.Fatalf("ParseTargetLSN failed: %v", err)
	}
	if canon1 != "0/0204D2E8" {
		t.Errorf("canonical LSN = %q, want 0/0204D2E8", canon1)
	}
	lsn2, canon2, err := ParseTargetLSN("1A/FE000010")
	if err != nil {
		t.Fatalf("ParseTargetLSN failed: %v", err)
	}
	if canon2 != "1A/FE000010" {
		t.Errorf("canonical LSN = %q, want 1A/FE000010", canon2)
	}
	if lsn1 >= lsn2 {
		t.Errorf("expected LSN1 (%x) < LSN2 (%x)", lsn1, lsn2)
	}

	invalid := []string{
		"",
		"0/0",
		"0/00000000",
		"204D2E8",
		"0/123456789",
		"123456789/1",
		"0/204G2E8",
		"0/204D2E8; drop database fvpn",
	}
	for _, bad := range invalid {
		if _, _, err := ParseTargetLSN(bad); err == nil {
			t.Errorf("ParseTargetLSN(%q) succeeded, expected error", bad)
		}
	}
}

func TestAT30_RestoreDestinationGuard_RefusesLivePrimary(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	primaryDir := filepath.Join(tmp, "primary-pgdata")
	if err := os.MkdirAll(primaryDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(primaryDir, "postmaster.pid"), []byte("1234\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 1. Go helper refuses directory equal to primaryDir or containing postmaster.pid.
	if err := ValidateRestoreDestination(primaryDir, primaryDir); err == nil {
		t.Fatal("expected ValidateRestoreDestination to refuse primaryDir")
	}
	if err := ValidateRestoreDestination(primaryDir, ""); err == nil {
		t.Fatal("expected ValidateRestoreDestination to refuse directory with postmaster.pid")
	}

	// 2. Empty fresh directory is accepted.
	cleanDir := filepath.Join(tmp, "clean-restore-target")
	if err := ValidateRestoreDestination(cleanDir, primaryDir); err != nil {
		t.Fatalf("ValidateRestoreDestination rejected clean target dir: %v", err)
	}

	// 3. restore-pitr.sh refuses a target directory containing postmaster.pid.
	scriptPath := filepath.Join("..", "..", "deploy", "backup", "restore-pitr.sh")
	cmd := exec.Command("bash", scriptPath,
		"--target-time", "2026-10-29T18:02:00Z",
		"--target-dir", primaryDir,
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected restore-pitr.sh to refuse directory with postmaster.pid, but it exited 0: %s", string(out))
	}
	if !strings.Contains(string(out), "postmaster.pid") {
		t.Errorf("expected postmaster.pid refusal message, got: %s", string(out))
	}

	// 4. restore-pitr.sh refuses when --isolated-container matches --primary-container.
	cmd2 := exec.Command("bash", scriptPath,
		"--target-time", "2026-10-29T18:02:00Z",
		"--isolated-container", "fvpn-w4-ops-backups-primary",
		"--primary-container", "fvpn-w4-ops-backups-primary",
		"--target-volume", "fvpn-w4-ops-backups-vol",
		"--repo-volume", "fvpn-w4-ops-backups-repo",
	)
	out2, err := cmd2.CombinedOutput()
	if err == nil {
		t.Fatalf("expected restore-pitr.sh to refuse live primary container, but it exited 0: %s", string(out2))
	}
	if !strings.Contains(string(out2), "refusing to restore over live primary container") {
		t.Errorf("expected live primary container refusal message, got: %s", string(out2))
	}
}

func TestSECOPS05_StaticAssetsAndChecklistContract(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Join("..", "..")
	requiredFiles := []string{
		"deploy/backup/postgres/Dockerfile",
		"deploy/backup/postgres/fvpn-archiving.conf",
		"deploy/backup/postgres/fvpn-postgres-entrypoint.sh",
		"deploy/backup/postgres/10-pgbackrest-stanza.sh",
		"deploy/backup/pgbackrest.conf.example",
		"deploy/backup/backup.sh",
		"deploy/backup/restore-pitr.sh",
		"deploy/backup/verify.sh",
		"deploy/backup/post-restore-checklist.sql",
		"deploy/backup/systemd/fvpn-backup-full.service",
		"deploy/backup/systemd/fvpn-backup-full.timer",
		"deploy/backup/systemd/fvpn-backup-diff.service",
		"deploy/backup/systemd/fvpn-backup-diff.timer",
		"deploy/backup/systemd/fvpn-backup-verify.service",
		"deploy/backup/systemd/fvpn-backup-verify.timer",
	}
	for _, rel := range requiredFiles {
		p := filepath.Join(repoRoot, rel)
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("missing required backup deliverable %s: %v", rel, err)
		}
		if info.Size() == 0 {
			t.Fatalf("backup deliverable %s is empty", rel)
		}
	}

	// Verify post-restore-checklist.sql covers all required §17.3/§17.4 queries and is read-only.
	sqlBytes, err := os.ReadFile(filepath.Join(repoRoot, "deploy/backup/post-restore-checklist.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sqlText := string(sqlBytes)
	for _, requiredFragment := range []string{
		"SET SESSION CHARACTERISTICS AS TRANSACTION READ ONLY;",
		"usage_ledger_state",
		"usage_checkpoints",
		"highest_contiguous_seq",
		"entitlements",
		"effective_service_end",
		"publication_intent",
		"update_batches",
		"update_jobs",
		"revoked_at",
		"admin_suspended",
		"fvpn-usage-admin",
		"begin-recovery",
	} {
		if !strings.Contains(sqlText, requiredFragment) {
			t.Errorf("post-restore-checklist.sql missing required fragment %q", requiredFragment)
		}
	}
}
