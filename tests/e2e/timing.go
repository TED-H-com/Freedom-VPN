package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// TimedSections admits before reading any scenario clock or deadline. Its
// conservative intervals include the entire scenario, including its cleanup.
// Child tool wrappers cannot wait for admission during these intervals.
// The acceptance runner is sequential; nested helpers inherit the section.
type TimedSections struct{ Root, Log, active string }

func (s *TimedSections) event(event, name string) error {
	if err := os.MkdirAll(filepath.Dir(s.Log), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.Log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(map[string]any{"event": event, "section": name, "at": time.Now().UTC().Format(time.RFC3339Nano)})
}

func (s *TimedSections) Admit(ctx context.Context, name string) error {
	if s.active != "" {
		return fmt.Errorf("admission inside section %s", s.active)
	}
	if err := os.MkdirAll(filepath.Dir(s.Log), 0o755); err != nil {
		return err
	}
	if os.Getenv("FVPN_HOLD_SH") == "" {
		return s.event("admission_not_configured", name)
	}
	cmd := exec.CommandContext(ctx, "python3", "-B", filepath.Join(s.Root, "deploy/devstack/admission.py"), "check", name)
	cmd.Env = append(os.Environ(), "FVPN_E2E_ADMISSION_LOG="+s.Log)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("section admission: %w: %s", err, Redact(string(output)))
	}
	return nil
}

func (s *TimedSections) Begin(ctx context.Context, name string) (func() error, error) {
	if s.active != "" {
		return nil, fmt.Errorf("nested section %s inside %s", name, s.active)
	}
	if err := s.Admit(ctx, name); err != nil {
		return nil, err
	}
	if err := os.Setenv("FVPN_E2E_TIMED_SECTION", name); err != nil {
		return nil, err
	}
	if err := s.event("section_start", name); err != nil {
		_ = os.Unsetenv("FVPN_E2E_TIMED_SECTION")
		return nil, err
	}
	s.active = name
	return func() error {
		if s.active != name {
			return fmt.Errorf("section %s already ended", name)
		}
		err := s.event("section_end", name)
		_ = os.Unsetenv("FVPN_E2E_TIMED_SECTION")
		s.active = ""
		return err
	}, nil
}
