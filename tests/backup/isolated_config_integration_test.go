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
	docker := func(args ...string) string { return runCmd(ctx, t, root, nil, "docker", args...) }
	sql := func(container, query string) string {
		return strings.TrimSpace(docker("exec", "--user", "postgres", container,
			"psql", "-X", "-U", "postgres", "-d", "fvpn", "-v", "ON_ERROR_STOP=1", "-tAc", query))
	}
	backup := func(action string) string {
		return runCmd(ctx, t, root, nil, "bash", "deploy/backup/backup.sh", "--container", names.PrimaryContainer, "--db-user", "postgres", action)
	}
	docker("build", "-t", names.Image, filepath.Join(root, "deploy/backup/postgres"))
	work := t.TempDir()
	migrate := filepath.Join(work, "fvpn-migrate")
	runCmd(ctx, t, root, []string{"GOOS=linux", "GOARCH=amd64"}, "go", "build", "-trimpath", "-o", migrate, "./internal/db/cmd/fvpn-migrate")
	docker("volume", "create", names.PrimaryVolume)
	docker("volume", "create", names.RepoVolume)
	docker("run", "-d", "--name", names.PrimaryContainer, "--network", "none",
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
	docker("exec", "--user", "postgres", names.PrimaryContainer, "createdb", "fvpn")
	docker("cp", migrate, names.PrimaryContainer+":/usr/local/bin/fvpn-migrate")
	docker("exec", "--user", "postgres", "-e", "FVPN_DATABASE_URL=postgres:///fvpn?host=/var/run/postgresql&port=55432", names.PrimaryContainer, "fvpn-migrate", "up")
	// A runtime-only password proves the hostssl/SCRAM rule works. Neither it
	// nor the generated TLS key is copied into PGDATA or returned to test logs.
	docker("exec", "--user", "postgres", names.PrimaryContainer, "bash", "-ceu", `
password=$(openssl rand -hex 24)
printf "CREATE ROLE fvpn_app LOGIN PASSWORD '%s';\n" "$password" | psql -U postgres -d fvpn -v ON_ERROR_STOP=1 >/dev/null
PGPASSWORD="$password" psql 'host=127.0.0.1 port=55432 user=fvpn_app dbname=fvpn sslmode=require' -tAc 'SELECT 1' | grep -qx 1
`)
	primarySettings := sql(names.PrimaryContainer, `SELECT name || '=' || setting FROM pg_settings WHERE name IN
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
	} {
		sql(names.PrimaryContainer, "ALTER SYSTEM SET "+setting)
	}
	sql(names.PrimaryContainer, "CREATE TABLE _backup_drill (marker text PRIMARY KEY); INSERT INTO _backup_drill VALUES ('A')")
	backup("stanza-create")
	_, _, _, stop := parsePgBackRestInfo(t, backup("full"))
	for time.Now().Unix() <= stop {
		time.Sleep(100 * time.Millisecond)
	}
	sql(names.PrimaryContainer, "INSERT INTO _backup_drill VALUES ('M')")
	targetTime := sql(names.PrimaryContainer, `SELECT to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`)
	// Exact record boundary used for exclusive LSN recovery before B.
	targetLSN := sql(names.PrimaryContainer, "SELECT pg_current_wal_insert_lsn()")
	sql(names.PrimaryContainer, "INSERT INTO _backup_drill VALUES ('B')")
	backup("wal-switch")
	docker("exec", names.PrimaryContainer, "bash", "-ceu", `for f in postgresql.conf pg_hba.conf pg_ident.conf; do test ! -e "$PGDATA/$f"; done`)
	t.Log("T1: primary PGDATA contains none of postgresql.conf, pg_hba.conf, pg_ident.conf; external TLS/SCRAM connection passed")
	// There are no further primary writes, and archive_timeout remains zero.
	// Compare every repository object's name and digest, not just manifests.
	digests := func() string {
		return docker("exec", names.PrimaryContainer, "bash", "-ceu", `cd /var/lib/pgbackrest; find . -type f -print0 | sort -z | xargs -0 sha256sum`)
	}
	before := digests()
	verifyCmd := exec.CommandContext(ctx, "bash", filepath.Join(root, "deploy/backup/verify.sh"),
		"--repo-volume", names.RepoVolume, "--primary-container", names.PrimaryContainer, "--image", names.Image,
		"--verify-prefix", names.VerifyPrefix, "--migrate-bin", migrate, "--db-user", "postgres")
	verifyBytes, verifyErr := verifyCmd.CombinedOutput()
	verifyOut := string(verifyBytes)
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
	t.Log("T5: every repository object name and SHA-256 digest unchanged after verify")

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
			runCmd(ctx, t, root, nil, "bash", args...)
			got := sql(container, "SELECT string_agg(marker, ',' ORDER BY marker) FROM _backup_drill")
			if got != target.want {
				t.Fatalf("markers %s, want %s", got, target.want)
			}
			t.Logf("T4 %s markers: %s", target.name, got)
			settings := sql(container, `SELECT name || '=' || setting || ' source=' || source FROM pg_settings WHERE name IN
('config_file','hba_file','ident_file','port','listen_addresses','unix_socket_directories','ssl','archive_mode','archive_command','archive_library','shared_preload_libraries','max_connections','max_worker_processes','max_wal_senders','max_prepared_transactions','max_locks_per_transaction') ORDER BY name`)
			for _, want := range []string{
				"config_file=/etc/postgresql/fvpn-isolated/postgresql.conf", "hba_file=/etc/postgresql/fvpn-isolated/pg_hba.conf", "ident_file=/etc/postgresql/fvpn-isolated/pg_ident.conf",
				"port=5432", "listen_addresses=127.0.0.1", "unix_socket_directories=/var/run/postgresql", "ssl=off", "archive_mode=off", "archive_command=(disabled)", "archive_library=", "shared_preload_libraries=",
				"max_connections=240", "max_worker_processes=16", "max_wal_senders=20", "max_prepared_transactions=12", "max_locks_per_transaction=128",
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
			if got := sql(container, "SELECT reset_val = '' FROM pg_settings WHERE name = 'archive_command'"); got != "t" {
				t.Fatalf("archive_command reset value is not empty: %s", got)
			}
			for _, parameter := range []string{"config_file", "hba_file", "port", "listen_addresses", "ssl", "archive_mode", "archive_command"} {
				t.Logf("SHOW %s: %s", parameter, sql(container, "SHOW "+parameter))
			}
			if got := strings.TrimSpace(docker("inspect", "--format", "{{.HostConfig.NetworkMode}} {{len .NetworkSettings.Ports}}", container)); got != "none 0" {
				t.Fatalf("isolation: %s", got)
			}
			if got := strings.TrimSpace(docker("inspect", "--format", `{{range .Mounts}}{{if eq .Destination "/var/lib/pgbackrest"}}{{.RW}}{{end}}{{end}}`, container)); got != "false" {
				t.Fatalf("repository writable: %s", got)
			}
			// Exercise both allowed transports and refuse other roles, databases,
			// OS identities, TLS, IPv6, and physical replication.
			docker("exec", container, "psql", "postgres://postgres@127.0.0.1:5432/fvpn?sslmode=disable", "-tAc", "SELECT 1")
			for _, denied := range []struct{ osUser, dsn string }{
				{"root", "user=postgres dbname=fvpn"},
				{"postgres", "user=postgres dbname=postgres"},
				{"postgres", "user=fvpn_app dbname=fvpn"},
				{"postgres", "host=127.0.0.1 user=fvpn_app dbname=fvpn sslmode=disable"},
				{"postgres", "host=127.0.0.1 user=postgres dbname=postgres sslmode=disable"},
				{"postgres", "host=127.0.0.1 user=postgres dbname=fvpn sslmode=require"},
				{"postgres", "host=::1 user=postgres dbname=fvpn sslmode=disable"},
				{"postgres", "host=127.0.0.1 user=postgres replication=true sslmode=disable"},
			} {
				out, err := exec.CommandContext(ctx, "docker", "exec", "--user", denied.osUser, container, "psql", "-w", denied.dsn, "-tAc", "SELECT 1").CombinedOutput()
				if err == nil {
					t.Fatalf("unexpected connection accepted %s %s: %s", denied.osUser, denied.dsn, out)
				}
			}
			t.Log("selected socket/TCP connections passed; all eight other connection probes refused")
			docker("rm", "-fv", container)
			docker("volume", "rm", volume)
		})
	}
	if after := digests(); before != after {
		t.Fatal("repository changed during targeted restores")
	}
}
