//go:build e2e

package e2e

import (
	"context"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (r *runner) gatewayPrivacy(ctx context.Context) {
	rec := Record{ID: "gateway-privacy", Title: "Gateway diagnostics exclude transient authentication query values", Spec: []string{"§17.2", "§19 AT28", "SECOPS-16"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	sentinel := "TEST-ONLY-" + uuid.NewString()
	client := &http.Client{Transport: r.lab.transport, Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", r.lab.netbird+"/oauth2/e2e-log-fault?user_code="+sentinel, nil)
	resp, err := client.Do(req)
	if err != nil {
		note("replay", "transport failed")
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	note("upstream_fault_status", itoa(resp.StatusCode))
	logs, err := exec.CommandContext(ctx, "docker", "logs", r.lab.prefix+"-gateway").CombinedOutput()
	if err != nil {
		note("logs", "unavailable")
		return
	}
	valueHits := strings.Count(string(logs), sentinel)
	queryHits := strings.Count(string(logs), "user_code=")
	budgetHits := strings.Count(string(logs), "worker_connections are not enough")
	note("sentinel_value_matches", itoa(valueHits))
	note("query_key_matches", itoa(queryHits))
	note("connection_budget_errors", itoa(budgetHits))
	note("safe_502_diagnostic", boolText(strings.Contains(string(logs), "netbird.example.invalid GET 502")))
	note("configured_budget", "2 workers x 4096 connections; 60 native peers plus service and login traffic")
	if resp.StatusCode == 502 && valueHits == 0 && queryHits == 0 && budgetHits == 0 && strings.Contains(string(logs), "netbird.example.invalid GET 502") {
		rec.Status = StatusPass
	}
}
