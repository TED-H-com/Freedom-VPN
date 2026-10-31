package updater

import (
	"bytes"
	"encoding/json"
	"encoding/pem"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

func TestUPD05_MissingResultURLKeepsReceiptPending(t *testing.T) {
	w := newWorld(t)
	t.Cleanup(func() { _ = w.eng.Close() })
	quietDesired(w)
	rel := w.makeRelease("TEST ONLY candidate", 1, nil)
	var logs bytes.Buffer
	w.eng.UseLog(slog.New(slog.NewJSONHandler(&logs, nil)))
	if err := w.eng.ApplyPending(t.Context(), stageInstaller(t, w, rel)); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(rel.id)
	before, err := w.eng.Journal.Outbox(t.Context())
	if err != nil || len(before) != 1 || before[0].Status != updatejournal.OutboxPending {
		t.Fatal("local success lost pending receipt", err)
	}
	if !strings.Contains(logs.String(), `"reason":"result_url_missing"`) {
		t.Fatal("missing reporting configuration was silent", &logs)
	}
	for range 2 {
		err := w.eng.Flush(t.Context())
		if op := asOp(err); op == nil || op.Code != apierr.ValidationFailed || ExitCode(err) != 1 || !strings.Contains(op.Detail, "result_url") {
			t.Fatalf("unconfigured delivery reported success: %v", err)
		}
	}
	after, err := w.eng.Journal.Outbox(t.Context())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("missing sink changed receipt or delivery state", err)
	}
	history, err := w.eng.Journal.History(t.Context(), "")
	if err != nil || len(history) != 0 {
		t.Fatal("missing sink created acceptance history", err)
	}
	// Repair only the reporting configuration: transmit the original seal once.
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		calls++
		body, err := io.ReadAll(req.Body)
		if err != nil || !bytes.Equal(body, before[0].Payload) || req.Header.Get("Idempotency-Key") != before[0].ResultID {
			t.Error("delivery changed the sealed receipt", err)
		}
		rw.WriteHeader(http.StatusAccepted)
		_, _ = rw.Write([]byte(`{"accepted":true,"accepted_as_history":false}`))
	}))
	defer srv.Close()
	w.eng.Sender = HTTPSink{URL: srv.URL, AllowLoopback: true}
	for range 2 {
		if err := w.eng.Flush(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	after, err = w.eng.Journal.Outbox(t.Context())
	if err != nil || len(after) != 0 || calls != 1 {
		t.Fatal("repaired delivery was lost or duplicated", calls, err)
	}
	history, err = w.eng.Journal.History(t.Context(), "")
	if err != nil || len(history) != 1 || history[0].PayloadHash != before[0].PayloadHash {
		t.Fatal("acceptance did not retain original receipt hash", err)
	}
	w.mustCurrent(rel.id)
}

func TestUPD05_ResultConfigPolicyAndServerTrust(t *testing.T) {
	root := t.TempDir()
	trust := filepath.Join(root, "release-trust.json")
	writeTrust(t, trust)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v1/agent/update-results" {
			t.Error("wrong result endpoint", req.URL.Path)
		}
		rw.WriteHeader(http.StatusAccepted)
		_, _ = rw.Write([]byte(`{"accepted":true,"accepted_as_history":false}`))
	}))
	defer srv.Close()
	ca := filepath.Join(root, "server-ca.pem")
	if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	base := Config{ReleaseTrust: trust, DesiredTrust: trust, ResultURL: srv.URL + "/api/v1/agent/update-results", MTLSCA: ca, AllowLoopback: true}
	for _, scenario := range []string{"configured", "missing_url", "loopback_denied", "origin_does_not_allow_loopback", "system_roots", "missing_ca_file"} {
		t.Run(scenario, func(t *testing.T) {
			cfg := base
			switch scenario {
			case "missing_url":
				cfg.ResultURL = ""
			case "loopback_denied":
				cfg.AllowLoopback = false
			case "origin_does_not_allow_loopback":
				cfg.AllowLoopback = false
				cfg.AllowedArtifactOrigins = []string{srv.URL}
			case "system_roots":
				cfg.MTLSCA = ""
			case "missing_ca_file":
				cfg.MTLSCA = filepath.Join(root, "absent.pem")
			}
			// Exercise the exact JSON field names rendered by the installer.
			raw, err := json.Marshal(cfg)
			if err != nil {
				t.Fatal(err)
			}
			var decoded Config
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatal(err)
			}
			eng, err := decoded.Engine()
			if scenario == "loopback_denied" || scenario == "origin_does_not_allow_loopback" || scenario == "missing_ca_file" {
				if err == nil {
					_ = eng.Close()
					t.Fatal("invalid reporting configuration accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer eng.Close()
			if scenario == "missing_url" {
				if eng.Sender != nil || ExitCode(eng.Flush(t.Context())) != 1 {
					t.Fatal("missing result_url did not fail delivery")
				}
				return
			}
			ack, conflict, err := eng.Sender.Submit(t.Context(), []byte(`{}`))
			if scenario == "system_roots" {
				if err == nil || conflict || len(ack.Raw) != 0 {
					t.Fatal("untrusted server was accepted without configured CA")
				}
				return
			}
			if err != nil || conflict || len(ack.Raw) == 0 {
				t.Fatal("configured CA did not reach the sink", err)
			}
		})
	}
}

func TestAT25_ReleaseRootCandidateSwitchAndHealth(t *testing.T) {
	w := newWorld(t)
	t.Cleanup(func() { _ = w.eng.Close() })
	quietDesired(w)
	baseline := w.makeRelease("TEST ONLY baseline", 1, nil)
	w.seed(baseline)
	candidate := w.makeRelease("TEST ONLY candidate", 1, nil)
	if err := w.eng.ApplyPending(t.Context(), stageInstaller(t, w, candidate)); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(candidate.id)
	root := w.layout.releaseDir(candidate.id.String())
	if w.svc.last != root {
		t.Fatal("service start was not given the release root", w.svc.last)
	}
	if _, err := os.Stat(filepath.Join(w.layout.Current, "fvpn-agent")); err != nil {
		t.Fatal("current does not expose release-root agent", err)
	}
	if _, err := os.Stat(filepath.Join(root, "bin", "fvpn-agent")); !os.IsNotExist(err) {
		t.Fatal("fixture retained the obsolete bin layout", err)
	}
	v := w.eng.viewFor(t.Context(), candidate.id.String(), &candidate.manifest)
	checks := NodeChecks{}
	if got, err := checks.Processes(t.Context(), v); err != nil || got != api.HealthCheckStatusPassed {
		t.Fatal("release-root tree failed health hash check", got, err)
	}
	if err := os.WriteFile(filepath.Join(root, "fvpn-agent"), []byte("TEST ONLY drift"), 0755); err != nil {
		t.Fatal(err)
	}
	if got, err := checks.Processes(t.Context(), v); err != nil || got != api.HealthCheckStatusFailed {
		t.Fatal("root executable drift escaped health hash check", got, err)
	}
}
