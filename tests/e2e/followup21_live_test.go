//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestE2E_Followup21DNS(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP21") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP21=1 with a fresh two-resolver stack")
	}
	runFixtureCampaign(t, true)
}

func TestE2E_Followup21Installer(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP21") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP21=1 with an owned stack")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Minute)
	defer cancel()
	head, err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	r := &runner{t: t, lab: l, runID: id, sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), "followup21", id)}
	defer r.finish()
	s := &Stack{lab: l}
	admin := r.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		return
	}
	b := BrowserSession{Client: admin.http, CSRF: admin.csrf}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(s.PrepareInstallerNode(ctx))
	releases, err := s.RegisterInstallerReleases(ctx, b)
	must(err)
	ids := []uuid.UUID{releases[0].ID, releases[1].ID}
	_, _, err = s.ActivatePolicy(ctx, b, "release_support", "installer-loopback", map[string]any{"approved_release_ids": ids, "revoked_release_ids": []string{}, "minimum_serving_version": map[string]any{"serving_release_ids": ids}})
	must(err)
	revision, _, err := s.ActivatePolicy(ctx, b, "domain_rule_set", "installer-loopback-rules", map[string]any{"rules": []any{map[string]any{"rule_id": uuid.New(), "domain": "feed-01.example.invalid", "include_subdomains": true, "action": "block", "profile_id": nil, "reason": "admin_rule", "revision": "1"}}})
	must(err)
	r.profileID, _, err = s.ActivatePolicy(ctx, b, "dns_profile", "installer-loopback-dns", map[string]any{"profile_code": "installer-loopback", "display_name": map[string]string{"zh-TW": "TEST ONLY installer proxy"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []uuid.UUID{revision}, "threat_feed_revision_ids": []string{}})
	must(err)
	r.catalogue(ctx, admin)
	rec := Record{ID: "installer-loopback-wrapper", Title: "TEST ONLY shared install wrapper and generated authenticated reporting", Spec: []string{"ENR02", "UPD05", "AT07", "D-102"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	token, _, err := r.issueInstallerToken(ctx, s, b, "TEST ONLY loopback wrapper", uuid.Nil)
	must(err)
	must(s.StageInstallerToken(ctx, "installer-node", token))
	token = ""
	installed, err := s.RerunInstaller(ctx, "installer-node")
	noteJSON(&rec, "install", installed)
	if err != nil || installed.Exit != 0 {
		recPathNote(&rec)("error", fmt.Sprint(err))
		return
	}
	defer func() {
		must(s.RestoreInstallerServices(context.WithoutCancel(ctx), "installer-node", installed.Services))
	}()
	raw, err := l.docker(s.installerContainer(), "cat", "/etc/vpn-node/updater.json")
	must(err)
	// Only the generated reporting fields, never the whole identity/config file.
	var cfg struct {
		ResultURL     string   `json:"result_url"`
		Origins       []string `json:"allowed_artifact_origins"`
		AllowLoopback bool     `json:"allow_loopback"`
		Cert          string   `json:"mtls_cert"`
		Key           string   `json:"mtls_key"`
		CA            string   `json:"mtls_ca"`
	}
	must(json.Unmarshal(raw, &cfg))
	noteJSON(&rec, "generated_reporting", cfg)
	before := sha256.Sum256(raw)
	recPathNote(&rec)("config_sha256_before", hex.EncodeToString(before[:]))
	expected := "https://127.0.0.1:9444/api/v1/agent/update-results"
	if cfg.ResultURL != expected || !cfg.AllowLoopback || cfg.CA == "" || cfg.Cert == "" || cfg.Key == "" || !slices.Contains(cfg.Origins, "https://127.0.0.1:9444") {
		return
	}
	curl := []string{"curl", "--silent", "--show-error", "--max-time", "10", "--cacert", cfg.CA, "-o", "/dev/null", "-w", "%{http_code}", "-H", "Content-Type: application/json", "-H", "Idempotency-Key: TEST-ONLY-invalid-receipt", "--data", "{}", cfg.ResultURL}
	unauth, e1 := l.docker(s.installerContainer(), curl...)
	authenticated := append(append([]string{}, curl...), "--cert", cfg.Cert, "--key", cfg.Key)
	auth, e2 := l.docker(s.installerContainer(), authenticated...)
	noteJSON(&rec, "endpoint_authentication", map[string]any{"without_client_certificate": string(unauth), "with_client_certificate_invalid_body": string(auth), "invalid_body": "empty object; no receipt created"})
	after, e3 := l.docker(s.installerContainer(), "cat", "/etc/vpn-node/updater.json")
	sum := sha256.Sum256(after)
	recPathNote(&rec)("config_sha256_after", hex.EncodeToString(sum[:]))
	if e1 == nil && e2 == nil && e3 == nil && string(unauth) == "403" && string(auth) == "400" && bytes.Equal(raw, after) {
		rec.Status = StatusPass
	}
}

// Check the delivered diagnostic separately from the unchanged DNS assertions.
// Fallback deliberately stops a relay; only the subsequent full matrix requires
// all three lifecycle files. Failed/incomplete first-attempt artifacts survive.
func (r *runner) verifyDNSHistory21() {
	rec := Record{ID: "dns-history-integrity", Title: "Whole-sequence authority lifecycle and explicit relay streams", Spec: []string{"DNS02", "AT20", "D-102"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	raw, err := os.ReadFile(filepath.Join(r.dir, "dns-negative.json"))
	if err != nil {
		recPathNote(&rec)("error", err.Error())
		return
	}
	var source Record
	if json.Unmarshal(raw, &source) != nil {
		return
	}
	count, complete := 0, 0
	joins := map[string]int{}
	var walk func(any)
	walk = func(value any) {
		switch value := value.(type) {
		case map[string]any:
			if trace, ok := value["dns_trace"].(map[string]any); ok {
				count++
				if trace["trace_complete"] == true {
					complete++
				}
				if history, ok := trace["authority_history"].([]any); ok {
					for _, item := range history {
						request, ok := item.(map[string]any)
						if !ok {
							continue
						}
						start, ok := request["start"].(map[string]any)
						if !ok {
							continue
						}
						id, _ := start["request_id"].(string)
						if id != "" {
							joins[id]++
						}
					}
				}
			}
			for _, v := range value {
				walk(v)
			}
		case []any:
			for _, v := range value {
				walk(v)
			}
		}
	}
	for _, observation := range source.Observations {
		var value any
		if json.Unmarshal([]byte(observation.Detail), &value) == nil {
			walk(value)
		}
	}
	repeated := 0
	for _, n := range joins {
		if n > 1 {
			repeated++
		}
	}
	noteJSON(&rec, "diagnostic_counts", map[string]int{"queries": count, "complete": complete, "distinct_authority_requests": len(joins), "requests_retained_across_queries": repeated})
	if count == 36 && complete == count && repeated > 0 {
		rec.Status = StatusPass
	}
}
