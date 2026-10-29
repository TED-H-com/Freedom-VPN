package updater

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

type memSource struct {
	mu   sync.Mutex
	work Work
	err  error
}

func (s *memSource) Current(context.Context) (Work, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return Work{}, s.err
	}
	return s.work, nil
}

func (s *memSource) set(w Work) {
	s.mu.Lock()
	s.work = w
	s.mu.Unlock()
}

type fakeGate struct {
	mu       sync.Mutex
	blocked  bool
	blocks   int
	unblocks int
	err      error
}

func (g *fakeGate) Block(context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.blocks++
	if g.err != nil {
		return g.err
	}
	g.blocked = true
	return nil
}

func (g *fakeGate) Unblock(context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.unblocks++
	g.blocked = false
	return nil
}

func (g *fakeGate) snapshot() (blocked bool, blocks, unblocks int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.blocked, g.blocks, g.unblocks
}

type fakeSvc struct {
	mu      sync.Mutex
	stops   int
	starts  int
	stopErr error
	last    string
}

func (s *fakeSvc) Stop(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stops++
	return s.stopErr
}

func (s *fakeSvc) Start(_ context.Context, dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.starts++
	s.last = dir
	return nil
}

func (s *fakeSvc) counts() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stops, s.starts
}

type fakeMeter struct {
	mu     sync.Mutex
	calls  int
	epochs int
	epoch  string
	err    error
	closed string
	sample UsageCheckpoint
}

func newMeter() *fakeMeter {
	closed := uuid.Must(uuid.NewV7()).String()
	return &fakeMeter{
		closed: closed,
		sample: UsageCheckpoint{
			ClosedMeterEpochID:  closed,
			LastDurableEventSeq: decstr.MustParse("10"),
			CentralAckedThrough: decstr.MustParse("9"),
		},
	}
}

func (m *fakeMeter) FinalSample(context.Context) (UsageCheckpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.err != nil {
		return UsageCheckpoint{}, m.err
	}
	return m.sample, nil
}

func (m *fakeMeter) OpenEpoch(context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.epochs++
	if m.epoch == "" {
		m.epoch = uuid.Must(uuid.NewV7()).String()
	}
	return m.epoch, nil
}

type fakeHealth struct {
	mu   sync.Mutex
	fail map[string]bool
}

func (h *fakeHealth) status(id string) api.HealthCheckStatus {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.fail[id] {
		return api.HealthCheckStatusFailed
	}
	return api.HealthCheckStatusPassed
}

func (h *fakeHealth) Processes(_ context.Context, v ReleaseView) (api.HealthCheckStatus, error) {
	return h.status(v.ReleaseID), nil
}
func (h *fakeHealth) PersistentData(_ context.Context, v ReleaseView) (api.HealthCheckStatus, error) {
	return h.status(v.ReleaseID), nil
}
func (h *fakeHealth) Isolation(_ context.Context, v ReleaseView) (api.HealthCheckStatus, error) {
	return h.status(v.ReleaseID), nil
}
func (h *fakeHealth) DNSPolicy(_ context.Context, v ReleaseView) (api.HealthCheckStatus, error) {
	return h.status(v.ReleaseID), nil
}
func (h *fakeHealth) TunnelProbe(_ context.Context, v ReleaseView) (api.HealthCheckStatus, error) {
	return h.status(v.ReleaseID), nil
}

type fakeMig struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (m *fakeMig) Migrate(context.Context, int, int, string, string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	return m.err
}

func writeMarker(t *testing.T, dir, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin", "fvpn-agent"), []byte(text), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte(text+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode().Perm())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func fixedNow() time.Time { return time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC) }

func testLayout(root string) Layout {
	return Layout{
		Releases: filepath.Join(root, "opt", "releases"),
		Current:  filepath.Join(root, "opt", "current"),
		Identity: filepath.Join(root, "var", "identity"),
		State:    filepath.Join(root, "var", "state"),
		Journal:  filepath.Join(root, "var", "journal"),
		Lock:     filepath.Join(root, "var", "update.lock"),
		Staging:  filepath.Join(root, "var", "staging"),
		Trust:    filepath.Join(root, "etc", "release-trust.json"),
	}
}

func testPlatform() Platform {
	return Platform{
		OSID: "ubuntu", OSVersion: "24.04", Architecture: "amd64",
		UpdaterVersion: "1.2.0", ControlAPIMajor: 1,
		PolicySchema: 1, UsageSchema: 1, StateSchema: 1,
	}
}

func fileBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func newTestServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewUnstartedServer(h)
	srv.EnableHTTP2 = false
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv
}

func tlsClient(srv *httptest.Server) *http.Client {
	base := srv.Client()
	if tr, ok := base.Transport.(*http.Transport); ok {
		tr.TLSClientConfig.MinVersion = tls.VersionTLS12
	}
	return base
}

// touchPersistent writes the identity and usage files the updater must not rewrite (§13.7).
func touchPersistent(t *testing.T, lay Layout) (idPath, usagePath string) {
	t.Helper()
	idPath = filepath.Join(lay.Identity, "node.key")
	usagePath = filepath.Join(lay.State, "usage-outbox")
	if err := os.MkdirAll(lay.Identity, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(lay.State, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(idPath, []byte("identity-v1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(usagePath, []byte("usage-v1"), 0o600); err != nil {
		t.Fatal(err)
	}
	return idPath, usagePath
}

func assertSameBytes(t *testing.T, path string, before []byte) {
	t.Helper()
	got := fileBytes(t, path)
	if string(got) != string(before) {
		t.Fatalf("%s changed", path)
	}
}

func openJournal(t *testing.T, dir string) *updatejournal.Journal {
	t.Helper()
	j, err := updatejournal.Open(dir, updatejournal.Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = j.Close() })
	return j
}

func treeOrDie(t *testing.T, dir string) string {
	t.Helper()
	sum, err := fsutil.TreeHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	return sum
}
