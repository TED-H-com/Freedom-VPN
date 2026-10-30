package updater

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestUPD_D063_EmptyNodeInventories(t *testing.T) {
	for _, ips := range [][]string{nil, {"127.0.0.1"}, {"169.254.1.1"}, {"2001:db8::1"}, {"127.0.0.1", "169.254.1.1", "2001:db8::1"}} {
		t.Run(strings.Join(ips, ","), func(t *testing.T) {
			cfg := probeConfigFixture(t)
			cfg.ProbePeer = probeFixture()
			cfg.probeSources.interfaces = probeInterfaces(ips...)
			for _, check := range []func() error{
				func() error { _, err := loadProbeConfig(t, cfg); return err },
				func() error { _, err := cfg.Engine(); return err },
			} {
				var reason probeRefusal
				if err := check(); !errors.As(err, &reason) || reason != "node_addresses_unavailable" || err.Error() != errProbePeer.Error() {
					t.Fatal("empty node inventory was accepted or leaked", err)
				}
			}
		})
	}
}

func TestUPD_D063_AgentConfigNonRegularAndReplaced(t *testing.T) {
	for _, replacement := range []string{"symlink", "fifo", "directory", "regular"} {
		t.Run(replacement, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "agent.json")
			writeProbeAgentConfig(t, path, `{"control_url":"https://control.example.invalid"}`)
			lst, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			// Keep the original inode alive so the replacement cannot reuse it.
			if err := os.Rename(path, path+".old"); err != nil {
				t.Fatal(err)
			}
			switch replacement {
			case "symlink":
				err = os.Symlink(path+".old", path)
			case "fifo":
				err = syscall.Mkfifo(path, 0o600)
			case "directory":
				err = os.Mkdir(path, 0o700)
			case "regular":
				writeProbeAgentConfig(t, path, `{"control_url":"https://different.example.invalid"}`)
			}
			if err != nil {
				t.Fatal(err)
			}
			// Exercise the race after Lstat as well as refusing a non-regular
			// path already present before Lstat. A FIFO must never block.
			done := make(chan error, 1)
			go func() { _, err := readProbeAgentFile(path, lst); done <- err }()
			select {
			case err := <-done:
				var reason probeRefusal
				if !errors.As(err, &reason) || reason != "agent_config_unavailable" {
					t.Fatal("replacement accepted", err)
				}
			case <-time.After(time.Second):
				t.Fatal("agent config replacement blocked the reader")
			}
			if replacement != "regular" {
				if _, err := readProbeAgentConfig(path); err == nil || err.Error() != errProbePeer.Error() {
					t.Fatal("non-regular file accepted or leaked", err)
				}
			}
		})
	}
}

func TestUPD_D063_OmissionSurvivesHealthAndConsumption(t *testing.T) {
	for _, reason := range []string{"node_addresses_unavailable", "agent_config_unavailable"} {
		t.Run(reason, func(t *testing.T) {
			cfg := probeConfigFixture(t)
			cfg.ProbePeer = probeFixture()
			configured, err := cfg.Engine()
			if err != nil {
				t.Fatal(err)
			}
			defer configured.Close()
			w := newWorld(t)
			next := w.makeRelease("omitted-probe", 1, []int{1})
			touchPersistent(t, w.layout)
			w.eng.ProbePeer, w.eng.probeEnv = configured.ProbePeer, configured.probeEnv
			if reason == "node_addresses_unavailable" {
				w.eng.probeEnv.sources.interfaces = probeInterfaces()
			} else {
				if err := os.Rename(cfg.AgentConfig, cfg.AgentConfig+".real"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(cfg.AgentConfig+".real", cfg.AgentConfig); err != nil {
					t.Fatal(err)
				}
			}
			var issued healthattest.Challenge
			useAttest(w, func(ch healthattest.Challenge) (healthattest.Attestation, bool) {
				if ch.Probe != nil || ch.ProbeOmitted == nil || ch.ProbeOmitted.Reason != reason || ch.ProbeOmitted.PeerID != cfg.ProbePeer.PeerID {
					t.Fatal("missing challenge omission")
				}
				// The audit is already durable when the service first sees the challenge.
				records, err := w.eng.Journal.ProbeOmissions(context.Background(), ch.TxnID)
				if err != nil || len(records) != 1 || records[0].Nonce != ch.Nonce {
					t.Fatal("challenge published before durable omission", err)
				}
				issued = ch
				return goodAttest(ch, api.PublicationIntentUnpublished), true
			})
			w.setWork(w.work(next, "1", api.PublicationIntentUnpublished))
			if err := w.run(); err != nil {
				t.Fatal("omission made issuance fatal", err)
			}
			res := w.oneResult()
			if res.State != api.UpdateResultStateSucceeded {
				t.Fatal("omission changed the health outcome")
			}
			// UpdateHealth has only status enums; no unsupported wire detail is added.
			if _, _, err := healthattest.ReadChallenge(w.layout.Journal); !os.IsNotExist(err) {
				t.Fatal("challenge not consumed", err)
			}
			events, err := w.eng.Journal.TxnEvents(context.Background(), issued.TxnID)
			if err != nil {
				t.Fatal(err)
			}
			var audit, health bool
			for _, event := range events {
				var d struct {
					Kind         string                      `json:"kind"`
					ProbeOmitted *healthattest.ProbeOmission `json:"probe_omitted"`
				}
				if json.Unmarshal([]byte(event.Detail), &d) != nil || d.ProbeOmitted == nil {
					continue
				}
				if *d.ProbeOmitted != *issued.ProbeOmitted {
					t.Fatal("health evidence changed omission")
				}
				audit = audit || d.Kind == "probe_omitted"
				health = health || d.Kind == "health_attest"
			}
			if !audit || !health {
				t.Fatal("omission missing from transaction or local health result evidence")
			}
			if err := w.eng.Close(); err != nil {
				t.Fatal(err)
			}
			j, err := updatejournal.Open(w.layout.Journal, updatejournal.Options{})
			if err != nil {
				t.Fatal(err)
			}
			defer j.Close()
			records, err := j.ProbeOmissions(context.Background(), issued.TxnID)
			if err != nil || len(records) != 0 {
				t.Fatal("sealed transaction kept probe_omission meta", len(records), err)
			}
			events, err = j.TxnEvents(context.Background(), issued.TxnID)
			if err != nil {
				t.Fatal(err)
			}
			audit = false
			for _, event := range events {
				var d struct {
					Kind         string                      `json:"kind"`
					ProbeOmitted *healthattest.ProbeOmission `json:"probe_omitted"`
				}
				if json.Unmarshal([]byte(event.Detail), &d) != nil || d.Kind != "probe_omitted" || d.ProbeOmitted == nil {
					continue
				}
				if *d.ProbeOmitted != *issued.ProbeOmitted {
					t.Fatal("reopened audit changed omission")
				}
				audit = true
			}
			if !audit {
				t.Fatal("txn_events omission lost after seal and reopen")
			}
		})
	}
}
