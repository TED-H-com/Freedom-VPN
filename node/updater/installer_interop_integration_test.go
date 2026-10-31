//go:build integration

package updater

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/installerjob"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func TestD110_RealInstallerReceiptInteroperability(t *testing.T) {
	for _, outcome := range []string{"succeeded", "failed", "rolled_back", "legacy_failed"} {
		t.Run(outcome, func(t *testing.T) {
			l := newCentralLab(t)
			w := newWorld(t)
			w.releaseSeq = "1"
			old := w.makeRelease("installed", 1, []int{1})
			w.releaseSeq = "2"
			w.rollbackAllow = []uuid.UUID{old.id}
			next := w.makeRelease("next", 1, []int{1})
			var err error
			l.rel.Validator, err = releases.NewValidator(w.trust, []string{w.srv.URL, "https://releases.example.invalid"})
			if err != nil {
				t.Fatal(err)
			}
			for _, rel := range []builtRel{old, next} {
				request, err := json.Marshal(map[string]any{"manifest": json.RawMessage(rel.env), "manifest_url": w.srv.URL + "/m/" + rel.id.String()})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := l.rel.Register(l.ctx, releasetest.Command(), request); err != nil {
					t.Fatal(err)
				}
			}
			l.old = old.id
			l.support([]uuid.UUID{old.id, next.id}, []uuid.UUID{old.id, next.id}, []uuid.UUID{})
			identity := l.node("amd64", "disabled", "registered", false)
			// Match the live F9 installer: baseline seeded locally before central knows it.
			if _, err := l.pool.Exec(l.ctx, `UPDATE nodes SET current_release_id=NULL,last_good_release_id=NULL WHERE id=$1`, identity.NodeID); err != nil {
				t.Fatal(err)
			}
			w.eng.Node.ID, w.eng.Node.Incarnation = identity.NodeID, identity.IncarnationID
			manifest := installSeedTree(t, w, old, true)
			if err := w.eng.Seed(l.ctx, manifest, w.layout.releaseDir(old.id.String())); err != nil {
				t.Fatal(err)
			}
			switch outcome {
			case "failed", "legacy_failed":
				w.meter.err = errors.New("TEST ONLY absent checkpoint")
			case "rolled_back":
				w.health.fail[next.id.String()] = true
			}
			applyErr := w.eng.ApplyPending(l.ctx, stageInstaller(t, w, next))
			if (applyErr != nil) != (outcome == "failed" || outcome == "legacy_failed") {
				t.Fatal("apply", applyErr)
			}
			raw, result := lastPayload(t, w)
			if result.JobId != installerjob.ID(next.id, next.sum, identity.IncarnationID) {
				t.Fatal("wrong installer job")
			}
			if outcome == "legacy_failed" {
				// A previously unaccepted seal is real failure evidence with the old ID.
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(raw, &fields); err != nil {
					t.Fatal(err)
				}
				legacy := uuid.New()
				fields["job_id"], _ = json.Marshal(legacy)
				fields["result_id"], _ = json.Marshal(resultID(legacy.String(), 1, result.State))
				var hash string
				raw, hash = sealTestFields(t, fields)
				if err := json.Unmarshal(raw, &result); err != nil {
					t.Fatal(err)
				}
				_, err := l.svc.SubmitResult(l.ctx, identity, raw)
				var ae *apierr.Error
				if !errors.As(err, &ae) || ae.Code != apierr.StateConflict || ae.Details["reason"] != "job_id_mismatch" {
					t.Fatal("legacy was not rejected", err)
				}
				// Separate outbox simulates retained receipt delivery after a binary upgrade.
				w.eng.Journal = openJournal(t, t.TempDir())
				if err := w.eng.Journal.Enqueue(l.ctx, updatejournal.OutboxEntry{ResultID: result.ResultId.String(), JobID: legacy.String(), Attempt: 1, Generation: result.Generation, State: string(result.State), Payload: raw, PayloadHash: hash}); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				calls++
				payload, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					rw.WriteHeader(500)
					return
				}
				ack, err := l.svc.SubmitResult(l.ctx, identity, payload)
				if err != nil {
					t.Error("real central rejected updater receipt", err)
					rw.WriteHeader(500)
					return
				}
				replay, err := l.svc.SubmitResult(l.ctx, identity, payload)
				if err != nil || !reflect.DeepEqual(ack, replay) {
					t.Error("central replay differs", err)
				}
				if outcome != "legacy_failed" && !bytes.Equal(payload, raw) {
					t.Error("D-110 seal rewritten")
				}
				rw.WriteHeader(202)
				_ = json.NewEncoder(rw).Encode(ack)
			}))
			defer srv.Close()
			w.eng.Sender = HTTPSink{URL: srv.URL, AllowLoopback: true}
			if err := w.eng.Flush(l.ctx); err != nil {
				t.Fatal(err)
			}
			if err := w.eng.Flush(l.ctx); err != nil || calls != 1 {
				t.Fatal("ACK resent", err, calls)
			}
			history, err := w.eng.Journal.History(l.ctx, "")
			if err != nil || len(history) != 1 {
				t.Fatal("no accepted receipt", history, err)
			}
			var count int
			if err := l.pool.QueryRow(l.ctx, `SELECT count(*) FROM installer_update_results WHERE node_id=$1`, identity.NodeID).Scan(&count); err != nil || count != 1 {
				t.Fatal("receipt cardinality", count, err)
			}
			var current, lastGood uuid.UUID
			if err := l.pool.QueryRow(l.ctx, `SELECT current_release_id,last_good_release_id FROM nodes WHERE id=$1`, identity.NodeID).Scan(&current, &lastGood); err != nil {
				t.Fatal(err)
			}
			want := old.id
			if outcome == "succeeded" {
				want = next.id
			}
			if current != want || lastGood != want {
				t.Fatal("wrong central pointers", current, lastGood, want)
			}
		})
	}
}
