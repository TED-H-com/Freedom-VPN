// Package counterread retains content-free counter read diagnostics (F10-02).
package counterread

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

type Heartbeat struct {
	ReadAt     time.Time `json:"read_at"`
	WallUnixNS string    `json:"wall_unix_ns"`
	Serving    bool      `json:"serving"`
	ReadOK     bool      `json:"read_ok"`
}
type Diagnostic struct {
	BootID       string         `json:"boot_id"`
	MonoStart    time.Duration  `json:"read_start_mono_ns,string"`
	MonoEnd      time.Duration  `json:"read_end_mono_ns,string"`
	TimingBefore TimingSnapshot `json:"enforcement_before"`
	TimingAfter  TimingSnapshot `json:"enforcement_after"`
	Started      time.Time      `json:"started"`
	Completed    time.Time      `json:"completed"`
	Stage        string         `json:"stage"`
	Category     string         `json:"category"`
	Exit         int            `json:"exit"`
	Before       Heartbeat      `json:"heartbeat_before"`
	After        Heartbeat      `json:"heartbeat_after"`
	Parse        *ParseFailure  `json:"parse_failure,omitempty"`
}
type Result struct {
	Upload     uint64 `json:"upload_bytes,string"`
	Download   uint64 `json:"download_bytes,string"`
	Present    bool
	Generation string     `json:"kernel_generation"`
	Diagnostic Diagnostic `json:"read_diagnostic"`
}

func heartbeat() Heartbeat {
	h := Heartbeat{ReadAt: time.Now().UTC()}
	raw, e := os.ReadFile("/run/vpn-node/heartbeat")
	if e != nil {
		return h
	}
	fields := strings.Fields(string(raw))
	var wall bool
	for i := 0; i+1 < len(fields); i++ {
		switch fields[i] {
		case "wall_unix_ns":
			h.WallUnixNS = fields[i+1]
			n, e := strconv.ParseInt(h.WallUnixNS, 10, 64)
			wall = e == nil && n > 0
		case "serving":
			h.Serving = fields[i+1] == "1"
		}
	}
	h.ReadOK = wall
	return h
}
func Category(ctx context.Context, err error, stderr string) string {
	if err == nil {
		return "none"
	}
	if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	s := strings.ToLower(stderr)
	switch {
	case strings.Contains(s, "permission denied"), strings.Contains(s, "operation not permitted"):
		return "permission"
	case strings.Contains(s, "resource temporarily unavailable"), strings.Contains(s, "interrupted system call"), strings.Contains(s, "input/output error"), strings.Contains(s, "no buffer space"):
		return "transient_io"
	default:
		return "nft_read"
	}
}
func Read(ctx context.Context, id uuid.UUID) Result {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out := Result{Diagnostic: Diagnostic{Started: time.Now().UTC(), Stage: "nft_read", Category: "none", Before: heartbeat()}}
	boot, _ := os.ReadFile("/proc/sys/kernel/random/boot_id")
	out.Diagnostic.BootID = strings.TrimSpace(string(boot))
	out.Diagnostic.TimingBefore = latestTiming()
	out.Diagnostic.MonoStart, _ = enforce.MonotonicNow()
	raw, err := exec.CommandContext(ctx, "nft", "-j", "list", "table", "inet", "fvpn").CombinedOutput()
	out.Diagnostic.MonoEnd, _ = enforce.MonotonicNow()
	out.Diagnostic.Completed = time.Now().UTC()
	out.Diagnostic.TimingAfter = latestTiming()
	out.Diagnostic.After = heartbeat()
	out.Diagnostic.Category = Category(ctx, err, string(raw))
	if err != nil {
		out.Diagnostic.Exit = -1
		var e *exec.ExitError
		if errors.As(err, &e) {
			out.Diagnostic.Exit = e.ExitCode()
		}
		return out
	}
	st, err := enforce.ParseState(raw)
	if err != nil {
		out.Diagnostic.Stage = "nft_parse"
		out.Diagnostic.Category = "parse"
		out.Diagnostic.Parse = retainParseFailure(PrivateDirectory, raw, err)
		return out
	}
	out.Generation = strconv.FormatUint(st.Generation, 10)
	suffix := strings.ReplaceAll(id.String(), "-", "")
	up, u := st.Counters["u_"+suffix]
	down, d := st.Counters["d_"+suffix]
	out.Present = u && d
	if out.Present {
		out.Upload = up.Bytes
		out.Download = down.Bytes
	}
	return out
}

// The complete record is detached by the product after its operation. Keeping
// both sides and the same CLOCK_MONOTONIC domain lets consumers join all
// overlapping operations from the full exported trace without wall inference.
type TimingSnapshot struct {
	Available bool            `json:"available"`
	Error     string          `json:"error,omitempty"`
	Record    json.RawMessage `json:"record,omitempty"`
}

func latestTiming() TimingSnapshot {
	raw, e := os.ReadFile("/var/lib/vpn-node/e2e/enforcement-timing.jsonl")
	if e != nil {
		return TimingSnapshot{Error: "timing_not_enabled_or_unreadable"}
	}
	out := TimingSnapshot{}
	scan := bufio.NewScanner(bytes.NewReader(raw))
	scan.Buffer(make([]byte, 4096), 1<<20)
	for scan.Scan() {
		var v struct {
			Timing json.RawMessage `json:"timing"`
		}
		if json.Unmarshal(scan.Bytes(), &v) != nil {
			return TimingSnapshot{Error: "timing_parse"}
		}
		if len(v.Timing) > 0 {
			out.Record = append(json.RawMessage(nil), scan.Bytes()...)
			out.Available = true
		}
	}
	if scan.Err() != nil {
		return TimingSnapshot{Error: "timing_read"}
	}
	return out
}
