package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// ServiceManager is the unit control the installer uses. Production is
// systemctl. Tests inject MemServices. FVPN_INSTALL_SERVICE_MANAGER=file
// is the documented fallback when systemd cannot be PID 1.
type ServiceManager interface {
	DaemonReload(ctx context.Context) error
	Enable(ctx context.Context, unit string) error
	Disable(ctx context.Context, unit string) error
	Start(ctx context.Context, unit string) error
	Stop(ctx context.Context, unit string) error
	Active(ctx context.Context, unit string) (string, error)
}

func (e *Env) services() ServiceManager {
	if e != nil && e.Services != nil {
		return e.Services
	}
	if os.Getenv("FVPN_INSTALL_SERVICE_MANAGER") == "file" {
		return &fileServices{path: e.layout().servicesFile()}
	}
	return systemdManager{env: e}
}

type systemdManager struct{ env *Env }

func (s systemdManager) DaemonReload(ctx context.Context) error {
	return s.env.run(ctx, "systemctl", "daemon-reload")
}

func (s systemdManager) Enable(ctx context.Context, unit string) error {
	return s.env.run(ctx, "systemctl", "enable", unit)
}

func (s systemdManager) Disable(ctx context.Context, unit string) error {
	return s.env.run(ctx, "systemctl", "disable", unit)
}

func (s systemdManager) Start(ctx context.Context, unit string) error {
	if err := s.env.run(ctx, "systemctl", "start", unit); err != nil {
		// systemctl does not forward the unit's own logs. Keep a short
		// tail so a failed install says which unit and why (§6.3).
		st, _ := s.env.output(ctx, "systemctl", "status", unit, "--no-pager", "-l", "--lines", "20")
		jrn, _ := s.env.output(ctx, "journalctl", "-u", unit, "--no-pager", "-n", "40", "-o", "cat")
		return fmt.Errorf("start %s: %w: %s", unit, err, oneLine(string(st)+" "+string(jrn)))
	}
	return nil
}

func oneLine(s string) string {
	s = string(bytes.Join(bytes.Fields([]byte(s)), []byte(" ")))
	const max = 600
	if len(s) > max {
		s = s[len(s)-max:]
	}
	return s
}

func (s systemdManager) Stop(ctx context.Context, unit string) error {
	return s.env.run(ctx, "systemctl", "stop", unit)
}

func (s systemdManager) Active(ctx context.Context, unit string) (string, error) {
	out, err := s.env.output(ctx, "systemctl", "is-active", unit)
	state := strings.TrimSpace(string(out))
	if state == "" {
		if err != nil {
			return "unknown", nil
		}
		return "unknown", nil
	}
	return state, nil
}

// MemServices is an in-memory service manager for tests.
type MemServices struct {
	mu      sync.Mutex
	active  map[string]string
	enabled map[string]bool
	Reloads int
}

func NewMemServices() *MemServices {
	return &MemServices{active: map[string]string{}, enabled: map[string]bool{}}
}

func (m *MemServices) DaemonReload(context.Context) error {
	m.mu.Lock()
	m.Reloads++
	m.mu.Unlock()
	return nil
}

func (m *MemServices) Enable(_ context.Context, unit string) error {
	m.mu.Lock()
	m.enabled[unit] = true
	m.mu.Unlock()
	return nil
}

func (m *MemServices) Disable(_ context.Context, unit string) error {
	m.mu.Lock()
	m.enabled[unit] = false
	m.mu.Unlock()
	return nil
}

func (m *MemServices) Start(_ context.Context, unit string) error {
	m.mu.Lock()
	m.active[unit] = "active"
	m.mu.Unlock()
	return nil
}

func (m *MemServices) Stop(_ context.Context, unit string) error {
	m.mu.Lock()
	m.active[unit] = "inactive"
	m.mu.Unlock()
	return nil
}

func (m *MemServices) Active(_ context.Context, unit string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.active[unit]; ok {
		return s, nil
	}
	return "inactive", nil
}

// SetActive lets a test inject drift.
func (m *MemServices) SetActive(unit, state string) {
	m.mu.Lock()
	m.active[unit] = state
	m.mu.Unlock()
}

type fileServices struct {
	path string
	mu   sync.Mutex
}

type fileState struct {
	Active  map[string]string `json:"active"`
	Enabled map[string]bool   `json:"enabled"`
}

func (f *fileServices) load() fileState {
	var st fileState
	raw, err := os.ReadFile(f.path)
	if err != nil {
		return fileState{Active: map[string]string{}, Enabled: map[string]bool{}}
	}
	if json.Unmarshal(raw, &st) != nil || st.Active == nil || st.Enabled == nil {
		return fileState{Active: map[string]string{}, Enabled: map[string]bool{}}
	}
	return st
}

func (f *fileServices) store(st fileState) error {
	if err := os.MkdirAll(filepathDir(f.path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(f.path, append(raw, '\n'), 0o600)
}

func (f *fileServices) DaemonReload(context.Context) error { return nil }

func (f *fileServices) Enable(_ context.Context, unit string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	st := f.load()
	st.Enabled[unit] = true
	return f.store(st)
}

func (f *fileServices) Disable(_ context.Context, unit string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	st := f.load()
	st.Enabled[unit] = false
	return f.store(st)
}

func (f *fileServices) Start(_ context.Context, unit string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	st := f.load()
	st.Active[unit] = "active"
	return f.store(st)
}

func (f *fileServices) Stop(_ context.Context, unit string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	st := f.load()
	st.Active[unit] = "inactive"
	return f.store(st)
}

func (f *fileServices) Active(_ context.Context, unit string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	st := f.load()
	if s, ok := st.Active[unit]; ok {
		return s, nil
	}
	return "inactive", nil
}

func filepathDir(path string) string {
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return "."
	}
	if i == 0 {
		return "/"
	}
	return path[:i]
}
