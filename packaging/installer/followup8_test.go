package installer

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestUPD05_RenderedReportingConfigShape(t *testing.T) {
	for _, tc := range []struct {
		name, origin string
		loopback     bool
	}{
		{"control_cgnat", "https://100.64.0.10:9443", false},
		{"test_only_loopback", "https://127.0.0.1:9444", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// TEST ONLY: fake host/enrollment commands; neither origin is dialed.
			root := t.TempDir()
			env := baseEnv(t, root)
			fx := mustRelease(t, root, uuid.NewString(), "1")
			req := mustReq(t, root, fx)
			req.ControlURL, req.AllowLoopback = tc.origin+"/", tc.loopback

			// Literal wire keys and paths pin updater follow-up 19's acceptance
			// independently of updater.Config tags and installer path helpers.
			want := map[string]any{
				"result_url":               tc.origin + "/api/v1/agent/update-results",
				"allowed_artifact_origins": []string{tc.origin},
				"allow_loopback":           tc.loopback,
				"mtls_ca":                  filepath.Join(root, "etc/vpn-node/ca.pem"),
				"mtls_cert":                filepath.Join(root, "var/lib/vpn-node/identity/cert.pem"),
				"mtls_key":                 filepath.Join(root, "var/lib/vpn-node/identity/key.pem"),
			}
			var identity map[string]string
			for _, step := range []struct{ name, status string }{
				{"initial", statusInstalled},
				{"same_release_reporting_repair", statusAlreadyCurrent},
				{"identical_rerun", statusAlreadyCurrent},
			} {
				t.Run(step.name, func(t *testing.T) {
					before := ""
					if step.name == "identical_rerun" {
						before = hashFile(t, env.Layout.updaterConfig())
					}
					res, err := Install(t.Context(), env, req)
					if err != nil || res.Status != step.status {
						t.Fatalf("install: result=%+v err=%v", res, err)
					}
					raw, err := os.ReadFile(env.Layout.updaterConfig())
					if err != nil {
						t.Fatal(err)
					}
					var fields map[string]json.RawMessage
					if err := json.Unmarshal(raw, &fields); err != nil {
						t.Fatal(err)
					}
					for name, value := range want {
						expected, err := json.Marshal(value)
						if err != nil {
							t.Fatal(err)
						}
						var actual bytes.Buffer
						if err := json.Compact(&actual, fields[name]); err != nil || !bytes.Equal(actual.Bytes(), expected) {
							t.Errorf("%s: got %s, want %s", name, fields[name], expected)
						}
					}
					if _, present := fields["boot_id"]; present {
						t.Error("rendered config must not pin a boot fallback")
					}
					if hashFile(t, want["mtls_ca"].(string)) != hashFile(t, req.ServerCA) {
						t.Error("server trust differs from enrollment input")
					}
					if hashFile(t, req.ServerCA) == hashFile(t, filepath.Join(env.Layout.IdentityDir, "ca.pem")) {
						t.Fatal("fixture must distinguish server trust from client issuer")
					}
					if identity == nil {
						identity = make(map[string]string)
						for _, name := range []string{"node.json", "cert.pem", "key.pem", "ca.pem"} {
							path := filepath.Join(env.Layout.IdentityDir, name)
							identity[path] = hashFile(t, path)
						}
					}
					for path, digest := range identity {
						if hashFile(t, path) != digest {
							t.Error("same-release install changed enrolled identity")
						}
					}
					if before != "" && hashFile(t, env.Layout.updaterConfig()) != before {
						t.Error("identical rerun changed updater config bytes")
					}
					if step.name == "initial" {
						// Reproduce legacy/misconfigured reporting while retaining
						// valid enrolled client paths; arbitrary identity repair is
						// outside the same-release reporting repair contract.
						delete(fields, "result_url")
						fields["mtls_ca"], err = json.Marshal(filepath.Join(env.Layout.IdentityDir, "ca.pem"))
						if err != nil {
							t.Fatal(err)
						}
						fields["allowed_artifact_origins"] = json.RawMessage(`["https://mirror.example.invalid"]`)
						fields["allow_loopback"], err = json.Marshal(!tc.loopback)
						if err != nil {
							t.Fatal(err)
						}
						raw, err = marshalIndent(fields)
						if err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(env.Layout.updaterConfig(), raw, 0644); err != nil {
							t.Fatal(err)
						}
					}
				})
				if t.Failed() {
					return
				}
			}
		})
	}
}
