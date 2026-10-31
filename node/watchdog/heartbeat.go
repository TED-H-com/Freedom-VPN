package watchdog

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Beat is one agent heartbeat (§16.3). Mono is CLOCK_MONOTONIC, shared with
// this process. PolicyExpires is the monotonic instant member forwarding must
// already have stopped; zero means the agent is not serving.
type Beat struct {
	BootID        uuid.UUID
	Mono          time.Duration
	Wall          time.Time
	PolicyExpires time.Duration
	BaselineOK    bool
	Serving       bool
	Generation    uint64
	Reason        string
	// Evidence is compact, sanitized JSON from the agent; the watchdog does not use it for admission.
	Evidence string
}

// ReadBootID reads /proc/sys/kernel/random/boot_id, or path when set.
// A missing boot id is fail-closed: the caller must not forward.
func ReadBootID(path string) (uuid.UUID, error) {
	if path == "" {
		path = "/proc/sys/kernel/random/boot_id"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return uuid.Nil, fmt.Errorf("watchdog: boot id: %w", err)
	}
	id, err := uuid.Parse(strings.TrimSpace(string(b)))
	if err != nil {
		return uuid.Nil, fmt.Errorf("watchdog: boot id: %w", err)
	}
	return id, nil
}

// WriteHeartbeat atomically replaces path. The watchdog only reads it.
func WriteHeartbeat(path string, b Beat) error {
	if b.BootID == uuid.Nil || b.Mono <= 0 {
		return fmt.Errorf("watchdog: incomplete heartbeat")
	}
	reason := b.Reason
	if reason == "" || strings.ContainsAny(reason, " \r\n") {
		reason = "-"
	}
	var body strings.Builder
	fmt.Fprintf(&body, "v1\n")
	fmt.Fprintf(&body, "boot_id %s\n", b.BootID)
	fmt.Fprintf(&body, "mono_ns %d\n", b.Mono)
	fmt.Fprintf(&body, "wall_unix_ns %d\n", b.Wall.UnixNano())
	fmt.Fprintf(&body, "policy_expires_mono_ns %d\n", b.PolicyExpires)
	fmt.Fprintf(&body, "baseline_ok %d\n", boolBit(b.BaselineOK))
	fmt.Fprintf(&body, "serving %d\n", boolBit(b.Serving))
	fmt.Fprintf(&body, "generation %d\n", b.Generation)
	fmt.Fprintf(&body, "reason %s\n", reason)
	if b.Evidence != "" && !strings.ContainsAny(b.Evidence, "\r\n") {
		fmt.Fprintf(&body, "evidence %s\n", b.Evidence)
	}
	return writeAtomic(path, []byte(body.String()), 0o644)
}

// ReadHeartbeat parses a heartbeat file. A malformed file is an error.
func ReadHeartbeat(path string) (Beat, error) {
	f, err := os.Open(path)
	if err != nil {
		return Beat{}, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() || sc.Text() != "v1" {
		return Beat{}, fmt.Errorf("watchdog: heartbeat version")
	}
	fields := map[string]string{}
	for sc.Scan() {
		line := sc.Text()
		k, v, ok := strings.Cut(line, " ")
		if !ok || k == "" || v == "" {
			return Beat{}, fmt.Errorf("watchdog: heartbeat line")
		}
		if _, dup := fields[k]; dup {
			return Beat{}, fmt.Errorf("watchdog: duplicate heartbeat field")
		}
		fields[k] = v
	}
	if err := sc.Err(); err != nil {
		return Beat{}, err
	}
	var b Beat
	var errp error
	b.BootID, errp = uuid.Parse(fields["boot_id"])
	if errp != nil {
		return Beat{}, fmt.Errorf("watchdog: heartbeat boot id")
	}
	mono, errp := strconv.ParseInt(fields["mono_ns"], 10, 64)
	if errp != nil || mono <= 0 {
		return Beat{}, fmt.Errorf("watchdog: heartbeat mono")
	}
	b.Mono = time.Duration(mono)
	wall, errp := strconv.ParseInt(fields["wall_unix_ns"], 10, 64)
	if errp != nil {
		return Beat{}, fmt.Errorf("watchdog: heartbeat wall")
	}
	b.Wall = time.Unix(0, wall).UTC()
	exp, errp := strconv.ParseInt(fields["policy_expires_mono_ns"], 10, 64)
	if errp != nil || exp < 0 {
		return Beat{}, fmt.Errorf("watchdog: heartbeat policy")
	}
	b.PolicyExpires = time.Duration(exp)
	b.BaselineOK, errp = bit(fields["baseline_ok"])
	if errp != nil {
		return Beat{}, errp
	}
	b.Serving, errp = bit(fields["serving"])
	if errp != nil {
		return Beat{}, errp
	}
	gen, errp := strconv.ParseUint(fields["generation"], 10, 64)
	if errp != nil {
		return Beat{}, fmt.Errorf("watchdog: heartbeat generation")
	}
	b.Generation = gen
	b.Evidence = fields["evidence"]
	b.Reason = fields["reason"]
	if b.Reason == "" {
		return Beat{}, fmt.Errorf("watchdog: heartbeat reason")
	}
	return b, nil
}

func bit(s string) (bool, error) {
	switch s {
	case "1":
		return true, nil
	case "0":
		return false, nil
	default:
		return false, fmt.Errorf("watchdog: heartbeat flag")
	}
}

func boolBit(v bool) int {
	if v {
		return 1
	}
	return 0
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".heartbeat-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = tmp.Close()
			_ = os.Remove(name)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	ok = true
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
