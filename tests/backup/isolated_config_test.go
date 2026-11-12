package backup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAT30_IsolatedNames(t *testing.T) {
	for _, option := range []string{"--db-user", "--db-name"} {
		for _, value := range []string{"", "a\nhost all all 0.0.0.0/0 trust", "a,b", `a"b`, "a/b", "+group", "a b", "é", "9vpn", "vpn.prod", strings.Repeat("a", 64)} {
			cmd := exec.Command("bash", "../../deploy/backup/restore-pitr.sh", "--latest", "--isolated-container", "unused", option, value)
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), "invalid isolated --db-user or --db-name") {
				t.Fatalf("%s %q not refused before Docker: %v %s", option, value, err, out)
			}
		}
	}
}

func TestAT30_IsolatedConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, connections, wantError string
	}{
		{"resources", "240", ""},
		{"defaults", "50", ""},
		{"missing", "", "max_connections"},
		{"malformed", "100\nport = 9000", "max_connections"},
		{"overflow", "9999999999", "max_connections"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			control := "max_connections setting: " + tc.connections + "\nmax_worker_processes setting: 16\nmax_wal_senders setting: 20\nmax_prepared_xacts setting: 12\nmax_locks_per_xact setting: 128\n"
			if tc.name == "malformed" {
				control = strings.ReplaceAll(control, "max_connections setting: 100\nport = 9000", "max_connections setting: not-a-number")
			}
			if err := os.WriteFile(filepath.Join(dir, "control"), []byte(control), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "pg_controldata"), []byte("#!/bin/sh\ncat \"$CONTROL_FIXTURE\"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "-c", `source ../../deploy/backup/postgres/fvpn-postgres-entrypoint.sh
prepare_isolated_restore "$1/config" "$1/data" all replication
printf '%s\n' "${ISOLATED_ARGS[@]}"
cat "$1/config/pg_hba.conf" "$1/config/pg_ident.conf"`, "_", dir)
			cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "CONTROL_FIXTURE="+filepath.Join(dir, "control"))
			out, err := cmd.CombinedOutput()
			if tc.wantError != "" {
				if err == nil || !strings.Contains(string(out), tc.wantError) {
					t.Fatalf("expected named resource error: %v %s", err, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("prepare: %v %s", err, out)
			}
			connections := tc.connections
			if tc.name == "defaults" {
				connections = "100"
			}
			for _, want := range []string{
				"max_connections=" + connections, "max_worker_processes=16", "max_wal_senders=20",
				"max_prepared_transactions=12", "max_locks_per_transaction=128",
				"ssl=off", "archive_mode=off", "archive_command=\n", "archive_library=\n",
				"port=5432", "listen_addresses=127.0.0.1", "shared_preload_libraries=\n",
				"log_min_messages=warning", "log_error_verbosity=default",
				"recovery_end_command=\n", "archive_cleanup_command=\n",
				`local "replication" "all" peer map=fvpn_verifier`,
				`hostnossl "replication" "all" 127.0.0.1/32 trust`, `fvpn_verifier postgres "all"`,
			} {
				if !strings.Contains(string(out), want) {
					t.Errorf("missing %q: %s", want, out)
				}
			}
		})
	}
}

func TestAT30_IsolatedRecoveryResourceFailure(t *testing.T) {
	// PostgreSQL can enter hot standby, encounter a higher resource value in
	// later WAL, and pause. The runner must return its diagnostic immediately.
	for _, parameter := range []string{"max_connections", "max_worker_processes", "max_wal_senders", "max_prepared_transactions", "max_locks_per_transaction"} {
		t.Run(parameter, func(t *testing.T) {
			dir := t.TempDir()
			stub := `#!/bin/bash
case "$1" in
ps|volume|run|rm) ;;
inspect)
  case "$3" in
    *NetworkMode*) echo none ;;
    *NetworkSettings.Ports*) echo 0 ;;
    *State.Running*) echo true ;;
  esac ;;
logs) printf 'DETAIL: %s = 100 is a lower setting than on the primary server, where its value was 240.\n' "$RESOURCE_PARAMETER" ;;
*) echo 'unexpected Docker call' >&2; exit 99 ;;
esac
`
			if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(stub), 0o700); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "../../deploy/backup/restore-pitr.sh", "--latest", "--isolated-container", "unused", "--target-volume", "unused", "--repo-volume", "unused")
			cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "RESOURCE_PARAMETER="+parameter)
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), "recovery resource mismatch:") || !strings.Contains(string(out), parameter+" = 100") || strings.Contains(string(out), "timed out") {
				t.Fatalf("expected immediate named recovery failure: %v %s", err, out)
			}
		})
	}
}
