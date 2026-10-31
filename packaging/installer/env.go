package installer

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Env is the host surface. Tests replace Command and LookPath so the
// installer never calls ip, nft or apt on the development host.
type Env struct {
	Layout   Layout
	Arch     string
	Kernel   string
	Now      func() time.Time
	Stdout   io.Writer
	Stderr   io.Writer
	Stdin    io.Reader
	LookPath func(string) (string, error)
	// Command, when set, replaces every external program. Return the
	// combined output.
	Command  func(ctx context.Context, name string, args ...string) ([]byte, error)
	Services ServiceManager
	// AssumeSystemd skips the PID 1 check. Set when Services is injected.
	AssumeSystemd bool
	// AllowHost permits real ip, nft, apt and service commands. The CLI
	// sets it. Tests leave it false and set Command, so a unit test cannot
	// change the development host (§6.3).
	AllowHost bool
}

func (e *Env) now() time.Time {
	if e != nil && e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func (e *Env) stdout() io.Writer {
	if e != nil && e.Stdout != nil {
		return e.Stdout
	}
	return os.Stdout
}

func (e *Env) stderr() io.Writer {
	if e != nil && e.Stderr != nil {
		return e.Stderr
	}
	return os.Stderr
}

func (e *Env) look(name string) (string, error) {
	if e != nil && e.LookPath != nil {
		return e.LookPath(name)
	}
	return exec.LookPath(name)
}

func (e *Env) output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if e != nil && e.Command != nil {
		return e.Command(ctx, name, args...)
	}
	if e == nil || !e.AllowHost {
		return nil, errf(ExitFailed, "failed", "host command refused")
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = cleanEnv(os.Environ())
	return cmd.Output()
}

func (e *Env) run(ctx context.Context, name string, args ...string) error {
	if e != nil && e.Command != nil {
		_, err := e.Command(ctx, name, args...)
		return err
	}
	if e == nil || !e.AllowHost {
		return errf(ExitFailed, "failed", "host command refused")
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = cleanEnv(os.Environ())
	cmd.Stdout = e.stdout()
	cmd.Stderr = e.stderr()
	return cmd.Run()
}

func (e *Env) layout() Layout {
	if e == nil || e.Layout.VarDir == "" {
		return DefaultLayout()
	}
	return e.Layout
}

// cleanEnv drops the enrollment token variable. The agent ignores it
// (§6.1); the installer still does not pass it to children (§18.2).
func cleanEnv(in []string) []string {
	out := make([]string, 0, len(in))
	for _, e := range in {
		if strings.HasPrefix(e, "FVPN_ENROLLMENT_TOKEN=") {
			continue
		}
		out = append(out, e)
	}
	return out
}
