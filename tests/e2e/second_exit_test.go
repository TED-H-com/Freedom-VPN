//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestD141_ClonePreparedConfigsDialContainerResolverProxy(t *testing.T) {
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

	const enrollPath = "etc/agent-enroll.json"
	const runPath = "etc/agent-run.json"
	const releasePath = "var/e2e/agent-00000000-0000-7000-8000-000000000001.json"
	durableFiles := map[string][]byte{
		"etc/machine-id":         []byte("0123456789abcdef0123456789abcdef\n"),
		"var/identity/node.json": []byte("{\"node_id\":\"00000000-0000-7000-8000-000000000002\"}\n"),
		"var/identity/cert.pem":  []byte("TEST ONLY - NOT A SECRET cert\n"),
		"var/identity/key.pem":   []byte("TEST ONLY - NOT A SECRET key\n"),
		"var/state/agent.db":     []byte("TEST ONLY durable sqlite state bytes"),
		"var/e2e/agent-selected": []byte("/usr/local\n"),
	}
	writeFiles := func(t *testing.T, base string, files map[string][]byte) {
		t.Helper()
		for path, data := range files {
			path = filepath.Join(base, path)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	readFile := func(t *testing.T, base, path string) []byte {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(base, path))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	decode := func(t *testing.T, raw []byte) map[string]any {
		t.Helper()
		var fields map[string]any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if err := d.Decode(&fields); err != nil {
			t.Fatal(err)
		}
		return fields
	}
	checkDurable := func(t *testing.T, base string) {
		t.Helper()
		for path, want := range durableFiles {
			if got := readFile(t, base, path); !bytes.Equal(got, want) {
				t.Fatalf("durable file %s changed", path)
			}
		}
	}
	for _, mode := range []struct {
		name  string
		clone bool
		snat  string
	}{
		{name: "clone", clone: true, snat: "198.51.100.23"},
		{name: "fresh", clone: false, snat: "198.51.100.22"},
	} {
		for _, format := range []string{"formatted", "compact"} {
			t.Run(mode.name+"/"+format, func(t *testing.T) {
				base := t.TempDir()
				files := map[string][]byte{enrollPath: sourceEnroll, runPath: sourceRun}
				if mode.clone {
					release := sourceRun
					if format == "compact" {
						// candidateConfig writes release configs with json.Marshal.
						var err error
						release, err = json.Marshal(decode(t, sourceRun))
						if err != nil {
							t.Fatal(err)
						}
						if bytes.Contains(release, []byte("\n")) {
							t.Fatal("release fixture is not compact")
						}
					}
					files[releasePath] = release
				} else if format == "compact" {
					for path, raw := range files {
						var compact bytes.Buffer
						if err := json.Compact(&compact, raw); err != nil {
							t.Fatal(err)
						}
						files[path] = compact.Bytes()
					}
				}
				writeFiles(t, base, files)
				writeFiles(t, base, durableFiles)
				if err := prepareExitSlotConfigs(base, mode.snat, mode.clone); err != nil {
					t.Fatal(err)
				}
				for path, before := range files {
					want := decode(t, before)
					if path == enrollPath || path == runPath {
						want["datapath"].(map[string]any)["snat"] = mode.snat
					}
					if path == runPath || path == releasePath {
						want["control_url"] = "https://127.0.0.1:9444"
					}
					// Compare complete decoded objects, including every key and
					// nested DNS value. Serialization need not preserve whitespace.
					got := decode(t, readFile(t, base, path))
					if !reflect.DeepEqual(got, want) {
						t.Errorf("%s: values changed outside the authorized field matrix\ngot: %#v\nwant: %#v", path, got, want)
					}
				}
				checkDurable(t, base)
			})
		}
	}
	// Exercise the same file preparation path without invoking startExitSlot or
	// Docker. A rejected file must not be overwritten; durable bytes stay intact.
	for _, invalid := range []struct{ name, raw string }{
		{"malformed", `{"datapath":`},
		{"null-config", `null`},
		{"array-config", `[]`},
		{"missing-datapath", `{"control_url":"https://127.0.0.1:9445"}`},
		{"null-datapath", `{"datapath":null}`},
		{"array-datapath", `{"datapath":[]}`},
	} {
		for _, path := range []string{enrollPath, runPath, releasePath} {
			t.Run("refuse/"+invalid.name+"/"+path, func(t *testing.T) {
				base := t.TempDir()
				files := map[string][]byte{enrollPath: sourceEnroll, runPath: sourceRun, releasePath: sourceRun}
				files[path] = []byte(invalid.raw)
				writeFiles(t, base, files)
				writeFiles(t, base, durableFiles)
				if err := prepareExitSlotConfigs(base, "198.51.100.23", true); err == nil {
					t.Fatal("accepted invalid config")
				}
				if got := readFile(t, base, path); !bytes.Equal(got, files[path]) {
					t.Fatal("rejected config was overwritten")
				}
				checkDurable(t, base)
			})
		}
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
