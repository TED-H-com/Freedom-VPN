//go:build integration

package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	drillPrefix           = "fvpn-w4-ops-backups"
	drillImage            = "fvpn-w4-ops-backups-postgres:17"
	drillPrimaryContainer = "fvpn-w4-ops-backups-primary"
	drillPrimaryVolume    = "fvpn-w4-ops-backups-primary-pgdata"
	drillRepoVolume       = "fvpn-w4-ops-backups-repo"
	drillRestoreContainer = "fvpn-w4-ops-backups-restore"
	drillRestoreVolume    = "fvpn-w4-ops-backups-restore-pgdata"
	drillVerifyPrefix     = "fvpn-w4-ops-backups-verify"
)

// BackupFileDigest records the path and SHA-256 digest of a file in the pgBackRest backup repository.
type BackupFileDigest struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// DrillRecord represents the JSON evidence written to docs/ops/backups/drill-<date>.json (§17.2, §17.5, AT30).
type DrillRecord struct {
	SchemaVersion         int                `json:"schema_version"`
	SpecReferences        []string           `json:"spec_references"`
	Requirements          []string           `json:"requirements"`
	DrillDate             string             `json:"drill_date"`
	ExecutedAtUTC         string             `json:"executed_at_utc"`
	PgBackRestVersion     string             `json:"pgbackrest_version"`
	PostgresVersion       string             `json:"postgres_version"`
	BaseImageDigest       string             `json:"base_image_digest"`
	BuiltImageID          string             `json:"built_image_id"`
	ArchiveMode           string             `json:"archive_mode"`
	ProductionTimeout     string             `json:"production_archive_timeout"`
	LabOverrideTimeout    string             `json:"lab_archive_timeout_test_only"`
	BackupLabel           string             `json:"backup_label"`
	BackupType            string             `json:"backup_type"`
	BackupStartLSN        string             `json:"backup_start_lsn"`
	BackupStopLSN         string             `json:"backup_stop_lsn"`
	TargetTimestampUTC    string             `json:"target_timestamp_utc"`
	TargetLSN             string             `json:"target_lsn"`
	PrimaryMarkers        []string           `json:"primary_markers"`
	RestoredMarkers       []string           `json:"restored_markers"`
	MarkerAPresent        bool               `json:"marker_a_present"`
	MarkerBAbsent         bool               `json:"marker_b_absent"`
	LivePrimaryProtected  bool               `json:"live_primary_overwrite_refused"`
	IsolatedNetworkMode   string             `json:"isolated_network_mode"`
	IsolatedPublishedPort int                `json:"isolated_published_ports"`
	MigrationsApplied     int                `json:"migrations_applied"`
	MigrationsPending     int                `json:"migrations_pending"`
	ChecklistVerified     bool               `json:"post_restore_checklist_verified"`
	VerifyScriptPassed    bool               `json:"verify_script_passed"`
	RPOTargetSeconds      int64              `json:"rpo_target_seconds"`
	RTOTargetSeconds      int64              `json:"rto_target_seconds"`
	MeasuredArchiveLagMs  int64              `json:"measured_rpo_archive_lag_ms"`
	MeasuredRestoreRTOMs  int64              `json:"measured_rto_restore_ms"`
	MeasuredVerifyRTOMs   int64              `json:"measured_rto_verify_ms"`
	MeasuredTotalDrillMs  int64              `json:"measured_total_drill_ms"`
	BackupSetDigests      []BackupFileDigest `json:"backup_set_digests"`
	Status                string             `json:"status"`
}

func TestAT30_BackupPITRDrill(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	if err := exec.CommandContext(ctx, "docker", "info").Run(); err != nil {
		t.Skipf("skipping backup PITR drill: docker daemon is not reachable: %v", err)
	}

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	// Ensure all fvpn-w4-ops-backups-* Docker objects are cleaned before and after the drill.
	cleanDrillDockerResources(ctx, t, false)
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cleanCancel()
		cleanDrillDockerResources(cleanCtx, t, true)
	})

	drillStart := time.Now().UTC()
	drillDate := os.Getenv("FVPN_BACKUP_DRILL_DATE")
	if drillDate == "" {
		drillDate = time.Now().Format("2006-01-02")
	}

	// 1. Build the drop-in PostgreSQL 17 + pgBackRest image.
	t.Log("building deploy/backup/postgres/Dockerfile as", drillImage)
	runCmd(ctx, t, repoRoot, nil, "docker", "build", "-t", drillImage, filepath.Join(repoRoot, "deploy", "backup", "postgres"))
	builtImageID := strings.TrimSpace(runCmd(ctx, t, repoRoot, nil, "docker", "image", "inspect", "--format", "{{.Id}}", drillImage))

	// 2. Build fvpn-migrate for linux/amd64 and linux/arm64 (§3 cross-compile check).
	workTmp := t.TempDir()
	migrateBinAmd64 := filepath.Join(workTmp, "fvpn-migrate-amd64")
	migrateBinArm64 := filepath.Join(workTmp, "fvpn-migrate-arm64")
	runCmd(ctx, t, repoRoot, []string{"GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "GOTOOLCHAIN=local"},
		"go", "build", "-trimpath", "-o", migrateBinAmd64, "./internal/db/cmd/fvpn-migrate")
	runCmd(ctx, t, repoRoot, []string{"GOOS=linux", "GOARCH=arm64", "CGO_ENABLED=0", "GOTOOLCHAIN=local"},
		"go", "build", "-trimpath", "-o", migrateBinArm64, "./internal/db/cmd/fvpn-migrate")

	// 3. Create volumes and start the primary container with archiving enabled.
	runCmd(ctx, t, repoRoot, nil, "docker", "volume", "create", drillPrimaryVolume)
	runCmd(ctx, t, repoRoot, nil, "docker", "volume", "create", drillRepoVolume)

	runCmd(ctx, t, repoRoot, nil, "docker", "run", "-d",
		"--name", drillPrimaryContainer,
		"--network", "none",
		"-e", "POSTGRES_USER=fvpn",
		"-e", "POSTGRES_DB=fvpn",
		"-e", "POSTGRES_HOST_AUTH_METHOD=trust",
		// TEST ONLY lab override for archive_timeout; production default in fvpn-archiving.conf is 300s.
		"-e", "FVPN_PG_ARCHIVE_TIMEOUT=10s",
		"-v", drillPrimaryVolume+":/var/lib/postgresql/data",
		"-v", drillRepoVolume+":/var/lib/pgbackrest",
		drillImage,
	)

	waitForPostgresReady(ctx, t, drillPrimaryContainer, "fvpn", "fvpn")

	pgVersion := strings.TrimSpace(execPsql(ctx, t, drillPrimaryContainer, "SELECT version();"))
	pgBackRestVer := strings.TrimSpace(runCmd(ctx, t, repoRoot, nil, "docker", "exec", "--user", "postgres", drillPrimaryContainer, "pgbackrest", "version"))
	archiveMode := strings.TrimSpace(execPsql(ctx, t, drillPrimaryContainer, "SHOW archive_mode;"))
	if archiveMode != "on" {
		t.Fatalf("expected primary archive_mode=on, got %q", archiveMode)
	}
	walLevel := strings.TrimSpace(execPsql(ctx, t, drillPrimaryContainer, "SHOW wal_level;"))
	if walLevel != "replica" {
		t.Fatalf("expected primary wal_level=replica, got %q", walLevel)
	}

	// 4. Apply all product forward migrations using fvpn-migrate up.
	runCmd(ctx, t, repoRoot, nil, "docker", "cp", migrateBinAmd64, drillPrimaryContainer+":/usr/local/bin/fvpn-migrate")
	migrateUpOut := runCmd(ctx, t, repoRoot, nil, "docker", "exec",
		"-e", "FVPN_DATABASE_URL=postgres://fvpn@127.0.0.1:5432/fvpn?sslmode=disable",
		drillPrimaryContainer, "/usr/local/bin/fvpn-migrate", "up")
	if !strings.Contains(migrateUpOut, "migrations applied") {
		t.Fatalf("unexpected fvpn-migrate up output:\n%s", migrateUpOut)
	}

	// 5. Create the drill-only table _backup_drill (never touching product tables per D-050) and write Marker A.
	execPsql(ctx, t, drillPrimaryContainer, `
		CREATE TABLE _backup_drill (
			marker TEXT PRIMARY KEY,
			recorded_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
			note TEXT NOT NULL
		);
		INSERT INTO _backup_drill (marker, note) VALUES ('A', 'pre-backup marker A');
	`)

	// 6. Initialize stanza, verify check, and take a physical full base backup using deploy/backup/backup.sh.
	backupScript := filepath.Join(repoRoot, "deploy", "backup", "backup.sh")
	runCmd(ctx, t, repoRoot, nil, "bash", backupScript, "--container", drillPrimaryContainer, "stanza-create")
	fullBackupInfoJSON := runCmd(ctx, t, repoRoot, nil, "bash", backupScript, "--container", drillPrimaryContainer, "full")

	backupLabel, backupStartLSN, backupStopLSN, backupStopUnix := parsePgBackRestInfo(t, fullBackupInfoJSON)

	// 7. Wait until wall-clock Unix seconds advance past pgBackRest's integer
	// backup.timestamp.stop, then record targetTimestamp and targetLSN,
	// write Marker B after that timestamp, and force a WAL switch.
	for time.Now().UTC().Unix() <= backupStopUnix {
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	targetTimestamp := strings.TrimSpace(execPsql(ctx, t, drillPrimaryContainer,
		`SELECT to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');`))
	targetLSN := strings.TrimSpace(execPsql(ctx, t, drillPrimaryContainer,
		`SELECT pg_current_wal_insert_lsn()::text;`))

	if _, _, err := ParseTargetTimestamp(targetTimestamp); err != nil {
		t.Fatalf("recorded targetTimestamp %q failed validation: %v", targetTimestamp, err)
	}
	if _, _, err := ParseTargetLSN(targetLSN); err != nil {
		t.Fatalf("recorded targetLSN %q failed validation: %v", targetLSN, err)
	}

	time.Sleep(200 * time.Millisecond)
	execPsql(ctx, t, drillPrimaryContainer, `
		INSERT INTO _backup_drill (marker, note) VALUES ('B', 'post-target marker B');
	`)

	primaryMarkersCSV := strings.TrimSpace(execPsql(ctx, t, drillPrimaryContainer,
		`SELECT string_agg(marker, ',' ORDER BY marker) FROM _backup_drill;`))
	if primaryMarkersCSV != "A,B" {
		t.Fatalf("expected primary _backup_drill markers 'A,B', got %q", primaryMarkersCSV)
	}

	// Force WAL switch and verify archiving via backup.sh wal-switch.
	walSwitchStart := time.Now()
	runCmd(ctx, t, repoRoot, nil, "bash", backupScript, "--container", drillPrimaryContainer, "wal-switch")
	rpoArchiveLagMs := time.Since(walSwitchStart).Milliseconds()
	if rpoArchiveLagMs < 1 {
		rpoArchiveLagMs = 1
	}

	// 8. Failure case: verify restore-pitr.sh refuses to overwrite the live primary container.
	restoreScript := filepath.Join(repoRoot, "deploy", "backup", "restore-pitr.sh")
	refuseCmd := exec.CommandContext(ctx, "bash", restoreScript,
		"--target-time", targetTimestamp,
		"--isolated-container", drillPrimaryContainer,
		"--primary-container", drillPrimaryContainer,
		"--target-volume", drillPrimaryVolume,
		"--repo-volume", drillRepoVolume,
		"--image", drillImage,
	)
	refuseOut, refuseErr := refuseCmd.CombinedOutput()
	if refuseErr == nil {
		t.Fatalf("expected restore-pitr.sh to refuse live primary container, but it succeeded:\n%s", string(refuseOut))
	}
	// Confirm primary still has both A and B untouched.
	afterRefuseCSV := strings.TrimSpace(execPsql(ctx, t, drillPrimaryContainer,
		`SELECT string_agg(marker, ',' ORDER BY marker) FROM _backup_drill;`))
	if afterRefuseCSV != "A,B" {
		t.Fatalf("primary markers altered after refused restore: got %q", afterRefuseCSV)
	}

	// 9. Execute PITR restore to targetTimestamp into an isolated container (--network none, no host ports).
	rtoStart := time.Now()
	runCmd(ctx, t, repoRoot, nil, "bash", restoreScript,
		"--stanza", "fvpn",
		"--target-time", targetTimestamp,
		"--isolated-container", drillRestoreContainer,
		"--target-volume", drillRestoreVolume,
		"--repo-volume", drillRepoVolume,
		"--primary-container", drillPrimaryContainer,
		"--image", drillImage,
	)
	rtoRestoreMs := time.Since(rtoStart).Milliseconds()

	// 10. Verify isolation invariants (--network none and zero published host ports).
	netMode := strings.TrimSpace(runCmd(ctx, t, repoRoot, nil, "docker", "inspect", "--format", "{{.HostConfig.NetworkMode}}", drillRestoreContainer))
	if netMode != "none" {
		t.Fatalf("expected restored container NetworkMode=none, got %q", netMode)
	}
	portCountStr := strings.TrimSpace(runCmd(ctx, t, repoRoot, nil, "docker", "inspect", "--format", "{{len .NetworkSettings.Ports}}", drillRestoreContainer))
	portCount, err := strconv.Atoi(portCountStr)
	if err != nil || portCount != 0 {
		t.Fatalf("expected 0 published ports on restored container, got %q (err=%v)", portCountStr, err)
	}

	// 11. Assert Marker A is present and Marker B is absent in the PITR-restored database.
	restoredMarkersCSV := strings.TrimSpace(execPsql(ctx, t, drillRestoreContainer,
		`SELECT COALESCE(string_agg(marker, ',' ORDER BY marker), '') FROM _backup_drill;`))
	if restoredMarkersCSV != "A" {
		t.Fatalf("PITR verification failed: expected restored _backup_drill markers 'A' (B absent), got %q (target=%s)", restoredMarkersCSV, targetTimestamp)
	}

	// 12. Assert fvpn-migrate status reports every migration applied on the restored container.
	runCmd(ctx, t, repoRoot, nil, "docker", "cp", migrateBinAmd64, drillRestoreContainer+":/usr/local/bin/fvpn-migrate")
	migrateStatusOut := runCmd(ctx, t, repoRoot, nil, "docker", "exec",
		"-e", "FVPN_DATABASE_URL=postgres://fvpn@127.0.0.1:5432/fvpn?sslmode=disable",
		drillRestoreContainer, "/usr/local/bin/fvpn-migrate", "status")
	appliedMigrations := strings.Count(migrateStatusOut, " applied\n")
	pendingMigrations := strings.Count(migrateStatusOut, " pending\n")
	if appliedMigrations == 0 || pendingMigrations != 0 {
		t.Fatalf("expected all migrations applied and 0 pending on restored DB, got applied=%d pending=%d:\n%s",
			appliedMigrations, pendingMigrations, migrateStatusOut)
	}

	// 13. Run post-restore-checklist.sql against the restored container and verify all sections execute cleanly.
	checklistBytes, err := os.ReadFile(filepath.Join(repoRoot, "deploy", "backup", "post-restore-checklist.sql"))
	if err != nil {
		t.Fatal(err)
	}
	checklistCmd := exec.CommandContext(ctx, "docker", "exec", "-i", "--user", "postgres",
		drillRestoreContainer,
		"psql", "-U", "fvpn", "-d", "fvpn", "-v", "ON_ERROR_STOP=1", "-v", "restore_point="+targetTimestamp,
	)
	checklistCmd.Stdin = bytes.NewReader(checklistBytes)
	checklistOutBytes, err := checklistCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("post-restore-checklist.sql failed: %v\n%s", err, string(checklistOutBytes))
	}
	checklistOut := string(checklistOutBytes)
	for _, sectionHeader := range []string{
		"[1/6] USAGE LEDGER EPOCH",
		"[2/6] ENTITLEMENTS ENDING AFTER RESTORE POINT",
		"[3/6] PUBLICATION INTENTS",
		"[4/6] UPDATE BATCHES",
		"[5/6] REVOCATIONS, RETIREMENTS & SUSPENSIONS",
		"[6/6] REQUIRED USAGE LEDGER FREEZE & EPOCH ROTATION COMMANDS",
		"fvpn-usage-admin",
	} {
		if !strings.Contains(checklistOut, sectionHeader) {
			t.Fatalf("post-restore-checklist.sql output missing %q:\n%s", sectionHeader, checklistOut)
		}
	}

	// 14. Exercise deploy/backup/verify.sh against the backup repository volume.
	verifyScript := filepath.Join(repoRoot, "deploy", "backup", "verify.sh")
	verifyJSONPath := filepath.Join(workTmp, "verify-result.json")
	verifyStart := time.Now()
	verifyOut := runCmd(ctx, t, repoRoot, nil, "bash", verifyScript,
		"--stanza", "fvpn",
		"--repo-volume", drillRepoVolume,
		"--primary-container", drillPrimaryContainer,
		"--image", drillImage,
		"--verify-prefix", drillVerifyPrefix,
		"--migrate-bin", migrateBinAmd64,
		"--json-out", verifyJSONPath,
	)
	verifyRTOMs := time.Since(verifyStart).Milliseconds()
	if !strings.Contains(verifyOut, "status=PASS") {
		t.Fatalf("verify.sh did not report status=PASS:\n%s", verifyOut)
	}

	// 15. Collect SHA-256 digests of the backup set files from the pgBackRest repository.
	digests := collectRepoDigests(ctx, t, drillPrimaryContainer)
	if len(digests) == 0 {
		t.Fatal("expected non-empty backup set digests from pgBackRest repository")
	}

	totalDrillMs := time.Since(drillStart).Milliseconds()

	record := DrillRecord{
		SchemaVersion:         1,
		SpecReferences:        []string{"§15.4", "§16.2", "§17.1", "§17.2", "§17.3", "§17.4", "§17.5", "§19 AT30"},
		Requirements:          []string{"DATA01", "BILL01", "MTR03", "SECOPS-05"},
		DrillDate:             drillDate,
		ExecutedAtUTC:         drillStart.Format(time.RFC3339),
		PgBackRestVersion:     pgBackRestVer,
		PostgresVersion:       pgVersion,
		BaseImageDigest:       "postgres@sha256:31ba9c94a03ada9988f2bb493a22a75eef3a185884533e5cdbcd498ea835b08f",
		BuiltImageID:          builtImageID,
		ArchiveMode:           archiveMode,
		ProductionTimeout:     "300s",
		LabOverrideTimeout:    "10s",
		BackupLabel:           backupLabel,
		BackupType:            "full",
		BackupStartLSN:        backupStartLSN,
		BackupStopLSN:         backupStopLSN,
		TargetTimestampUTC:    targetTimestamp,
		TargetLSN:             targetLSN,
		PrimaryMarkers:        strings.Split(primaryMarkersCSV, ","),
		RestoredMarkers:       strings.Split(restoredMarkersCSV, ","),
		MarkerAPresent:        restoredMarkersCSV == "A",
		MarkerBAbsent:         !strings.Contains(restoredMarkersCSV, "B"),
		LivePrimaryProtected:  true,
		IsolatedNetworkMode:   netMode,
		IsolatedPublishedPort: portCount,
		MigrationsApplied:     appliedMigrations,
		MigrationsPending:     pendingMigrations,
		ChecklistVerified:     true,
		VerifyScriptPassed:    true,
		RPOTargetSeconds:      900,  // 15 minutes (§17.2)
		RTOTargetSeconds:      7200, // 2 hours (§17.2 / docs/handover/restore-drill.md)
		MeasuredArchiveLagMs:  rpoArchiveLagMs,
		MeasuredRestoreRTOMs:  rtoRestoreMs,
		MeasuredVerifyRTOMs:   verifyRTOMs,
		MeasuredTotalDrillMs:  totalDrillMs,
		BackupSetDigests:      digests,
		Status:                "PASS",
	}

	outputDir := os.Getenv("FVPN_BACKUP_DRILL_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = filepath.Join(repoRoot, "docs", "ops", "backups")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	jsonPath := filepath.Join(outputDir, fmt.Sprintf("drill-%s.json", drillDate))
	mdPath := filepath.Join(outputDir, fmt.Sprintf("drill-%s.md", drillDate))

	jsonBytes, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, append(jsonBytes, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	mdContent := renderDrillMarkdown(record)
	if err := os.WriteFile(mdPath, []byte(mdContent), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Logf("PITR Drill Summary: backup=%s target=%s markerA=%v markerB_absent=%v migrations=%d/%d RPO=%dms RTO=%dms (verify=%dms, total=%dms) record=%s",
		record.BackupLabel,
		record.TargetTimestampUTC,
		record.MarkerAPresent,
		record.MarkerBAbsent,
		record.MigrationsApplied,
		record.MigrationsApplied+record.MigrationsPending,
		record.MeasuredArchiveLagMs,
		record.MeasuredRestoreRTOMs,
		record.MeasuredVerifyRTOMs,
		record.MeasuredTotalDrillMs,
		jsonPath,
	)
}

func runCmd(ctx context.Context, t *testing.T, dir string, extraEnv []string, name string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %s %s: %v\n%s", name, strings.Join(args, " "), err, string(out))
	}
	return string(out)
}

func execPsql(ctx context.Context, t *testing.T, container, sql string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, "docker", "exec", "--user", "postgres", container,
		"psql", "-U", "fvpn", "-d", "fvpn", "-v", "ON_ERROR_STOP=1", "-tAc", sql)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("psql on %s failed: %v\nSQL: %s\nOutput: %s", container, err, sql, string(out))
	}
	return string(out)
}

func waitForPostgresReady(ctx context.Context, t *testing.T, container, user, dbName string) {
	t.Helper()
	for i := 0; i < 80; i++ {
		cmd := exec.CommandContext(ctx, "docker", "exec", "--user", "postgres", container,
			"pg_isready", "-U", user, "-d", dbName)
		if cmd.Run() == nil {
			// Also ensure SQL queries succeed (after docker-entrypoint init finishes).
			check := exec.CommandContext(ctx, "docker", "exec", "--user", "postgres", container,
				"psql", "-U", user, "-d", dbName, "-tAc", "SELECT 1;")
			if out, err := check.CombinedOutput(); err == nil && strings.TrimSpace(string(out)) == "1" {
				return
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	logs, _ := exec.CommandContext(ctx, "docker", "logs", container).CombinedOutput()
	t.Fatalf("timed out waiting for PostgreSQL container %s to become ready:\n%s", container, string(logs))
}

func parsePgBackRestInfo(t *testing.T, rawOutput string) (label, startLSN, stopLSN string, stopUnix int64) {
	t.Helper()
	// Find JSON array in output (backup.sh full prints backup logs followed by JSON info array).
	idx := strings.Index(rawOutput, "[{")
	if idx < 0 {
		t.Fatalf("pgBackRest info output missing JSON array:\n%s", rawOutput)
	}
	jsonPart := rawOutput[idx:]
	var stanzas []struct {
		Name   string `json:"name"`
		Backup []struct {
			Label string `json:"label"`
			Type  string `json:"type"`
			LSN   struct {
				Start string `json:"start"`
				Stop  string `json:"stop"`
			} `json:"lsn"`
			Timestamp struct {
				Start int64 `json:"start"`
				Stop  int64 `json:"stop"`
			} `json:"timestamp"`
		} `json:"backup"`
	}
	if err := json.Unmarshal([]byte(jsonPart), &stanzas); err != nil {
		t.Fatalf("unmarshal pgBackRest info JSON: %v\n%s", err, jsonPart)
	}
	if len(stanzas) == 0 || len(stanzas[0].Backup) == 0 {
		t.Fatalf("pgBackRest info contains no backups:\n%s", jsonPart)
	}
	last := stanzas[0].Backup[len(stanzas[0].Backup)-1]
	return last.Label, last.LSN.Start, last.LSN.Stop, last.Timestamp.Stop
}

func collectRepoDigests(ctx context.Context, t *testing.T, container string) []BackupFileDigest {
	t.Helper()
	script := `
		cd /var/lib/pgbackrest
		find archive backup -type f \( -name 'archive.info*' -o -name 'backup.info*' -o -name 'backup.manifest*' \) | sort | while read -r f; do
			sha=$(sha256sum "$f" | awk '{print $1}')
			sz=$(stat -c '%s' "$f")
			printf '%s\t%s\t%s\n' "$f" "$sha" "$sz"
		done
	`
	out, err := exec.CommandContext(ctx, "docker", "exec", "--user", "postgres", container, "bash", "-c", script).CombinedOutput()
	if err != nil {
		t.Fatalf("collectRepoDigests failed: %v\n%s", err, string(out))
	}
	var digests []BackupFileDigest
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			continue
		}
		sz, _ := strconv.ParseInt(parts[2], 10, 64)
		digests = append(digests, BackupFileDigest{
			Path:   parts[0],
			SHA256: parts[1],
			Bytes:  sz,
		})
	}
	sort.Slice(digests, func(i, j int) bool {
		return digests[i].Path < digests[j].Path
	})
	return digests
}

func cleanDrillDockerResources(ctx context.Context, t *testing.T, removeImage bool) {
	t.Helper()
	// Remove any containers matching fvpn-w4-ops-backups-*
	if out, err := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name="+drillPrefix, "--format", "{{.Names}}").Output(); err == nil {
		for _, c := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			c = strings.TrimSpace(c)
			if strings.HasPrefix(c, drillPrefix) {
				_ = exec.CommandContext(ctx, "docker", "rm", "-fv", c).Run()
			}
		}
	}
	// Remove any volumes matching fvpn-w4-ops-backups-*
	if out, err := exec.CommandContext(ctx, "docker", "volume", "ls", "--filter", "name="+drillPrefix, "--format", "{{.Name}}").Output(); err == nil {
		for _, v := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			v = strings.TrimSpace(v)
			if strings.HasPrefix(v, drillPrefix) {
				_ = exec.CommandContext(ctx, "docker", "volume", "rm", "-f", v).Run()
			}
		}
	}
	// Remove any networks matching fvpn-w4-ops-backups-*
	if out, err := exec.CommandContext(ctx, "docker", "network", "ls", "--filter", "name="+drillPrefix, "--format", "{{.Name}}").Output(); err == nil {
		for _, n := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			n = strings.TrimSpace(n)
			if strings.HasPrefix(n, drillPrefix) {
				_ = exec.CommandContext(ctx, "docker", "network", "rm", n).Run()
			}
		}
	}
	if removeImage {
		_ = exec.CommandContext(ctx, "docker", "rmi", "-f", drillImage).Run()
	}
}

func renderDrillMarkdown(r DrillRecord) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Central PostgreSQL 17 PITR Restore Drill Record (`%s`)\n\n", r.DrillDate)
	fmt.Fprintf(&b, "- **Spec References**: `%s`\n", strings.Join(r.SpecReferences, "`, `"))
	fmt.Fprintf(&b, "- **Requirement & AT Coverage**: `%s`, `AT30`\n", strings.Join(r.Requirements, "`, `"))
	fmt.Fprintf(&b, "- **Executed At (UTC)**: `%s`\n", r.ExecutedAtUTC)
	fmt.Fprintf(&b, "- **PostgreSQL Version**: `%s`\n", r.PostgresVersion)
	fmt.Fprintf(&b, "- **pgBackRest Version**: `%s`\n", r.PgBackRestVersion)
	fmt.Fprintf(&b, "- **Base Image Digest**: `%s`\n", r.BaseImageDigest)
	fmt.Fprintf(&b, "- **Status**: **%s**\n\n", r.Status)

	b.WriteString("## 1. Measured RPO & RTO vs. Targets (`§17.2`)\n\n")
	b.WriteString("| Metric | Spec / Handover Target | Measured Drill Value | Result |\n")
	b.WriteString("|---|---|---|---|\n")
	fmt.Fprintf(&b, "| **Continuous WAL Archive Lag (RPO)** | `<= 15 minutes` (`%d s`) | **`%d ms`** (`%.3f s`) | **PASS** |\n",
		r.RPOTargetSeconds, r.MeasuredArchiveLagMs, float64(r.MeasuredArchiveLagMs)/1000.0)
	fmt.Fprintf(&b, "| **Physical PITR Restore + Promotion Time (Database RTO)** | `<= 2 hours` (`%d s`) | **`%d ms`** (`%.3f s`) | **PASS** |\n",
		r.RTOTargetSeconds, r.MeasuredRestoreRTOMs, float64(r.MeasuredRestoreRTOMs)/1000.0)
	fmt.Fprintf(&b, "| **Full Throwaway Verification (`verify.sh`: Restore + `fvpn-migrate status` + Checklist)** | `<= 2 hours` (`%d s`) | **`%d ms`** (`%.3f s`) | **PASS** |\n",
		r.RTOTargetSeconds, r.MeasuredVerifyRTOMs, float64(r.MeasuredVerifyRTOMs)/1000.0)
	fmt.Fprintf(&b, "| **Total End-to-End Automated Drill Duration** | `<= 10 minutes` CI budget | **`%d ms`** (`%.3f s`) | **PASS** |\n\n",
		r.MeasuredTotalDrillMs, float64(r.MeasuredTotalDrillMs)/1000.0)

	b.WriteString("## 2. Point-in-Time Recovery (PITR) & Safety Verification\n\n")
	b.WriteString("| Verification Check | Expected | Observed | Result |\n")
	b.WriteString("|---|---|---|---|\n")
	fmt.Fprintf(&b, "| Primary `archive_mode` | `on` | `%s` (prod `archive_timeout=%s`, lab `TEST ONLY` `%s`) | **PASS** |\n",
		r.ArchiveMode, r.ProductionTimeout, r.LabOverrideTimeout)
	fmt.Fprintf(&b, "| Full Base Backup Label & LSN Window | Valid pgBackRest backup | `%s` (`%s` -> `%s`) | **PASS** |\n",
		r.BackupLabel, r.BackupStartLSN, r.BackupStopLSN)
	fmt.Fprintf(&b, "| Target Recovery Point (`restore-pitr.sh --target-time`) | Between Marker `A` and Marker `B` | `%s` (LSN `%s`) | **PASS** |\n",
		r.TargetTimestampUTC, r.TargetLSN)
	fmt.Fprintf(&b, "| Live Primary Markers (`_backup_drill`) | `A, B` | `%s` | **PASS** |\n",
		strings.Join(r.PrimaryMarkers, ", "))
	fmt.Fprintf(&b, "| Restored Instance Markers (`_backup_drill`) | `A` present, `B` absent | `%s` (`marker_a_present=%v`, `marker_b_absent=%v`) | **PASS** |\n",
		strings.Join(r.RestoredMarkers, ", "), r.MarkerAPresent, r.MarkerBAbsent)
	fmt.Fprintf(&b, "| Live-Primary Overwrite Guard | Refuses live primary container/dir | `live_primary_overwrite_refused=%v` | **PASS** |\n",
		r.LivePrimaryProtected)
	fmt.Fprintf(&b, "| Isolated Restore Container Network & Ports | `--network none`, `0` host ports | `network=%s`, `published_ports=%d` | **PASS** |\n",
		r.IsolatedNetworkMode, r.IsolatedPublishedPort)
	fmt.Fprintf(&b, "| Restored Schema Migration Status (`fvpn-migrate status`) | All migrations `applied`, `0` `pending` | `%d applied`, `%d pending` | **PASS** |\n",
		r.MigrationsApplied, r.MigrationsPending)
	fmt.Fprintf(&b, "| Post-Restore Reconciliation Checklist (`post-restore-checklist.sql`) | All 6 read-only sections succeed | `verified=%v` | **PASS** |\n\n",
		r.ChecklistVerified)

	b.WriteString("## 3. Backup Set Artifact Digests (pgBackRest Repository)\n\n")
	b.WriteString("| Repository Path | Size (Bytes) | SHA-256 Digest |\n")
	b.WriteString("|---|---:|---|\n")
	for _, d := range r.BackupSetDigests {
		fmt.Fprintf(&b, "| `%s` | `%d` | `%s` |\n", d.Path, d.Bytes, d.SHA256)
	}
	return b.String()
}
