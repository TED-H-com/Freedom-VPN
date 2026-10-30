//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func (r *runner) backup(ctx context.Context, admin *session) {
	rec := Record{ID: "backup", Title: "Archived WAL and isolated physical PITR", Spec: []string{"§17.2", "§17.3", "§19 AT30"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	ctx, cancel := context.WithTimeout(ctx, 6*time.Minute)
	defer cancel()
	started := time.Now()
	primary, restored, volume, repo := r.lab.prefix+"-postgres", r.lab.prefix+"-restore-verify", r.lab.prefix+"-restore-pg", r.lab.prefix+"-pgbackrest-repo"
	// Only this run's prefix; no host networking or shared PostgreSQL access.
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 20*time.Second)
		defer stop()
		_ = exec.CommandContext(cleanup, "docker", "rm", "-f", restored, restored+"-init").Run()
		_ = exec.CommandContext(cleanup, "docker", "volume", "rm", volume).Run()
	}()
	command := func(label string, limit time.Duration, name string, args ...string) ([]byte, bool) {
		c, stop := context.WithTimeout(ctx, limit)
		defer stop()
		begin := time.Now()
		cmd := exec.CommandContext(c, name, args...)
		cmd.Dir = r.lab.root
		out, err := cmd.CombinedOutput()
		note(label+"_milliseconds", fmt.Sprint(time.Since(begin).Milliseconds()))
		if err != nil {
			note(label+"_error", clipText(string(out), 1200))
			return out, false
		}
		note(label+"_exit", "0")
		return out, true
	}
	marker := func(label string) uuid.UUID {
		resp, err := admin.call(ctx, "POST", r.lab.public+"/api/v1/admin/enrollments", map[string]string{"delivery": "installer", "name": "TEST ONLY PITR " + label, "region": "lab"}, map[string]string{"Idempotency-Key": uuid.NewString()})
		note("enrollment_"+label, httpDetail(resp, err))
		var created api.EnrollmentCreated
		if err != nil || resp.Status != 201 || json.Unmarshal(resp.Body, &created) != nil {
			return uuid.Nil
		}
		return created.Enrollment.NodeId
	}
	before := marker("before")
	if before == uuid.Nil {
		return
	}
	if _, ok := command("archive_check", 90*time.Second, "deploy/backup/backup.sh", "--container", primary, "check"); !ok {
		return
	}
	if _, ok := command("physical_full", 150*time.Second, "deploy/backup/backup.sh", "--container", primary, "full"); !ok {
		return
	}
	inventory, ok := command("repository_info", 10*time.Second, "deploy/backup/backup.sh", "--container", primary, "info")
	if !ok {
		return
	}
	var stanzas []struct {
		Backup []struct {
			Label     string `json:"label"`
			Type      string `json:"type"`
			Timestamp struct {
				Stop int64 `json:"stop"`
			} `json:"timestamp"`
		} `json:"backup"`
	}
	if json.Unmarshal(inventory, &stanzas) != nil || len(stanzas) != 1 || len(stanzas[0].Backup) == 0 {
		note("backup_inventory", "invalid")
		return
	}
	b := stanzas[0].Backup[len(stanzas[0].Backup)-1]
	note("backup_label", b.Label)
	note("backup_type", b.Type)
	conn, err := r.lab.db(ctx)
	if err != nil {
		return
	}
	defer conn.Close(ctx)
	// A target strictly after the full backup, then an API write strictly after
	// that target. PostgreSQL time determines the boundary, not a host clock guess.
	wait, stop := context.WithTimeout(ctx, 5*time.Second)
	ok = waitUntil(wait, 100*time.Millisecond, func() bool {
		var after bool
		return conn.QueryRow(ctx, `SELECT clock_timestamp()>to_timestamp($1)+interval '1 second'`, b.Timestamp.Stop).Scan(&after) == nil && after
	})
	stop()
	if !ok {
		return
	}
	var target string
	if conn.QueryRow(ctx, `SELECT to_char(clock_timestamp() AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`).Scan(&target) != nil {
		return
	}
	note("target_utc", target)
	after := marker("after")
	if after == uuid.Nil {
		return
	}
	var primaryCount int
	if conn.QueryRow(ctx, `SELECT count(*) FROM nodes WHERE id=ANY($1::uuid[])`, []uuid.UUID{before, after}).Scan(&primaryCount) != nil {
		return
	}
	note("primary_marker_count", itoa(primaryCount))
	if _, ok = command("wal_switch_archive", 90*time.Second, "deploy/backup/backup.sh", "--container", primary, "wal-switch"); !ok {
		return
	}
	image, ok := command("primary_image", 5*time.Second, "docker", "inspect", "--format", "{{.Image}}", primary)
	if !ok {
		return
	}
	if _, ok = command("physical_pitr_restore", 150*time.Second, "deploy/backup/restore-pitr.sh", "--image", strings.TrimSpace(string(image)), "--repo-volume", repo, "--target-volume", volume, "--isolated-container", restored, "--primary-container", primary, "--target-time", target, "--db-user", "fvpn", "--db-name", "fvpn"); !ok {
		return
	}
	query := fmt.Sprintf("SELECT (SELECT count(*) FROM nodes WHERE id='%s'),(SELECT count(*) FROM nodes WHERE id='%s'),pg_is_in_recovery(),current_setting('archive_mode');", before, after)
	rows, ok := command("restored_readback", 5*time.Second, "docker", "exec", restored, "psql", "-U", "fvpn", "-d", "fvpn", "-Atc", query)
	if !ok {
		return
	}
	got := strings.TrimSpace(string(rows))
	note("restored_before_after_recovery_archive", got)
	isolation, ok := command("restore_isolation", 5*time.Second, "docker", "inspect", "--format", "{{.HostConfig.NetworkMode}}|{{len .HostConfig.PortBindings}}", restored)
	if !ok {
		return
	}
	isolated := strings.TrimSpace(string(isolation))
	note("network_and_published_ports", isolated)
	note("total_milliseconds", fmt.Sprint(time.Since(started).Milliseconds()))
	note("scope", "Physical backup + WAL replay; API-created pre-target enrollment retained, post-target enrollment absent. AT30 ledger freeze/recovery remains with at-secops.")
	if primaryCount == 2 && got == "1|0|f|off" && isolated == "none|0" && b.Type == "full" {
		rec.Status = StatusPass
	}
}

func (r *runner) metrics(ctx context.Context) {
	rec := Record{ID: "control-metrics", Title: "Control loopback metrics listener", Spec: []string{"§17.1"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	note("config_metrics_address", "127.0.0.1:9091")
	note("compose_override", "none; worker alone overrides FVPN_METRICS_ADDRESS=:9092")
	probe, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probe, "docker", "exec", r.lab.control, "wget", "--server-response", "--timeout=3", "--tries=1", "-O-", "http://127.0.0.1:9091/metrics")
	raw, err := cmd.CombinedOutput()
	status := ""
	count := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "HTTP/") {
			status = strings.TrimSpace(line)
		}
		if strings.HasPrefix(line, "# HELP ") {
			count++
		}
	}
	note("response", status)
	note("metric_families", itoa(count))
	if err == nil && strings.Contains(status, "200") && count > 0 {
		rec.Status = StatusPass
	} else {
		rec.Status = StatusBlocked
		note("owner", "apps/control: loopback metrics listener")
		note("probe_error", clipText(string(raw), 500))
	}
}
