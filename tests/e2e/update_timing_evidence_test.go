package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// Marshal the real producer types: unavailable fields, decimal strings and
// optional command lists must match what the running agent actually writes.
func updateTimingFixture() agent.Measurement {
	const ns = time.Duration(9007199254740993)
	stamp := func(n time.Duration) enforce.LifecycleStamp {
		return enforce.LifecycleStamp{NS: n, Available: true}
	}
	shape := enforce.CommandLifecycle{
		Kind: "tc", Phase: "shape", Started: true, OK: true,
		ContextDeadline: enforce.LifecycleBracket{Before: stamp(ns + 1000), After: stamp(ns + 1002)},
		StartAttempt:    stamp(ns), StartReturn: stamp(ns + 1), ChildStartedBy: stamp(ns + 1), WaitReturn: stamp(ns + 2),
		ContextBefore: "none", ContextAfter: "none", ContextAtCancel: "unavailable",
		ExitAvailable: true, CPUAvailable: true, SwitchesAvailable: true,
		UserCPU: 1, SystemCPU: 2, VoluntarySwitches: 3, InvoluntarySwitches: 4,
	}
	install := shape
	install.Kind, install.Phase, install.OK = "nft", "install", false
	install.ContextDeadline = enforce.LifecycleBracket{Before: stamp(ns + 100), After: stamp(ns + 102)}
	install.ContextAfter, install.ContextAtCancel = "deadline_exceeded", "deadline_exceeded"
	install.CancelObserved, install.WaitReturn = stamp(ns+103), stamp(ns+105)
	install.ExitCode, install.SignalAvailable, install.Signal = -1, true, 9
	return agent.Measurement{
		RunID: uuid.New(), BootID: uuid.New(), Sequence: decstr.U64(^uint64(0)), Source: "apply",
		Anchor: agent.MeasurementAnchor{MonoBefore: ns - 1, MonoAfter: ns + 1, Wall: time.Date(2026, 1, 1, 0, 0, 0, 123, time.UTC)},
		Timing: &enforce.TimingEvidence{
			Version: 1, Clock: "CLOCK_MONOTONIC", Operation: "apply", CandidateGeneration: decstr.U64(^uint64(0)), Start: ns, End: ns + 200,
			Deadline: enforce.DeadlineEvidence{Granularity: time.Millisecond, InstallBound: 100 * time.Millisecond, ApplyStart: ns, RenderStart: ns + 1, InstallStart: ns + 2, InstalledBy: ns + 105},
			Commands: []enforce.CommandLifecycle{shape, install},
		},
	}
}

func timingJSONLine(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}

func TestSEC02_RecoveryTimingProducerProjection(t *testing.T) {
	m := updateTimingFixture()
	m.Lease = &agent.MeasurementLease{ID: uuid.New(), Members: 4}
	m.TC = "TEST_ONLY_PRIVATE_HASH"
	// A real command resource field can make an Apply much larger than a log
	// line. The bounded public projection must neither drop it nor copy it.
	for depth := range 32 {
		m.Timing.Commands[0].Before.Cgroups = append(m.Timing.Commands[0].Before.Cgroups, enforce.CgroupCPU{Depth: depth, Stat: enforce.ResourceText{Available: true, Value: strings.Repeat("TEST_ONLY_PRIVATE_RESOURCE", 400)}})
	}
	raw := timingJSONLine(t, m)
	raw = bytes.Replace(raw, []byte(`"kind":"tc"`), []byte(`"kind":"tc","argv":"TEST_ONLY_PRIVATE_ARGV","stderr":"TEST_ONLY_PRIVATE_STDERR","path":"TEST_ONLY_PRIVATE_PATH"`), 1)
	if len(raw) <= 256<<10 || len(raw) >= updateTimingLineLimit {
		t.Fatalf("fixture does not exercise large producer row: %d", len(raw))
	}
	original := bytes.Clone(raw)
	outerRun := uuid.NewString()
	out := projectRecoveryContext(outerRun, recoveryTestCommit, m.Anchor.Wall, m.Anchor.Wall, true, true, map[string]recoveryInput{"enforcement_timing": {Data: raw, Available: true}})
	s := out.EnforcementTiming
	if !out.BindingAvailable || out.RunID != outerRun || out.Source != recoveryTestCommit || !s.Available || s.Truncated || s.Incomplete != 0 || s.Malformed != 0 || len(s.Records) != 1 || !bytes.Equal(original, raw) {
		t.Fatalf("producer projection failed: %+v", s)
	}
	r := s.Records[0]
	for key, want := range map[string]any{"run_id": m.RunID.String(), "boot_id": m.BootID.String(), "sequence": "18446744073709551615", "candidate_generation": "18446744073709551615", "start_ns": "9007199254740993", "succeeded": false, "verified": false, "commands_available": true, "window_position": "within"} {
		if r[key] != want {
			t.Fatalf("field %s: got %v want %v", key, r[key], want)
		}
	}
	d := r["deadline"].(map[string]any)
	if d["install_ok"] != false || d["guard_released"] != false || d["release_start_ns"] != "0" || d["install_bound_ns"] != "100000000" {
		t.Fatalf("failed/absent stages changed: %+v", d)
	}
	commands := r["commands"].([]map[string]any)
	if len(commands) != 2 || commands[0]["phase"] != "shape" || commands[1]["phase"] != "install" || commands[1]["ok"] != false || commands[1]["started"] != true || commands[1]["exit_code"] != int64(-1) || commands[1]["signal"] != int64(9) || commands[1]["context_after"] != "deadline_exceeded" {
		t.Fatalf("command results changed: %+v", commands)
	}
	for i, delta := range []time.Duration{1000, 100} {
		before := commands[i]["context_deadline"].(map[string]any)["before"].(map[string]any)
		if before["ns"] != strconv.FormatInt(int64(m.Timing.Start+delta), 10) || before["available"] != true {
			t.Fatalf("parent/effective deadline changed: %+v", before)
		}
		exact := commands[i]["child_start_exact"].(map[string]any)
		if exact["ns"] != "0" || exact["available"] != false {
			t.Fatalf("unavailable observation fabricated: %+v", exact)
		}
	}
	encoded := timingJSONLine(t, out)
	for _, forbidden := range []string{"TEST_ONLY_PRIVATE", "lease_id", "cgroups", "tc_sha256", "verifications"} {
		if bytes.Contains(encoded, []byte(forbidden)) {
			t.Fatalf("unsafe projection field %s", forbidden)
		}
	}
	if len(encoded) > 16<<10 || !strings.Contains(s.Scope, "Non-Apply records are intentionally omitted") || !strings.Contains(s.Scope, "not kernel events") {
		t.Fatal("projection size or interpretation scope changed")
	}
}

func TestSEC02_RecoveryTimingRejectsMissingNullAndInvalidFields(t *testing.T) {
	paths := [][]string{
		{"run_id"}, {"boot_id"}, {"sequence"}, {"anchor"}, {"anchor", "wall"}, {"anchor", "mono_before_ns"},
		{"timing"}, {"timing", "version"}, {"timing", "clock"}, {"timing", "operation"}, {"timing", "candidate_generation"},
		{"timing", "succeeded"}, {"timing", "verified"}, {"timing", "start_ns"}, {"timing", "deadline"}, {"timing", "deadline", "install_ok"}, {"timing", "deadline", "install_start_ns"},
		{"timing", "commands", "0", "kind"}, {"timing", "commands", "0", "phase"}, {"timing", "commands", "0", "started"}, {"timing", "commands", "0", "ok"},
		{"timing", "commands", "0", "exit_available"}, {"timing", "commands", "0", "exit_code"}, {"timing", "commands", "0", "signal"}, {"timing", "commands", "0", "user_cpu_ns"},
		{"timing", "commands", "0", "context_deadline"}, {"timing", "commands", "0", "context_deadline", "before", "available"},
		{"timing", "commands", "0", "context_before"}, {"timing", "commands", "0", "start_attempt", "ns"}, {"timing", "commands", "0", "child_started_by", "available"},
	}
	fixture := timingJSONLine(t, updateTimingFixture())
	for _, path := range paths {
		for _, mode := range []string{"missing", "null", "wrong type"} {
			t.Run(strings.Join(path, "/")+"/"+mode, func(t *testing.T) {
				var fields any
				if err := json.Unmarshal(fixture, &fields); err != nil {
					t.Fatal(err)
				}
				cur := fields
				for _, key := range path[:len(path)-1] {
					if key == "0" {
						cur = cur.([]any)[0]
					} else {
						cur = cur.(map[string]any)[key]
					}
				}
				obj, key := cur.(map[string]any), path[len(path)-1]
				switch mode {
				case "missing":
					delete(obj, key)
				case "null":
					obj[key] = nil
				default:
					obj[key] = []any{"TEST_ONLY_PRIVATE"}
				}
				out := projectUpdateTiming(recoveryInput{Data: timingJSONLine(t, fields), Available: true}, time.Time{}, time.Time{})
				if len(out.Records) != 0 || out.Incomplete != 1 || out.Malformed != 0 {
					t.Fatalf("invalid row became evidence: %+v", out)
				}
			})
		}
	}
	for name, replace := range map[string][2]string{
		"zero sequence":   {`"sequence":"18446744073709551615"`, `"sequence":"0"`},
		"overflow":        {`"sequence":"18446744073709551615"`, `"sequence":"18446744073709551616"`},
		"rounded number":  {`"start_ns":"9007199254740993"`, `"start_ns":9007199254740993`},
		"noncanonical":    {`"start_ns":"9007199254740993"`, `"start_ns":"01"`},
		"negative":        {`"start_ns":"9007199254740993"`, `"start_ns":"-1"`},
		"fraction":        {`"start_ns":"9007199254740993"`, `"start_ns":"1.5"`},
		"nil UUID":        {updateTimingUUID(t, fixture), uuid.Nil.String()},
		"unknown phase":   {`"phase":"shape"`, `"phase":"TEST_ONLY_PRIVATE"`},
		"unknown context": {`"context_before":"none"`, `"context_before":"TEST_ONLY_PRIVATE"`},
		"exit range":      {`"exit_code":0`, `"exit_code":256`},
		"signal range":    {`"signal":0`, `"signal":65`},
	} {
		t.Run(name, func(t *testing.T) {
			raw := bytes.Replace(fixture, []byte(replace[0]), []byte(replace[1]), 1)
			if bytes.Equal(raw, fixture) {
				t.Fatal("replacement did not exercise fixture")
			}
			out := projectUpdateTiming(recoveryInput{Data: raw, Available: true}, time.Time{}, time.Time{})
			if len(out.Records) != 0 || out.Incomplete != 1 {
				t.Fatalf("invalid value accepted: %+v", out)
			}
		})
	}
}

func updateTimingUUID(t *testing.T, raw []byte) string {
	t.Helper()
	var m agent.Measurement
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m.RunID.String()
}

func TestSEC02_RecoveryTimingUnavailableAndPartialCommands(t *testing.T) {
	m := updateTimingFixture()
	m.Timing.Commands = nil
	out := projectUpdateTiming(recoveryInput{Data: timingJSONLine(t, m)}, m.Anchor.Wall.Add(time.Second), m.Anchor.Wall.Add(time.Minute))
	if out.Available || len(out.Records) != 1 || out.Records[0]["commands_available"] != false || out.Records[0]["window_position"] != "before" || out.OutsideWindow != 1 {
		t.Fatalf("missing command stream claimed available: %+v", out)
	}
	// A Start failure has no child/Wait/cancellation samples or exit data.
	m.Timing.Commands = []enforce.CommandLifecycle{{Kind: "nft", Phase: "install", ContextBefore: "deadline_exceeded", ContextAfter: "deadline_exceeded", ContextAtCancel: "unavailable"}}
	out = projectUpdateTiming(recoveryInput{Data: timingJSONLine(t, m), Available: true}, m.Anchor.Wall.Add(-time.Minute), m.Anchor.Wall.Add(-time.Second))
	c := out.Records[0]["commands"].([]map[string]any)[0]
	if c["started"] != false || c["exit_available"] != false || c["wait_return"].(map[string]any)["available"] != false || c["context_deadline"].(map[string]any)["before"].(map[string]any)["available"] != false || out.Records[0]["window_position"] != "after" {
		t.Fatalf("Start failure fabricated child evidence: %+v", c)
	}
	var fields map[string]any
	if err := json.Unmarshal(timingJSONLine(t, m), &fields); err != nil {
		t.Fatal(err)
	}
	fields["timing"].(map[string]any)["commands"] = nil
	out = projectUpdateTiming(recoveryInput{Data: timingJSONLine(t, fields), Available: true}, time.Time{}, time.Time{})
	if len(out.Records) != 0 || out.Incomplete != 1 {
		t.Fatal("null commands became empty successful list")
	}
	out = projectUpdateTiming(recoveryInput{}, time.Time{}, time.Time{})
	if out.Available || len(out.Records) != 0 {
		t.Fatal("absent stream claimed available")
	}
}

func TestSEC02_RecoveryTimingInputAndProjectionBounds(t *testing.T) {
	m := updateTimingFixture()
	commands := m.Timing.Commands
	m.Timing.Commands = nil
	for range 100 {
		c := commands[0]
		c.Phase = "prepare"
		m.Timing.Commands = append(m.Timing.Commands, c)
	}
	for range updateTimingCommandLimit/2 + 2 {
		m.Timing.Commands = append(m.Timing.Commands, commands...)
	}
	for _, phase := range []string{"retire", "release", "verify", "revoke", "quarantine", "rollback"} {
		c := commands[0]
		c.Phase = phase
		m.Timing.Commands = append(m.Timing.Commands, c)
	}
	out := projectUpdateTiming(recoveryInput{Data: timingJSONLine(t, m), Available: true}, m.Anchor.Wall, m.Anchor.Wall)
	if len(out.Records) != 1 || out.DroppedCommands != 4 || !out.Truncated || len(out.Records[0]["commands"].([]map[string]any)) != updateTimingCommandLimit || out.Records[0]["commands"].([]map[string]any)[1]["phase"] != "install" {
		t.Fatalf("non-target phases hid install evidence: %+v", out)
	}
	m.Timing.Commands = commands
	var raw []byte
	for i := range updateTimingRecordLimit + 3 {
		m.Sequence = decstr.U64(i + 1)
		raw = append(raw, timingJSONLine(t, m)...)
	}
	out = projectUpdateTiming(recoveryInput{Data: raw, Available: true}, m.Anchor.Wall, m.Anchor.Wall)
	if !out.Truncated || out.DroppedRecords != 3 || len(out.Records) != updateTimingRecordLimit || out.Records[0]["sequence"] != "4" || out.Records[updateTimingRecordLimit-1]["sequence"] != "67" {
		t.Fatalf("newest record cap: %+v", out)
	}
	encoded := timingJSONLine(t, out)
	if len(encoded) > 1<<20 {
		t.Fatalf("unbounded public projection: %d", len(encoded))
	}
	line := timingJSONLine(t, m)
	// cap+1 proves a prefix was omitted. The first suffix fragment and final
	// incomplete row must not become complete observations.
	raw = append(bytes.Repeat([]byte{'x'}, updateTimingInputLimit+20), '\n')
	raw = append(raw, line...)
	raw = append(raw, line[:len(line)-1]...)
	raw = raw[len(raw)-updateTimingInputLimit-1:]
	out = projectUpdateTiming(recoveryInput{Data: raw, Available: true}, m.Anchor.Wall, m.Anchor.Wall)
	if !out.Truncated || out.Incomplete != 2 || out.InputBytes != updateTimingInputLimit+1 || out.ScannedBytes > updateTimingInputLimit || len(out.Records) != 1 {
		t.Fatalf("partial suffix boundary: %+v", out)
	}
	raw = append(bytes.Repeat([]byte{'x'}, updateTimingLineLimit+1), '\n')
	raw = append(raw, []byte("not-json\nnull\n")...)
	raw = append(raw, line...)
	out = projectUpdateTiming(recoveryInput{Data: raw, Available: true}, m.Anchor.Wall, m.Anchor.Wall)
	if !out.Truncated || out.Malformed != 3 || len(out.Records) != 1 {
		t.Fatalf("oversized/malformed input: %+v", out)
	}
	m.Source = "heartbeat"
	raw = append(timingJSONLine(t, m), line...)
	out = projectUpdateTiming(recoveryInput{Data: raw, Available: true}, time.Time{}, time.Time{})
	if len(out.Records) != 1 || out.Incomplete != 0 || out.Records[0]["window_position"] != "unavailable" {
		t.Fatalf("non-Apply or absent window interpretation: %+v", out)
	}
}

func TestSEC02_RecoveryTimingBoundedContainerRead(t *testing.T) {
	root := t.TempDir()
	fixture := filepath.Join(root, "timing.jsonl")
	if err := os.WriteFile(fixture, timingJSONLine(t, updateTimingFixture()), 0o600); err != nil {
		t.Fatal(err)
	}
	// This executable replaces Docker entirely; unexpected arguments fail.
	script := `#!/bin/sh
set -eu
[ "$#" = 6 ] && [ "$1" = exec ] && [ "$2" = fvpn-timing-test-node ] && [ "$3" = tail ] && [ "$4" = -c ] && [ "$5" = 4194305 ] && [ "$6" = /var/lib/vpn-node/e2e/enforcement-timing.jsonl ] || exit 91
printf 'TEST_ONLY_PRIVATE_STDERR' >&2
if [ "${FVPN_TEST_TIMING_OVERFLOW:-}" = 1 ]; then
  head -c 4194400 /dev/zero
else
  cat "$FVPN_TEST_TIMING_FIXTURE"
fi
exit "${FVPN_TEST_TIMING_EXIT:-0}"
`
	if err := os.WriteFile(filepath.Join(root, "docker"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FVPN_TEST_TIMING_FIXTURE", fixture)
	t.Setenv("FVPN_TEST_TIMING_EXIT", "0")
	t.Setenv("FVPN_TEST_TIMING_OVERFLOW", "")
	in := readUpdateTiming(context.Background(), "fvpn-timing-test-node")
	if !in.Available || bytes.Contains(in.Data, []byte("STDERR")) || len(projectUpdateTiming(in, time.Time{}, time.Time{}).Records) != 1 {
		t.Fatalf("bounded read mixed output or wrong invocation: available=%v bytes=%d", in.Available, len(in.Data))
	}
	t.Setenv("FVPN_TEST_TIMING_EXIT", "7")
	in = readUpdateTiming(context.Background(), "fvpn-timing-test-node")
	if in.Available || len(in.Data) == 0 {
		t.Fatal("failed read claimed available or lost partial evidence")
	}
	t.Setenv("FVPN_TEST_TIMING_EXIT", "0")
	t.Setenv("FVPN_TEST_TIMING_OVERFLOW", "1")
	in = readUpdateTiming(context.Background(), "fvpn-timing-test-node")
	if in.Available || len(in.Data) != updateTimingInputLimit+1 || !projectUpdateTiming(in, time.Time{}, time.Time{}).Truncated {
		t.Fatalf("misbehaving subprocess escaped bound: available=%v bytes=%d", in.Available, len(in.Data))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if in := readUpdateTiming(ctx, "fvpn-timing-test-node"); in.Available {
		t.Fatal("canceled read claimed available")
	}
}
