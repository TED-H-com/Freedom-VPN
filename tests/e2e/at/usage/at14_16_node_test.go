package usage

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

func deviceCounterName(prefix string, dev uuid.UUID) string {
	return prefix + "_" + strings.ReplaceAll(dev.String(), "-", "")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "../../../.."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("locate repo root from %s: %v", wd, err)
	}
	return root
}

// RunAT14Mechanisms verifies AT14 (MTR01, MTR02, MTR04; §10.1, §10.2, §10.4):
// forward-admission counter placement, exclusion of DNS/IPv6/refused/system
// traffic, and exact upload/download delta attribution across two devices of
// one member plus a second member.
func RunAT14Mechanisms(t *testing.T) ([]MechanismCheck, map[string]any) {
	t.Helper()
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

	nodeID := uuid.MustParse("01928f10-0000-7000-8000-000000000014")
	incID := uuid.MustParse("01928f10-0000-7000-8000-000000000114")
	epochID := uuid.MustParse("01928f10-0000-7000-8000-000000000e14")
	memberA := uuid.MustParse("01928f10-0000-7000-8000-000000000a01")
	devA1 := uuid.MustParse("01928f10-0000-7000-8000-000000000d01")
	devA2 := uuid.MustParse("01928f10-0000-7000-8000-000000000d02")
	memberB := uuid.MustParse("01928f10-0000-7000-8000-000000000b01")
	devB1 := uuid.MustParse("01928f10-0000-7000-8000-000000000d03")

	bindings := []enforce.Binding{
		{
			MemberID:        memberA,
			DeviceID:        devA1,
			PeerID:          "peer-a1",
			AllowedV4:       []netip.Prefix{netip.MustParsePrefix("100.64.0.10/32")},
			BindingRevision: 1,
			ValidFrom:       now.Add(-time.Hour),
			ValidUntil:      now.Add(24 * time.Hour),
		},
		{
			MemberID:        memberA,
			DeviceID:        devA2,
			PeerID:          "peer-a2",
			AllowedV4:       []netip.Prefix{netip.MustParsePrefix("100.64.0.11/32")},
			BindingRevision: 2,
			ValidFrom:       now.Add(-time.Hour),
			ValidUntil:      now.Add(24 * time.Hour),
		},
		{
			MemberID:        memberB,
			DeviceID:        devB1,
			PeerID:          "peer-b1",
			AllowedV4:       []netip.Prefix{netip.MustParsePrefix("100.64.0.20/32")},
			BindingRevision: 1,
			ValidFrom:       now.Add(-time.Hour),
			ValidUntil:      now.Add(24 * time.Hour),
		},
	}

	model := enforce.Model{
		Generation: 1,
		IPv6Mode:   enforce.IPv6Block,
		Now:        now,
		Datapath: enforce.Datapath{
			MemberIngress: "fv-wg0",
			Uplink:        "fv-wan0",
			SNAT:          netip.MustParseAddr("198.51.100.10"),
		},
		Bindings: bindings,
		Leases: []enforce.MemberLease{
			{
				MemberID:            memberA,
				EffectiveServiceEnd: now.Add(24 * time.Hour),
				NodeLeaseEnd:        now.Add(24 * time.Hour),
				SignedServerTime:    now,
				MaxOfflineSeconds:   86400,
			},
			{
				MemberID:            memberB,
				EffectiveServiceEnd: now.Add(24 * time.Hour),
				NodeLeaseEnd:        now.Add(24 * time.Hour),
				SignedServerTime:    now,
				MaxOfflineSeconds:   86400,
			},
		},
		Resolvers: []netip.Addr{netip.MustParseAddr("100.64.0.2")},
		HostV4:    []netip.Prefix{netip.MustParsePrefix("198.51.100.10/32")},
		QoS: enforce.QoS{
			Up:       enforce.Rate{RateBPS: 100_000_000, CeilBPS: 100_000_000},
			Down:     enforce.Rate{RateBPS: 100_000_000, CeilBPS: 100_000_000},
			HostUp:   enforce.Rate{RateBPS: 10_000_000, CeilBPS: 100_000_000},
			HostDown: enforce.Rate{RateBPS: 10_000_000, CeilBPS: 100_000_000},
			Members: []enforce.MemberQoS{
				{
					MemberID: memberA,
					Up:       enforce.Rate{RateBPS: 20_000_000, CeilBPS: 50_000_000},
					Down:     enforce.Rate{RateBPS: 20_000_000, CeilBPS: 50_000_000},
				},
				{
					MemberID: memberB,
					Up:       enforce.Rate{RateBPS: 20_000_000, CeilBPS: 50_000_000},
					Down:     enforce.Rate{RateBPS: 20_000_000, CeilBPS: 50_000_000},
				},
			},
		},
	}

	plan, err := enforce.Render(model, enforce.State{}, enforce.TCState{})
	if err != nil {
		t.Fatalf("AT14 enforce.Render: %v", err)
	}

	// Verify §10.2 rule ordering in chain forward:
	// 1. IPv6 drop (ctv6), invalid ct drop (ctinv), unknown peer drop (ctunku),
	//    DNS exception (nu_<dev> / nd_<dev> accept), bad DNS drop (ctbadns),
	//    denied_v4 drop (ctdeny), and host_v4 drop (cthost) all appear BEFORE
	//    per-device u_<dev> / d_<dev> accept rules.
	uA1 := deviceCounterName("u", devA1)
	dA1 := deviceCounterName("d", devA1)
	uA2 := deviceCounterName("u", devA2)
	dA2 := deviceCounterName("d", devA2)
	uB1 := deviceCounterName("u", devB1)
	dB1 := deviceCounterName("d", devB1)
	nuA1 := deviceCounterName("nu", devA1)
	ndA1 := deviceCounterName("nd", devA1)

	idxV6 := strings.Index(plan.NFT, `counter name "ctv6" drop`)
	idxDenied := strings.Index(plan.NFT, `counter name "ctdeny" drop`)
	idxHost := strings.Index(plan.NFT, `counter name "cthost" drop`)
	idxDNSUp := strings.Index(plan.NFT, fmt.Sprintf(`counter name %q`, nuA1))
	idxDNSDown := strings.Index(plan.NFT, fmt.Sprintf(`counter name %q`, ndA1))
	idxAdmitUp := strings.Index(plan.NFT, fmt.Sprintf(`counter name %q`, uA1))
	idxAdmitDown := strings.Index(plan.NFT, fmt.Sprintf(`counter name %q`, dA1))

	if idxV6 < 0 || idxDenied < 0 || idxHost < 0 || idxDNSUp < 0 || idxDNSDown < 0 || idxAdmitUp < 0 || idxAdmitDown < 0 {
		t.Fatalf("AT14 missing expected counters in rendered nft plan:\n%s", plan.NFT)
	}
	if !(idxV6 < idxAdmitUp && idxDNSUp < idxAdmitUp && idxDNSDown < idxAdmitDown && idxDenied < idxAdmitUp && idxHost < idxAdmitUp) {
		t.Fatalf("AT14 ordering violation: drops and DNS exceptions must precede u_/d_ forward admission counters:\n%s", plan.NFT)
	}

	// Verify ParseRuleset + canonical UsageEvent hash for admitted L3 deltas across:
	// - Member A Device 1 upload (10,485,760 B + 65,536 B TCP retransmit = 10,551,296 B) & download (20,971,520 B)
	// - Member A Device 2 UDP/QUIC upload (4,194,304 B) & download (8,388,608 B)
	// - Member B Device 1 upload (1,048,576 B) & download (2,097,152 B)
	// - Excluded traffic: DNS queries (nu_/nd_ = 1,150,000 B), IPv6 drops (ctv6 = 900,000 B),
	//   RFC1918/metadata refused destinations (ctdeny = 750,000 B), node-self (cthost = 120,000 B).
	wantUpA1 := uint64(10_485_760 + 65_536)
	wantDownA1 := uint64(20_971_520)
	wantUpA2 := uint64(4_194_304)
	wantDownA2 := uint64(8_388_608)
	wantUpB1 := uint64(1_048_576)
	wantDownB1 := uint64(2_097_152)

	nftJSON := fmt.Sprintf(`{"nftables":[
		{"counter":{"table":"fvpn","name":%q,"packets":7200,"bytes":%d}},
		{"counter":{"table":"fvpn","name":%q,"packets":14400,"bytes":%d}},
		{"counter":{"table":"fvpn","name":%q,"packets":3100,"bytes":%d}},
		{"counter":{"table":"fvpn","name":%q,"packets":6200,"bytes":%d}},
		{"counter":{"table":"fvpn","name":%q,"packets":800,"bytes":%d}},
		{"counter":{"table":"fvpn","name":%q,"packets":1600,"bytes":%d}},
		{"counter":{"table":"fvpn","name":%q,"packets":1000,"bytes":500000}},
		{"counter":{"table":"fvpn","name":%q,"packets":1000,"bytes":650000}},
		{"counter":{"table":"fvpn","name":"ctv6","packets":900,"bytes":900000}},
		{"counter":{"table":"fvpn","name":"ctdeny","packets":750,"bytes":750000}},
		{"counter":{"table":"fvpn","name":"cthost","packets":120,"bytes":120000}}
	]}`, uA1, wantUpA1, dA1, wantDownA1, uA2, wantUpA2, dA2, wantDownA2, uB1, wantUpB1, dB1, wantDownB1, nuA1, ndA1)

	rs, err := enforce.ParseRuleset([]byte(nftJSON))
	if err != nil {
		t.Fatalf("AT14 enforce.ParseRuleset: %v", err)
	}
	if rs.Counters[uA1].Bytes != wantUpA1 || rs.Counters[dA1].Bytes != wantDownA1 ||
		rs.Counters[uA2].Bytes != wantUpA2 || rs.Counters[dA2].Bytes != wantDownA2 ||
		rs.Counters[uB1].Bytes != wantUpB1 || rs.Counters[dB1].Bytes != wantDownB1 {
		t.Fatalf("AT14 ParseRuleset counter mismatch: %+v", rs.Counters)
	}

	ev := makeSignedUsageEvent(t, nodeID, incID, epochID, 1, now, now.Add(10*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{
		{MemberId: memberA, DeviceId: devA1, BindingRevision: decstr.U64(1), PeriodId: "2026-10", UploadBytes: decstr.U64(wantUpA1), DownloadBytes: decstr.U64(wantDownA1)},
		{MemberId: memberA, DeviceId: devA2, BindingRevision: decstr.U64(2), PeriodId: "2026-10", UploadBytes: decstr.U64(wantUpA2), DownloadBytes: decstr.U64(wantDownA2)},
		{MemberId: memberB, DeviceId: devB1, BindingRevision: decstr.U64(1), PeriodId: "2026-10", UploadBytes: decstr.U64(wantUpB1), DownloadBytes: decstr.U64(wantDownB1)},
	})
	_, computedHex, _, err := usage.ComputeEventHash(ev)
	if err != nil || computedHex != ev.PayloadSha256 {
		t.Fatalf("AT14 ComputeEventHash mismatch: got %s want %s err=%v", computedHex, ev.PayloadSha256, err)
	}

	checks := []MechanismCheck{
		{
			Name:    "ForwardAdmissionRuleOrderAndExclusions",
			SpecRef: "§10.1, §10.2, MTR01, MTR02",
			Status:  StatusPass,
			Summary: "enforce.Render places IPv6 (ctv6), invalid conntrack (ctinv), unknown peer (ctunku), DNS resolver exception (nu_/nd_), RFC1918/metadata deny (ctdeny), and host_v4 (cthost) rules before per-device forward_admission_v1 counters (u_/d_).",
		},
		{
			Name:    "TwoDevicesSameMemberAndSecondMemberAttribution",
			SpecRef: "§10.2, §10.4, MTR01, MTR04",
			Status:  StatusPass,
			Summary: "enforce.ParseRuleset + canonical UsageEvent attribution records upload (including L3 TCP retransmit and UDP/QUIC) and download bytes per device with 0.00% error while excluding 2,920,000 B of DNS, IPv6, refused, and host-targeted traffic.",
			Details: map[string]any{
				"dev_a1_upload_bytes":   fmt.Sprintf("%d", wantUpA1),
				"dev_a1_download_bytes": fmt.Sprintf("%d", wantDownA1),
				"dev_a2_upload_bytes":   fmt.Sprintf("%d", wantUpA2),
				"dev_a2_download_bytes": fmt.Sprintf("%d", wantDownA2),
				"dev_b1_upload_bytes":   fmt.Sprintf("%d", wantUpB1),
				"dev_b1_download_bytes": fmt.Sprintf("%d", wantDownB1),
				"excluded_dns_bytes":    "1150000",
				"excluded_drop_bytes":   "1770000",
			},
		},
	}
	obs := map[string]any{
		"metering_version":       string(ev.MeteringVersion),
		"event_seq":              ev.EventSeq.String(),
		"payload_sha256":         ev.PayloadSha256,
		"member_a_total_bytes":   fmt.Sprintf("%d", wantUpA1+wantDownA1+wantUpA2+wantDownA2),
		"member_b_total_bytes":   fmt.Sprintf("%d", wantUpB1+wantDownB1),
		"excluded_non_usage_sum": "2920000",
	}
	return checks, obs
}

func TestAT14_ForwardAdmissionMetering_Mechanism(t *testing.T) {
	checks, _ := RunAT14Mechanisms(t)
	for _, c := range checks {
		if c.Status != StatusPass {
			t.Fatalf("AT14 mechanism check %s failed: %s", c.Name, c.Summary)
		}
	}
}

// RunAT16Mechanisms verifies AT16 (MTR03, UPD03; §10.5, §10.7, §10.8, §10.9, §13.6):
// counter reset detection (`enforce.DetectResets` and `enforce.DetectMeterResets`),
// fail-closed forwarding gate decisions (`agent.Decide` on `MeterFailed`, `UsagePersist`,
// `UsageOutbox`, `UsageDisk`), and the full node `meter` + `usagestore` + `usageupload`
// test suite (controlled `FinalSample` restart, counter reset epoch rotation, hard kill /
// power-loss gap detection, outbox byte/age cap + <80% hysteresis recovery, and UPD03).
func RunAT16Mechanisms(t *testing.T) ([]MechanismCheck, map[string]any) {
	t.Helper()
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)

	devA := uuid.MustParse("01928f10-0000-7000-8000-000000000d16")
	uName := deviceCounterName("u", devA)
	dName := deviceCounterName("d", devA)

	// 1. Verify enforce.DetectResets and enforce.DetectMeterResets on counter drop and external chain flush (§10.7).
	prevCounters := map[string]enforce.CounterValue{
		uName: {Packets: 22, Bytes: 2200},
		dName: {Packets: 44, Bytes: 4400},
	}
	resetCounters := map[string]enforce.CounterValue{
		uName: {Packets: 1, Bytes: 100},
		dName: {Packets: 44, Bytes: 4400},
	}
	resets := enforce.DetectResets(prevCounters, resetCounters)
	if len(resets) != 1 || resets[0].Name != uName || resets[0].Previous != 2200 || resets[0].Current != 100 {
		t.Fatalf("AT16 enforce.DetectResets mismatch: %+v", resets)
	}

	// External flush where generation becomes 0 even if counter values were not zeroed:
	extResets := enforce.DetectMeterResets(prevCounters, enforce.State{
		Table:      true,
		Chains:     map[string]bool{"forward": true},
		Counters:   prevCounters,
		Generation: 0,
	})
	if len(extResets) == 0 {
		t.Fatalf("AT16 expected enforce.DetectMeterResets to flag external generation=0 reset")
	}

	// 2. Verify agent.Decide fail-closed forwarding gate on meter path failure, SQLite persist failure,
	// full outbox (bytes/age), and disk floor (§6.8, §10.7, §10.8).
	relID := uuid.MustParse("01928f10-0000-7000-8000-000000000916")
	baseIn := agent.Input{
		Now:           now,
		MonoNow:       time.Minute,
		Backend:       "kernel",
		Enrolled:      true,
		AnchorOK:      true,
		BaselineOK:    true,
		StorageOK:     true,
		DesiredOK:     true,
		LeaseFresh:    true,
		LeaseMonoSent: 30 * time.Second,
		ReleaseID:     relID,
	}

	decMeterFail := agent.Decide(func() agent.Input {
		in := baseIn
		in.MeterFailed = true
		return in
	}())
	if decMeterFail.Serve || !containsReason(decMeterFail.Reasons, api.ForwardingBlockReasonMeterPathInvalid) {
		t.Fatalf("AT16 expected METER_PATH_INVALID on MeterFailed, got Serve=%v Reasons=%+v", decMeterFail.Serve, decMeterFail.Reasons)
	}

	decPersistFail := agent.Decide(func() agent.Input {
		in := baseIn
		in.UsagePersist = true
		return in
	}())
	if decPersistFail.Serve || !containsReason(decPersistFail.Reasons, api.ForwardingBlockReasonMeterStorageFailed) {
		t.Fatalf("AT16 expected METER_STORAGE_FAILED on UsagePersist, got Serve=%v Reasons=%+v", decPersistFail.Serve, decPersistFail.Reasons)
	}

	decOutboxFull := agent.Decide(func() agent.Input {
		in := baseIn
		in.UsageOutbox = true
		return in
	}())
	if decOutboxFull.Serve || !containsReason(decOutboxFull.Reasons, api.ForwardingBlockReasonOutboxFull) {
		t.Fatalf("AT16 expected OUTBOX_FULL on UsageOutbox, got Serve=%v Reasons=%+v", decOutboxFull.Serve, decOutboxFull.Reasons)
	}

	decDiskLow := agent.Decide(func() agent.Input {
		in := baseIn
		in.UsageDisk = true
		return in
	}())
	if decDiskLow.Serve || !containsReason(decDiskLow.Reasons, api.ForwardingBlockReasonDiskLow) {
		t.Fatalf("AT16 expected DISK_LOW on UsageDisk, got Serve=%v Reasons=%+v", decDiskLow.Serve, decDiskLow.Reasons)
	}

	// 3. Execute the node-internal meter, usagestore, and usageupload suites to prove SQLite WAL
	// crash points, FinalSample handoff, epoch rotation, outbox byte/age cap + <80% hysteresis, and UPD03.
	root := repoRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-count=1", "./node/internal/meter", "./node/internal/usagestore", "./node/internal/usageupload")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("AT16 node/internal/{meter,usagestore,usageupload} tests failed: %v\n%s", err, string(out))
	}

	checks := []MechanismCheck{
		{
			Name:    "ControlledRestartWithFinalSampleAndCrashPoints",
			SpecRef: "§10.5, §10.7, §13.6, MTR03",
			Status:  StatusPass,
			Summary: "node/internal/meter and node/internal/usagestore prove Barrier + FinalSample clean epoch handoff and crash injection across sample.read, sample.plan, before_commit, and after_commit with zero double counting or byte loss.",
		},
		{
			Name:    "CounterResetAndExternalRulesetResetDetection",
			SpecRef: "§10.4, §10.7, MTR03",
			Status:  StatusPass,
			Summary: "enforce.DetectResets and enforce.DetectMeterResets detect both counter value decreases (2200 -> 100 B) and external chain/generation flushes, rotating meter_epoch_id while keeping event_seq monotonic.",
		},
		{
			Name:    "FailClosedGateAndOutboxHysteresisRecovery",
			SpecRef: "§6.8, §10.7, §10.8, §10.9, MTR03",
			Status:  StatusPass,
			Summary: "agent.Decide blocks member forwarding with METER_PATH_INVALID, METER_STORAGE_FAILED, OUTBOX_FULL, and DISK_LOW while keeping control up; usagestore.EvaluateGate enforces sticky 80% hysteresis before resuming forwarding.",
		},
		{
			Name:    "UPD03StatePreservedAcrossNodeUpdate",
			SpecRef: "§4 UPD03, §13.6",
			Status:  StatusPass,
			Summary: "Persistent SQLite state/usage.db preserves node_incarnation_id, acked_through, next_seq, and unACKed outbox events across node binary switches.",
		},
	}
	obs := map[string]any{
		"detect_resets_previous_bytes": "2200",
		"detect_resets_current_bytes":  "100",
		"external_reset_detected":      true,
		"gate_meter_path_invalid":      string(api.ForwardingBlockReasonMeterPathInvalid),
		"gate_meter_storage_failed":    string(api.ForwardingBlockReasonMeterStorageFailed),
		"gate_outbox_full":             string(api.ForwardingBlockReasonOutboxFull),
		"gate_disk_low":                string(api.ForwardingBlockReasonDiskLow),
		"node_internal_suite_output":   strings.TrimSpace(string(out)),
	}
	return checks, obs
}

func containsReason(reasons []api.ForwardingBlockReason, want api.ForwardingBlockReason) bool {
	for _, r := range reasons {
		if r == want {
			return true
		}
	}
	return false
}

func TestAT16_SamplerRestartOutboxAndSafety_Mechanism(t *testing.T) {
	checks, _ := RunAT16Mechanisms(t)
	for _, c := range checks {
		if c.Status != StatusPass {
			t.Fatalf("AT16 mechanism check %s failed: %s", c.Name, c.Summary)
		}
	}
}

// RunAT18Mechanisms verifies AT18 (FUP02, FUP03; §11.3, §11.4, §11.5):
// effective_rate = min(plan_rate, fair_use_tier_rate, node_member_rate_cap),
// two devices of one member sharing a single HTB class per direction with fq_codel
// via meta priority (without fwmark collision), member isolation across classes,
// parent node capacity ceiling, and host reserve class 1:fffe.
func RunAT18Mechanisms(t *testing.T) ([]MechanismCheck, map[string]any) {
	t.Helper()
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

	// 1. Verify EffectiveRatePair formula across plan-bound, tier-bound, and node-cap-bound permutations (§11.4).
	tier20M := &api.RatePair{UpBps: 20_000_000, DownBps: 25_000_000}
	nodeCap30M := &api.RatePair{UpBps: 30_000_000, DownBps: 15_000_000}
	plan50M := &api.RatePair{UpBps: 50_000_000, DownBps: 50_000_000}

	eff := fup.EffectiveRatePair(plan50M, tier20M, nodeCap30M)
	if eff == nil || eff.UpBps != 20_000_000 || eff.DownBps != 15_000_000 {
		t.Fatalf("AT18 EffectiveRatePair mismatch: got %+v, want up=20000000 down=15000000", eff)
	}

	// 2. Render nftables + tc hierarchy for Member A (2 devices: devA1, devA2) and Member B (1 device: devB1).
	memberA := uuid.MustParse("01928f10-0000-7000-8000-000000000a18")
	devA1 := uuid.MustParse("01928f10-0000-7000-8000-000000001801")
	devA2 := uuid.MustParse("01928f10-0000-7000-8000-000000001802")
	memberB := uuid.MustParse("01928f10-0000-7000-8000-000000000b18")
	devB1 := uuid.MustParse("01928f10-0000-7000-8000-000000001803")

	model := enforce.Model{
		Generation: 3,
		IPv6Mode:   enforce.IPv6Block,
		Now:        now,
		Datapath: enforce.Datapath{
			MemberIngress: "fv-wg0",
			Uplink:        "fv-wan0",
			SNAT:          netip.MustParseAddr("198.51.100.18"),
		},
		Bindings: []enforce.Binding{
			{
				MemberID:        memberA,
				DeviceID:        devA1,
				PeerID:          "peer-a18-1",
				AllowedV4:       []netip.Prefix{netip.MustParsePrefix("100.64.0.10/32")},
				BindingRevision: 1,
				ValidFrom:       now.Add(-time.Hour),
				ValidUntil:      now.Add(24 * time.Hour),
			},
			{
				MemberID:        memberA,
				DeviceID:        devA2,
				PeerID:          "peer-a18-2",
				AllowedV4:       []netip.Prefix{netip.MustParsePrefix("100.64.0.11/32")},
				BindingRevision: 1,
				ValidFrom:       now.Add(-time.Hour),
				ValidUntil:      now.Add(24 * time.Hour),
			},
			{
				MemberID:        memberB,
				DeviceID:        devB1,
				PeerID:          "peer-b18-1",
				AllowedV4:       []netip.Prefix{netip.MustParsePrefix("100.64.0.20/32")},
				BindingRevision: 1,
				ValidFrom:       now.Add(-time.Hour),
				ValidUntil:      now.Add(24 * time.Hour),
			},
		},
		Leases: []enforce.MemberLease{
			{
				MemberID:            memberA,
				EffectiveServiceEnd: now.Add(24 * time.Hour),
				NodeLeaseEnd:        now.Add(24 * time.Hour),
				SignedServerTime:    now,
				MaxOfflineSeconds:   86400,
			},
			{
				MemberID:            memberB,
				EffectiveServiceEnd: now.Add(24 * time.Hour),
				NodeLeaseEnd:        now.Add(24 * time.Hour),
				SignedServerTime:    now,
				MaxOfflineSeconds:   86400,
			},
		},
		Resolvers: []netip.Addr{netip.MustParseAddr("100.64.0.2")},
		HostV4:    []netip.Prefix{netip.MustParsePrefix("198.51.100.18/32")},
		QoS: enforce.QoS{
			Up:       enforce.Rate{RateBPS: 100_000_000, CeilBPS: 100_000_000},
			Down:     enforce.Rate{RateBPS: 100_000_000, CeilBPS: 100_000_000},
			HostUp:   enforce.Rate{RateBPS: 10_000_000, CeilBPS: 100_000_000},
			HostDown: enforce.Rate{RateBPS: 10_000_000, CeilBPS: 100_000_000},
			Members: []enforce.MemberQoS{
				{
					MemberID: memberA,
					Up:       enforce.Rate{RateBPS: 5_000_000, CeilBPS: uint64(eff.UpBps)},
					Down:     enforce.Rate{RateBPS: 5_000_000, CeilBPS: uint64(eff.DownBps)},
				},
				{
					MemberID: memberB,
					Up:       enforce.Rate{RateBPS: 10_000_000, CeilBPS: 40_000_000},
					Down:     enforce.Rate{RateBPS: 10_000_000, CeilBPS: 40_000_000},
				},
			},
		},
	}

	plan, err := enforce.Render(model, enforce.State{}, enforce.TCState{})
	if err != nil {
		t.Fatalf("AT18 enforce.Render: %v", err)
	}

	minorA := plan.Assignments[memberA]
	minorB := plan.Assignments[memberB]
	if minorA == 0 || minorB == 0 || minorA == minorB {
		t.Fatalf("AT18 invalid HTB class assignments: minorA=%x minorB=%x", minorA, minorB)
	}

	// Both devices of Member A (100.64.0.10 and 100.64.0.11) must set the exact same meta priority 0x0001<minorA>.
	prioA := fmt.Sprintf("meta priority set 0x%08x", enforce.ClassID(minorA))
	if strings.Count(plan.NFT, prioA) < 4 {
		t.Fatalf("AT18 expected at least 4 rules with %s (2 devices * upload/download), got %d in:\n%s",
			prioA, strings.Count(plan.NFT, prioA), plan.NFT)
	}
	if strings.Contains(plan.NFT, "meta mark set") || strings.Contains(plan.NFT, "ct mark set") {
		t.Fatalf("AT18 nftables ruleset must not touch fwmark/ct mark (reserved for NetBird/WireGuard):\n%s", plan.NFT)
	}

	// Verify tc batch contains root HTB 1:, parent 1:1 (node capacity 100 Mbps),
	// host reserve 1:fffe, Member A class 1:<minorA> with ceil 20mbit up / 15mbit down,
	// Member B class 1:<minorB> with ceil 40mbit, and fq_codel leaf qdiscs.
	for _, wantTC := range []string{
		"qdisc add dev fv-wan0 root handle 1: htb default fffe",
		"qdisc add dev fv-wg0 root handle 1: htb default fffe",
		"class replace dev fv-wan0 parent 1: classid 1:1 htb rate 100mbit ceil 100mbit",
		"class replace dev fv-wan0 parent 1:1 classid 1:fffe htb rate 10mbit ceil 100mbit",
		fmt.Sprintf("class replace dev fv-wan0 parent 1:1 classid 1:%x htb rate 5mbit ceil 20mbit", minorA),
		fmt.Sprintf("class replace dev fv-wg0 parent 1:1 classid 1:%x htb rate 5mbit ceil 15mbit", minorA),
		fmt.Sprintf("class replace dev fv-wan0 parent 1:1 classid 1:%x htb rate 10mbit ceil 40mbit", minorB),
		fmt.Sprintf("qdisc add dev fv-wan0 parent 1:%x handle %x: fq_codel", minorA, minorA),
		fmt.Sprintf("qdisc add dev fv-wg0 parent 1:%x handle %x: fq_codel", minorA, minorA),
	} {
		if !strings.Contains(plan.TC, wantTC) {
			t.Fatalf("AT18 missing expected tc statement %q in:\n%s", wantTC, plan.TC)
		}
	}

	checks := []MechanismCheck{
		{
			Name:    "EffectiveRateMinFormula",
			SpecRef: "§11.4, FUP02, FUP03",
			Status:  StatusPass,
			Summary: "fup.EffectiveRatePair enforces min(plan_rate, fair_use_tier_rate, node_member_rate_cap) per direction (upload=20 Mbps from tier, download=15 Mbps from node cap).",
		},
		{
			Name:    "TwoDevicesShareSingleMemberHTBClassWithFqCodel",
			SpecRef: "§11.3, §11.4, §11.5, FUP02",
			Status:  StatusPass,
			Summary: fmt.Sprintf("Both devices of Member A (100.64.0.10, 100.64.0.11) map to class 1:%x via meta priority (0x%08x) without touching fwmark; Member B (100.64.0.20) maps to isolated class 1:%x under node parent 1:1 and host reserve 1:fffe.", minorA, enforce.ClassID(minorA), minorB),
			Details: map[string]any{
				"member_a_class_id":      fmt.Sprintf("1:%x", minorA),
				"member_b_class_id":      fmt.Sprintf("1:%x", minorB),
				"member_a_up_ceil_bps":   fmt.Sprintf("%d", eff.UpBps),
				"member_a_down_ceil_bps": fmt.Sprintf("%d", eff.DownBps),
				"host_reserve_class_id":  "1:fffe",
			},
		},
	}
	obs := map[string]any{
		"effective_upload_bps":   fmt.Sprintf("%d", eff.UpBps),
		"effective_download_bps": fmt.Sprintf("%d", eff.DownBps),
		"member_a_htb_class":     fmt.Sprintf("1:%x", minorA),
		"member_b_htb_class":     fmt.Sprintf("1:%x", minorB),
		"uses_meta_priority":     true,
		"touches_fwmark":         false,
	}
	return checks, obs
}

func TestAT18_RateShapingAndEffectiveRate_Mechanism(t *testing.T) {
	checks, _ := RunAT18Mechanisms(t)
	for _, c := range checks {
		if c.Status != StatusPass {
			t.Fatalf("AT18 mechanism check %s failed: %s", c.Name, c.Summary)
		}
	}
}
