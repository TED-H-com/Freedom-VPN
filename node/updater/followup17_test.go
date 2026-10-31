package updater

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func TestD125_RuleCorrelatedFreshPostBlockAccounting(t *testing.T) { correlatedApply(t, "fresh") }
func TestD125_RuleCorrelatedLateWithinBound(t *testing.T)          { correlatedApply(t, "late") }
func TestD125_RuleCorrelatedAfterBound(t *testing.T)               { correlatedApply(t, "expired") }
func TestD125_RulePreviousBlockIdentity(t *testing.T)              { correlatedApply(t, "foreign") }
func TestD125_RuleMalformedIdentity(t *testing.T)                  { correlatedApply(t, "malformed") }
func TestD125_RuleLegacyFourMembers(t *testing.T)                  { correlatedApply(t, "legacy") }
func TestD125_RuleWithdrawnBeforeCorrelatedPublication(t *testing.T) {
	correlatedApply(t, "withdrawn")
}
func TestD125_RuleRestartRechecksCorrelatedFile(t *testing.T) { correlatedApply(t, "restart") }
func TestD125_RuleIdenticalRepublicationAccountsOnce(t *testing.T) {
	correlatedApply(t, "republished")
}
func TestD125_RuleCorrelatedPriorBootRejected(t *testing.T) { correlatedApply(t, "boot") }
func TestD125_RuleByteEqualUUIDSpelling(t *testing.T)       { correlatedApply(t, "spelling") }
func TestD125_RuleFinalSampledRestartRechecksFile(t *testing.T) {
	correlatedApply(t, "sampled_restart")
}

func TestD125_RuleFinalSampledRestartDiscardsWithdrawnCheckpoint(t *testing.T) {
	correlatedApply(t, "sampled_withdrawn")
}

func correlatedApply(t *testing.T, scenario string) {
	t.Helper()
	w := newWorld(t)
	t.Cleanup(func() { _ = w.eng.Close() })
	old := w.makeRelease("baseline", 1, nil)
	w.seed(old)
	next := w.makeRelease("candidate", 1, nil)
	pending := stageInstaller(t, w, next)
	g := FileGate{Path: filepath.Join(w.layout.Journal, "forwarding.blocked")}
	m := FileMeter{Path: filepath.Join(w.layout.State, "meter.checkpoint"), Wait: 2 * time.Second}
	if scenario == "expired" || scenario == "foreign" || scenario == "malformed" || scenario == "legacy" || scenario == "spelling" {
		m.Wait = 250 * time.Millisecond
	}
	w.eng.Gate, w.eng.Meter = g, m
	// An actual previous block, not merely a different parsed UUID.
	if err := g.Block(t.Context()); err != nil {
		t.Fatal(err)
	}
	previous := strings.TrimSuffix(string(fileBytes(t, g.Path)), "\n")
	if err := g.Unblock(t.Context()); err != nil {
		t.Fatal(err)
	}
	// Real meter/store starts before the block; final delta and close happen
	// inside the blocked hook. Only kernel counters/withdrawal are fixtures.
	closeMeter := agentBarrier(t, w.node.BootID)
	var cp usagestore.MeterCheckpoint
	var raw []byte
	done := make(chan error, 1)
	w.eng.Hooks.AfterStage = func(stage string) error {
		if stage == string(updatejournal.StageFinalSampled) {
			if scenario == "sampled_restart" || scenario == "sampled_withdrawn" {
				return ErrCrashed
			}
			if scenario == "republished" {
				return fsutil.WriteFileAtomic(m.Path, raw, 0600)
			}
		}
		if stage != string(updatejournal.StageBlocked) {
			return nil
		}
		cp = closeMeter()
		cp.BarrierID = strings.TrimSuffix(string(fileBytes(t, g.Path)), "\n")
		if err := uuid.Validate(cp.BarrierID); err != nil || cp.BarrierID == previous {
			t.Fatal("missing new latch identity", err)
		}
		switch scenario {
		case "foreign":
			cp.BarrierID = previous
		case "malformed":
			cp.BarrierID = "TEST ONLY invalid UUID"
		case "legacy":
			cp.BarrierID = ""
		case "spelling":
			// Braces are a valid alternative spelling even for an all-digit ID.
			cp.BarrierID = "{" + cp.BarrierID + "}"
		case "withdrawn":
			if err := g.Unblock(t.Context()); err != nil {
				return err
			}
		}
		var err error
		raw, err = json.Marshal(cp) // exact producer type, ordering and newline
		if err != nil {
			return err
		}
		raw = append(raw, '\n')
		switch scenario {
		case "expired":
			return nil // publish only after the original bounded wait ends
		case "late":
			go func() {
				time.Sleep(150 * time.Millisecond)
				done <- fsutil.WriteFileAtomic(m.Path, raw, 0600)
			}()
			return nil
		default:
			// fsutil uses temporary file fsync, atomic rename and directory fsync,
			// as agent.handoff does; its publication helper is unexported.
			if err := fsutil.WriteFileAtomic(m.Path, raw, 0600); err != nil {
				return err
			}
		}
		if scenario == "restart" || scenario == "boot" {
			return ErrCrashed
		}
		return nil
	}
	err := w.eng.ApplyPending(t.Context(), pending)
	if scenario == "late" {
		if publishErr := <-done; publishErr != nil {
			t.Fatal(publishErr)
		}
	}
	if scenario == "restart" || scenario == "boot" || scenario == "sampled_restart" || scenario == "sampled_withdrawn" {
		if !errors.Is(err, ErrCrashed) {
			t.Fatal("did not stop mid-handshake", err)
		}
		if err := w.eng.Close(); err != nil {
			t.Fatal(err)
		}
		restarted := *w.eng
		restarted.Journal = openJournal(t, w.layout.Journal)
		restarted.Hooks = Hooks{}
		restarted.Meter = FileMeter{Path: m.Path, Wait: time.Minute}
		w.eng = &restarted
		if scenario == "boot" {
			w.eng.Node.BootID = uuid.New()
		}
		if scenario == "sampled_withdrawn" {
			if err := g.Unblock(t.Context()); err != nil {
				t.Fatal(err)
			}
		}
		err = w.eng.Recover(t.Context())
	}
	success := scenario == "fresh" || scenario == "late" || scenario == "restart" || scenario == "republished" || scenario == "sampled_restart"
	if success {
		// Installer intent is unpublished: recovery can complete the apply
		// successfully while correctly reporting that members stay blocked.
		if err != nil && !errors.Is(err, ErrMembersBlocked) {
			t.Fatal(err)
		}
		w.mustCurrent(next.id)
		res := w.oneResult()
		usage := requiredValue(t, res.UsageCheckpoint)
		if res.State != api.UpdateResultStateSucceeded || usage.LastDurableEventSeq != cp.LastDurableEventSeq || usage.CentralAckedThrough != cp.CentralAckedThrough || requiredValue(t, usage.ClosedMeterEpochId).String() != cp.ClosedMeterEpochID || requiredValue(t, usage.NewMeterEpochId) == uuid.Nil {
			t.Fatalf("successful receipt lost the post-block close: %+v", res)
		}
		if stops, starts := w.svc.counts(); stops == 0 || starts == 0 {
			t.Fatal("successful apply did not replace services")
		}
		// Finalizing again must return the same close and exactly one 123-byte
		// durable usage event (agentBarrier asserts its store, not just receipt).
		again := closeMeter()
		again.BarrierID = cp.BarrierID
		if again != cp {
			t.Fatal("same close changed accounting")
		}
	} else {
		if err == nil {
			t.Fatal("invalid barrier allowed apply")
		}
		code, reason := apierr.CheckpointFailed, "meter_checkpoint_uncorrelated"
		switch scenario {
		case "expired":
			reason = "meter_checkpoint_timeout"
			if err := fsutil.WriteFileAtomic(m.Path, raw, 0600); err != nil {
				t.Fatal(err)
			}
		case "withdrawn", "sampled_withdrawn":
			code, reason = apierr.StateConflict, "meter_barrier_withdrawn"
		case "boot":
			reason = "meter_wait_invalid"
		}
		assertMeterFailure(t, w, old.id, code, reason)
	}
	before, err := w.eng.Journal.Outbox(t.Context())
	if err != nil || len(before) != 1 {
		t.Fatal("missing sealed receipt", err)
	}
	w.eng.Hooks = Hooks{}
	if err := w.eng.Recover(t.Context()); err != nil && !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	after, err := w.eng.Journal.Outbox(t.Context())
	if err != nil || len(after) != 1 || !bytes.Equal(before[0].Payload, after[0].Payload) {
		t.Fatal("replay changed decision or accounted twice", err)
	}
	if got := fileBytes(t, m.Path); !bytes.Equal(got, raw) {
		t.Fatal("updater changed producer evidence")
	}
	if st, err := os.Stat(m.Path); err != nil || st.Mode().Perm() != 0600 {
		t.Fatal("publication mode", err)
	}
}
