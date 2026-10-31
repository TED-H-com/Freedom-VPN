//go:build e2e

package e2e

import (
	"context"
	"os/exec"
	"time"

	"github.com/jackc/pgx/v5"
)

// workerRestart kills the worker process inside the container. unless-stopped
// must bring that process back within 30s, and the new process must deliver or
// claim an outbox event. Readiness is the in-container /readyz endpoint, not
// docker inspect.
func (r *runner) workerRestart(ctx context.Context) {
	rec := Record{ID: "worker-restart", Title: "Worker process crash restarts and claims events", Spec: []string{"§17"}, Status: StatusFail}
	note := recPathNote(&rec)
	defer func() { r.record(rec) }()
	worker := r.lab.prefix + "-worker"
	conn, err := r.lab.db(ctx)
	if err != nil {
		note("database", "unavailable")
		return
	}
	defer conn.Close(ctx)
	kill := exec.CommandContext(ctx, "docker", "exec", worker, "sh", "-c", "kill -KILL 1")
	_ = kill.Run()
	// The clock and the attempt snapshot are taken after SIGKILL returns, so a
	// delivery or claim from the dead process is not the proof.
	var mark time.Time
	if err = conn.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&mark); err != nil {
		note("database", "clock unavailable")
		return
	}
	baseline, err := outboxAttempts(ctx, conn)
	if err != nil {
		note("database", "attempt snapshot failed")
		return
	}
	wait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	ready := false
	deliveries := 0
	claims := 0
	back := waitUntil(wait, time.Second, func() bool {
		_, err := r.lab.docker(worker, "wget", "-q", "-O", "/dev/null", "--timeout=2", "--tries=1", "http://127.0.0.1:9092/readyz")
		ready = err == nil
		if scanErr := conn.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE delivered_at > $1`, mark).Scan(&deliveries); scanErr != nil {
			deliveries = 0
		}
		claims = outboxClaimAdvances(ctx, conn, baseline)
		return ready && (deliveries > 0 || claims > 0)
	})
	note("readyz", boolText(ready))
	note("deliveries_after_kill", itoa(deliveries))
	note("claims_after_kill", itoa(claims))
	note("within_30s", boolText(back))
	if !ready {
		note("owner", "w3-devstack-e2e: worker restart unless-stopped did not serve /readyz within 30s")
		return
	}
	if deliveries == 0 && claims == 0 {
		note("owner", "w3-wire-nodes: worker became ready but neither delivered nor claimed an outbox event within 30s of the process kill")
		return
	}
	rec.Status = StatusPass
	note("note", "SIGKILL of pid 1; /readyz on 127.0.0.1:9092; a post-kill delivery or a new claim")
}

func outboxAttempts(ctx context.Context, conn *pgx.Conn) (map[string]int, error) {
	rows, err := conn.Query(ctx, `SELECT id::text, attempt FROM outbox_events WHERE delivered_at IS NULL AND dead_lettered_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var attempt int
		if err = rows.Scan(&id, &attempt); err != nil {
			return nil, err
		}
		out[id] = attempt
	}
	return out, rows.Err()
}

func outboxClaimAdvances(ctx context.Context, conn *pgx.Conn, baseline map[string]int) int {
	rows, err := conn.Query(ctx, `SELECT id::text, attempt FROM outbox_events WHERE delivered_at IS NULL AND dead_lettered_at IS NULL AND lease_until > clock_timestamp()`)
	if err != nil {
		return 0
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id string
		var attempt int
		if rows.Scan(&id, &attempt) != nil {
			continue
		}
		prev, ok := baseline[id]
		if !ok || attempt > prev {
			n++
		}
	}
	return n
}
