package members

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// The aggregate stream grows through the assembled run. Bound each record,
// never the whole export, and reject any partial trailing append.
func parseTimingStream(reader io.Reader) ([]json.RawMessage, error) {
	rows := []json.RawMessage{}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		raw := scanner.Bytes()
		if !json.Valid(raw) {
			return nil, fmt.Errorf("incomplete producer record")
		}
		if err := validateProducerCommands(raw); err != nil {
			return nil, err
		}
		rows = append(rows, append(json.RawMessage(nil), raw...))
	}
	if e := scanner.Err(); e != nil {
		return nil, e
	}
	return rows, nil
}

func TestAT02_F12TimingStreamExceedsDiagnosticLimit(t *testing.T) {
	row := []byte(`{"source":"TEST ONLY","sequence":"9007199254740993"}` + "\n")
	raw := bytes.Repeat(row, 40000)
	if len(raw) <= 1<<20 {
		t.Fatal("fixture too small")
	}
	rows, e := parseTimingStream(bytes.NewReader(raw))
	if e != nil || len(rows) != 40000 || !bytes.Equal(rows[0], bytes.TrimSpace(row)) {
		t.Fatal("large stream lost", len(rows), e)
	}
	if _, e = parseTimingStream(bytes.NewReader(append(raw, []byte(`{"source":`)...))); e == nil {
		t.Fatal("partial tail accepted")
	}
}

func TestAT02_F18RetireStreamAfterNFTCommit(t *testing.T) {
	stamp := func(n time.Duration) enforce.LifecycleStamp {
		return enforce.LifecycleStamp{NS: n, Available: true}
	}
	command := func(kind, phase string, start time.Duration) enforce.CommandLifecycle {
		return enforce.CommandLifecycle{Kind: kind, Phase: phase, Started: true, OK: true,
			StartAttempt: stamp(start), StartReturn: stamp(start + 1),
			ChildStartedBy: stamp(start + 1), WaitReturn: stamp(start + 2),
			ContextBefore: "none", ContextAfter: "none", ContextAtCancel: "unavailable",
			ExitAvailable: true, ExitCode: 0}
	}
	// TEST ONLY: the relevant sequence-582 ordering, with synthetic clocks.
	// The completed nft install precedes the tc retirement batch, then release.
	timing := enforce.TimingEvidence{Version: 1, Clock: "CLOCK_MONOTONIC", Operation: "apply",
		CandidateGeneration: 33, Start: 1, End: 20, Succeeded: true, Verified: true,
		Deadline: enforce.DeadlineEvidence{InstallStart: 2, InstalledBy: 5, InstallOK: true,
			ReleaseStart: 10, ReleaseEnd: 13, GuardReleased: true},
		Commands: []enforce.CommandLifecycle{command("nft", "install", 2), command("tc", "retire", 6), command("nft", "release", 10)}}
	encode := func(v enforce.TimingEvidence) []byte {
		raw, err := json.Marshal(map[string]any{"source": "apply", "sequence": "582", "timing": v})
		if err != nil {
			t.Fatal(err)
		}
		return append(raw, '\n')
	}
	raw := encode(timing)
	rows, err := parseTimingStream(bytes.NewReader(raw))
	if err != nil || len(rows) != 1 || !bytes.Equal(rows[0], bytes.TrimSpace(raw)) {
		t.Fatalf("valid post-install retire rejected: rows=%d err=%v", len(rows), err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*enforce.CommandLifecycle)
	}{
		{"unknown phase", func(c *enforce.CommandLifecycle) { c.Phase = "TEST ONLY unknown" }},
		{"unknown kind", func(c *enforce.CommandLifecycle) { c.Kind = "TEST ONLY unknown" }},
		{"outside Apply", func(c *enforce.CommandLifecycle) { c.WaitReturn = stamp(21) }},
		{"before prior command", func(c *enforce.CommandLifecycle) { c.StartAttempt = stamp(3) }},
		{"invented child start", func(c *enforce.CommandLifecycle) { c.ChildStartExact = stamp(7) }},
		{"negative CPU", func(c *enforce.CommandLifecycle) { c.UserCPU = -1 }},
		{"cancel after wait", func(c *enforce.CommandLifecycle) { c.CancelObserved = stamp(9) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := timing
			invalid.Commands = append([]enforce.CommandLifecycle(nil), timing.Commands...)
			tc.mutate(&invalid.Commands[1])
			stream := append(append([]byte(nil), raw...), encode(invalid)...)
			if got, err := parseTimingStream(bytes.NewReader(stream)); err == nil || got != nil {
				t.Fatalf("invalid retire must reject the complete stream: rows=%d err=%v", len(got), err)
			}
		})
	}
}

func TestAT02_F18RecordedRetireLifecycle(t *testing.T) {
	raw, err := readEvidence("../../../../docs/e2e/at/members/history/follow-up-18-run1/F18-lifecycle-phase-compatibility.json")
	if err != nil {
		t.Fatal(err)
	}
	var evidence struct {
		Commands []struct {
			Record json.RawMessage `json:"original_record"`
		} `json:"unsupported_phase_commands"`
	}
	if err := json.Unmarshal(raw, &evidence); err != nil || len(evidence.Commands) != 1 {
		t.Fatal("missing original sequence-582 reproduction", err)
	}
	// The published wrapper is indented; restore the producer's JSONL framing
	// without changing any value in the retained original record.
	var line bytes.Buffer
	if err := json.Compact(&line, evidence.Commands[0].Record); err != nil {
		t.Fatal(err)
	}
	rows, err := parseTimingStream(&line)
	if err != nil || len(rows) != 1 {
		t.Fatalf("original retire Apply rejected: rows=%d err=%v", len(rows), err)
	}
}
