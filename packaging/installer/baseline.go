package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/google/uuid"
)

// BaselineResult records the installed updater's own verification, never a
// journal entry manufactured by the installer. Probe stderr is retained so
// an older updater is distinguishable from a broken seed invocation.
type BaselineResult struct {
	Status       string         `json:"status"`
	ManifestPath string         `json:"manifest_path,omitempty"`
	TreePath     string         `json:"tree_path,omitempty"`
	Probe        CommandResult  `json:"probe"`
	Seed         *CommandResult `json:"seed,omitempty"`
}

// CommandResult is the exit code and stderr of a seed probe or invocation.
// ExitCode -1 means the process could not be started.
type CommandResult struct {
	ExitCode int    `json:"exit_code"`
	Stderr   string `json:"stderr"`
}

func captureCommand(ctx context.Context, env *Env, name string, args ...string) CommandResult {
	var stderr bytes.Buffer
	var err error
	r := CommandResult{}
	if env.Command != nil {
		var out []byte
		out, err = env.Command(ctx, name, args...)
		stderr.Write(out)
	} else if !env.AllowHost {
		err = errors.New("host command refused")
	} else {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Env = cleanEnv(os.Environ())
		cmd.Stderr = &stderr
		err = cmd.Run()
	}
	r.Stderr = stderr.String()
	if err != nil {
		r.ExitCode = -1
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			r.ExitCode = ee.ExitCode()
		} else if r.Stderr == "" {
			r.Stderr = err.Error()
		}
	}
	return r
}

func seedBaseline(ctx context.Context, env *Env) (*BaselineResult, error) {
	l := env.layout()
	r := &BaselineResult{Status: "failed"}
	finish := func(cause error) (*BaselineResult, error) {
		raw, err := marshalIndent(r)
		if err == nil {
			err = fsutil.WriteFileAtomic(l.baselineFile(), raw, 0o600)
		}
		fmt.Fprintf(env.stderr(), "baseline: %s (seed probe exit %d, stderr %q)\n", r.Status, r.Probe.ExitCode, r.Probe.Stderr)
		if err != nil {
			return r, err
		}
		return r, cause
	}
	r.Probe = captureCommand(ctx, env, l.updaterBin(), "seed", "--help")
	if r.Probe.ExitCode == 64 && strings.Contains(r.Probe.Stderr, `unknown command "seed"`) {
		r.Status = "not seeded (updater without seed)"
		return finish(nil)
	}
	// Go flag help may exit 64. Recognize seed-specific help, not general
	// usage or an arbitrary failed probe. Follow-up 14 owns the CLI contract.
	help := strings.Contains(r.Probe.Stderr, "-manifest") && strings.Contains(r.Probe.Stderr, "-tree")
	if r.Probe.ExitCode != 0 && !(r.Probe.ExitCode == 64 && help) {
		return finish(errf(ExitFailed, "baseline_failed", "updater seed probe failed"))
	}
	tree, err := filepath.EvalSymlinks(l.CurrentLink)
	if err != nil {
		return finish(errf(ExitFailed, "baseline_failed", "current tree cannot be resolved"))
	}
	id, err := uuid.Parse(filepath.Base(tree))
	if err != nil || id == uuid.Nil || tree != l.releaseDir(id.String()) {
		return finish(errf(ExitFailed, "baseline_failed", "current tree is not a release directory"))
	}
	r.TreePath, r.ManifestPath = tree, l.releaseManifest(id.String())
	seed := captureCommand(ctx, env, l.updaterBin(), "seed", "--config", l.updaterConfig(), "--manifest", r.ManifestPath, "--tree", tree)
	r.Seed = &seed
	if seed.ExitCode != 0 {
		return finish(errf(ExitFailed, "baseline_failed", "updater seed failed; inspect check --json baseline"))
	}
	r.Status = "seeded"
	return finish(nil)
}

func readBaseline(l Layout) *BaselineResult {
	raw, err := os.ReadFile(l.baselineFile())
	if err != nil {
		return &BaselineResult{Status: "not recorded"}
	}
	var r BaselineResult
	if json.Unmarshal(raw, &r) != nil || r.Status == "" {
		return &BaselineResult{Status: "unreadable"}
	}
	return &r
}
