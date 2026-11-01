//go:build linux

package enforce

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func commandProcessState(ps *os.ProcessState, r *CommandLifecycle) {
	if status, ok := ps.Sys().(syscall.WaitStatus); ok {
		r.SignalAvailable = true
		if status.Signaled() {
			r.Signal = int(status.Signal())
		}
	}
	if usage, ok := ps.SysUsage().(*syscall.Rusage); ok {
		r.SwitchesAvailable = true
		r.VoluntarySwitches, r.InvoluntarySwitches = usage.Nvcsw, usage.Nivcsw
	}
}

// Resolve only a visible cgroup-v2 mount whose root covers membership. Hidden
// namespace ancestors and cgroup v1 remain explicitly unavailable. The paths
// are used locally and never exported. Do not fall back to an unrelated root.
func cgroupLocation(membership, mounts string) (string, string, bool) {
	var member string
	for line := range strings.SplitSeq(membership, "\n") {
		if strings.HasPrefix(line, "0::/") {
			member = strings.TrimPrefix(line, "0::")
		}
	}
	if member == "" || filepath.Clean(member) != member {
		return "", "", false
	}
	for line := range strings.SplitSeq(mounts, "\n") {
		fields := strings.Fields(line)
		sep := -1
		for i, f := range fields {
			if f == "-" {
				sep = i
				break
			}
		}
		if sep < 6 || len(fields) <= sep+1 || fields[sep+1] != "cgroup2" {
			continue
		}
		// Escaped or unusual paths are unsupported, not silently misresolved.
		root, mount := fields[3], fields[4]
		if strings.ContainsAny(root+mount, "\\") || !filepath.IsAbs(mount) {
			continue
		}
		rel, err := filepath.Rel(root, member)
		if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
			continue
		}
		return filepath.Join(mount, rel), mount, true
	}
	return "", "", false
}

func commandResources(ctx context.Context) CommandResources {
	return readCommandResources(ctx, os.ReadFile)
}

func readCommandResources(ctx context.Context, readFile func(string) ([]byte, error)) CommandResources {
	if state := contextState(ctx); state != "none" {
		return CommandResources{SkipReason: "context_" + state}
	}
	// A synchronous kernel read already in flight cannot be interrupted by
	// context cancellation. Check before every read and discard the entire
	// sample if cancellation is observed before it completes. Never run this
	// walk in the install phase.
	read := func(path string) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return readFile(path)
	}
	r := CommandResources{Bracket: LifecycleBracket{Before: lifecycleNow()}}
	membership, e1 := read("/proc/self/cgroup")
	mounts, e2 := read("/proc/self/mountinfo")
	current, root, ok := cgroupLocation(string(membership), string(mounts))
	if e1 == nil && e2 == nil && ok {
		// A successful membership resolution is not evidence the directory exists.
		if _, err := read(filepath.Join(current, "cgroup.controllers")); err == nil {
			r.CgroupV2Available = true
			field := func(dir, name string) ResourceText {
				b, err := read(filepath.Join(dir, name))
				if err != nil || len(b) > 16384 {
					return ResourceText{}
				}
				// CPU control files contain fixed kernel keys/numbers; no identifiers.
				return ResourceText{true, strings.TrimSpace(string(b))}
			}
			for depth := 0; depth < 64; depth++ {
				r.Cgroups = append(r.Cgroups, CgroupCPU{depth, field(current, "cpu.max"), field(current, "cpu.weight"), field(current, "cpu.stat"), field(current, "cpu.pressure")})
				if current == root {
					r.VisibleAncestorsComplete = true
					break
				}
				current = filepath.Dir(current)
			}
		}
	}
	r.Bracket.After = lifecycleNow()
	if state := contextState(ctx); state != "none" {
		return CommandResources{SkipReason: "context_" + state}
	}
	return r
}
