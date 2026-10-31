package updatejournal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// SnapshotTimeout bounds an observation even when the caller has no deadline.
const SnapshotTimeout = time.Second

// SnapshotState is local durable evidence, not central acceptance. Stage/State
// use API vocabulary; JournalStage exposes the exact transaction stage.
// Nil Generation means no job/result has been recorded, not generation zero.
type SnapshotState struct {
	Generation       *decstr.U64     `json:"generation"`
	JobID            string          `json:"job_id"`
	Source           string          `json:"source,omitempty"`
	RequestedRelease string          `json:"requested_release_id,omitempty"`
	Stage            string          `json:"stage"`
	State            string          `json:"state"`
	JournalStage     Stage           `json:"journal_stage,omitempty"`
	CurrentRelease   string          `json:"current_release_id"`
	LastGoodRelease  string          `json:"last_good_release_id"`
	RecoveryRequired bool            `json:"recovery_required"`
	LastResultState  string          `json:"last_result_state"`
	LastRejection    *Rejection      `json:"last_rejection"`
	Health           json.RawMessage `json:"health"`
	CandidateHealth  json.RawMessage `json:"candidate_health_json"`
}

// Rejection contains only whitelisted delivery diagnostics.
type Rejection struct {
	ResultID   string    `json:"result_id"`
	HTTPStatus int       `json:"http_status"`
	APICode    string    `json:"api_code"`
	Code       string    `json:"code"`
	At         time.Time `json:"at"`
}

// Snapshot reads an existing journal database file (not a directory) using
// mode=ro and query_only. It creates no schema, performs no migration, takes no
// updater flock, and reads one consistent SQLite snapshot while the updater
// can continue writing WAL. Missing/unsupported/unreadable journals return an
// error; callers must not turn that into a successful update observation.
func Snapshot(ctx context.Context, path string) (SnapshotState, error) {
	ctx, cancel := context.WithTimeout(ctx, SnapshotTimeout)
	defer cancel()
	path, err := filepath.Abs(path)
	if err != nil {
		return SnapshotState{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return SnapshotState{}, err
	}
	if !info.Mode().IsRegular() {
		return SnapshotState{}, errors.New("updatejournal: snapshot requires a regular database file")
	}
	u := url.URL{Scheme: "file", Path: path}
	q := url.Values{"mode": {"ro"}, "_pragma": {"query_only(1)", "busy_timeout(50)"}}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return SnapshotState{}, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return SnapshotState{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var version int
	if err := tx.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return SnapshotState{}, err
	}
	if version < 1 || version > schemaVersion {
		return SnapshotState{}, fmt.Errorf("updatejournal: unsupported snapshot schema %d", version)
	}
	return readSnapshot(ctx, tx, version)
}

func readSnapshot(ctx context.Context, tx *sql.Tx, version int) (SnapshotState, error) {
	var s SnapshotState
	var err error
	if s.CurrentRelease, _, err = getMeta(ctx, tx, metaCurrentRelease); err != nil {
		return s, err
	}
	if s.LastGoodRelease, _, err = getMeta(ctx, tx, metaLastGoodRelease); err != nil {
		return s, err
	}
	outcome, _, err := getMeta(ctx, tx, metaTerminalOutcome)
	if err != nil {
		return s, err
	}
	s.RecoveryRequired = outcome == OutcomeRecoveryRequired
	// Local installer requests reuse the accepted generation without claiming
	// a central retry attempt. Within that generation, show the last claim.
	job, jobErr := scanJob(tx.QueryRowContext(ctx, jobSelect+` ORDER BY length(generation) DESC, generation DESC, rowid DESC LIMIT 1`))
	if jobErr != nil && !errors.Is(jobErr, ErrNotFound) {
		return s, jobErr
	}
	if jobErr == nil {
		var intent struct {
			Source string `json:"source"`
		}
		if err := json.Unmarshal(job.IntentJSON, &intent); err != nil {
			return s, err
		}
		s.Source, s.RequestedRelease = intent.Source, job.ReleaseID
		s.Generation, s.JobID = &job.Generation, job.JobID
		s.Stage, s.State = jobProgress(job)
		selectTxn := txnSelect
		if version == 1 {
			selectTxn = strings.Replace(selectTxn, "candidate_health_json", "NULL", 1)
		}
		t, err := scanTxn(tx.QueryRowContext(ctx, selectTxn+` WHERE job_id = ? AND attempt = ? ORDER BY rowid DESC LIMIT 1`, job.JobID, job.Attempt))
		if err != nil && !errors.Is(err, ErrNotFound) {
			return s, err
		}
		if err == nil {
			s.JournalStage = t.Stage
			s.Stage, s.State = txnProgress(t)
			s.RecoveryRequired = s.RecoveryRequired || t.Stage == StageRecoveryRequired
			s.Health, s.CandidateHealth = t.Health, t.CandidateHealth
			if version == 1 {
				s.CandidateHealth = t.Health
				if t.Stage == StageRolledBack && t.Destructive {
					s.Health = nil // v1 did not store restored health here
				}
			}
		}
	}
	historyStage := "stage"
	if version == 1 {
		historyStage = "NULL"
	}
	var gen, resultJob, state string
	var resultStage sql.NullString
	var attempt int
	err = tx.QueryRowContext(ctx, `SELECT generation, job_id, attempt, state, stage FROM (
		SELECT generation, job_id, attempt, state, CASE WHEN json_valid(payload) THEN json_extract(payload, '$.stage') END AS stage, created_at FROM result_outbox
		UNION ALL SELECT generation, job_id, attempt, state, `+historyStage+`, created_at FROM result_history)
		ORDER BY length(generation) DESC, generation DESC, attempt DESC, created_at DESC LIMIT 1`).Scan(&gen, &resultJob, &attempt, &state, &resultStage)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return s, err
	}
	if err == nil {
		generation, err := decstr.Parse(gen)
		if err != nil {
			return s, err
		}
		s.LastResultState = state
		if s.Generation == nil || generation > *s.Generation || (generation == *s.Generation && attempt >= job.Attempt) {
			if s.Generation == nil || generation != *s.Generation || resultJob != s.JobID || attempt != job.Attempt {
				s.JournalStage, s.Health, s.CandidateHealth = "", nil, nil
			}
			s.Generation, s.JobID, s.State = &generation, resultJob, state
			if resultStage.Valid {
				s.Stage = resultStage.String
			} else if s.Stage == "" {
				s.Stage = "report"
			}
		}
	}
	if version >= 2 {
		var r Rejection
		var code, apiCode, at sql.NullString
		err = tx.QueryRowContext(ctx, `SELECT result_id, last_http_status, last_api_code, last_error, last_attempt_at FROM result_outbox
			WHERE status IN ('rejected', 'conflict') ORDER BY last_attempt_at DESC, seq DESC LIMIT 1`).Scan(&r.ResultID, &r.HTTPStatus, &apiCode, &code, &at)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return s, err
		}
		if err == nil {
			r.APICode, r.Code, r.At = apiCode.String, code.String, parseTS(at)
			s.LastRejection = &r
		}
	}
	return s, nil
}

func jobProgress(j Job) (stage, state string) {
	if j.Outcome != "" {
		return "report", j.Outcome
	}
	switch j.Phase {
	case JobDownloading:
		return "download", "downloading"
	case JobVerifying:
		return "verify", "verifying"
	case JobVerified, JobApplying:
		return "prepare", "applying"
	case JobChecking:
		return "check", "checking"
	default:
		return "claim", "queued"
	}
}

func txnProgress(t Txn) (stage, state string) {
	switch t.Stage {
	case StageLocallySucceeded:
		return "report", "succeeded"
	case StageRolledBack:
		if t.Destructive {
			return "report", "rolled_back"
		}
		return t.ErrorStage, "failed"
	case StageRecoveryRequired:
		return "rollback", "failed"
	case StageChecking:
		return "check", "checking"
	case StageRollingBack:
		return "rollback", "applying"
	case StagePrepared:
		return "prepare", "applying"
	case StageBlocked:
		return "block_forwarding", "applying"
	case StageFinalSampled:
		return "final_sample", "applying"
	case StageStopped:
		return "stop_services", "applying"
	case StageMigrated:
		return "migrate", "applying"
	case StageSwitched:
		return "switch", "applying"
	default:
		return "", ""
	}
}
