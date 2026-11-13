//go:build integration

package backup

import (
	"context"
	cryptorand "crypto/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAT30_IsolatedHostConfig reproduces a native primary's external config/TLS
// layout without requiring a host PostgreSQL service or changing its settings.
func TestAT30_IsolatedHostConfig(t *testing.T) {
	// TEST ONLY - NOT A SECRET; supplied only to restore/verify processes.
	const canary = "TEST_ONLY_UNUSED_REPOSITORY_OPTION"
	restoreEnv := []string{"PGBACKREST_REPO1_S3_KEY_SECRET=" + canary}
	markers := []string{"FVPN_TEST_ONLY_RECOVERY_END", "FVPN_TEST_ONLY_ARCHIVE_CLEANUP", "FVPN_TEST_ONLY_IMITATION"}
	assertNoHook := func(t *testing.T, where, output string) {
		t.Helper()
		for _, marker := range append(markers, canary) {
			if strings.Contains(output, marker) {
				t.Fatalf("hostile restored marker/canary %s in %s", marker, where)
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	if err := exec.CommandContext(ctx, "docker", "info").Run(); err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatal(err)
		}
		t.Skipf("Docker unavailable: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	names, err := drillNamesFromEnv(os.Getenv("FVPN_BACKUP_DRILL_PREFIX"), cryptorand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	// Sequential with the original drill; still derive all objects from its
	// caller-selected prefix, including every throwaway restore below.
	t.Logf("host-config drill prefix %s", names.Prefix)
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cleanCancel()
		if err := removeDrillResources(cleanCtx, names.Prefix); err != nil {
			t.Error(err)
		}
	})
	docker := func(t *testing.T, args ...string) string {
		t.Helper()
		return runCmd(ctx, t, root, nil, "docker", args...)
	}
	sql := func(t *testing.T, container, query string) string {
		t.Helper()
		return strings.TrimSpace(docker(t, "exec", "--user", "postgres", container,
			"psql", "-X", "-U", "postgres", "-d", "fvpn", "-v", "ON_ERROR_STOP=1", "-tAc", query))
	}
	backup := func(action string) string {
		return runCmd(ctx, t, root, nil, "bash", "deploy/backup/backup.sh", "--container", names.PrimaryContainer, "--db-user", "postgres", action)
	}
	docker(t, "build", "-t", names.Image, filepath.Join(root, "deploy/backup/postgres"))
	work := t.TempDir()
	migrate := filepath.Join(work, "fvpn-migrate")
	runCmd(ctx, t, root, []string{"GOOS=linux", "GOARCH=amd64"}, "go", "build", "-trimpath", "-o", migrate, "./internal/db/cmd/fvpn-migrate")
	docker(t, "volume", "create", names.PrimaryVolume)
	docker(t, "volume", "create", names.RepoVolume)
	docker(t, "run", "-d", "--name", names.PrimaryContainer, "--network", "none",
		"-e", "POSTGRES_USER=postgres", "-e", "POSTGRES_DB=fvpn", "-e", "PGPORT=55432", "-e", "PGBACKREST_PG1_PORT=55432",
		"-v", names.PrimaryVolume+":/var/lib/postgresql/data", "-v", names.RepoVolume+":/var/lib/pgbackrest",
		"--entrypoint", "bash", names.Image, "-ceu", `
install -d -o postgres -g postgres -m 0700 "$PGDATA" /etc/postgresql/17/main
runuser -u postgres -- initdb -D "$PGDATA" --auth-local=peer --auth-host=scram-sha-256 > /var/log/pgbackrest/fixture-init.log
mv "$PGDATA/postgresql.conf" "$PGDATA/pg_hba.conf" "$PGDATA/pg_ident.conf" /etc/postgresql/17/main/
openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj /CN=example.invalid \
  -keyout /etc/postgresql/17/main/server.key -out /etc/postgresql/17/main/server.crt >/dev/null 2>&1
chown postgres:postgres /etc/postgresql/17/main/server.*
chmod 0600 /etc/postgresql/17/main/server.key
printf 'local all postgres peer\nhostssl fvpn fvpn_app 127.0.0.1/32 scram-sha-256\n' > /etc/postgresql/17/main/pg_hba.conf
exec /usr/local/bin/fvpn-postgres-entrypoint.sh postgres \
  -c config_file=/etc/postgresql/17/main/postgresql.conf \
  -c hba_file=/etc/postgresql/17/main/pg_hba.conf -c ident_file=/etc/postgresql/17/main/pg_ident.conf \
  -c port=55432 -c listen_addresses=127.0.0.1 -c ssl=on -c archive_timeout=0 \
  -c ssl_cert_file=/etc/postgresql/17/main/server.crt -c ssl_key_file=/etc/postgresql/17/main/server.key \
  -c max_connections=240 -c max_worker_processes=16 -c max_wal_senders=20 \
  -c max_prepared_transactions=12 -c max_locks_per_transaction=128
`)
	waitForPostgresReady(ctx, t, names.PrimaryContainer, "postgres", "postgres")
	docker(t, "exec", "--user", "postgres", names.PrimaryContainer, "createdb", "fvpn")
	docker(t, "cp", migrate, names.PrimaryContainer+":/usr/local/bin/fvpn-migrate")
	docker(t, "exec", "--user", "postgres", "-e", "FVPN_DATABASE_URL=postgres:///fvpn?host=/var/run/postgresql&port=55432", names.PrimaryContainer, "fvpn-migrate", "up")
	// A runtime-only password proves the hostssl/SCRAM rule works. Neither it
	// nor the generated TLS key is copied into PGDATA or returned to test logs.
	docker(t, "exec", "--user", "postgres", names.PrimaryContainer, "bash", "-ceu", `
password=$(openssl rand -hex 24)
printf "CREATE ROLE fvpn_app LOGIN PASSWORD '%s';\n" "$password" | psql -U postgres -d fvpn -v ON_ERROR_STOP=1 >/dev/null
PGPASSWORD="$password" psql 'host=127.0.0.1 port=55432 user=fvpn_app dbname=fvpn sslmode=require' -tAc 'SELECT 1' | grep -qx 1
`)
	primarySettings := sql(t, names.PrimaryContainer, `SELECT name || '=' || setting FROM pg_settings WHERE name IN
('config_file','hba_file','ident_file','port','ssl','max_connections','max_worker_processes','max_wal_senders','max_prepared_transactions','max_locks_per_transaction') ORDER BY name`)
	t.Logf("primary effective settings:\n%s", primarySettings)
	for _, want := range []string{"port=55432", "ssl=on", "max_connections=240", "max_worker_processes=16", "max_wal_senders=20", "max_prepared_transactions=12", "max_locks_per_transaction=128"} {
		if !strings.Contains(primarySettings, want) {
			t.Fatalf("primary missing %s", want)
		}
	}
	// These values are recorded in the physical backup but are not activated
	// on the primary. Several would prevent the isolated image from starting.
	for _, setting := range []string{
		"port = '6543'", "listen_addresses = '*'", "ssl = 'on'",
		"ssl_cert_file = '/etc/primary-only/missing.crt'", "ssl_key_file = '/etc/primary-only/missing.key'",
		"hba_file = '/etc/primary-only/pg_hba.conf'", "ident_file = '/etc/primary-only/pg_ident.conf'",
		"unix_socket_directories = '/etc/primary-only'", "archive_mode = 'always'", "archive_command = 'false'",
		"archive_library = 'missing_archive_library'", "shared_preload_libraries = 'missing_primary_library'",
		"max_connections = '100'", "max_worker_processes = '8'", "max_wal_senders = '5'",
		"max_prepared_transactions = '0'", "max_locks_per_transaction = '64'",
		"log_min_messages = 'panic'", "log_error_verbosity = 'terse'",
		"zero_damaged_pages = 'on'", "ignore_checksum_failure = 'on'",
		"recovery_end_command = 'echo FVPN_TEST_ONLY_RECOVERY_END > /var/lib/postgresql/data/FVPN_TEST_ONLY_RECOVERY_END; echo FVPN_TEST_ONLY_RECOVERY_END >&2; printenv PGBACKREST_REPO1_S3_KEY_SECRET >&2'",
		"archive_cleanup_command = 'echo FVPN_TEST_ONLY_ARCHIVE_CLEANUP > /var/lib/postgresql/data/FVPN_TEST_ONLY_ARCHIVE_CLEANUP; echo FVPN_TEST_ONLY_ARCHIVE_CLEANUP >&2; printenv PGBACKREST_REPO1_S3_KEY_SECRET >&2'",
	} {
		sql(t, names.PrimaryContainer, "ALTER SYSTEM SET "+setting)
	}
	// Imitate pgBackRest's marker BEFORE this restore's genuine appended block.
	// Both the lookalike and the include are backed up, without primary reload.
	docker(t, "exec", "--user", "postgres", names.PrimaryContainer, "bash", "-ceu", `
cat >> "$PGDATA/postgresql.auto.conf" <<'CONF'
# Recovery settings generated by pgBackRest restore on 2000-01-01 00:00:00
recovery_end_command = 'echo FVPN_TEST_ONLY_IMITATION > /var/lib/postgresql/data/FVPN_TEST_ONLY_IMITATION; echo FVPN_TEST_ONLY_IMITATION >&2; printenv PGBACKREST_REPO1_S3_KEY_SECRET >&2'
include_if_exists = '/var/lib/postgresql/data/hostile-include.conf'
CONF
printf "ignore_invalid_pages = 'on'\n" > "$PGDATA/hostile-include.conf"
`)
	sql(t, names.PrimaryContainer, "CREATE TABLE _backup_drill (marker text PRIMARY KEY); INSERT INTO _backup_drill VALUES ('A')")
	backup("stanza-create")
	_, _, _, stop := parsePgBackRestInfo(t, backup("full"))
	for time.Now().Unix() <= stop {
		time.Sleep(100 * time.Millisecond)
	}
	sql(t, names.PrimaryContainer, "INSERT INTO _backup_drill VALUES ('M')")
	targetTime := sql(t, names.PrimaryContainer, `SELECT to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`)
	// Exact record boundary used for exclusive LSN recovery before B.
	targetLSN := sql(t, names.PrimaryContainer, "SELECT pg_current_wal_insert_lsn()")
	sql(t, names.PrimaryContainer, "INSERT INTO _backup_drill VALUES ('B')")
	backup("wal-switch")
	docker(t, "exec", names.PrimaryContainer, "bash", "-ceu", `for f in postgresql.conf pg_hba.conf pg_ident.conf; do test ! -e "$PGDATA/$f"; done`)
	t.Log("T1: primary PGDATA contains none of postgresql.conf, pg_hba.conf, pg_ident.conf; external TLS/SCRAM connection passed")
	// There are no further primary writes, and archive_timeout remains zero.
	// Compare every repository object's name and digest, not just manifests.
	digests := func() string {
		return docker(t, "exec", names.PrimaryContainer, "bash", "-ceu", `cd /var/lib/pgbackrest; find . -type f -print0 | sort -z | xargs -0 sha256sum`)
	}
	before := digests()
	verifyCmd := exec.CommandContext(ctx, "bash", filepath.Join(root, "deploy/backup/verify.sh"),
		"--repo-volume", names.RepoVolume, "--primary-container", names.PrimaryContainer, "--image", names.Image,
		"--verify-prefix", names.VerifyPrefix, "--migrate-bin", migrate, "--db-user", "postgres")
	verifyCmd.Env = append(os.Environ(), restoreEnv...)
	verifyBytes, verifyErr := verifyCmd.CombinedOutput()
	verifyOut := string(verifyBytes)
	assertNoHook(t, "verify.sh output", verifyOut)
	// Check verify.sh's own EXIT trap before the test's broader cleanup can
	// hide leaked resources, including on an injected preparation failure.
	for _, object := range []struct{ kind, name string }{
		{"container", names.VerifyPrefix + "-pg"},
		{"container", names.VerifyPrefix + "-pg-init"},
		{"volume", names.VerifyPrefix + "-pgdata"},
	} {
		if err := exec.CommandContext(ctx, "docker", object.kind, "inspect", object.name).Run(); err == nil {
			t.Errorf("verify.sh leaked %s %s", object.kind, object.name)
		}
	}
	t.Logf("verify.sh exit=%v; verification containers and volume absent before test cleanup", verifyErr)
	if verifyErr != nil || !strings.Contains(verifyOut, "status=PASS") {
		t.Fatalf("latest verifier did not pass: %s", verifyOut)
	}
	t.Logf("T4 latest verifier:\n%s", verifyOut)
	if after := digests(); before != after {
		t.Fatal("T5: repository object list/digests changed during verify")
	}
	t.Logf("T5: all %d repository object names and SHA-256 digests unchanged after verify", len(strings.Split(strings.TrimSpace(before), "\n")))

	for _, target := range []struct {
		name string
		args []string
		want string
	}{
		{"time", []string{"--target-time", targetTime}, "A,M"},
		{"lsn", []string{"--target-lsn", targetLSN, "--target-exclusive"}, "A,M"},
		{"latest", []string{"--latest"}, "A,B,M"},
	} {
		t.Run(target.name, func(t *testing.T) {
			container := names.Prefix + "-" + target.name
			volume := container + "-pgdata"
			args := append([]string{"deploy/backup/restore-pitr.sh", "--isolated-container", container, "--target-volume", volume,
				"--repo-volume", names.RepoVolume, "--image", names.Image, "--db-user", "postgres"}, target.args...)
			restoreOut := runCmd(ctx, t, root, restoreEnv, "bash", args...)
			assertNoHook(t, "restore-pitr.sh output", restoreOut)
			assertNoHook(t, "container log", docker(t, "logs", container))
			for _, marker := range markers {
				docker(t, "exec", container, "test", "!", "-e", "/var/lib/postgresql/data/"+marker)
			}
			containerEnv := docker(t, "inspect", "--format", "{{range .Config.Env}}{{println .}}{{end}}", container)
			if strings.Contains(containerEnv, "PGBACKREST_REPO1_S3_KEY_SECRET=") || strings.Contains(containerEnv, canary) {
				t.Fatal("unneeded repository canary variable reached the server container")
			}
			for _, key := range []string{"zero_damaged_pages", "ignore_checksum_failure", "ignore_invalid_pages"} {
				if got := sql(t, container, "SHOW "+key); got != "off" {
					t.Fatalf("hostile restored setting %s=%s, want default off", key, got)
				}
			}
			autoConf := docker(t, "exec", container, "cat", "/var/lib/postgresql/data/postgresql.auto.conf")
			allowed := map[string]bool{"restore_command": true, "recovery_target_time": true, "recovery_target_lsn": true, "recovery_target_action": true, "recovery_target_inclusive": true}
			for _, line := range strings.Split(strings.TrimSpace(autoConf), "\n") {
				key, _, ok := strings.Cut(line, " = ")
				if !ok || !allowed[key] {
					t.Fatalf("auto.conf retains an unapproved key or line: %q", key)
				}
			}
			t.Logf("%s recovery settings only:\n%s", target.name, autoConf)
			docker(t, "exec", "--user", "postgres", container, "bash", "-ceu", `
for file in /etc/postgresql/fvpn-isolated /etc/postgresql/fvpn-isolated/*; do
  test "$(stat -c %u "$file")" = 0
  test -r "$file"
  test ! -w "$file"
done
`)
			t.Log("hostile hooks/includes/settings discarded; remote canary absent; root-owned config is postgres-readable and not writable")
			got := sql(t, container, "SELECT string_agg(marker, ',' ORDER BY marker) FROM _backup_drill")
			if got != target.want {
				t.Fatalf("markers %s, want %s", got, target.want)
			}
			t.Logf("T4 %s markers: %s", target.name, got)
			settings := sql(t, container, `SELECT name || '=' || setting || ' source=' || source FROM pg_settings WHERE name IN
('config_file','hba_file','ident_file','port','listen_addresses','unix_socket_directories','ssl','archive_mode','archive_command','archive_library','shared_preload_libraries','recovery_end_command','archive_cleanup_command','max_connections','max_worker_processes','max_wal_senders','max_prepared_transactions','max_locks_per_transaction') ORDER BY name`)
			for _, want := range []string{
				"config_file=/etc/postgresql/fvpn-isolated/postgresql.conf", "hba_file=/etc/postgresql/fvpn-isolated/pg_hba.conf", "ident_file=/etc/postgresql/fvpn-isolated/pg_ident.conf",
				"port=5432", "listen_addresses=127.0.0.1", "unix_socket_directories=/var/run/postgresql", "ssl=off", "archive_mode=off", "archive_command=(disabled)", "archive_library=", "shared_preload_libraries=",
				"max_connections=240", "max_worker_processes=16", "max_wal_senders=20", "max_prepared_transactions=12", "max_locks_per_transaction=128",
				"recovery_end_command=", "archive_cleanup_command=",
			} {
				source := "command line"
				if strings.HasPrefix(want, "config_file=") || strings.HasPrefix(want, "hba_file=") || strings.HasPrefix(want, "ident_file=") {
					// PostgreSQL resolves these file paths with PGC_S_OVERRIDE.
					source = "override"
				}
				if !strings.Contains(settings, want+" source="+source) {
					t.Errorf("missing %s from %s:\n%s", want, source, settings)
				}
			}
			t.Logf("T2/T3/T5 effective settings:\n%s", settings)
			// SHOW uses PostgreSQL's archive_command display hook: with
			// archive_mode=off it returns "(disabled)" even for an empty value.
			if got := sql(t, container, "SELECT reset_val = '' FROM pg_settings WHERE name = 'archive_command'"); got != "t" {
				t.Fatalf("archive_command reset value is not empty: %s", got)
			}
			for _, parameter := range []string{"config_file", "hba_file", "port", "listen_addresses", "ssl", "archive_mode", "archive_command"} {
				t.Logf("SHOW %s: %s", parameter, sql(t, container, "SHOW "+parameter))
			}
			if got := strings.TrimSpace(docker(t, "inspect", "--format", "{{.HostConfig.NetworkMode}} {{len .NetworkSettings.Ports}}", container)); got != "none 0" {
				t.Fatalf("isolation: %s", got)
			}
			if got := strings.TrimSpace(docker(t, "inspect", "--format", `{{range .Mounts}}{{if eq .Destination "/var/lib/pgbackrest"}}{{.RW}}{{end}}{{end}}`, container)); got != "false" {
				t.Fatalf("repository writable: %s", got)
			}
			// Exercise both allowed transports and refuse other roles, databases,
			// OS identities, TLS, IPv6, and physical replication.
			docker(t, "exec", container, "psql", "postgres://postgres@127.0.0.1:5432/fvpn?sslmode=disable", "-tAc", "SELECT 1")
			for _, denied := range []struct{ name, osUser, dsn, query, refusal string }{
				{"peer", "root", "user=postgres dbname=fvpn", "SELECT 1", `FATAL:  Peer authentication failed for user "postgres"`},
				{"socket_database", "postgres", "user=postgres dbname=postgres", "SELECT 1", `FATAL:  no pg_hba.conf entry for host "[local]", user "postgres", database "postgres"`},
				{"socket_role", "postgres", "user=fvpn_app dbname=fvpn", "SELECT 1", `FATAL:  no pg_hba.conf entry for host "[local]", user "fvpn_app", database "fvpn"`},
				{"tcp_role", "postgres", "host=127.0.0.1 user=fvpn_app dbname=fvpn sslmode=disable", "SELECT 1", `FATAL:  no pg_hba.conf entry for host "127.0.0.1", user "fvpn_app", database "fvpn"`},
				{"tcp_database", "postgres", "host=127.0.0.1 user=postgres dbname=postgres sslmode=disable", "SELECT 1", `FATAL:  no pg_hba.conf entry for host "127.0.0.1", user "postgres", database "postgres"`},
				{"tls", "postgres", "host=127.0.0.1 user=postgres dbname=fvpn sslmode=require", "SELECT 1", "server does not support SSL, but SSL was required"},
				{"ipv6", "postgres", "host=::1 user=postgres dbname=fvpn sslmode=disable", "SELECT 1", `connection to server at "::1", port 5432 failed: Connection refused`},
				{"replication", "postgres", "host=127.0.0.1 user=postgres replication=true sslmode=disable", "IDENTIFY_SYSTEM", `FATAL:  no pg_hba.conf entry for replication connection from host "127.0.0.1", user "postgres"`},
			} {
				out, err := exec.CommandContext(ctx, "docker", "exec", "--user", denied.osUser, "-e", "LC_ALL=C", container, "psql", "-X", "-w", denied.dsn, "-tAc", denied.query).CombinedOutput()
				if err == nil || !strings.Contains(string(out), denied.refusal) || strings.Contains(string(out), "ERROR:") {
					t.Fatalf("%s: expected connection refusal %q, got %v: %s", denied.name, denied.refusal, err, out)
				}
				t.Logf("%s refused: %s", denied.name, strings.TrimSpace(string(out)))
			}
			t.Log("selected socket/TCP connections passed; all eight other connection probes refused")
			docker(t, "rm", "-fv", container)
			docker(t, "volume", "rm", volume)
		})
	}
	// These assertions run before the fixture cleanup, so a direct caller
	// cannot rely on verify.sh or removeDrillResources to hide its leaks.
	t.Run("direct_failure_cleanup", func(t *testing.T) {
		container := names.Prefix + "-past-end"
		volume := container + "-pgdata"
		cmd := exec.CommandContext(ctx, "bash", filepath.Join(root, "deploy/backup/restore-pitr.sh"),
			"--target-lsn", "FFFFFFFF/FFFFFFFF", "--isolated-container", container, "--target-volume", volume,
			"--repo-volume", names.RepoVolume, "--image", names.Image, "--db-user", "postgres")
		cmd.Env = append(os.Environ(), restoreEnv...)
		out, err := cmd.CombinedOutput()
		assertNoHook(t, "failed restore output", string(out))
		if err == nil || !strings.Contains(string(out), "recovery ended before configured recovery target was reached") || strings.Contains(string(out), "timed out") {
			t.Fatalf("expected PostgreSQL unreachable-target failure after restore: %v %s", err, out)
		}
		for _, name := range []string{container, container + "-init"} {
			if got := strings.TrimSpace(docker(t, "ps", "-a", "--filter", "name=^/"+name+"$", "--format", "{{.Names}}")); got != "" {
				t.Errorf("direct failed restore leaked container %s before test cleanup", got)
			}
		}
		if got := strings.TrimSpace(docker(t, "volume", "ls", "--filter", "name=^"+volume+"$", "--format", "{{.Name}}")); got != "" {
			t.Errorf("direct failed restore leaked volume %s before test cleanup", got)
		}
		t.Logf("unreachable LSN failure and own cleanup checked before test cleanup:\n%s", out)
	})
	t.Run("preexisting_volume", func(t *testing.T) {
		container := names.Prefix + "-nonempty"
		volume := container + "-pgdata"
		docker(t, "volume", "create", volume)
		docker(t, "run", "--rm", "--name", names.Prefix+"-seed", "--network", "none", "--entrypoint", "bash", "-v", volume+":/data", names.Image,
			"-ceu", "printf preserved > /data/marker")
		cmd := exec.CommandContext(ctx, "bash", filepath.Join(root, "deploy/backup/restore-pitr.sh"),
			"--latest", "--isolated-container", container, "--target-volume", volume,
			"--repo-volume", names.RepoVolume, "--image", names.Image, "--db-user", "postgres")
		cmd.Env = append(os.Environ(), restoreEnv...)
		out, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(out), "refusing non-empty target volume") {
			t.Fatalf("expected nonempty target refusal: %v %s", err, out)
		}
		docker(t, "volume", "inspect", volume)
		contents := docker(t, "run", "--rm", "--name", names.Prefix+"-read", "--network", "none", "--entrypoint", "cat", "-v", volume+":/data:ro", names.Image, "/data/marker")
		if contents != "preserved" {
			t.Fatalf("pre-existing volume contents changed: %q", contents)
		}
		for _, name := range []string{container, container + "-init"} {
			if got := strings.TrimSpace(docker(t, "ps", "-a", "--filter", "name=^/"+name+"$", "--format", "{{.Names}}")); got != "" {
				t.Errorf("nonempty refusal leaked container %s", got)
			}
		}
		t.Log("nonempty volume refused; pre-existing volume and marker preserved before test cleanup")
		docker(t, "volume", "rm", volume)
	})
	if after := digests(); before != after {
		t.Fatal("repository changed during targeted restores")
	}
}
