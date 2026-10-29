//go:build e2e

package updates

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	_ "modernc.org/sqlite"
)

type nodeIdentity struct {
	NodeID      uuid.UUID
	Incarnation uuid.UUID
	Publication string
}

func readIdentity(path string) (nodeIdentity, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nodeIdentity{}, err
	}
	var f struct {
		NodeID            string `json:"node_id"`
		NodeIncarnationID string `json:"node_incarnation_id"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nodeIdentity{}, err
	}
	id, err := uuid.Parse(f.NodeID)
	if err != nil {
		return nodeIdentity{}, err
	}
	inc, err := uuid.Parse(f.NodeIncarnationID)
	if err != nil {
		return nodeIdentity{}, err
	}
	return nodeIdentity{NodeID: id, Incarnation: inc}, nil
}

func (l *lab) pause(varDir string) error {
	if err := os.MkdirAll(filepath.Join(varDir, "e2e"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(varDir, "e2e", "pause-agent"), []byte("1\n"), 0o644)
}

func (l *lab) unpause(varDir string) {
	_ = os.Remove(filepath.Join(varDir, "e2e", "pause-agent"))
}

type updaterRun struct {
	Exit     int
	Journal  journalSnap
	Out      string
	Finished bool
}

type journalSnap struct {
	Stage, ErrorCode, ErrorStage, ErrorDetail, Health string
	LastGood, Current, RollbackRelease, Forwarding    string
	ArtifactSHA, OutboxState, OutboxError             string
	Events                                            string
	Text                                              string
}

type runOpts struct {
	container    string
	varDir       string
	node         nodeIdentity
	arch         string
	releaseTrust string
	staging      string
	releases     string
	stop         []string
	start        []string
	timeout      time.Duration
}

func (l *lab) runUpdater(ctx context.Context, opt runOpts) (updaterRun, error) {
	var out updaterRun
	if opt.timeout <= 0 {
		opt.timeout = 4 * time.Minute
	}
	if opt.arch == "" {
		opt.arch = "amd64"
	}
	if len(opt.stop) == 0 {
		opt.stop = []string{"/bin/true"}
	}
	if len(opt.start) == 0 {
		opt.start = []string{"/bin/true"}
	}
	bootRaw, err := l.exec(15*time.Second, opt.container, "cat", "/etc/vpn-node/boot-id")
	if err != nil {
		return out, fmt.Errorf("boot id: %w", err)
	}
	bootID, err := uuid.Parse(strings.TrimSpace(string(bootRaw)))
	if err != nil {
		return out, fmt.Errorf("boot id parse: %w", err)
	}
	trust := opt.releaseTrust
	if trust == "" {
		trust = "/etc/vpn-node/trust.json"
	}
	cfg := map[string]any{
		"node_id": opt.node.NodeID.String(), "incarnation_id": opt.node.Incarnation.String(),
		"boot_id": bootID.String(), "revoked": false,
		"platform": map[string]any{
			"os_id": "ubuntu", "os_version": "24.04", "architecture": opt.arch, "updater_version": "1.0.0",
			"control_api_major": 1, "policy_schema": 1, "usage_schema": 1, "state_schema": 1,
		},
		"release_trust": trust, "desired_trust": "/etc/vpn-node/trust.json",
		"desired_state":    "/var/lib/vpn-node/state/desired-state.json",
		"meter_checkpoint": "/var/lib/vpn-node/e2e/meter.json",
		"stop_command":     opt.stop, "start_command": opt.start,
		"result_url":               l.agent + "/api/v1/agent/update-results",
		"allowed_artifact_origins": []string{l.agent, l.releases},
		"mtls_cert":                "/var/lib/vpn-node/identity/cert.pem",
		"mtls_key":                 "/var/lib/vpn-node/identity/key.pem",
		"mtls_ca":                  "/etc/vpn-node/ca.pem",
	}
	if opt.staging != "" || opt.releases != "" {
		layout := map[string]string{}
		if opt.staging != "" {
			layout["staging"] = opt.staging
		}
		if opt.releases != "" {
			layout["releases"] = opt.releases
		}
		cfg["layout"] = layout
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return out, err
	}
	e2e := filepath.Join(opt.varDir, "e2e")
	if err := os.MkdirAll(e2e, 0o755); err != nil {
		return out, err
	}
	_ = os.Remove(filepath.Join(e2e, "updater-result.json"))
	if err := os.WriteFile(filepath.Join(e2e, "updater.json"), append(raw, '\n'), 0o644); err != nil {
		return out, err
	}
	wait, cancel := context.WithTimeout(ctx, opt.timeout)
	defer cancel()
	out.Finished = waitUntil(wait, 2*time.Second, func() bool {
		_, err := os.Stat(filepath.Join(e2e, "updater-result.json"))
		return err == nil
	})
	if !out.Finished {
		return out, nil
	}
	body, _ := os.ReadFile(filepath.Join(e2e, "updater-result.json"))
	var exit struct {
		Code int `json:"exit_code"`
	}
	_ = json.Unmarshal(body, &exit)
	out.Exit = exit.Code
	logBody, _ := os.ReadFile(filepath.Join(e2e, "updater.out"))
	out.Out = clip(string(logBody), 700)
	out.Journal = readJournal(opt.varDir)
	return out, nil
}

func readJournal(varDir string) journalSnap {
	var view journalSnap
	src := filepath.Join(varDir, "update-journal", "journal.db")
	if _, err := os.Stat(src); err != nil {
		return view
	}
	tmp := filepath.Join(varDir, "e2e", "journal-copy.db")
	_ = copyFile(src, tmp)
	if b, err := os.ReadFile(src + "-wal"); err == nil {
		_ = os.WriteFile(tmp+"-wal", b, 0o644)
	}
	db, err := sql.Open("sqlite", "file:"+tmp+"?_pragma=busy_timeout(2000)")
	if err != nil {
		return view
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = db.QueryRowContext(ctx, `SELECT stage, COALESCE(error_code,''), COALESCE(error_stage,''), COALESCE(error_detail,''), COALESCE(health_json,''), COALESCE(rollback_release_id,'') FROM txns ORDER BY updated_at DESC LIMIT 1`).
		Scan(&view.Stage, &view.ErrorCode, &view.ErrorStage, &view.ErrorDetail, &view.Health, &view.RollbackRelease)
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(value,'') FROM meta WHERE key='current_release_id'`).Scan(&view.Current)
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(value,'') FROM meta WHERE key='last_good_release_id'`).Scan(&view.LastGood)
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(value,'') FROM meta WHERE key='forwarding_blocked'`).Scan(&view.Forwarding)
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(sha256,'') FROM artifacts ORDER BY verified_at DESC LIMIT 1`).Scan(&view.ArtifactSHA)
	_ = db.QueryRowContext(ctx, `SELECT state, COALESCE(last_error,'') FROM result_outbox ORDER BY seq DESC LIMIT 1`).Scan(&view.OutboxState, &view.OutboxError)
	rows, err := db.QueryContext(ctx, `SELECT seq, stage, COALESCE(detail,'') FROM txn_events ORDER BY seq`)
	if err == nil {
		var b strings.Builder
		for rows.Next() {
			var seq int
			var stage, detail string
			if rows.Scan(&seq, &stage, &detail) == nil {
				fmt.Fprintf(&b, "seq=%d stage=%s detail=%s\n", seq, stage, detail)
			}
		}
		rows.Close()
		view.Events = b.String()
	}
	view.Text = fmt.Sprintf("stage=%s error_code=%s error_stage=%s error_detail=%s health=%s current=%s last_good=%s rollback_release=%s forwarding_blocked=%s artifact=%s outbox=%s %s",
		view.Stage, view.ErrorCode, view.ErrorStage, view.ErrorDetail, view.Health, view.Current, view.LastGood, view.RollbackRelease, view.Forwarding, view.ArtifactSHA, view.OutboxState, view.OutboxError)
	return view
}

func journalForRelease(varDir, releaseID string) journalSnap {
	var view journalSnap
	src := filepath.Join(varDir, "update-journal", "journal.db")
	tmp := filepath.Join(varDir, "e2e", "journal-copy.db")
	if err := copyFile(src, tmp); err != nil {
		return view
	}
	if b, err := os.ReadFile(src + "-wal"); err == nil {
		_ = os.WriteFile(tmp+"-wal", b, 0o644)
	}
	db, err := sql.Open("sqlite", "file:"+tmp+"?_pragma=busy_timeout(2000)")
	if err != nil {
		return view
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = db.QueryRowContext(ctx, `SELECT stage, COALESCE(error_code,''), COALESCE(error_stage,''), COALESCE(error_detail,''), COALESCE(health_json,''), COALESCE(rollback_release_id,'') FROM txns WHERE release_id=? ORDER BY updated_at DESC LIMIT 1`, releaseID).
		Scan(&view.Stage, &view.ErrorCode, &view.ErrorStage, &view.ErrorDetail, &view.Health, &view.RollbackRelease)
	if err != nil {
		view.ErrorDetail = "no txn for release"
		return view
	}
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(value,'') FROM meta WHERE key='current_release_id'`).Scan(&view.Current)
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(value,'') FROM meta WHERE key='last_good_release_id'`).Scan(&view.LastGood)
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(value,'') FROM meta WHERE key='forwarding_blocked'`).Scan(&view.Forwarding)
	rows, qerr := db.QueryContext(ctx, `SELECT e.seq, e.stage, COALESCE(e.detail,'') FROM txn_events e JOIN txns t ON t.txn_id=e.txn_id WHERE t.release_id=? ORDER BY e.seq`, releaseID)
	if qerr == nil {
		var b strings.Builder
		for rows.Next() {
			var seq int
			var stage, detail string
			if rows.Scan(&seq, &stage, &detail) == nil {
				fmt.Fprintf(&b, "seq=%d stage=%s detail=%s\n", seq, stage, detail)
			}
		}
		rows.Close()
		view.Events = b.String()
	}
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(sha256,'') FROM artifacts WHERE release_id=? ORDER BY verified_at DESC LIMIT 1`, releaseID).Scan(&view.ArtifactSHA)
	view.Text = fmt.Sprintf("stage=%s error_code=%s error_stage=%s error_detail=%s health=%s current=%s last_good=%s rollback_release=%s artifact=%s",
		view.Stage, view.ErrorCode, view.ErrorStage, view.ErrorDetail, view.Health, view.Current, view.LastGood, view.RollbackRelease, view.ArtifactSHA)
	return view
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

func (l *lab) desiredHas(varDir, releaseID string) bool {
	b, err := os.ReadFile(filepath.Join(varDir, "state", "desired-state.json"))
	return err == nil && strings.Contains(string(b), releaseID)
}

type issuedJob struct {
	JobID, BatchID, TargetID, ReleaseID, Manifest, Generation, Publication, ConfigGeneration string
	Attempt                                                                                  int
	Issued, Until                                                                            time.Time
}

func parseIssued(varDir string) (issuedJob, bool) {
	b, err := os.ReadFile(filepath.Join(varDir, "state", "desired-state.json"))
	if err != nil {
		return issuedJob{}, false
	}
	var env struct {
		Signed struct {
			ConfigGeneration string          `json:"config_generation"`
			Update           json.RawMessage `json:"update"`
		} `json:"signed"`
	}
	if json.Unmarshal(b, &env) != nil || len(env.Signed.Update) == 0 || string(env.Signed.Update) == "null" {
		return issuedJob{}, false
	}
	var intent struct {
		JobID       string  `json:"job_id"`
		Attempt     int     `json:"attempt"`
		Generation  string  `json:"generation"`
		ReleaseID   string  `json:"release_id"`
		Manifest    string  `json:"manifest_sha256"`
		BatchID     *string `json:"batch_id"`
		TargetID    *string `json:"target_id"`
		Issued      string  `json:"issued_at"`
		Until       string  `json:"authorization_valid_until"`
		Publication string  `json:"publication_intent"`
	}
	if json.Unmarshal(env.Signed.Update, &intent) != nil || intent.JobID == "" {
		return issuedJob{}, false
	}
	issued, _ := time.Parse(time.RFC3339, intent.Issued)
	if issued.IsZero() {
		issued, _ = time.Parse(time.RFC3339Nano, intent.Issued)
	}
	until, _ := time.Parse(time.RFC3339, intent.Until)
	if until.IsZero() {
		until, _ = time.Parse(time.RFC3339Nano, intent.Until)
	}
	job := issuedJob{
		JobID: intent.JobID, ReleaseID: intent.ReleaseID, Manifest: intent.Manifest,
		Generation: intent.Generation, Publication: intent.Publication, Attempt: intent.Attempt,
		Issued: issued, Until: until, ConfigGeneration: env.Signed.ConfigGeneration,
	}
	if intent.BatchID != nil {
		job.BatchID = *intent.BatchID
	}
	if intent.TargetID != nil {
		job.TargetID = *intent.TargetID
	}
	return job, true
}

func (l *lab) postResult(container, varDir string, node nodeIdentity, job issuedJob, state string, terminal bool) (int, string, error) {
	started := job.Issued.Add(time.Second)
	if !started.Before(job.Until) {
		started = job.Issued
	}
	gen, err := decstr.Parse(job.Generation)
	if err != nil {
		gen = decstr.MustParse("0")
	}
	policy, err := decstr.Parse(zero(job.ConfigGeneration))
	if err != nil {
		policy = decstr.MustParse("0")
	}
	batch := presence.NewNullNullable[openapi_types.UUID]()
	if id, err := uuid.Parse(job.BatchID); err == nil {
		batch = presence.NewNullableWithValue(openapi_types.UUID(id))
	}
	target := presence.NewNullNullable[openapi_types.UUID]()
	if id, err := uuid.Parse(job.TargetID); err == nil {
		target = presence.NewNullableWithValue(openapi_types.UUID(id))
	}
	current := presence.NewNullNullable[openapi_types.UUID]()
	finished := presence.NewNullNullable[time.Time]()
	health := presence.NewNullNullable[api.UpdateHealth]()
	errCode := presence.NewNullNullable[api.ErrorCode]()
	if terminal {
		if id, err := uuid.Parse(job.ReleaseID); err == nil {
			current = presence.NewNullableWithValue(openapi_types.UUID(id))
		}
		finished = presence.NewNullableWithValue(time.Now().UTC())
		health = presence.NewNullableWithValue(api.UpdateHealth{
			Processes: api.HealthCheckStatusFailed, IdentityAndState: api.HealthCheckStatusFailed,
			InternetOnly: api.HealthCheckStatusFailed, DnsPolicy: api.HealthCheckStatusFailed,
			TunnelProbe: api.HealthCheckStatusFailed,
		})
		errCode = presence.NewNullableWithValue(api.ErrorCode("HASH_MISMATCH"))
	}
	pub := api.PublicationIntent(job.Publication)
	if pub == "" {
		pub = api.PublicationIntentUnpublished
	}
	resultID, err := uuid.NewV7()
	if err != nil {
		return 0, "", err
	}
	jobID, err := uuid.Parse(job.JobID)
	if err != nil {
		return 0, "", err
	}
	relID, err := uuid.Parse(job.ReleaseID)
	if err != nil {
		return 0, "", err
	}
	r := api.UpdateResult{
		ResultId: resultID, NodeId: openapi_types.UUID(node.NodeID), IncarnationId: openapi_types.UUID(node.Incarnation),
		BootId: uuid.New(), BatchId: batch, TargetId: target, JobId: openapi_types.UUID(jobID),
		Attempt: job.Attempt, Generation: gen, State: api.UpdateResultState(state),
		RequestedReleaseId: openapi_types.UUID(relID), CurrentReleaseId: current,
		ManifestSha256: job.Manifest, PublicationIntent: pub, StartedAt: started.UTC(),
		StateSchemaVersion: 1, Health: health,
		UsageCheckpoint: presence.NewNullNullable[api.UpdateUsageCheckpoint](),
		ErrorCode:       errCode, FinishedAt: finished,
		PreviousReleaseId: presence.NewNullNullable[openapi_types.UUID](),
		RecoveryRequired:  presence.NewRequired(false),
		Stage:             presence.NewNullNullable[api.UpdateStage](),
		PolicyGeneration:  policy,
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return 0, "", err
	}
	r.PayloadHash, err = canonjson.HashHex(raw, "payload_hash")
	if err != nil {
		return 0, "", err
	}
	raw, err = json.Marshal(r)
	if err != nil {
		return 0, "", err
	}
	e2e := filepath.Join(varDir, "e2e")
	if err := os.WriteFile(filepath.Join(e2e, "result.json"), raw, 0o644); err != nil {
		return 0, "", err
	}
	_ = os.Remove(filepath.Join(e2e, "result-ack.json"))
	out, err := l.exec(30*time.Second, container, "wget", "-S", "-O", "/var/lib/vpn-node/e2e/result-ack.json",
		"--ca-certificate=/etc/vpn-node/ca.pem",
		"--certificate=/var/lib/vpn-node/identity/cert.pem",
		"--private-key=/var/lib/vpn-node/identity/key.pem",
		"--header=Content-Type: application/json",
		"--post-file=/var/lib/vpn-node/e2e/result.json",
		l.agent+"/api/v1/agent/update-results")
	ack, _ := os.ReadFile(filepath.Join(e2e, "result-ack.json"))
	status := httpStatus(string(out))
	if err != nil && status == 0 {
		return 0, clip(string(out)+" "+string(ack), 400), err
	}
	return status, clip(string(ack), 500), nil
}

func httpStatus(stderr string) int {
	// wget -S prints the status line. The last one is the final response.
	status := 0
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "HTTP/") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				fmt.Sscanf(fields[1], "%d", &status)
			}
		}
	}
	return status
}

func zero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

func (l *lab) currentLink(container string) string {
	out, err := l.exec(10*time.Second, container, "readlink", "/opt/vpn-node/current")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (l *lab) releaseTree(container, id string) bool {
	_, err := l.exec(10*time.Second, container, "test", "-d", "/opt/vpn-node/releases/"+id)
	return err == nil
}

func (l *lab) identitySHA(varDir string) string {
	b, err := os.ReadFile(filepath.Join(varDir, "identity", "node.json"))
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (l *lab) forwardSummary(container string) string {
	out, err := l.exec(15*time.Second, container, "nft", "list", "chain", "inet", "fvpn", "forward")
	if err != nil {
		return "absent"
	}
	text := string(out)
	accepts := strings.Count(text, "accept")
	return fmt.Sprintf("present accepts=%d", accepts)
}

type auxNode struct {
	name      string
	container string
	ip        string
	varDir    string
	etcDir    string
	identity  nodeIdentity
}

func (l *lab) startAux(role, ip string, pause bool) (auxNode, error) {
	name := l.prefix + "-" + role
	base := filepath.Join(l.state, "aux", role)
	etcDir := filepath.Join(base, "etc")
	varDir := filepath.Join(base, "var")
	_ = os.RemoveAll(base)
	if err := os.MkdirAll(filepath.Join(varDir, "e2e"), 0o755); err != nil {
		return auxNode{}, err
	}
	if err := copyTree(filepath.Join(l.state, "node", "etc"), etcDir); err != nil {
		return auxNode{}, err
	}
	if pause {
		if err := l.pause(varDir); err != nil {
			return auxNode{}, err
		}
	}
	_ = exec.Command("docker", "rm", "-f", name).Run()
	cmd := exec.Command("docker", "run", "-d", "--name", name, "--privileged",
		"--network", l.prefix, "--ip", ip,
		"-v", etcDir+":/etc/vpn-node",
		"-v", varDir+":/var/lib/vpn-node",
		l.prefix+"-node:local")
	if out, err := cmd.CombinedOutput(); err != nil {
		return auxNode{}, fmt.Errorf("docker run %s: %s", role, clip(string(out), 300))
	}
	connect := exec.Command("docker", "network", "connect", l.prefix+"-origin", name)
	if out, err := connect.CombinedOutput(); err != nil {
		return auxNode{}, fmt.Errorf("origin network %s: %s", role, clip(string(out), 200))
	}
	return auxNode{name: role, container: name, ip: ip, varDir: varDir, etcDir: etcDir}, nil
}

func (l *lab) removeAux(names ...string) {
	for _, name := range names {
		_ = exec.Command("docker", "rm", "-f", name).Run()
	}
}

func copyTree(src, dst string) error {
	cmd := exec.Command("cp", "-a", src, dst)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("copy etc: %s", clip(string(out), 200))
	}
	return nil
}

func (a *auxNode) waitEnrolled(ctx context.Context) bool {
	return waitUntil(ctx, time.Second, func() bool {
		b, err := os.ReadFile(filepath.Join(a.varDir, "e2e", "status.json"))
		return err == nil && strings.Contains(string(b), `"enrolled"`)
	})
}

func (l *lab) writeToken(varDir, token string) error {
	if err := os.MkdirAll(filepath.Join(varDir, "e2e"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(varDir, "e2e", "enrollment.token"), []byte(token+"\n"), 0o600)
}

func (l *lab) hideToken(varDir string) {
	_ = os.Remove(filepath.Join(varDir, "e2e", "enrollment.token"))
}

// stopUpdater ends a run that outlived its wait. The entrypoint writes
// updater-result.json only after the process exits, so the next config must
// not be dropped by that deferred rename (§13.7).
func (l *lab) stopUpdater(container, varDir string) {
	result := filepath.Join(varDir, "e2e", "updater-result.json")
	if _, err := os.Stat(result); err == nil {
		return
	}
	script := `for d in /proc/[0-9]*; do comm=$(cat "$d/comm" 2>/dev/null || true); if [ "$comm" = "fvpn-updater" ]; then kill "$(basename "$d")" 2>/dev/null || true; fi; done`
	_, _ = l.exec(15*time.Second, container, "sh", "-c", script)
	wait, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = waitUntil(wait, time.Second, func() bool {
		_, err := os.Stat(result)
		return err == nil
	})
}

func (l *lab) imageDigests() string {
	out, err := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}} {{.ID}}").Output()
	if err != nil {
		return ""
	}
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, l.prefix+"-") {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	return strings.Join(lines, "; ")
}

func gitHead(root string) string {
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
