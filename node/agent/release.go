package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

var errActiveRelease = errors.New("agent: active release identity is inconsistent")

// Resolve once per process: an old process must not claim the candidate merely
// because the updater moved current. The journal and pointer must name the
// executable actually running. Configuration files stay read-only.
func (e *Engine) loadActiveRelease(ctx context.Context, executable string) error {
	current, err := filepath.EvalSymlinks(e.cfg.CurrentReleasePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} // unpackaged development
	if err != nil {
		return errActiveRelease
	}
	id, err := uuid.Parse(filepath.Base(current))
	if err != nil || id == uuid.Nil {
		return errActiveRelease
	}
	running, err := os.Stat(executable)
	if err != nil {
		return errActiveRelease
	}
	active, err := os.Stat(filepath.Join(current, "fvpn-agent"))
	if err != nil || !os.SameFile(running, active) {
		return errActiveRelease
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	snap, err := updatejournal.Snapshot(ctx, e.cfg.UpdaterJournal)
	if errors.Is(err, os.ErrNotExist) && id == e.cfg.ReleaseUUID() {
		return nil
	} // initial install, before seed
	if err != nil {
		return errActiveRelease
	}
	// Installed releases are immutable records of the updater's verified
	// envelope. Check the recorded digest before reading component identity.
	u := url.URL{Scheme: "file", Path: e.cfg.UpdaterJournal, RawQuery: "mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(50)"}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return errActiveRelease
	}
	defer db.Close()
	if snap.CurrentRelease == "" && snap.LastGoodRelease == "" && snap.Generation == nil && snap.JobID == "" && !snap.RecoveryRequired && snap.SeededBaseline == nil && snap.LastResultState == "" && snap.LastRejection == nil && id == e.cfg.ReleaseUUID() {
		// The packaged recover service initializes an empty journal before
		// bootstrap. Installer seed waits for that bootstrap's setup key.
		// Only the configured initial executable may cross this boundary;
		// any durable update history requires a real current-release record.
		var history int
		if err := db.QueryRowContext(ctx, `SELECT
			(SELECT count(*) FROM releases) + (SELECT count(*) FROM jobs) +
			(SELECT count(*) FROM txns) + (SELECT count(*) FROM result_outbox) +
			(SELECT count(*) FROM result_history)`).Scan(&history); err == nil && history == 0 {
			return nil
		}
		return errActiveRelease
	}
	if snap.CurrentRelease != id.String() {
		return errActiveRelease
	}
	var raw []byte
	var digest string
	if err := db.QueryRowContext(ctx, `SELECT manifest_envelope, manifest_sha256 FROM releases WHERE release_id = ?`, id.String()).Scan(&raw, &digest); err != nil {
		return errActiveRelease
	}
	hash, err := canonjson.HashHex(raw)
	if err != nil || hash != digest {
		return errActiveRelease
	}
	var env struct {
		Signed api.ReleaseManifest `json:"signed"`
	}
	if json.Unmarshal(raw, &env) != nil || env.Signed.ReleaseId != id || !releaseRE.MatchString(env.Signed.Components.NodeAgent) {
		return errActiveRelease
	}
	e.cfg.ReleaseID, e.cfg.AgentRelease = id.String(), env.Signed.Components.NodeAgent
	return nil
}
