package backup

import (
	"strings"
	"testing"
	"time"
)

// postgresTwoReadyLinesLog is the docker-entrypoint sequence for a fresh data
// directory: the temporary server logs postgresReadyToAccept, shuts down, init
// completes, and the real server logs postgresReadyToAccept again.
const postgresTwoReadyLinesLog = `The files belonging to this database system will be owned by user "postgres".
This user must also own the server process.
initdb: warning: enabling "trust" authentication for local connections

waiting for server to start....2026-10-29 22:17:01.140 UTC [48] LOG:  database system is ready to accept connections
 done
server started
/usr/local/bin/docker-entrypoint.sh: running /docker-entrypoint-initdb.d/10-pgbackrest-stanza.sh

waiting for server to shut down....2026-10-29 22:17:02.010 UTC [49] LOG:  database system is shutting down
2026-10-29 22:17:02.210 UTC [48] LOG:  database system is shut down
 done
server stopped

PostgreSQL init process complete; ready for start up.

2026-10-29 22:17:02.330 UTC [1] LOG:  database system is ready to accept connections
`

// postgresRestoredReadyLog is a promoted PITR container. PGDATA already exists,
// so there is no initdb temporary server. Recovery logs the read-only line,
// then promote logs the final ready line.
const postgresRestoredReadyLog = `PostgreSQL Database directory appears to contain a database; Skipping initialization

2026-10-29 22:20:01.100 UTC [1] LOG:  starting PostgreSQL 17.6
2026-10-29 22:20:01.200 UTC [1] LOG:  listening on Unix socket "/var/run/postgresql/.s.PGSQL.5432"
2026-10-29 22:20:01.300 UTC [30] LOG:  database system is ready to accept read-only connections
2026-10-29 22:20:02.300 UTC [1] LOG:  archive recovery complete
2026-10-29 22:20:02.400 UTC [1] LOG:  database system is ready to accept connections
`

func TestAT30_PostgresFinalServerReady(t *testing.T) {
	t.Parallel()

	if strings.Count(postgresTwoReadyLinesLog, postgresReadyToAccept) != 2 {
		t.Fatalf("two-ready fixture contains %d ready lines, want 2", strings.Count(postgresTwoReadyLinesLog, postgresReadyToAccept))
	}
	initAt := strings.Index(postgresTwoReadyLinesLog, postgresInitComplete)
	secondReady := strings.LastIndex(postgresTwoReadyLinesLog, postgresReadyToAccept)
	if initAt < 0 || secondReady < initAt {
		t.Fatal("fixture must place the second ready line after init completion")
	}
	restoredFinal := strings.LastIndex(postgresRestoredReadyLog, postgresReadyToAccept)
	if restoredFinal < 0 {
		t.Fatal("restored fixture missing final ready line")
	}

	tests := []struct {
		name string
		logs string
		want bool
	}{
		{name: "empty", logs: "", want: false},
		{name: "temp server ready before init completes", logs: postgresTwoReadyLinesLog[:initAt], want: false},
		{name: "init complete without the final ready line", logs: postgresTwoReadyLinesLog[:secondReady], want: false},
		{name: "two ready lines and the final server stays up", logs: postgresTwoReadyLinesLog, want: true},
		{
			name: "final server shut down after the second ready line",
			logs: postgresTwoReadyLinesLog + "LOG:  database system is shut down\n",
			want: false,
		},
		{
			name: "final server shutting down after the second ready line",
			logs: postgresTwoReadyLinesLog + "LOG:  database system is shutting down\n",
			want: false,
		},
		{name: "restored cluster after promote", logs: postgresRestoredReadyLog, want: true},
		{name: "restored cluster still read-only", logs: postgresRestoredReadyLog[:restoredFinal], want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := postgresFinalServerReady(tt.logs); got != tt.want {
				t.Fatalf("postgresFinalServerReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAT30_PostgresReadyTimeoutBound(t *testing.T) {
	t.Parallel()
	if postgresReadyTimeout <= 0 || postgresReadyTimeout > 120*time.Second {
		t.Fatalf("readiness deadline = %s, want a positive bound of at most 120s", postgresReadyTimeout)
	}
	if postgresReadyConfirmGap != time.Second {
		t.Fatalf("confirm gap = %s, want 1s", postgresReadyConfirmGap)
	}
}
