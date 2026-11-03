//go:build netlab

package at

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/at/isolation"
)

//go:embed d147_runner.py
var d147RunnerPy []byte

type d147CounterDelta struct {
	Packets int `json:"packets"`
	Bytes   int `json:"bytes"`
}

type d147TCLeaf struct {
	Packets int `json:"packets"`
	Bytes   int `json:"bytes"`
	Drops   int `json:"drops"`
}

type d147UsageReconciliation struct {
	DeviceID                      string     `json:"device_id"`
	AssignedPriority              string     `json:"assigned_priority"`
	AssignedTCClass               string     `json:"assigned_tc_class"`
	UploadCounterMatchesCapture   bool       `json:"upload_counter_matches_capture"`
	DownloadCounterMatchesCapture bool       `json:"download_counter_matches_capture"`
	DownloadTCPIPBytes            int        `json:"download_tcp_ip_bytes"`
	DownloadFragNeededICMPPackets int        `json:"download_frag_needed_icmp_packets"`
	DownloadFragNeededICMPIPBytes int        `json:"download_frag_needed_icmp_ip_bytes"`
	DownloadTotalIPBytes          int        `json:"download_total_ip_bytes"`
	TCUploadLeaf                  d147TCLeaf `json:"tc_upload_leaf"`
	TCDownloadLeaf                d147TCLeaf `json:"tc_download_leaf"`
}

type d147Transfer struct {
	OK        bool    `json:"ok"`
	Status    string  `json:"status"`
	Body      string  `json:"body"`
	ElapsedMS float64 `json:"elapsed_ms"`
	PMTU      *int    `json:"pmtu"`
	Error     string  `json:"error,omitempty"`
}

type d147Subcase struct {
	Name                string                      `json:"name"`
	Passed              bool                        `json:"passed"`
	PMTUBefore          *int                        `json:"pmtu_before,omitempty"`
	Transfer            *d147Transfer               `json:"transfer,omitempty"`
	DroppedAtCounter    string                      `json:"dropped_at_counter,omitempty"`
	Captures            map[string]json.RawMessage  `json:"captures"`
	Counters            map[string]d147CounterDelta `json:"counters"`
	UsageReconciliation *d147UsageReconciliation    `json:"usage_reconciliation,omitempty"`
}

type d147SetMembership struct {
	InResolverV4 bool     `json:"in_resolver_v4"`
	InDeniedV4   bool     `json:"in_denied_v4"`
	InHostV4     bool     `json:"in_host_v4"`
	MatchedSets  []string `json:"matched_sets"`
}

type d147Row struct {
	ID                    string            `json:"id"`
	SourceIP              string            `json:"source_ip"`
	ExpectedSets          []string          `json:"expected_sets"`
	SetMembership         d147SetMembership `json:"set_membership"`
	SetMembershipVerified bool              `json:"set_membership_verified"`
	UploadDropCounter     string            `json:"upload_drop_counter"`
	Passed                bool              `json:"passed"`
	Subcases              []d147Subcase     `json:"subcases"`
}

type d147Report struct {
	Passed           bool      `json:"passed"`
	MemberNS         string    `json:"member_ns"`
	MemberIP         string    `json:"member_ip"`
	DeviceID         string    `json:"device_id"`
	SNATIP           string    `json:"snat_ip"`
	OriginIP         string    `json:"origin_ip"`
	OriginPort       int       `json:"origin_port"`
	PathMTU          int       `json:"path_mtu"`
	UploadBytes      int       `json:"upload_bytes"`
	AssignedPriority string    `json:"assigned_priority"`
	AssignedTCClass  string    `json:"assigned_tc_class"`
	Rows             []d147Row `json:"rows"`
}

func d147LabModel(gen uint64) enforce.Model {
	m := labModel(gen, allBinds(), nil)
	// Add 203.0.113.53 to Resolvers so it is ONLY in @resolver_v4 (203.0.113.0/24
	// is omitted from DeniedV4, and 203.0.113.53 is not in HostV4).
	m.Resolvers = []netip.Addr{
		netip.MustParseAddr("198.51.100.53"),
		netip.MustParseAddr("203.0.113.53"),
	}
	// Add 203.0.113.33/32 to HostV4 so it is ONLY in @host_v4, and 192.168.34.1/32
	// so it is in BOTH @denied_v4 (192.168.0.0/16) and @host_v4.
	m.HostV4 = append(
		append([]netip.Prefix{}, m.HostV4...),
		netip.MustParsePrefix("203.0.113.33/32"),
		netip.MustParsePrefix("192.168.34.1/32"),
	)
	return m
}

func TestAT13_D147RelatedFragNeeded(t *testing.T) {
	skipLab(t)
	rec := labMeta("generation 4200")
	rec.ID = "lab-AT13-d147"
	rec.Title = "Mechanism: D-147 RELATED IPv4 fragmentation-needed per source set and overlap"
	rec.Spec = []string{"§19 AT13", "ISO01", "ISO02", "MTR01", "D-146", "D-147"}
	rec.Status = isolation.StatusFail
	rec.Expected = "Genuine router IPv4 ICMP fragmentation-needed (type 3, code 4, MTU 1000) from resolver_v4-only, denied_v4-only, host_v4-only, and denied_v4+host_v4 overlap sources reaches the admitted member when conntrack marks it RELATED, is counted once in d_<device> at the device's assigned priority (1:2f), and matches independent captures; time-exceeded is dropped at ctdenyd, unsolicited fragmentation-needed at ctinv, and member upload to that source is dropped with 0 upload usage."
	rec.Actual = "not finished"
	t.Cleanup(func() {
		_, _, _ = dockerExec("fvpn-netlab", "baseline")
		finish(t, rec)
	})

	flush(t)
	m := d147LabModel(4200)
	doc := applyModel(t, m)
	if !doc.Verified || doc.Admitted != 3 {
		fail(t, &rec, "d147 apply generation 4200: verified=%v admitted=%d", doc.Verified, doc.Admitted)
	}
	d147Rules := ruleset(t)
	_ = isolation.WriteExtra(evidenceDir(), "lab-AT13-d147-ruleset.nft", d147Rules)
	fview := isolation.ParseForwardChain(d147Rules)
	if !fview.Present || fview.Policy != "drop" || fview.Baseline || fview.Generation != 4200 || fview.MSSClamp != 6 || fview.PMTUAccept != 9 || fview.LabMemberAccept != 27 {
		fail(t, &rec, "d147 forward chain view mismatch: %+v", fview)
	}

	out, errOut, err := dockerExecStdin([]string{"python3", "-B", "-"}, d147RunnerPy)
	if err != nil {
		fail(t, &rec, "d147 runner failed: %v\nstderr:\n%s\nstdout:\n%s", err, errOut, out)
	}
	var report d147Report
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		fail(t, &rec, "d147 runner json decode: %v\n%s", err, clipOne(out))
	}
	pretty, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fail(t, &rec, "d147 runner json marshal: %v", err)
	}
	if err := isolation.WriteExtra(evidenceDir(), "lab-AT13-d147-detail.json", string(pretty)+"\n"); err != nil {
		fail(t, &rec, "write lab-AT13-d147-detail.json: %v", err)
	}
	if !report.Passed || len(report.Rows) != 4 {
		fail(t, &rec, "d147 report did not pass all 4 rows: passed=%v rows=%d", report.Passed, len(report.Rows))
	}

	rec.Observations = append(rec.Observations, isolation.Observation{
		Name:   "forward_view",
		Detail: fmt.Sprintf("generation=%d lab_member_accept=%d mss_clamp=%d pmtu_accept=%d", fview.Generation, fview.LabMemberAccept, fview.MSSClamp, fview.PMTUAccept),
	})

	var summaryParts []string
	for _, row := range report.Rows {
		if !row.Passed || len(row.Subcases) != 4 {
			fail(t, &rec, "d147 row %s (%s) failed", row.ID, row.SourceIP)
		}
		pos := row.Subcases[0]
		te := row.Subcases[1]
		unsol := row.Subcases[2]
		up := row.Subcases[3]
		if pos.UsageReconciliation == nil || pos.Transfer == nil || pos.Transfer.PMTU == nil {
			fail(t, &rec, "d147 row %s missing positive usage or transfer detail", row.ID)
		}
		uDelta := pos.Counters["u_"+report.DeviceID]
		dDelta := pos.Counters["d_"+report.DeviceID]
		ur := pos.UsageReconciliation
		detail := fmt.Sprintf(
			"src=%s sets=%s pmtu=%d upload_bytes=%d u_%s=%dp/%dB (tc %s %dp/%dB drops=%d) d_%s=%dp/%dB [tcp=%dB + frag_needed=%dp/%dB] (tc %s %dp/%dB drops=%d); negatives: time_exceeded->ctdenyd=%dp/%dB (d=0), unsolicited_frag_needed->ctinv=%dp/%dB (d=0), member_upload->%s=%dp/%dB (u=0)",
			row.SourceIP,
			strings.Join(row.SetMembership.MatchedSets, "+"),
			*pos.Transfer.PMTU,
			report.UploadBytes,
			report.DeviceID,
			uDelta.Packets,
			uDelta.Bytes,
			ur.AssignedTCClass,
			ur.TCUploadLeaf.Packets,
			ur.TCUploadLeaf.Bytes,
			ur.TCUploadLeaf.Drops,
			report.DeviceID,
			dDelta.Packets,
			dDelta.Bytes,
			ur.DownloadTCPIPBytes,
			ur.DownloadFragNeededICMPPackets,
			ur.DownloadFragNeededICMPIPBytes,
			ur.AssignedTCClass,
			ur.TCDownloadLeaf.Packets,
			ur.TCDownloadLeaf.Bytes,
			ur.TCDownloadLeaf.Drops,
			te.Counters["ctdenyd"].Packets,
			te.Counters["ctdenyd"].Bytes,
			unsol.Counters["ctinv"].Packets,
			unsol.Counters["ctinv"].Bytes,
			up.DroppedAtCounter,
			up.Counters[up.DroppedAtCounter].Packets,
			up.Counters[up.DroppedAtCounter].Bytes,
		)
		rec.Observations = append(rec.Observations, isolation.Observation{
			Name:   "d147_" + row.ID,
			Detail: detail,
		})
		totalAdmissionBytes := uDelta.Bytes + dDelta.Bytes
		rec.Checks = append(rec.Checks, isolation.Check{
			Name:   "D-147 " + row.ID + " (" + row.SourceIP + ")",
			Status: isolation.StatusPass,
			At:     time.Now().UTC().Format(time.RFC3339),
			Actual: detail,
			Positive: isolation.Positive{
				DeviceID:  devA1,
				Container: report.MemberNS,
				At:        time.Now().UTC().Format(time.RFC3339),
				Counted:   totalAdmissionBytes > 0,
				Before:    "0",
				After:     strconv.Itoa(totalAdmissionBytes),
				Observations: []isolation.Observation{
					{Name: "sets", Detail: strings.Join(row.SetMembership.MatchedSets, ",")},
					{Name: "frag_needed_packets", Detail: strconv.Itoa(ur.DownloadFragNeededICMPPackets)},
					{Name: "frag_needed_ip_bytes", Detail: strconv.Itoa(ur.DownloadFragNeededICMPIPBytes)},
					{Name: "priority", Detail: ur.AssignedPriority + " (" + ur.AssignedTCClass + ")"},
				},
			},
		})
		summaryParts = append(summaryParts, fmt.Sprintf("%s(%s)", row.ID, row.SourceIP))
	}
	rec.Observations = append(rec.Observations, isolation.Observation{
		Name:   "scope",
		Detail: "mechanism lab; D-147 per-source-set and overlap verification; product status is AT13.json",
	})
	rec.Status = isolation.StatusPass
	rec.Actual = "Verified D-147 on generation 4200 across all 4 source-set rows (" + strings.Join(summaryParts, ", ") + ") with genuine router frag-needed (MTU 1000) counted once in d_<device> at 1:2f matching independent captures, and all 3 negatives dropped per row."
}
