//go:build e2e

package members

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
)

type acceptanceCase struct {
	id, title, requirements, expected, next string
	operator                                bool
}

// §19.1 requires unavailable provenance to remain explicitly unavailable.
// A prepare-binary digest is not a control/node build or a container digest.
type provenance struct {
	SourceCommit      string   `json:"source_commit"`
	SourceDirty       bool     `json:"source_dirty"`
	PrepareSHA256     string   `json:"prepare_binary_sha256"`
	ImageDigests      []string `json:"image_digests"`
	PolicyRevision    *string  `json:"policy_revision"`
	NodeIncarnation   *string  `json:"node_incarnation"`
	UnavailableReason string   `json:"unavailable_reason"`
}

type evidence struct {
	e2e.Record
	Scope          string     `json:"scope"`
	Build          provenance `json:"build"`
	Expected       string     `json:"expected"`
	Actual         string     `json:"actual"`
	Commands       []string   `json:"commands"`
	OperatorStatus string     `json:"operator_status,omitempty"`
}

// TestATMembers_AssignedStackGate intentionally fails when acceptance is blocked.
// It calls the owner's validator, not a copy of its slot arithmetic. No Docker,
// alternative ports, generated identity rows or standalone lab are substituted.
func TestATMembers_AssignedStackGate(t *testing.T) {
	if os.Getenv("FVPN_E2E") != "1" {
		t.Skip("set FVPN_E2E=1 through make -f mk/at-members.mk at-members")
	}
	if os.Getenv("FVPN_E2E_PREFIX") != "w4-members" || os.Getenv("FVPN_E2E_PORT_BASE") != "19000" {
		t.Fatal("this assignment requires prefix w4-members and port base 19000")
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "../../../.."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatal("repository root unavailable")
	}
	base := filepath.Join(root, ".agent", "tmp")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	scratch, err := os.MkdirTemp(base, "at-members-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(scratch); err != nil {
			t.Error("remove owned disk scratch failed")
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	bin := filepath.Join(scratch, "prepare")
	build := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", bin, "./tests/e2e/cmd/fvpn-e2e-prepare")
	build.Dir = root
	build.Env = append(os.Environ(), "TMPDIR="+scratch, "GOTMPDIR="+scratch, "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	if _, err := build.CombinedOutput(); err != nil {
		t.Fatal("build existing prepare command failed; no acceptance evidence recorded")
	}
	bytes, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bytes)
	commitCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	commitCmd.Dir = root
	commit, err := commitCmd.Output()
	if err != nil {
		t.Fatal("source commit unavailable")
	}
	dirtyCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	dirtyCmd.Dir = root
	dirty, err := dirtyCmd.Output()
	if err != nil {
		t.Fatal("source worktree status unavailable")
	}
	prepare := exec.CommandContext(ctx, bin, "-state", filepath.Join(scratch, "state"), "-prefix", "w4-members", "-port-base", "19000")
	prepare.Dir = root
	out, runErr := prepare.CombinedOutput()
	const diagnostic = "fvpn-e2e-prepare: port base must be 18000 plus a multiple of 6000"
	status := e2e.StatusFail
	actual := "Unexpected prepare result; inspect the owner command locally. No live acceptance was executed."
	var exit *exec.ExitError
	if errors.As(runErr, &exit) && exit.ExitCode() == 2 && strings.TrimSpace(string(out)) == diagnostic {
		status = e2e.StatusBlocked
		actual = diagnostic + "; exit 2 before stack creation. Needs devstack follow-up 2."
	} else if runErr == nil {
		// Do not turn a repaired startup gate into a false AT pass. The live
		// scenarios still need implementation after the owner's hooks land.
		status = e2e.StatusBlocked
		actual = "Layout is now accepted. Resume the live acceptance implementation and operator scripts; this startup gate alone proves no AT."
	}
	commands := []string{
		"go build -trimpath -o <owned-disk-scratch>/prepare ./tests/e2e/cmd/fvpn-e2e-prepare",
		"<owned-disk-scratch>/prepare -state <owned-disk-scratch>/state -prefix w4-members -port-base 19000",
	}
	for _, c := range catalogue() {
		t.Run(c.id, func(t *testing.T) {
			rec := evidence{
				Record: e2e.Record{ID: c.id, Title: c.title, Spec: []string{"§19.1", "§19 " + c.id, c.requirements}, Status: status, At: time.Now().UTC().Format(time.RFC3339),
					Observations: []e2e.Observation{
						{Name: "scope", Detail: "startup prerequisite only; no live scenario executed"},
						{Name: "expected", Detail: c.expected},
						{Name: "actual", Detail: actual},
						{Name: "remaining_preconditions", Detail: c.next},
						{Name: "source_commit", Detail: strings.TrimSpace(string(commit))},
						{Name: "prepare_binary_sha256", Detail: hex.EncodeToString(sum[:])},
						{Name: "runtime_provenance", Detail: "image digests, policy revision and node incarnation unavailable: no stack observed"},
						{Name: "commands", Detail: strings.Join(commands, " ; ")},
						{Name: "procedure", Detail: "See PROCEDURES.md and BLOCKERS.md in this directory"},
					}},
				Scope: "startup prerequisite only", Expected: c.expected, Actual: actual, Commands: commands,
				Build: provenance{SourceCommit: strings.TrimSpace(string(commit)), SourceDirty: len(dirty) != 0, PrepareSHA256: hex.EncodeToString(sum[:]), ImageDigests: []string{}, UnavailableReason: "No running stack observed; image, policy and incarnation provenance unavailable"},
			}
			if c.operator {
				rec.OperatorStatus = "needs operator run"
				rec.Observations = append(rec.Observations, e2e.Observation{Name: "operator_status", Detail: rec.OperatorStatus})
			}
			dir := filepath.Join(root, "docs", "e2e", "at", "members")
			if err := e2e.Write(dir, rec.Record); err != nil {
				t.Fatal("write shared runner evidence failed")
			}
			// The shared writer appends a blank line at EOF; normalize only our
			// pages so the public-repository whitespace check stays clean.
			page := filepath.Join(dir, c.id+".md")
			markdown, err := os.ReadFile(page)
			if err != nil {
				t.Fatal("read generated evidence page failed")
			}
			if err := os.WriteFile(page, []byte(strings.TrimRight(string(markdown), "\n")+"\n"), 0o644); err != nil {
				t.Fatal("normalize generated evidence page failed")
			}
			// Keep the runner's Markdown format and add typed provenance to JSON.
			raw, err := json.MarshalIndent(rec, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, c.id+".json"), append(raw, '\n'), 0o644); err != nil {
				t.Fatal("write acceptance provenance failed")
			}
			t.Errorf("%s: %s", status, actual)
		})
	}
}

func catalogue() []acceptanceCase {
	return []acceptanceCase{
		{"AT01", "Identity and first use", "APP01 ID01 MEM01", "Two OAuth clients map through verified identity to the correct member/device; selected exit forwards; sign-out, network change and wake recover correctly.", "Second OAuth client and native user-peer fixture; serving node and bound device; four platform runs.", true},
		{"AT02", "Device quota race and isolation", "MEM01 MEM02 MEM03 SEC01", "One final-slot winner, loser denied at gateway, no cross-member disclosure, exactly one slot released by revoke.", "Three member identities and native peers; verified binding; concurrent admission hook; serving gateway. See AT02-TRACE.md.", false},
		{"AT03", "Payment replay and ordering", "BILL01 BILL03 ENT01", "Ten identical paid events and all six created/paid/failed orders preserve one paid interval; bad signature and forged return grant nothing.", "Mutable authoritative payment snapshot/event hook; bound device and serving gateway.", false},
		{"AT04", "Expiry, cancellation, refunds and suspension", "BILL02 ENT02 MEM03", "Cancellation preserves paid time; explicit refund policy applies; payment never clears suspension; existing/new TCP and UDP stop on every online node within the configured 60-second target.", "Payment/refund fixture, finite entitlement, two serving exits, controlled long-flow endpoint and portal observation.", false},
		{"AT05", "Three node join sources", "NODE01 NODE04 ENR01 ENR04 PKG01", "Preinstalled, cloud-init and installer nodes use the same protocol, become unpublished, then classify and publish through the same API.", "Three node source flavours; package digests; publication readback; physical arm64 execution.", true},
		{"AT08", "Publication, groups and disable", "NODE02 NODE03 EXIT01 EXIT03", "Only group A forwards; rename/regroup converges; disabled intent and gateway denial survive reconnect and reboot; pending and applied stay distinct.", "Publication worker follow-up 5 and devstack follow-up 3; group A/B user peers and serving probe.", false},
		{"AT09", "Exit selection, NAT, relay and MTU", "EXIT02 APP01 NFR01", "Two exits select correctly with skip_auto_apply, failure behaviour is measured, direct/relay TCP UDP QUIC and PMTU baselines are recorded.", "Second exit, working relay, controlled flow/PMTU fixtures and second NAT; native platform tests.", true},
		{"AT20", "DNS rules and record types", "DNS01 DNS02 ISO04", "Case/dot/IDNA and label boundaries match; URL input rejects; all RR types obey block/TTL; allowlisted rebinding never opens internal networks.", "Two filtering resolvers, controlled RR fixtures, applied policy and actual member DNS/forwarding path.", false},
		{"AT21", "DNS failure and encrypted DNS boundary", "DNS03 DNS04 APP01", "Resolver/policy-sync faults keep controlled fallback or fail closed; Android Private DNS and browser DoH limitations match guidance.", "Second filtering resolver; client DNS enabled; fault hooks and controlled DoH/DoT endpoints; platform operator runs.", true},
		{"AT22", "Privacy, feed sources and retention", "SEC02 DNS01 DNS04", "Synthetic qnames/URLs absent from query history, logs, metrics, audit and backups; bounded retention purges and licensed last-good feed handling verified.", "Candidate configuration, backup/retention fixture and approved feed inventory; queried sentinel traffic and update-failure hook.", false},
	}
}
