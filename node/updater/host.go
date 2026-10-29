package updater

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// Config is the updater's on-node configuration. It holds paths, never key
// bytes. The release private key is not a field: it lives only on the build
// machine (§16.2). allow_loopback is TEST ONLY.
type Config struct {
	Layout            Layout      `json:"layout"`
	NodeID            uuid.UUID   `json:"node_id"`
	IncarnationID     uuid.UUID   `json:"incarnation_id"`
	BootID            uuid.UUID   `json:"boot_id"`
	Revoked           bool        `json:"revoked"`
	Platform          Platform    `json:"platform"`
	ReleaseTrust      string      `json:"release_trust"`
	DesiredTrust      string      `json:"desired_trust"`
	DesiredState      string      `json:"desired_state"`
	RevokedReleaseIDs []uuid.UUID `json:"revoked_release_ids"`
	MeterCheckpoint   string      `json:"meter_checkpoint"`
	StopCommand       []string    `json:"stop_command"`
	StartCommand      []string    `json:"start_command"`
	ResultURL         string      `json:"result_url"`
	MTLSCert          string      `json:"mtls_cert"`
	MTLSKey           string      `json:"mtls_key"`
	MTLSCA            string      `json:"mtls_ca"`
	AllowLoopback     bool        `json:"allow_loopback"`
}

// LoadConfig reads a config file. Unknown fields are rejected. Empty layout
// fields fall back to DefaultLayout.
func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var c Config
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("updater: config: %w", err)
	}
	d := DefaultLayout()
	if c.Layout.Releases == "" {
		c.Layout.Releases = d.Releases
	}
	if c.Layout.Current == "" {
		c.Layout.Current = d.Current
	}
	if c.Layout.Identity == "" {
		c.Layout.Identity = d.Identity
	}
	if c.Layout.State == "" {
		c.Layout.State = d.State
	}
	if c.Layout.Journal == "" {
		c.Layout.Journal = d.Journal
	}
	if c.Layout.Lock == "" {
		c.Layout.Lock = d.Lock
	}
	if c.Layout.Staging == "" {
		c.Layout.Staging = d.Staging
	}
	if c.Layout.Trust == "" {
		c.Layout.Trust = d.Trust
	}
	if c.NodeID == uuid.Nil || c.IncarnationID == uuid.Nil || c.BootID == uuid.Nil {
		return Config{}, errors.New("updater: node_id, incarnation_id and boot_id are required")
	}
	if c.Platform.OSID == "" || c.Platform.OSVersion == "" || c.Platform.Architecture == "" || c.Platform.UpdaterVersion == "" {
		return Config{}, errors.New("updater: platform is incomplete")
	}
	if c.ReleaseTrust == "" || c.DesiredTrust == "" || c.DesiredState == "" || c.MeterCheckpoint == "" {
		return Config{}, errors.New("updater: release_trust, desired_trust, desired_state and meter_checkpoint are required")
	}
	return c, nil
}

// Engine builds the node engine. Health checks are NodeChecks, which fail
// closed for isolation, DNS and the tunnel probe until the agent supplies
// its own (§13.9). Service commands that are empty fail closed at stop.
func (c Config) Engine() (*Engine, error) {
	rel, err := readTrust(c.ReleaseTrust)
	if err != nil {
		return nil, err
	}
	desired, err := readTrust(c.DesiredTrust)
	if err != nil {
		return nil, err
	}
	client, err := NewMTLSClient(c.MTLSCert, c.MTLSKey, c.MTLSCA)
	if err != nil {
		return nil, err
	}
	node := Node{ID: c.NodeID, Incarnation: c.IncarnationID, BootID: c.BootID, Revoked: c.Revoked}
	gate := FileGate{Path: filepath.Join(c.Layout.Journal, "forwarding.blocked")}
	var sender ResultSink
	if c.ResultURL != "" {
		if err := checkURL(c.ResultURL, c.AllowLoopback); err != nil {
			return nil, err
		}
		sender = HTTPSink{URL: c.ResultURL, Client: client, AllowLoopback: c.AllowLoopback}
	}
	return &Engine{
		Layout:       c.Layout,
		Node:         node,
		Platform:     c.Platform,
		ReleaseTrust: rel,
		DesiredTrust: desired,
		Source: GuardSource{
			Inner: FileSource{
				Path: c.DesiredState, Trust: desired, Node: node,
				Revoked: append([]uuid.UUID(nil), c.RevokedReleaseIDs...),
			},
			Gate: gate,
		},
		HTTP:          client,
		Meter:         FileMeter{Path: c.MeterCheckpoint},
		Services:      CommandServices{StopCmd: c.StopCommand, StartCmd: c.StartCommand},
		Health:        NodeChecks{},
		Gate:          gate,
		Migrator:      noMigrator{},
		Sender:        sender,
		Download:      DefaultDownload(),
		AllowLoopback: c.AllowLoopback,
	}, nil
}

func readTrust(path string) (*envelope.TrustStore, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ts, err := envelope.ParseNodeTrustStoreJSON(b)
	if err != nil {
		return nil, fmt.Errorf("updater: trust %s: %w", filepath.Base(path), err)
	}
	return ts, nil
}

// ExitCode maps an engine error to a process status. 2 means member
// forwarding must stay off (§13.5, §13.9).
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, ErrMembersBlocked) {
		return 2
	}
	return 1
}

// FileGate is the stand-in forwarding latch the agent reads before it
// enables member forwarding. The real nftables gate belongs to the agent.
// Block writes a file; Unblock removes it. Both are idempotent.
type FileGate struct {
	Path string
}

func (g FileGate) Block(context.Context) error {
	if g.Path == "" {
		return errors.New("updater: no forwarding gate")
	}
	if err := os.MkdirAll(filepath.Dir(g.Path), 0o700); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(g.Path, []byte("blocked\n"), 0o600)
}

func (g FileGate) Unblock(context.Context) error {
	if g.Path == "" {
		return errors.New("updater: no forwarding gate")
	}
	err := os.Remove(g.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// Blocked reports whether the latch file is present.
func (g FileGate) Blocked() bool {
	if g.Path == "" {
		return true
	}
	_, err := os.Lstat(g.Path)
	return err == nil
}

// noMigrator refuses a schema change. Config.Engine installs it so a release
// that needs a migration fails before services stop when no migrator was
// supplied (D-034c).
type noMigrator struct{}

func (noMigrator) Migrate(context.Context, int, int, string, string) error {
	return permanent(apierr.MigrationFailed, "no state migrator configured")
}

func (e *Engine) migrationUnsupported() bool {
	if e.Migrator == nil {
		return true
	}
	_, ok := e.Migrator.(noMigrator)
	return ok
}

// FileMeter reads a metering barrier the meter process wrote. It does not
// invent sequence numbers. OpenEpoch returns a new id and writes nothing:
// the per-transaction journal note is the idempotency record, and it is
// removed when the transaction completes (D-034c).
type FileMeter struct {
	Path string
}

func (m FileMeter) FinalSample(context.Context) (UsageCheckpoint, error) {
	b, err := os.ReadFile(m.Path)
	if err != nil {
		return UsageCheckpoint{}, fmt.Errorf("updater: meter checkpoint: %w", err)
	}
	var c UsageCheckpoint
	if err := json.Unmarshal(b, &c); err != nil {
		return UsageCheckpoint{}, permanent(apierr.CheckpointFailed, "meter checkpoint")
	}
	if _, err := uuid.Parse(c.ClosedMeterEpochID); err != nil {
		return UsageCheckpoint{}, permanent(apierr.CheckpointFailed, "meter checkpoint has no closed epoch")
	}
	return c, nil
}

func (m FileMeter) OpenEpoch(context.Context) (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// CommandServices runs the configured stop and start commands. An empty
// command fails closed: the updater does not guess systemd. Start appends
// the release directory as the last argument and does not execute anything
// from the manifest (§13.5).
type CommandServices struct {
	StopCmd  []string
	StartCmd []string
}

func (c CommandServices) Stop(ctx context.Context) error {
	return runArgv(ctx, c.StopCmd, nil)
}

func (c CommandServices) Start(ctx context.Context, releaseDir string) error {
	if releaseDir == "" {
		return errors.New("updater: empty release directory")
	}
	return runArgv(ctx, c.StartCmd, []string{releaseDir})
}

func runArgv(ctx context.Context, argv, extra []string) error {
	if len(argv) == 0 || argv[0] == "" {
		return errors.New("updater: service command is not configured")
	}
	args := append(append([]string{}, argv[1:]...), extra...)
	cmd := exec.CommandContext(ctx, argv[0], args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("updater: service command failed: %w", err)
	}
	return nil
}

// FileSource reads a desired-state envelope from disk and verifies it on
// every read (§13.4, §14.7). The agent writes the file; the updater does
// not trust it until the signature checks.
type FileSource struct {
	Path    string
	Trust   *envelope.TrustStore
	Node    Node
	Revoked []uuid.UUID
	Now     func() time.Time
}

func (s FileSource) Current(context.Context) (Work, error) {
	b, err := os.ReadFile(s.Path)
	if err != nil {
		return Work{}, err
	}
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	return WorkFromEnvelope(b, s.Trust, s.Node, now, s.Revoked)
}

// GuardSource turns member forwarding off when the desired state cannot be
// verified. ErrNoWork is not a failure: a valid state with no job leaves
// the latch to the last publication decision (§13.2).
type GuardSource struct {
	Inner Source
	Gate  Gate
}

func (s GuardSource) Current(ctx context.Context) (Work, error) {
	w, err := s.Inner.Current(ctx)
	if err != nil && !errors.Is(err, ErrNoWork) && s.Gate != nil {
		_ = s.Gate.Block(ctx)
	}
	return w, err
}

// WorkFromEnvelope verifies a desired-state envelope and builds the work
// item. revoked is the unsigned release-support list from the agent. A
// present signed revoked_release_ids member, including an empty array, is
// returned in SignedRevoked and does not replace that list. An absent
// member leaves SignedRevoked nil so the caller keeps today's unsigned
// behaviour. PolicyGeneration is the envelope's config_generation (D-043),
// not the update job generation. ErrNoWork still carries that policy and
// any signed revocations so a revoked current release can be stopped
// without an install job.
func WorkFromEnvelope(raw []byte, trust *envelope.TrustStore, node Node, now time.Time, revoked []uuid.UUID) (Work, error) {
	signed, _, err := envelope.Verify(raw, trust, envelope.PurposeDesiredState, now)
	if err != nil {
		if errors.Is(err, envelope.ErrKeyUntrusted) {
			return Work{}, permanent(apierr.KeyUntrusted, "desired state")
		}
		return Work{}, permanent(apierr.SignatureInvalid, "desired state")
	}
	var ds api.DesiredState
	if err := json.Unmarshal(signed, &ds); err != nil {
		return Work{}, permanent(apierr.SignatureInvalid, "desired state")
	}
	if ds.NodeId != node.ID || ds.NodeIncarnationId != node.Incarnation {
		return Work{}, permanent(apierr.IncarnationMismatch, "desired state")
	}
	// Validity is half-open [issued_at, expires_at) (§14.7).
	if now.Before(ds.IssuedAt) || !now.Before(ds.ExpiresAt) {
		return Work{}, permanent(apierr.AuthorizationExpired, "desired state")
	}
	if len(ds.ServingReleaseIds) == 0 {
		return Work{}, permanent(apierr.MinimumVersionViolation, "empty serving set")
	}
	work := Work{
		Serving:          append([]uuid.UUID(nil), ds.ServingReleaseIds...),
		Revoked:          append([]uuid.UUID(nil), revoked...),
		PolicyGeneration: ds.ConfigGeneration,
		PolicyKnown:      true,
	}
	if ds.RevokedReleaseIds != nil {
		ids := append([]uuid.UUID(nil), (*ds.RevokedReleaseIds)...)
		work.SignedRevoked = &ids
	}
	// The disk reader previously treated both omitted and null update as no
	// work. Keep that behavior; central producers must emit the required member.
	if !ds.Update.IsSpecified() || ds.Update.IsNull() {
		work.Intent.PublicationIntent = ds.PublicationIntent
		work.Intent.NodeId = ds.NodeId
		work.Intent.IncarnationId = ds.NodeIncarnationId
		return work, ErrNoWork
	}
	update, _ := ds.Update.Get()
	if update.NodeId != node.ID || update.IncarnationId != node.Incarnation {
		return Work{}, permanent(apierr.IncarnationMismatch, "update intent")
	}
	work.Intent = update
	return work, nil
}

// HTTPSink posts an update result. The contract answers 202. 409 is
// ACK_CONFLICT. Other statuses stay pending and are retried (§13.10.3).
type HTTPSink struct {
	URL           string
	Client        *http.Client
	AllowLoopback bool
}

func (s HTTPSink) Submit(ctx context.Context, payload []byte) (Accepted, bool, error) {
	if err := checkURL(s.URL, s.AllowLoopback); err != nil {
		return Accepted{}, false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, bytes.NewReader(payload))
	if err != nil {
		return Accepted{}, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "fvpn-updater/"+Version)
	var probe struct {
		ResultID string `json:"result_id"`
	}
	if json.Unmarshal(payload, &probe) == nil {
		if id, err := uuid.Parse(probe.ResultID); err == nil {
			req.Header.Set("Idempotency-Key", id.String())
		}
	}
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Accepted{}, false, transient(apierr.StoreUnavailable, "result submit")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Accepted{}, false, transient(apierr.StoreUnavailable, "result submit")
	}
	switch resp.StatusCode {
	case http.StatusAccepted:
		var ack api.UpdateResultAck
		if err := json.Unmarshal(body, &ack); err != nil {
			return Accepted{}, false, transient(apierr.StoreUnavailable, "result ack")
		}
		return Accepted{AcceptedAsHistory: ack.AcceptedAsHistory, Raw: append([]byte(nil), body...)}, false, nil
	case http.StatusConflict:
		return Accepted{}, true, nil
	default:
		return Accepted{}, false, transient(apierr.StoreUnavailable, fmt.Sprintf("result submit status %d", resp.StatusCode))
	}
}

// NewMTLSClient builds the agent HTTP client. Empty paths mean the default
// system roots and no client certificate. The error text does not include
// file contents.
func NewMTLSClient(certFile, keyFile, caFile string) (*http.Client, error) {
	if certFile == "" && keyFile == "" && caFile == "" {
		return &http.Client{Timeout: 30 * time.Second}, nil
	}
	if (certFile == "") != (keyFile == "") {
		return nil, errors.New("updater: mtls cert and key must both be set")
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if certFile != "" {
		pair, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, errors.New("updater: mtls certificate")
		}
		cfg.Certificates = []tls.Certificate{pair}
	}
	if caFile != "" {
		b, err := os.ReadFile(caFile)
		if err != nil {
			return nil, errors.New("updater: mtls ca")
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(b) {
			return nil, errors.New("updater: mtls ca")
		}
		cfg.RootCAs = pool
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: &http.Transport{TLSClientConfig: cfg},
	}, nil
}
