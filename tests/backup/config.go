// Package backup validates configuration rendering, PITR target parsing,
// and live-primary safety guards for Freedom VPN PostgreSQL 17 backups (§15.4, §17.2, AT30).
package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	stanzaPattern         = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	archiveTimeoutPattern = regexp.MustCompile(`^[1-9][0-9]*(ms|s|min)$`)
	safeValuePattern      = regexp.MustCompile(`^[a-zA-Z0-9_./:@-]+$`)
	utcTimestampPattern   = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2})[T ]([0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]{1,6})?)(Z|\+00(?::00)?)$`)
	lsnPattern            = regexp.MustCompile(`^([0-9A-Fa-f]{1,8})/([0-9A-Fa-f]{1,8})$`)
)

// ArchivingConfig holds settings for /etc/postgresql/conf.d/50-fvpn-archiving.conf (§3.1, §17.2).
type ArchivingConfig struct {
	Stanza          string
	ArchiveTimeout  string
	IsolatedRestore bool
}

// RenderArchivingConfig validates and renders the PostgreSQL WAL archiving configuration fragment.
func RenderArchivingConfig(cfg ArchivingConfig) (string, error) {
	stanza := cfg.Stanza
	if stanza == "" {
		stanza = "fvpn"
	}
	if !stanzaPattern.MatchString(stanza) {
		return "", fmt.Errorf("backup: invalid stanza %q", stanza)
	}

	timeout := cfg.ArchiveTimeout
	if timeout == "" {
		timeout = "300s"
	}
	if !archiveTimeoutPattern.MatchString(timeout) {
		return "", fmt.Errorf("backup: invalid archive_timeout %q (expected positive integer with ms, s, or min)", timeout)
	}

	if cfg.IsolatedRestore {
		return strings.Join([]string{
			"# Isolated PITR verification instance (§17.2, AT30).",
			"wal_level = replica",
			"archive_mode = off",
			"archive_timeout = 0",
			"archive_command = ''",
			"max_wal_senders = 5",
			"",
		}, "\n"), nil
	}

	return strings.Join([]string{
		"# Freedom VPN PostgreSQL 17 WAL Archiving Configuration (§3.1, §15.4, §17.2)",
		"wal_level = replica",
		"archive_mode = on",
		fmt.Sprintf("archive_timeout = %s", timeout),
		fmt.Sprintf("archive_command = 'pgbackrest --stanza=%s archive-push %%p'", stanza),
		"max_wal_senders = 5",
		"",
	}, "\n"), nil
}

// PgBackRestConfig holds parameters for rendering pgbackrest.conf (§17.2).
type PgBackRestConfig struct {
	Stanza            string
	RepoType          string // "posix" (lab) or "s3" (production)
	RepoPath          string
	S3Bucket          string
	S3Endpoint        string
	S3Region          string
	CipherType        string // "aes-256-cbc" or "" (posix lab only)
	RetentionFullType string // "time" or "count"
	RetentionFull     int
	RetentionDiff     int
	PGPath            string
	PGPort            int
	PGUser            string
	PGDatabase        string
}

// RenderPgBackRestConfig validates and renders a pgBackRest configuration file.
func RenderPgBackRestConfig(cfg PgBackRestConfig) (string, error) {
	if cfg.Stanza == "" {
		cfg.Stanza = "fvpn"
	}
	if !stanzaPattern.MatchString(cfg.Stanza) {
		return "", fmt.Errorf("backup: invalid stanza %q", cfg.Stanza)
	}
	if cfg.RepoPath == "" || !strings.HasPrefix(cfg.RepoPath, "/") || !safeValuePattern.MatchString(cfg.RepoPath) {
		return "", fmt.Errorf("backup: invalid repo1-path %q", cfg.RepoPath)
	}
	if cfg.RetentionFull <= 0 || cfg.RetentionDiff <= 0 {
		return "", errors.New("backup: retention counts/days must be positive")
	}
	if cfg.RetentionFullType == "" {
		cfg.RetentionFullType = "time"
	}
	if cfg.RetentionFullType != "time" && cfg.RetentionFullType != "count" {
		return "", fmt.Errorf("backup: invalid repo1-retention-full-type %q", cfg.RetentionFullType)
	}
	if cfg.PGPath == "" || !strings.HasPrefix(cfg.PGPath, "/") || !safeValuePattern.MatchString(cfg.PGPath) {
		return "", fmt.Errorf("backup: invalid pg1-path %q", cfg.PGPath)
	}
	if cfg.PGPort <= 0 || cfg.PGPort > 65535 {
		cfg.PGPort = 5432
	}
	if cfg.PGUser == "" || !stanzaPattern.MatchString(cfg.PGUser) {
		return "", fmt.Errorf("backup: invalid pg1-user %q", cfg.PGUser)
	}
	if cfg.PGDatabase == "" || !stanzaPattern.MatchString(cfg.PGDatabase) {
		return "", fmt.Errorf("backup: invalid pg1-database %q", cfg.PGDatabase)
	}

	var lines []string
	lines = append(lines, "[global]")

	switch cfg.RepoType {
	case "s3":
		if cfg.S3Bucket == "" || !safeValuePattern.MatchString(cfg.S3Bucket) {
			return "", fmt.Errorf("backup: s3 repo requires valid bucket, got %q", cfg.S3Bucket)
		}
		if cfg.S3Endpoint == "" || !safeValuePattern.MatchString(cfg.S3Endpoint) {
			return "", fmt.Errorf("backup: s3 repo requires valid endpoint, got %q", cfg.S3Endpoint)
		}
		if cfg.S3Region == "" || !stanzaPattern.MatchString(cfg.S3Region) {
			return "", fmt.Errorf("backup: s3 repo requires valid region, got %q", cfg.S3Region)
		}
		if cfg.CipherType != "aes-256-cbc" {
			return "", fmt.Errorf("backup: production s3 repo requires cipher-type aes-256-cbc (§16.4, §17.2), got %q", cfg.CipherType)
		}
		lines = append(lines,
			"repo1-type=s3",
			fmt.Sprintf("repo1-path=%s", cfg.RepoPath),
			fmt.Sprintf("repo1-s3-bucket=%s", cfg.S3Bucket),
			fmt.Sprintf("repo1-s3-endpoint=%s", cfg.S3Endpoint),
			fmt.Sprintf("repo1-s3-region=%s", cfg.S3Region),
			"repo1-s3-uri-style=host",
			"repo1-s3-verify-tls=y",
			"repo1-s3-key-type=auto",
			"repo1-cipher-type=aes-256-cbc",
		)
	case "posix":
		lines = append(lines,
			"repo1-type=posix",
			fmt.Sprintf("repo1-path=%s", cfg.RepoPath),
		)
		if cfg.CipherType != "" {
			if cfg.CipherType != "aes-256-cbc" {
				return "", fmt.Errorf("backup: unsupported cipher-type %q", cfg.CipherType)
			}
			lines = append(lines, "repo1-cipher-type=aes-256-cbc")
		}
	default:
		return "", fmt.Errorf("backup: unsupported repo1-type %q", cfg.RepoType)
	}

	lines = append(lines,
		fmt.Sprintf("repo1-retention-full-type=%s", cfg.RetentionFullType),
		fmt.Sprintf("repo1-retention-full=%d", cfg.RetentionFull),
		fmt.Sprintf("repo1-retention-diff=%d", cfg.RetentionDiff),
		"compress-type=zst",
		"process-max=2",
		"start-fast=y",
		"log-path=/var/log/pgbackrest",
		"spool-path=/var/spool/pgbackrest",
		"",
		fmt.Sprintf("[%s]", cfg.Stanza),
		fmt.Sprintf("pg1-path=%s", cfg.PGPath),
		fmt.Sprintf("pg1-port=%d", cfg.PGPort),
		fmt.Sprintf("pg1-user=%s", cfg.PGUser),
		fmt.Sprintf("pg1-database=%s", cfg.PGDatabase),
		"pg1-socket-path=/var/run/postgresql",
		"",
	)
	return strings.Join(lines, "\n"), nil
}

// ParseTargetTimestamp validates a UTC recovery target timestamp and returns
// both the parsed UTC time.Time and the canonical pgBackRest target string
// ("YYYY-MM-DD HH:MM:SS[.ffffff]+00").
func ParseTargetTimestamp(raw string) (time.Time, string, error) {
	m := utcTimestampPattern.FindStringSubmatch(raw)
	if m == nil {
		return time.Time{}, "", fmt.Errorf("backup: invalid UTC timestamp %q (expected YYYY-MM-DDTHH:MM:SS[.ffffff]Z)", raw)
	}
	datePart := m[1]
	timePart := m[2]
	rfc3339Candidate := datePart + "T" + timePart + "Z"
	parsed, err := time.Parse(time.RFC3339Nano, rfc3339Candidate)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("backup: invalid calendar timestamp %q: %w", raw, err)
	}
	pgTarget := fmt.Sprintf("%s %s+00", datePart, timePart)
	return parsed.UTC(), pgTarget, nil
}

// ParseTargetLSN validates a PostgreSQL WAL LSN ("X/YYYYYYYY") and returns
// its 64-bit numeric LSN and canonical uppercase representation.
func ParseTargetLSN(raw string) (uint64, string, error) {
	m := lsnPattern.FindStringSubmatch(raw)
	if m == nil {
		return 0, "", fmt.Errorf("backup: invalid WAL LSN %q (expected X/YYYYYYYY)", raw)
	}
	hi, err := strconv.ParseUint(m[1], 16, 32)
	if err != nil {
		return 0, "", fmt.Errorf("backup: invalid WAL LSN high segment %q: %w", m[1], err)
	}
	lo, err := strconv.ParseUint(m[2], 16, 32)
	if err != nil {
		return 0, "", fmt.Errorf("backup: invalid WAL LSN low segment %q: %w", m[2], err)
	}
	val := (hi << 32) | lo
	if val == 0 {
		return 0, "", errors.New("backup: WAL LSN must be non-zero")
	}
	canonical := fmt.Sprintf("%X/%08X", hi, lo)
	return val, canonical, nil
}

// ValidateRestoreDestination verifies that a local restore target directory is
// not the live primary directory, does not contain postmaster.pid, and is empty.
func ValidateRestoreDestination(targetDir, primaryDir string) error {
	if strings.TrimSpace(targetDir) == "" {
		return errors.New("backup: target directory is required")
	}
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("backup: resolve target directory: %w", err)
	}
	if strings.TrimSpace(primaryDir) != "" {
		absPrimary, err := filepath.Abs(primaryDir)
		if err != nil {
			return fmt.Errorf("backup: resolve primary directory: %w", err)
		}
		if absTarget == absPrimary {
			return fmt.Errorf("backup: refusing to restore over live primary directory %q", targetDir)
		}
	}
	if _, err := os.Stat(filepath.Join(absTarget, "postmaster.pid")); err == nil {
		return fmt.Errorf("backup: refusing to restore over active PostgreSQL directory %q (postmaster.pid present)", targetDir)
	}
	entries, err := os.ReadDir(absTarget)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("backup: inspect target directory: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("backup: target directory %q is not empty", targetDir)
	}
	return nil
}
