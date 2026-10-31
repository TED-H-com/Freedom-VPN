package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

const (
	MeterCheckpointWait = 30 * time.Second // 待實測; configurable, not an SLO
	meterCheckpointPoll = 100 * time.Millisecond
	meterWaitPrefix     = "meter-wait-"
)

func meterWaitDuration(ms int) (time.Duration, error) {
	if ms < 0 || ms > 300000 {
		return 0, errors.New("updater: meter_checkpoint_wait_ms must be 0..300000")
	}
	if ms == 0 {
		return MeterCheckpointWait, nil
	}
	return time.Duration(ms) * time.Millisecond, nil
}

// FileMeter decodes the agent's durable closed-epoch snapshot. FinalSample is
// a legacy single-read decoder, not proof that a snapshot belongs to an update.
// Correlated snapshots require the persisted identity supplied by Engine's wait;
// FinalSample alone has no such authority and continues to refuse them.
// OpenEpoch writes nothing; the existing epoch-<txn> note owns that decision.
type FileMeter struct {
	Path string
	Wait time.Duration // zero uses MeterCheckpointWait
}

func (m FileMeter) FinalSample(ctx context.Context) (UsageCheckpoint, error) {
	return m.finalSample(ctx, "")
}

// finalSample makes one decision from one file snapshot. The identity comes
// only from the validated durable wait note, never from the checkpoint itself.
func (m FileMeter) finalSample(ctx context.Context, identity string) (UsageCheckpoint, error) {
	if err := ctx.Err(); err != nil {
		return UsageCheckpoint{}, err
	}
	b, err := readMeterFile(m.Path)
	if err != nil {
		return UsageCheckpoint{}, err
	}
	var c struct {
		UsageCheckpoint
		BarrierID string `json:"barrier_id,omitempty"`
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF {
		return UsageCheckpoint{}, permanent(apierr.CheckpointFailed, "meter checkpoint format")
	}
	id, err := uuid.Parse(c.ClosedMeterEpochID)
	if err != nil || id == uuid.Nil || c.NewMeterEpochID != "" || c.CentralAckedThrough > c.LastDurableEventSeq {
		return UsageCheckpoint{}, permanent(apierr.CheckpointFailed, "meter checkpoint values")
	}
	// All three snapshot values must be present, even when the sequence is 0.
	var fields map[string]json.RawMessage
	if json.Unmarshal(b, &fields) != nil || fields["last_durable_event_seq"] == nil || fields["central_acked_through"] == nil {
		return UsageCheckpoint{}, permanent(apierr.CheckpointFailed, "meter checkpoint watermarks")
	}
	if identity == "" {
		if fields["barrier_id"] != nil {
			return UsageCheckpoint{}, permanent(apierr.CheckpointFailed, "meter checkpoint requires journaled barrier")
		}
	} else if uuid.Validate(c.BarrierID) != nil || c.BarrierID != identity {
		return UsageCheckpoint{}, permanent(apierr.CheckpointFailed, "meter_checkpoint_uncorrelated")
	}
	return c.UsageCheckpoint, nil
}

// readMeterFile bounds each read and refuses special files without blocking on
// a FIFO. Agent publication uses atomic rename, so one descriptor is one
// snapshot/decision; there is no second checkpoint read to consume other bytes.
func readMeterFile(path string) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() > 4096 {
		return nil, errors.New("updater: invalid meter handshake file")
	}
	b, err := io.ReadAll(io.LimitReader(f, 4097))
	if err != nil || len(b) > 4096 {
		return nil, errors.New("updater: invalid meter handshake file")
	}
	return b, nil
}

// meterWait is updater-local evidence. D-125 echoes Identity in barrier_id.
// Persist before Block so a restart cannot obtain another wait budget. The
// opaque latch identity detects Unblock/Block between reads and across restart.
type meterWait struct {
	BootID   uuid.UUID `json:"boot_id"`
	Identity string    `json:"identity"`
	Started  time.Time `json:"started"`
	Deadline time.Time `json:"deadline"`
}

func (e *Engine) fileMeter() (FileMeter, bool) {
	switch m := e.Meter.(type) {
	case FileMeter:
		return m, true
	case *FileMeter:
		if m != nil {
			return *m, true
		}
	}
	return FileMeter{}, false
}

func (e *Engine) fileGate() (FileGate, bool) {
	switch g := e.Gate.(type) {
	case FileGate:
		return g, true
	case *FileGate:
		if g != nil {
			return *g, true
		}
	}
	return FileGate{}, false
}

func (e *Engine) prepareMeterWait(txn updatejournal.Txn, m FileMeter) error {
	g, ok := e.fileGate()
	if !ok || g.Path == "" {
		return permanent(apierr.CheckpointFailed, "meter_gate_unavailable")
	}
	path := filepath.Join(e.Layout.Journal, meterWaitPrefix+txn.TxnID)
	if _, err := os.Lstat(path); err == nil {
		// Never republish the block on recovery: absence/replacement may be
		// withdrawal, including a crash after the note but before Block.
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	wait := m.Wait
	if wait == 0 {
		wait = MeterCheckpointWait
	}
	if wait < 0 || wait > 5*time.Minute {
		return permanent(apierr.CheckpointFailed, "meter_wait_invalid")
	}
	now := time.Now()
	note := meterWait{BootID: e.Node.BootID, Identity: uuid.NewString(), Started: now, Deadline: now.Add(wait)}
	raw, err := json.Marshal(note)
	if err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomic(path, raw, 0o600); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(g.Path, []byte(note.Identity+"\n"), 0o600)
}

func (e *Engine) waitMeter(ctx context.Context, txn updatejournal.Txn, m FileMeter) (UsageCheckpoint, error) {
	fail := func(reason string) (UsageCheckpoint, error) {
		return UsageCheckpoint{}, permanent(apierr.CheckpointFailed, reason)
	}
	g, ok := e.fileGate()
	if !ok {
		return fail("meter_gate_unavailable")
	}
	raw, err := readMeterFile(filepath.Join(e.Layout.Journal, meterWaitPrefix+txn.TxnID))
	var note meterWait
	if err != nil || json.Unmarshal(raw, &note) != nil || uuid.Validate(note.Identity) != nil ||
		note.BootID == uuid.Nil || note.BootID != e.Node.BootID || note.Started.IsZero() ||
		!note.Deadline.After(note.Started) || note.Deadline.Sub(note.Started) > 5*time.Minute || time.Now().Before(note.Started) {
		return fail("meter_wait_invalid")
	}
	// WithTimeout also bounds a running process with Go's monotonic clock.
	// The persisted wall deadline prevents restart from renewing the budget.
	ctx, cancel := context.WithTimeout(ctx, time.Until(note.Deadline))
	defer cancel()
	reason := "meter_checkpoint_timeout"
	for {
		latch, err := readMeterFile(g.Path)
		if err != nil || string(latch) != note.Identity+"\n" {
			return UsageCheckpoint{}, permanent(apierr.StateConflict, "meter_barrier_withdrawn")
		}
		if ctx.Err() != nil || !time.Now().Before(note.Deadline) {
			return fail(reason)
		}
		cp, sampleErr := m.finalSample(ctx, note.Identity)
		if op := asOp(sampleErr); op != nil && op.Detail == "meter_checkpoint_uncorrelated" {
			reason = "meter_checkpoint_uncorrelated"
		}
		latch, err = readMeterFile(g.Path)
		if err != nil || string(latch) != note.Identity+"\n" {
			return UsageCheckpoint{}, permanent(apierr.StateConflict, "meter_barrier_withdrawn")
		}
		// A read that finishes after the bound cannot authorize a switch.
		if ctx.Err() != nil || !time.Now().Before(note.Deadline) {
			return fail(reason)
		}
		if sampleErr == nil {
			return cp, nil
		}
		timer := time.NewTimer(meterCheckpointPoll)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}

func (e *Engine) failMeterWait(ctx context.Context, txn updatejournal.Txn, err error) (updatejournal.Txn, error) {
	code, detail := apierr.CheckpointFailed, "meter_wait_invalid"
	if op := asOp(err); op != nil {
		code, detail = op.Code, op.Detail
	}
	return e.advance(ctx, txn, updatejournal.StageRollingBack, updatejournal.StageUpdate{
		ErrorCode: string(code), ErrorStage: string(api.UpdateStageFinalSample), ErrorDetail: detail,
		Detail: detail, Checkpoint: json.RawMessage("null"),
	})
}

func meterWaitFailed(txn updatejournal.Txn) bool {
	if txn.Destructive || txn.ErrorStage != string(api.UpdateStageFinalSample) ||
		(txn.ErrorCode != string(apierr.CheckpointFailed) && txn.ErrorCode != string(apierr.StateConflict)) {
		return false
	}
	switch txn.ErrorDetail {
	case "meter_gate_unavailable", "meter_wait_invalid", "meter_checkpoint_timeout", "meter_checkpoint_uncorrelated", "meter_barrier_withdrawn":
		return true
	default:
		return false
	}
}
