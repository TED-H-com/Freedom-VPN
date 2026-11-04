//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestD141_ClonePreparedConfigsDialContainerResolverProxy(t *testing.T) {
	base := t.TempDir()
	for _, dir := range []string{"etc", "var/e2e", "var/identity", "var/state"} {
		if err := os.MkdirAll(filepath.Join(base, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	sourceEnroll, err := json.MarshalIndent(map[string]any{
		"agent_release":             "0.1.0",
		"boot_id_file":              "/etc/vpn-node/boot-id",
		"control_url":               "https://portal.example.invalid:8443",
		"datapath":                  map[string]string{"member_ingress": "fvmb0", "snat": "198.51.100.20", "uplink": "fvup0"},
		"heartbeat_seconds":         5,
		"host_reserve_bps":          1000000,
		"identity_dir":              "/var/lib/vpn-node/identity",
		"journal_dir":               "/var/lib/vpn-node/update-journal",
		"lock_path":                 "/var/lib/vpn-node/update.lock",
		"netbird_backend":           "kernel",
		"observation_seconds":       5,
		"poll_seconds":              5,
		"release_id":                "00000000-0000-7000-8000-000000000001",
		"server_ca_file":            "/etc/vpn-node/ca.pem",
		"state_dir":                 "/var/lib/vpn-node/state",
		"test_mode":                 true,
		"trust_bundle":              "/etc/vpn-node/trust.json",
		"updater_config":            "/etc/vpn-node/updater-recover.json",
		"updater_recover":           "/usr/local/bin/fvpn-updater",
		"watchdog_interval_seconds": 5,
		"watchdog_stale_seconds":    30,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	sourceRun, err := json.MarshalIndent(map[string]any{
		"agent_release": "0.1.0",
		"boot_id_file":  "/etc/vpn-node/boot-id",
		"control_url":   "https://127.0.0.1:9445",
		"datapath":      map[string]string{"member_ingress": "fvmb0", "snat": "198.51.100.20", "uplink": "fvup0"},
		"dns": map[string]any{
			"resolver_credentials_file": "/etc/vpn-node/resolver-credentials.json",
			"resolvers": []any{map[string]string{
				"admin_url": "http://127.0.0.1:8080",
				"dns_addr":  "127.0.0.1:53",
				"id":        "lab",
			}},
		},
		"heartbeat_seconds":         1,
		"host_reserve_bps":          1000000,
		"identity_dir":              "/var/lib/vpn-node/identity",
		"journal_dir":               "/var/lib/vpn-node/update-journal",
		"lock_path":                 "/var/lib/vpn-node/update.lock",
		"netbird_backend":           "kernel",
		"observation_seconds":       1,
		"poll_seconds":              1,
		"release_id":                "00000000-0000-7000-8000-000000000001",
		"server_ca_file":            "/etc/vpn-node/ca.pem",
		"state_dir":                 "/var/lib/vpn-node/state",
		"test_mode":                 true,
		"test_timing_path":          "/var/lib/vpn-node/e2e/isolation-operations.jsonl",
		"trust_bundle":              "/etc/vpn-node/trust.json",
		"updater_config":            "/etc/vpn-node/updater-recover.json",
		"updater_recover":           "/usr/local/bin/fvpn-updater",
		"watchdog_interval_seconds": 5,
		"watchdog_stale_seconds":    30,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	enrollPath := filepath.Join(base, "etc", "agent-enroll.json")
	runPath := filepath.Join(base, "etc", "agent-run.json")
	releasePath := filepath.Join(base, "var", "e2e", "agent-00000000-0000-7000-8000-000000000001.json")
	for path, body := range map[string][]byte{
		enrollPath:  sourceEnroll,
		runPath:     sourceRun,
		releasePath: sourceRun,
	} {
		if err := os.WriteFile(path, body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	durableFiles := map[string][]byte{
		filepath.Join(base, "etc", "machine-id"):            []byte("0123456789abcdef0123456789abcdef\n"),
		filepath.Join(base, "var", "identity", "node.json"): []byte("{\"node_id\":\"00000000-0000-7000-8000-000000000002\"}\n"),
		filepath.Join(base, "var", "identity", "cert.pem"):  []byte("TEST ONLY - NOT A SECRET cert\n"),
		filepath.Join(base, "var", "identity", "key.pem"):   []byte("TEST ONLY - NOT A SECRET key\n"),
		filepath.Join(base, "var", "state", "agent.db"):     []byte("TEST ONLY durable sqlite state bytes"),
		filepath.Join(base, "var", "e2e", "agent-selected"): []byte("/usr/local\n"),
	}
	for path, body := range durableFiles {
		if err := os.WriteFile(path, body, 0600); err != nil {
			t.Fatal(err)
		}
	}

	if err := prepareExitSlotConfigs(base, "198.51.100.23", true); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name        string
		path        string
		sourceBytes []byte
		sourceURL   string
	}{
		{name: "agent-enroll.json", path: enrollPath, sourceBytes: sourceEnroll, sourceURL: "https://portal.example.invalid:8443"},
		{name: "agent-run.json", path: runPath, sourceBytes: sourceRun, sourceURL: "https://127.0.0.1:9445"},
		{name: "agent-release.json", path: releasePath, sourceBytes: sourceRun, sourceURL: "https://127.0.0.1:9445"},
	} {
		gotBytes, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatal(err)
		}
		var before, after map[string]json.RawMessage
		if json.Unmarshal(tc.sourceBytes, &before) != nil || json.Unmarshal(gotBytes, &after) != nil {
			t.Fatalf("%s: invalid JSON", tc.name)
		}
		if string(after["control_url"]) != `"https://127.0.0.1:9444"` {
			t.Fatalf("%s: control_url = %s, want %q", tc.name, after["control_url"], localNodeControlURL)
		}
		var dpBefore, dpAfter map[string]json.RawMessage
		if json.Unmarshal(before["datapath"], &dpBefore) != nil || json.Unmarshal(after["datapath"], &dpAfter) != nil {
			t.Fatalf("%s: invalid datapath", tc.name)
		}
		if string(dpAfter["snat"]) != `"198.51.100.23"` {
			t.Fatalf("%s: datapath.snat = %s, want 198.51.100.23", tc.name, dpAfter["snat"])
		}
		delete(dpBefore, "snat")
		delete(dpAfter, "snat")
		if len(dpBefore) != len(dpAfter) {
			t.Fatalf("%s: datapath key count changed", tc.name)
		}
		for k, v := range dpBefore {
			if !bytes.Equal(v, dpAfter[k]) {
				t.Fatalf("%s: datapath[%s] changed: got %s, want %s", tc.name, k, dpAfter[k], v)
			}
		}
		delete(before, "control_url")
		delete(after, "control_url")
		delete(before, "datapath")
		delete(after, "datapath")
		if len(before) != len(after) {
			t.Fatalf("%s: top-level key count changed", tc.name)
		}
		for k, v := range before {
			if !bytes.Equal(v, after[k]) {
				t.Fatalf("%s: field %s changed: got %s, want %s", tc.name, k, after[k], v)
			}
		}
		restored := bytes.ReplaceAll(gotBytes, []byte(`"control_url": "https://127.0.0.1:9444"`), []byte(`"control_url": "`+tc.sourceURL+`"`))
		restored = bytes.ReplaceAll(restored, []byte(`"snat": "198.51.100.23"`), []byte(`"snat": "198.51.100.20"`))
		if !bytes.Equal(restored, tc.sourceBytes) {
			t.Fatalf("%s: bytes outside control_url and datapath.snat changed", tc.name)
		}
	}

	for path, want := range durableFiles {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("durable file %s changed", path)
		}
	}

	// Clean second-exit mode (clone == false) keeps agent-enroll.json's enrollment URL
	// while still pointing agent-run.json at the container's local resolver-proxy.
	if err := os.WriteFile(enrollPath, sourceEnroll, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runPath, sourceRun, 0600); err != nil {
		t.Fatal(err)
	}
	if err := prepareExitSlotConfigs(base, "198.51.100.22", false); err != nil {
		t.Fatal(err)
	}
	cleanEnroll, err := os.ReadFile(enrollPath)
	if err != nil {
		t.Fatal(err)
	}
	cleanRun, err := os.ReadFile(runPath)
	if err != nil {
		t.Fatal(err)
	}
	var enrollFields, runFields map[string]json.RawMessage
	if json.Unmarshal(cleanEnroll, &enrollFields) != nil || json.Unmarshal(cleanRun, &runFields) != nil {
		t.Fatal("invalid clean exit JSON")
	}
	if string(enrollFields["control_url"]) != `"https://portal.example.invalid:8443"` {
		t.Fatalf("clean agent-enroll.json control_url = %s", enrollFields["control_url"])
	}
	if string(runFields["control_url"]) != `"https://127.0.0.1:9444"` {
		t.Fatalf("clean agent-run.json control_url = %s", runFields["control_url"])
	}

	if _, err := rewriteExitConfig([]byte(`{"control_url":"https://127.0.0.1:9445"}`), "198.51.100.23", true); err == nil {
		t.Fatal("accepted config without datapath")
	}
	if _, err := rewriteExitConfig([]byte(`{"datapath":`), "198.51.100.23", true); err == nil {
		t.Fatal("accepted malformed config")
	}
}

func (r *runner) secondExit(ctx context.Context) {
	rec := Record{ID: "second-exit", Title: "Second real exit enrollment and independent resolver", Spec: []string{"§6", "§19 AT09", "F3-03"}, Status: StatusBlocked}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	s := &Stack{lab: r.lab}
	if err := s.StartSecondExit(ctx); err != nil {
		note("start", err.Error())
		return
	}
	admin := r.mustAdmin(ctx)
	if admin == nil {
		note("login", "failed")
		return
	}
	node, err := s.EnrollAt(ctx, BrowserSession{Client: admin.http, CSRF: admin.csrf}, "TEST ONLY second real exit", "node2")
	if err != nil {
		note("enroll", err.Error())
		return
	}
	note("distinct_identity", boolText(node.NodeID != r.nodeID && node.IncarnationID != r.incID))
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var detail map[string]json.RawMessage
	ready := waitUntil(wait, time.Second, func() bool {
		res, e := s.Call(wait, BrowserSession{Client: admin.http, CSRF: admin.csrf}, "GET", "/api/v1/admin/nodes/"+node.NodeID.String(), nil, nil)
		if e != nil || res.Status != 200 || json.Unmarshal(res.Body, &detail) != nil {
			return false
		}
		return string(detail["provisioning_state"]) == `"ready"`
	})
	for _, key := range []string{"provisioning_state", "safety_status", "data_health", "applied_config_generation"} {
		note(key, string(detail[key]))
	}
	raw, e := r.lab.docker(r.lab.prefix+"-node2", "nft", "-j", "list", "table", "inet", "fvpn")
	note("independent_nft_table", boolText(e == nil && len(raw) > 0))
	note("scope", "real second node hook; publication and simultaneous two-exit traffic remain harness scenarios, physical topology and boot flavours need operator runs")
	if ready && node.NodeID != r.nodeID && node.IncarnationID != r.incID && e == nil {
		rec.Status = StatusPass
	} else {
		note("owner", "w3-devstack-e2e hook / w2-agent-core readiness; see exact state")
	}
}
