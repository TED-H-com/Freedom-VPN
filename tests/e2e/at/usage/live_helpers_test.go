//go:build e2e

package usage

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"

	_ "modernc.org/sqlite"
)

const flowtableBypassBlocker = "w1-netlab / w3-devstack-e2e: PR #220 (2473acf, node/enforce/verify_foreign.go) added whole-ruleset foreign-chain verification to VerifyServing; in deploy/devstack, Docker Engine installs table ip nat (POSTROUTING -> DOCKER_POSTROUTING -> xt target SNAT for 127.0.0.11 loopback DNS) inside node-netns and node2-netns, causing VerifyServing to fail with FLOWTABLE_BYPASS (\"table nat family ip hook postrouting priority 100 chain POSTROUTING statement xt\") and roll back table inet fvpn to fvpn_generation=0 whenever d.Serve=true."

type d067CycleMeasurement struct {
	Cycle                            string            `json:"cycle"`
	NodeID                           uuid.UUID         `json:"node_id"`
	IncarnationID                    uuid.UUID         `json:"incarnation_id"`
	OldAppliedGeneration             uint64            `json:"old_applied_generation"`
	TargetDesiredGeneration          uint64            `json:"target_desired_generation"`
	ProvisionalLeaseID               uuid.UUID         `json:"provisional_lease_id"`
	ProvisionalLeaseGeneration       uint64            `json:"provisional_lease_generation"`
	ProvisionalLeaseIssuedAt         string            `json:"provisional_lease_issued_at"`
	ProvisionalLeaseExpiresAt        string            `json:"provisional_lease_expires_at"`
	ProvisionalLeaseTTLSeconds       int64             `json:"provisional_lease_ttl_seconds"`
	ProvisionalLeaseMemberCount      int               `json:"provisional_lease_member_count"`
	ReadyObservationReportID         uuid.UUID         `json:"ready_observation_report_id"`
	ReadyObservationReceivedAt       string            `json:"ready_observation_received_at"`
	ForwardingBlockReasons           []string          `json:"forwarding_block_reasons"`
	CheckErrorCodes                  map[string]string `json:"check_error_codes"`
	Errors                           []api.NodeError   `json:"errors"`
	LeaseExpiredReportedDuringWindow bool              `json:"lease_expired_reported_during_window"`
	PopulatedLeaseID                 uuid.UUID         `json:"populated_lease_id"`
	PopulatedLeaseGeneration         uint64            `json:"populated_lease_generation"`
	PopulatedLeaseIssuedAt           string            `json:"populated_lease_issued_at"`
	PopulatedLeaseExpiresAt          string            `json:"populated_lease_expires_at"`
	PopulatedLeaseTTLSeconds         int64             `json:"populated_lease_ttl_seconds"`
	PopulatedLeaseMemberCount        int               `json:"populated_lease_member_count"`
	PopulatedLeaseInForceAt          string            `json:"populated_lease_in_force_at"`
	FirstCountedByteAt               string            `json:"first_counted_byte_at"`
	ReadyToPopulatedLeaseMs          int64             `json:"ready_to_populated_lease_ms"`
	ReadyToFirstCountedByteMs        int64             `json:"ready_to_first_counted_byte_ms"`
	MaxAllowedWindowMs               int64             `json:"max_allowed_window_ms"`
	FirstCountedTransfer             transferResult    `json:"first_counted_transfer"`
	ApplyBlockerObservationReportID  uuid.UUID         `json:"apply_blocker_observation_report_id,omitempty"`
	ApplyBlockerObservationAt        string            `json:"apply_blocker_observation_at,omitempty"`
	ApplyBlockerErrorCode            string            `json:"apply_blocker_error_code,omitempty"`
	ApplyBlockerDetail               string            `json:"apply_blocker_detail,omitempty"`
}

type fupTransitionMeasurement struct {
	Transition                     string            `json:"transition"`
	Trigger                        string            `json:"trigger"`
	StartedAt                      string            `json:"started_at"`
	FUPMemberGenerationBefore      uint64            `json:"fup_member_generation_before"`
	FUPMemberGenerationAfter       uint64            `json:"fup_member_generation_after"`
	EnforcedTierIndex              int               `json:"enforced_tier_index"`
	EnforcedDownBPS                int64             `json:"enforced_down_bps"`
	EnforcedUpBPS                  int64             `json:"enforced_up_bps"`
	DesiredGenBefore               map[string]uint64 `json:"desired_gen_before"`
	DesiredGenAfter                map[string]uint64 `json:"desired_gen_after"`
	DesiredGenAfterReplay          map[string]uint64 `json:"desired_gen_after_replay"`
	FUPMemberGenerationAfterReplay uint64            `json:"fup_member_generation_after_replay"`
	ApplyingObservedTrue           bool              `json:"applying_observed_true"`
	ApplyingClearedToFalse         bool              `json:"applying_cleared_to_false"`
	ApplyingClearedAt              string            `json:"applying_cleared_at"`
	WallClockMsToApplyingFalse     int64             `json:"wall_clock_ms_to_applying_false"`
	ClearingObservationReportID    uuid.UUID         `json:"clearing_observation_report_id"`
	ClearingObservationNodeID      uuid.UUID         `json:"clearing_observation_node_id"`
	ClearingObservationGeneration  uint64            `json:"clearing_observation_generation"`
	ClearingObservationReceivedAt  string            `json:"clearing_observation_received_at"`
	Node1CompiledMemberRate        string            `json:"node1_compiled_member_rate"`
	Node2CompiledMemberRate        string            `json:"node2_compiled_member_rate"`
	Node1TCClassRate               string            `json:"node1_tc_class_rate"`
	Node2TCClassRate               string            `json:"node2_tc_class_rate"`
	BothExitsCompiledTier2Rate     bool              `json:"both_exits_compiled_tier2_rate"`
	BothExitsInstalledTier2TC      bool              `json:"both_exits_installed_tier2_tc"`
}

type incarnationRetentionMeasurement struct {
	NodeID                         uuid.UUID               `json:"node_id"`
	OldIncarnationID               uuid.UUID               `json:"old_incarnation_id"`
	NewIncarnationID               uuid.UUID               `json:"new_incarnation_id"`
	ActiveReenrollRejected409Code  string                  `json:"active_reenroll_rejected_409_code"`
	IncarnationsBeforeRevoke       api.NodeIncarnationList `json:"incarnations_before_revoke"`
	IncarnationsAfterRevoke        api.NodeIncarnationList `json:"incarnations_after_revoke"`
	IncarnationsAfterReinstall     api.NodeIncarnationList `json:"incarnations_after_reinstall"`
	IncarnationsAfterNewTraffic    api.NodeIncarnationList `json:"incarnations_after_new_traffic"`
	MemberTotalsBeforeReinstall    api.UsageTotals         `json:"member_totals_before_reinstall"`
	MemberTotalsAfterReinstall     api.UsageTotals         `json:"member_totals_after_reinstall"`
	MemberTotalsAfterNewTraffic    api.UsageTotals         `json:"member_totals_after_new_traffic"`
	DBPeriodTotalsBeforeReinstall  map[string]string       `json:"db_period_totals_before_reinstall"`
	DBPeriodTotalsAfterReinstall   map[string]string       `json:"db_period_totals_after_reinstall"`
	DBPeriodTotalsAfterNewTraffic  map[string]string       `json:"db_period_totals_after_new_traffic"`
	NewIncarnationCountedTransfer  transferResult          `json:"new_incarnation_counted_transfer"`
	OldIncarnationUsageFixed       bool                    `json:"old_incarnation_usage_fixed"`
	MemberTotalUnchangedOnReenroll bool                    `json:"member_total_unchanged_on_reenroll"`
	NewTrafficAccumulatedSameMonth bool                    `json:"new_traffic_accumulated_same_month"`
}

type sqliteUsageSnapshot struct {
	NextSeq         uint64   `json:"next_seq"`
	OpenEpoch       string   `json:"open_epoch"`
	OpenReason      string   `json:"open_reason"`
	LastCloseReason string   `json:"last_close_reason"`
	LedgerEpoch     string   `json:"ledger_epoch"`
	AckedThrough    uint64   `json:"acked_through"`
	GateStopped     int      `json:"gate_stopped"`
	GateReason      string   `json:"gate_reason"`
	EventsCount     int      `json:"events_count"`
	OutboxCount     int      `json:"outbox_count"`
	ArchivedCount   int      `json:"archived_count"`
	GapsCount       int      `json:"gaps_count"`
	GapReasons      []string `json:"gap_reasons,omitempty"`
	CountersCount   int      `json:"counters_count"`
	DeviceUpBytes   string   `json:"device_up_bytes,omitempty"`
	DeviceDownBytes string   `json:"device_down_bytes,omitempty"`
}

type transferResult struct {
	MemberService  string     `json:"member_service"`
	NodeService    string     `json:"node_service"`
	ProbeID        string     `json:"probe_id"`
	HTTP200        bool       `json:"http_200"`
	OriginAuth     bool       `json:"origin_authenticated"`
	OriginSource   string     `json:"origin_source"`
	ExpectedSNAT   string     `json:"expected_snat"`
	Requests       int        `json:"requests"`
	Generation     decstr.U64 `json:"generation"`
	BeforeUp       decstr.U64 `json:"before_upload_bytes"`
	BeforeDown     decstr.U64 `json:"before_download_bytes"`
	AfterUp        decstr.U64 `json:"after_upload_bytes"`
	AfterDown      decstr.U64 `json:"after_download_bytes"`
	DeltaUp        decstr.U64 `json:"delta_upload_bytes"`
	DeltaDown      decstr.U64 `json:"delta_download_bytes"`
	MemberTunnelTx uint64     `json:"member_fvnb0_tx_bytes_delta"`
	MemberTunnelRx uint64     `json:"member_fvnb0_rx_bytes_delta"`
	UplinkL3Tx     uint64     `json:"node_fvup0_l3_tx_bytes_delta"`
	UplinkL3Rx     uint64     `json:"node_fvup0_l3_rx_bytes_delta"`
	Counted        bool       `json:"counted"`
}

func (s *liveSuite) adminHook() e2e.BrowserSession {
	return e2e.BrowserSession{Client: s.admin.client, CSRF: s.admin.csrf, MemberID: s.admin.memberID}
}

func (s *liveSuite) memberAHook() e2e.BrowserSession {
	return e2e.BrowserSession{Client: s.memberA.client, CSRF: s.memberA.csrf, MemberID: s.memberA.memberID}
}

func (s *liveSuite) memberBHook() e2e.BrowserSession {
	return e2e.BrowserSession{Client: s.memberB.client, CSRF: s.memberB.csrf, MemberID: s.memberB.memberID}
}

func (s *liveSuite) buildNodeMTLSClient(nodeService string) *http.Client {
	crtPEM, codeCrt, errCrt := s.execInContainer(nodeService, "cat", "/var/lib/vpn-node/identity/cert.pem")
	keyPEM, codeKey, errKey := s.execInContainer(nodeService, "cat", "/var/lib/vpn-node/identity/key.pem")
	if errCrt != nil || codeCrt != 0 || errKey != nil || codeKey != 0 {
		s.t.Fatalf("read %s mTLS identity failed: crt=%v key=%v", nodeService, errCrt, errKey)
	}
	s.extraRedactions = append(s.extraRedactions, string(keyPEM))
	nodeCert, err := tls.X509KeyPair(crtPEM, keyPEM)
	if err != nil {
		s.t.Fatalf("parse %s X509KeyPair: %v", nodeService, err)
	}
	tr := s.transport.Clone()
	tr.TLSClientConfig = &tls.Config{
		RootCAs:      s.roots,
		Certificates: []tls.Certificate{nodeCert},
		MinVersion:   tls.VersionTLS12,
	}
	s.t.Cleanup(tr.CloseIdleConnections)
	return &http.Client{Transport: tr, Timeout: 20 * time.Second}
}

func (s *liveSuite) originAndSNAT(nodeService string) (string, string) {
	raw, code, err := s.execInContainer(nodeService, "cat", "/etc/vpn-node/agent-run.json")
	if err != nil || code != 0 {
		return "", ""
	}
	var cfg struct {
		Datapath struct {
			SNAT string `json:"snat"`
		} `json:"datapath"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return "", ""
	}
	i := strings.LastIndex(cfg.Datapath.SNAT, ".")
	if i < 0 {
		return "", ""
	}
	return cfg.Datapath.SNAT[:i] + ".10", cfg.Datapath.SNAT
}

func exitNetworkID(exitID uuid.UUID) string {
	sum := sha256.Sum256([]byte(exitID.String()))
	return "fvpn-exit-" + hex.EncodeToString(sum[:12])
}

func (s *liveSuite) deviceCounters(nodeService string, deviceID uuid.UUID) (up uint64, down uint64, gen uint64, all map[string]uint64, ok bool) {
	raw, code, err := s.execInContainer(nodeService, "nft", "-j", "list", "table", "inet", "fvpn")
	if err != nil || code != 0 {
		return 0, 0, 0, nil, false
	}
	st, err := enforce.ParseState(raw)
	if err != nil {
		return 0, 0, 0, nil, false
	}
	all = make(map[string]uint64, len(st.Counters))
	for k, v := range st.Counters {
		all[k] = v.Bytes
	}
	key := strings.ReplaceAll(deviceID.String(), "-", "")
	u, okU := st.Counters["u_"+key]
	d, okD := st.Counters["d_"+key]
	return u.Bytes, d.Bytes, st.Generation, all, okU && okD
}

func (s *liveSuite) ifaceBytes(service, iface string) (rx uint64, tx uint64) {
	out, code, err := s.execInContainer(service, "sh", "-c", fmt.Sprintf("cat /sys/class/net/%s/statistics/rx_bytes /sys/class/net/%s/statistics/tx_bytes", iface, iface))
	if err != nil || code != 0 {
		return 0, 0
	}
	_, _ = fmt.Sscanf(strings.TrimSpace(string(out)), "%d\n%d", &rx, &tx)
	return rx, tx
}

func (s *liveSuite) ifaceL3Bytes(service, iface string) (rxL3 uint64, txL3 uint64) {
	out, code, err := s.execInContainer(service, "sh", "-c", fmt.Sprintf("cat /sys/class/net/%s/statistics/rx_bytes /sys/class/net/%s/statistics/rx_packets /sys/class/net/%s/statistics/tx_bytes /sys/class/net/%s/statistics/tx_packets", iface, iface, iface, iface))
	if err != nil || code != 0 {
		return 0, 0
	}
	var rxB, rxP, txB, txP uint64
	_, _ = fmt.Sscanf(strings.TrimSpace(string(out)), "%d\n%d\n%d\n%d", &rxB, &rxP, &txB, &txP)
	if rxB >= rxP*14 {
		rxL3 = rxB - rxP*14
	}
	if txB >= txP*14 {
		txL3 = txB - txP*14
	}
	return rxL3, txL3
}

func (s *liveSuite) selectExitRoute(memberService string, wantNetID string, originIP string) (string, bool) {
	var listOut string
	var selectedID string
	re := regexp.MustCompile(`ID:\s+(fvpn-exit-[a-z0-9]+)\s+Network:\s+0\.0\.0\.0/0`)
	found := s.waitFor(10*time.Second, func() bool {
		raw, code, err := s.execInContainer(memberService, "netbird", "networks", "list")
		if err != nil || code != 0 {
			return false
		}
		listOut = strings.TrimSpace(string(raw))
		if wantNetID != "" {
			if strings.Contains(listOut, wantNetID) {
				selectedID = wantNetID
				return true
			}
			return false
		}
		matches := re.FindAllStringSubmatch(listOut, -1)
		if len(matches) >= 1 {
			selectedID = matches[0][1]
			return true
		}
		return false
	})
	if !found || selectedID == "" {
		return listOut, false
	}
	_, _, _ = s.execInContainer(memberService, "netbird", "networks", "deselect", "all")
	selOut, selCode, selErr := s.execInContainer(memberService, "netbird", "networks", "select", selectedID)
	if selErr != nil || selCode != 0 {
		return string(selOut), false
	}
	routed := s.waitFor(12*time.Second, func() bool {
		rt, code, err := s.execInContainer(memberService, "ip", "route", "get", originIP)
		return err == nil && code == 0 && strings.Contains(string(rt), "dev fvnb0")
	})
	afterOut, _, _ := s.execInContainer(memberService, "netbird", "networks", "list")
	return strings.TrimSpace(string(afterOut)), routed
}

func (s *liveSuite) originProbe(method, probeID string) (map[string]any, bool) {
	secret := s.readSecret("origin.secret")
	s.extraRedactions = append(s.extraRedactions, secret)
	raw, _ := json.Marshal(map[string]string{"id": probeID})
	req, err := http.NewRequestWithContext(s.ctx, method, "https://releases.example.invalid/e2e/traffic", bytes.NewReader(raw))
	if err != nil {
		return nil, false
	}
	req.Header.Set("X-E2E-Secret", secret)
	client := &http.Client{Transport: s.transport, Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if method == http.MethodPut {
		return nil, resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK
	}
	var out map[string]any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&out); err != nil || resp.StatusCode != http.StatusOK {
		return nil, false
	}
	return out, true
}

func (s *liveSuite) runCountedTransfer(memberService, nodeService string, deviceID uuid.UUID, count int) transferResult {
	originIP, snat := s.originAndSNAT(nodeService)
	probeID := uuid.NewString()
	_, armed := s.originProbe(http.MethodPut, probeID)
	beforeUp, beforeDown, genBefore, _, okBefore := s.deviceCounters(nodeService, deviceID)
	rxBefore, txBefore := s.ifaceBytes(memberService, "fvnb0")
	upL3RxBefore, upL3TxBefore := s.ifaceL3Bytes(nodeService, "fvup0")

	all200 := armed
	if count <= 0 {
		count = 1
	}
	for i := 0; i < count; i++ {
		out, code, err := s.execInContainer(memberService, "wget", "--server-response", "--timeout=2", "--tries=1",
			"--ca-certificate=/etc/vpn-node/ca.pem", "--header=X-E2E-Probe: "+probeID, "-O", "-", "https://"+originIP+"/healthz")
		if err != nil || code != 0 || !bytes.Contains(out, []byte("200 OK")) {
			all200 = false
		}
	}

	afterUp, afterDown, genAfter, _, okAfter := s.deviceCounters(nodeService, deviceID)
	rxAfter, txAfter := s.ifaceBytes(memberService, "fvnb0")
	upL3RxAfter, upL3TxAfter := s.ifaceL3Bytes(nodeService, "fvup0")
	arrival, okOrigin := s.originProbe(http.MethodGet, probeID)

	src, _ := arrival["source_address"].(string)
	reqs := 0
	if rf, ok := arrival["requests"].(float64); ok {
		reqs = int(rf)
	}
	var deltaUp, deltaDown uint64
	if afterUp >= beforeUp {
		deltaUp = afterUp - beforeUp
	}
	if afterDown >= beforeDown {
		deltaDown = afterDown - beforeDown
	}
	var deltaTx, deltaRx uint64
	if txAfter >= txBefore {
		deltaTx = txAfter - txBefore
	}
	if rxAfter >= rxBefore {
		deltaRx = rxAfter - rxBefore
	}
	var deltaUpL3Tx, deltaUpL3Rx uint64
	if upL3TxAfter >= upL3TxBefore {
		deltaUpL3Tx = upL3TxAfter - upL3TxBefore
	}
	if upL3RxAfter >= upL3RxBefore {
		deltaUpL3Rx = upL3RxAfter - upL3RxBefore
	}
	gen := genAfter
	if gen == 0 {
		gen = genBefore
	}
	counted := all200 && okOrigin && arrival["probe_id"] == probeID && src == snat && reqs >= count && okBefore && okAfter && deltaUp > 0 && deltaDown > 0
	return transferResult{
		MemberService:  memberService,
		NodeService:    nodeService,
		ProbeID:        probeID,
		HTTP200:        all200,
		OriginAuth:     okOrigin,
		OriginSource:   src,
		ExpectedSNAT:   snat,
		Requests:       reqs,
		Generation:     decstr.U64(gen),
		BeforeUp:       decstr.U64(beforeUp),
		BeforeDown:     decstr.U64(beforeDown),
		AfterUp:        decstr.U64(afterUp),
		AfterDown:      decstr.U64(afterDown),
		DeltaUp:        decstr.U64(deltaUp),
		DeltaDown:      decstr.U64(deltaDown),
		MemberTunnelTx: deltaTx,
		MemberTunnelRx: deltaRx,
		UplinkL3Tx:     deltaUpL3Tx,
		UplinkL3Rx:     deltaUpL3Rx,
		Counted:        counted,
	}
}

func (s *liveSuite) inspectNodeUsageSQLite(nodeService string, deviceID uuid.UUID) sqliteUsageSnapshot {
	tmp := s.t.TempDir()
	_, err := s.runHostCmd("docker", "cp", s.containerName(nodeService)+":/var/lib/vpn-node/state/.", tmp)
	if err != nil {
		return sqliteUsageSnapshot{}
	}
	dbPath := filepath.Join(tmp, "usage.db")
	if _, err := os.Stat(dbPath); err != nil {
		return sqliteUsageSnapshot{}
	}
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return sqliteUsageSnapshot{}
	}
	defer db.Close()

	var snap sqliteUsageSnapshot
	var openEpoch, ledgerEpoch sql.NullString
	_ = db.QueryRowContext(s.ctx, `
		SELECT next_seq, open_epoch, open_reason, last_close_reason, ledger_epoch, acked_through, gate_stopped, gate_reason
		FROM meta WHERE id = 1
	`).Scan(&snap.NextSeq, &openEpoch, &snap.OpenReason, &snap.LastCloseReason, &ledgerEpoch, &snap.AckedThrough, &snap.GateStopped, &snap.GateReason)
	snap.OpenEpoch = openEpoch.String
	snap.LedgerEpoch = ledgerEpoch.String

	_ = db.QueryRowContext(s.ctx, `SELECT count(*) FROM events`).Scan(&snap.EventsCount)
	_ = db.QueryRowContext(s.ctx, `SELECT count(*) FROM events WHERE state = 'outbox'`).Scan(&snap.OutboxCount)
	_ = db.QueryRowContext(s.ctx, `SELECT count(*) FROM events WHERE state = 'archived'`).Scan(&snap.ArchivedCount)
	_ = db.QueryRowContext(s.ctx, `SELECT count(*) FROM gaps`).Scan(&snap.GapsCount)
	_ = db.QueryRowContext(s.ctx, `SELECT count(*) FROM counters`).Scan(&snap.CountersCount)

	rows, err := db.QueryContext(s.ctx, `SELECT reason FROM gaps ORDER BY id ASC`)
	if err == nil {
		for rows.Next() {
			var r string
			if rows.Scan(&r) == nil {
				snap.GapReasons = append(snap.GapReasons, r)
			}
		}
		rows.Close()
	}

	if deviceID != uuid.Nil && snap.OpenEpoch != "" {
		_ = db.QueryRowContext(s.ctx, `SELECT bytes FROM counters WHERE epoch_id = ? AND device_id = ? AND direction = 'u'`, snap.OpenEpoch, deviceID.String()).Scan(&snap.DeviceUpBytes)
		_ = db.QueryRowContext(s.ctx, `SELECT bytes FROM counters WHERE epoch_id = ? AND device_id = ? AND direction = 'd'`, snap.OpenEpoch, deviceID.String()).Scan(&snap.DeviceDownBytes)
	}
	return snap
}

func (s *liveSuite) stopNodeAgent(nodeService string) {
	_, _, _ = s.execInContainer(nodeService, "touch", "/var/lib/vpn-node/e2e/pause-agent")
	_ = s.waitFor(12*time.Second, func() bool {
		_, code, _ := s.execInContainer(nodeService, "test", "-s", "/var/lib/vpn-node/e2e/agent.pid")
		return code != 0
	})
	_, _, _ = s.execInContainer(nodeService, "rm", "-f", "/run/vpn-node/heartbeat")
}

func (s *liveSuite) startNodeAgent(nodeService string) {
	_, _, _ = s.execInContainer(nodeService, "rm", "-f", "/var/lib/vpn-node/e2e/pause-agent")
	_ = s.waitFor(12*time.Second, func() bool {
		_, code, _ := s.execInContainer(nodeService, "test", "-s", "/var/lib/vpn-node/e2e/agent.pid")
		return code == 0
	})
}

func (s *liveSuite) buildAndInstallUsageAdmin() {
	binPath := filepath.Join(s.root, ".agent/e2e-build/fvpn-usage-admin")
	cmd := exec.CommandContext(s.ctx, "go", "build", "-trimpath", "-o", binPath, "./internal/usage/cmd/fvpn-usage-admin")
	cmd.Dir = s.root
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "GOTOOLCHAIN=local")
	if out, err := cmd.CombinedOutput(); err != nil {
		s.t.Fatalf("build fvpn-usage-admin: %v (%s)", err, string(out))
	}
	if out, err := s.runHostCmd("docker", "cp", binPath, s.containerName("control")+":/tmp/fvpn-usage-admin"); err != nil {
		s.t.Fatalf("docker cp fvpn-usage-admin: %v (%s)", err, string(out))
	}
}

func (s *liveSuite) runUsageAdmin(args ...string) ([]byte, int, error) {
	fullArgs := append([]string{"/tmp/fvpn-usage-admin", "-config", "/etc/fvpn/central.json"}, args...)
	return s.execInContainer("control", fullArgs...)
}

func (s *liveSuite) ackNodeGeneration(nodeID, incID uuid.UUID, nodeClient *http.Client, targetGen uint64) (uuid.UUID, uint64, time.Time, api.DesiredState) {
	dsResp := s.agentCallWithClient(nodeClient, http.MethodGet, "/api/v1/agent/desired-state", nil, nil)
	if dsResp.status != http.StatusOK {
		s.t.Fatalf("ackNodeGeneration GET /desired-state for %s failed: HTTP %d %s", nodeID, dsResp.status, string(dsResp.body))
	}
	verifiedBytes, _, err := envelope.Verify(dsResp.body, s.nodeTrust, envelope.PurposeDesiredState, time.Now().UTC())
	if err != nil {
		s.t.Fatalf("ackNodeGeneration Verify DesiredState for %s: %v", nodeID, err)
	}
	var ds api.DesiredState
	if err := json.Unmarshal(verifiedBytes, &ds); err != nil {
		s.t.Fatalf("ackNodeGeneration unmarshal DesiredState for %s: %v", nodeID, err)
	}

	var lastObsAt time.Time
	var templateRaw []byte
	if err := s.pool.QueryRow(s.ctx, `
		SELECT COALESCE(n.last_observed_at, '1970-01-01T00:00:00Z'::timestamptz),
		       (SELECT report_payload FROM node_observations WHERE node_id = n.id AND incarnation_id = $2 AND safety_status = 'passed' ORDER BY received_at DESC LIMIT 1)
		FROM nodes n
		WHERE n.id = $1
	`, nodeID, incID).Scan(&lastObsAt, &templateRaw); err != nil || len(templateRaw) == 0 {
		s.t.Fatalf("ackNodeGeneration query template observation for %s: err=%v len=%d", nodeID, err, len(templateRaw))
	}

	var rep api.ObservationReport
	if err := json.Unmarshal(templateRaw, &rep); err != nil {
		s.t.Fatalf("ackNodeGeneration unmarshal template report: %v", err)
	}
	rep.ReportId = uuid.New()
	rep.NodeId = nodeID
	rep.NodeIncarnationId = incID
	rep.SafetyStatus = api.SafetyStatusPassed
	rep.DataHealth = api.DataHealthHealthy
	rep.ProvisioningState = api.ProvisioningStateReady
	appliedGen := uint64(ds.ConfigGeneration)
	if targetGen > appliedGen {
		appliedGen = targetGen
	}
	rep.AppliedConfigGeneration.Set(decstr.U64(appliedGen))
	rep.AppliedBindingRevision.Set(ds.BindingRevision)
	rep.AppliedPolicyRefs.Set(ds.PolicyRefs)

	var obsResp httpResponse
	var obsAt time.Time
	for attempt := 0; attempt < 3; attempt++ {
		obsAt = time.Now().UTC()
		if !obsAt.After(lastObsAt) {
			obsAt = lastObsAt.UTC().Add(10 * time.Millisecond)
			time.Sleep(15 * time.Millisecond)
		}
		rep.ReportId = uuid.New()
		rep.ObservedAt = obsAt
		for i := range rep.Checks {
			rep.Checks[i].ObservedAt = obsAt
		}
		lastObsAt = obsAt
		obsResp = s.agentCallWithClient(nodeClient, http.MethodPost, "/api/v1/agent/observations", rep, nil)
		if obsResp.status == http.StatusAccepted {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if obsResp.status != http.StatusAccepted {
		s.t.Fatalf("ackNodeGeneration POST /observations for %s failed: HTTP %d %s", nodeID, obsResp.status, string(obsResp.body))
	}
	var receivedAt time.Time
	_ = s.pool.QueryRow(s.ctx, `SELECT received_at FROM node_observations WHERE report_id = $1`, rep.ReportId).Scan(&receivedAt)
	if receivedAt.IsZero() {
		receivedAt = obsAt
	}
	return rep.ReportId, appliedGen, receivedAt, ds
}

func (s *liveSuite) ingestNodeUsageEvent(
	nodeID, incID uuid.UUID,
	nodeClient *http.Client,
	memberID, deviceID uuid.UUID,
	uploadBytes, downloadBytes uint64,
) (api.UsageEvent, api.UsageBatchResult, uuid.UUID) {
	cpResp := s.agentCallWithClient(nodeClient, http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)
	if cpResp.status != http.StatusOK {
		s.t.Fatalf("ingestNodeUsageEvent GET /usage-checkpoint for %s failed: HTTP %d %s", nodeID, cpResp.status, string(cpResp.body))
	}
	verifiedCP, _, err := envelope.Verify(cpResp.body, s.nodeTrust, envelope.PurposeRecovery, time.Now().UTC())
	if err != nil {
		s.t.Fatalf("ingestNodeUsageEvent Verify UsageCheckpoint for %s: %v", nodeID, err)
	}
	var cp api.UsageCheckpoint
	if err := json.Unmarshal(verifiedCP, &cp); err != nil {
		s.t.Fatalf("ingestNodeUsageEvent unmarshal UsageCheckpoint: %v", err)
	}
	nextSeq := uint64(cp.MaxSeenSeq) + 1

	var bindingRev uint64 = 1
	var validFrom time.Time
	_ = s.pool.QueryRow(s.ctx, `
		SELECT binding_revision::bigint, valid_from
		FROM device_bindings
		WHERE device_id = $1 AND member_id = $2
		ORDER BY binding_revision DESC
		LIMIT 1
	`, deviceID, memberID).Scan(&bindingRev, &validFrom)

	nowUTC := time.Now().UTC().Truncate(time.Second)
	iStart := nowUTC.Add(-4 * time.Second)
	if !validFrom.IsZero() && iStart.Before(validFrom.UTC().Add(1*time.Second)) {
		iStart = validFrom.UTC().Truncate(time.Second).Add(1 * time.Second)
	}
	iEnd := iStart.Add(2 * time.Second)
	if iEnd.After(nowUTC) {
		time.Sleep(2 * time.Second)
		nowUTC = time.Now().UTC().Truncate(time.Second)
		iEnd = nowUTC
		iStart = iEnd.Add(-1 * time.Second)
	}
	curPeriod := fmt.Sprintf("%04d-%02d", iStart.Year(), int(iStart.Month()))
	epochUUID := uuid.Must(uuid.NewV7())

	ev := makeSignedUsageEvent(s.t, nodeID, incID, epochUUID, nextSeq, iStart, iEnd, api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        memberID,
			DeviceId:        deviceID,
			BindingRevision: decstr.U64(bindingRev),
			PeriodId:        curPeriod,
			UploadBytes:     decstr.U64(uploadBytes),
			DownloadBytes:   decstr.U64(downloadBytes),
		},
	})
	resp := s.agentCallWithClient(nodeClient, http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		LedgerEpoch:       presence.NewNullableWithValue(cp.LedgerEpoch),
		Events:            []api.UsageEvent{ev},
	}, idemHeaders())
	var ack api.UsageBatchResult
	_ = json.Unmarshal(resp.body, &ack)
	if resp.status != http.StatusOK || len(ack.Results) != 1 || ack.Results[0].Status != api.UsageEventResultStatusAccepted {
		s.t.Fatalf("ingestNodeUsageEvent POST /usage-batches for %s (seq %d) failed: HTTP %d %s", nodeID, nextSeq, resp.status, string(resp.body))
	}
	return ev, ack, cp.LedgerEpoch
}

func (s *liveSuite) measureD067ProvisionalWindow(
	cycleName string,
	nodeService string,
	nodeID uuid.UUID,
	incarnationID uuid.UUID,
	nodeClient *http.Client,
	advanceDesired func(),
	needSelectRoute bool,
) d067CycleMeasurement {
	s.stopNodeAgent(nodeService)

	var oldDesiredGen, oldAppliedGen uint64
	var lastObsAt time.Time
	var lastPayloadRaw []byte
	if err := s.pool.QueryRow(s.ctx, `
		SELECT n.desired_config_generation::bigint,
		       n.applied_config_generation::bigint,
		       COALESCE(n.last_observed_at, '1970-01-01T00:00:00Z'::timestamptz),
		       (SELECT report_payload FROM node_observations WHERE node_id = n.id AND incarnation_id = $2 AND safety_status = 'passed' ORDER BY received_at DESC LIMIT 1)
		FROM nodes n
		WHERE n.id = $1
	`, nodeID, incarnationID).Scan(&oldDesiredGen, &oldAppliedGen, &lastObsAt, &lastPayloadRaw); err != nil || len(lastPayloadRaw) == 0 {
		s.t.Fatalf("D-067 %s: query baseline observation for %s failed: err=%v payloadLen=%d", cycleName, nodeID, err, len(lastPayloadRaw))
	}

	advanceDesired()

	var targetDesiredGen uint64
	if !s.waitFor(45*time.Second, func() bool {
		err := s.pool.QueryRow(s.ctx, `SELECT desired_config_generation::bigint FROM nodes WHERE id = $1`, nodeID).Scan(&targetDesiredGen)
		return err == nil && targetDesiredGen > oldDesiredGen
	}) {
		s.t.Fatalf("D-067 %s: desired_config_generation did not advance beyond %d (got %d)", cycleName, oldDesiredGen, targetDesiredGen)
	}

	// Ensure nodes.provisioning_state is 'applying' and safety_status is 'passed' before starting fvpn-agent
	var provState, safetyState string
	_ = s.pool.QueryRow(s.ctx, `SELECT provisioning_state, safety_status FROM nodes WHERE id = $1`, nodeID).Scan(&provState, &safetyState)
	for attempt := 0; attempt < 3 && (provState != "applying" || safetyState != "passed"); attempt++ {
		var staleRep api.ObservationReport
		if err := json.Unmarshal(lastPayloadRaw, &staleRep); err != nil {
			s.t.Fatalf("D-067 %s: unmarshal baseline observation report: %v", cycleName, err)
		}
		staleRep.ReportId = uuid.New()
		obsAt := time.Now().UTC()
		if !obsAt.After(lastObsAt) {
			obsAt = lastObsAt.UTC().Add(10 * time.Millisecond)
			time.Sleep(15 * time.Millisecond)
		}
		staleRep.ObservedAt = obsAt
		staleRep.SafetyStatus = api.SafetyStatusPassed
		staleRep.ProvisioningState = api.ProvisioningStateApplying
		staleRep.MemberForwardingEnabled = false
		staleRep.ForwardingBlockReasons = []api.ForwardingBlockReason{api.ForwardingBlockReasonProvisioningIncomplete}
		staleRep.Errors = []api.NodeError{}
		staleRep.AppliedConfigGeneration.Set(decstr.U64(oldDesiredGen))
		for i := range staleRep.Checks {
			staleRep.Checks[i].ObservedAt = obsAt
		}
		lastObsAt = obsAt
		resp := s.agentCallWithClient(nodeClient, http.MethodPost, "/api/v1/agent/observations", staleRep, nil)
		if resp.status != http.StatusAccepted {
			s.t.Fatalf("D-067 %s: stale observation %d rejected: HTTP %d %s", cycleName, attempt, resp.status, string(resp.body))
		}
		time.Sleep(450 * time.Millisecond)
		_ = s.pool.QueryRow(s.ctx, `SELECT provisioning_state, safety_status FROM nodes WHERE id = $1`, nodeID).Scan(&provState, &safetyState)
	}
	if provState != "applying" {
		s.t.Fatalf("D-067 %s: node %s did not enter provisioning_state=applying before agent start (got %q)", cycleName, nodeID, provState)
	}

	var maxLeaseBefore, maxObsBefore time.Time
	_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(MAX(issued_at), '1970-01-01T00:00:00Z'::timestamptz) FROM serving_leases WHERE node_id = $1 AND incarnation_id = $2`, nodeID, incarnationID).Scan(&maxLeaseBefore)
	_ = s.pool.QueryRow(s.ctx, `SELECT COALESCE(MAX(received_at), '1970-01-01T00:00:00Z'::timestamptz) FROM node_observations WHERE node_id = $1 AND incarnation_id = $2`, nodeID, incarnationID).Scan(&maxObsBefore)

	s.startNodeAgent(nodeService)

	// 1. Wait for provisional lease (members: [], 60s expiry) and awaiting_member_grants observation that flips node to ready
	var provLeaseID uuid.UUID
	var provLeaseGen uint64
	var provLeaseIssued, provLeaseExpires time.Time
	var provLeaseMembers int
	var readyObsID uuid.UUID
	var readyObsReceived time.Time
	var readyObsRaw []byte

	gotProvisionalAndReady := s.waitFor(45*time.Second, func() bool {
		err1 := s.pool.QueryRow(s.ctx, `
			SELECT id, policy_generation::bigint, issued_at, expires_at, jsonb_array_length(envelope->'signed'->'members')
			FROM serving_leases
			WHERE node_id = $1 AND incarnation_id = $2 AND issued_at > $3 AND policy_generation >= $4
			  AND jsonb_array_length(envelope->'signed'->'members') = 0
			ORDER BY issued_at ASC
			LIMIT 1
		`, nodeID, incarnationID, maxLeaseBefore, targetDesiredGen).Scan(&provLeaseID, &provLeaseGen, &provLeaseIssued, &provLeaseExpires, &provLeaseMembers)
		if err1 != nil {
			return false
		}
		err2 := s.pool.QueryRow(s.ctx, `
			SELECT report_id, received_at, report_payload
			FROM node_observations
			WHERE node_id = $1 AND incarnation_id = $2 AND received_at > $3 AND applied_generation >= $4
			  AND safety_status = 'passed'
			  AND report_payload->>'provisioning_state' = 'ready'
			  AND report_payload->'forwarding_block_reasons' @> '["awaiting_member_grants"]'::jsonb
			ORDER BY received_at ASC
			LIMIT 1
		`, nodeID, incarnationID, maxObsBefore, targetDesiredGen).Scan(&readyObsID, &readyObsReceived, &readyObsRaw)
		return err2 == nil
	})
	if !gotProvisionalAndReady {
		s.t.Fatalf("D-067 %s: did not observe provisional lease (members:[]) and awaiting_member_grants observation on %s", cycleName, nodeID)
	}

	var readyObs api.ObservationReport
	if err := json.Unmarshal(readyObsRaw, &readyObs); err != nil {
		s.t.Fatalf("D-067 %s: unmarshal ready observation: %v", cycleName, err)
	}
	blockReasons := make([]string, 0, len(readyObs.ForwardingBlockReasons))
	for _, r := range readyObs.ForwardingBlockReasons {
		blockReasons = append(blockReasons, string(r))
	}
	checkCodes := make(map[string]string)
	for _, ch := range readyObs.Checks {
		if ch.Name == api.NodeCheckNameLease || ch.Name == api.NodeCheckNameQos || ch.Name == api.NodeCheckNameSourceBinding {
			if code, err := ch.ErrorCode.Get(); err == nil {
				checkCodes[string(ch.Name)] = string(code)
			} else {
				checkCodes[string(ch.Name)] = string(ch.Status)
			}
		}
	}
	provTTL := int64(provLeaseExpires.Sub(provLeaseIssued).Seconds())
	if provTTL != 60 || provLeaseMembers != 0 || readyObs.MemberForwardingEnabled ||
		checkCodes["lease"] != "AWAITING_MEMBER_GRANTS" ||
		checkCodes["qos"] != "AWAITING_MEMBER_GRANTS" ||
		checkCodes["source_binding"] != "AWAITING_MEMBER_GRANTS" ||
		len(readyObs.Errors) != 1 || readyObs.Errors[0].Stage != "lease" || readyObs.Errors[0].Code != api.ErrorCodeAWAITINGMEMBERGRANTS {
		s.t.Fatalf("D-067 %s: invalid provisional lease/observation evidence: ttl=%ds members=%d checks=%v errors=%+v reasons=%v",
			cycleName, provTTL, provLeaseMembers, checkCodes, readyObs.Errors, blockReasons)
	}

	// 2. Wait for populated lease (members > 0, 86400s expiry) to be issued by central RenewTx within <= 65000ms
	var popLeaseID uuid.UUID
	var popLeaseGen uint64
	var popLeaseIssued, popLeaseExpires, popInForceAt time.Time
	var popLeaseMembers int

	gotPopulatedLease := s.waitFor(70*time.Second, func() bool {
		err := s.pool.QueryRow(s.ctx, `
			SELECT id, policy_generation::bigint, issued_at, expires_at, jsonb_array_length(envelope->'signed'->'members')
			FROM serving_leases
			WHERE node_id = $1 AND incarnation_id = $2 AND issued_at >= $3 AND policy_generation >= $4
			  AND jsonb_array_length(envelope->'signed'->'members') > 0
			ORDER BY issued_at ASC
			LIMIT 1
		`, nodeID, incarnationID, provLeaseIssued, targetDesiredGen).Scan(&popLeaseID, &popLeaseGen, &popLeaseIssued, &popLeaseExpires, &popLeaseMembers)
		return err == nil && popLeaseMembers > 0
	})
	if !gotPopulatedLease {
		s.t.Fatalf("D-067 %s: populated lease was not issued on %s after ready observation %s", cycleName, nodeID, readyObsID)
	}
	popInForceAt = popLeaseIssued

	// 3. Wait for the agent's apply result after receiving the populated lease (either serving=1 + device counters, or FLOWTABLE_BYPASS apply failure)
	var applyBlockerObsID uuid.UUID
	var applyBlockerObsAt time.Time
	var applyBlockerCode, applyBlockerDetail string
	_ = s.waitFor(15*time.Second, func() bool {
		hb, code, _ := s.execInContainer(nodeService, "cat", "/run/vpn-node/heartbeat")
		if code == 0 && strings.Contains(string(hb), "serving 1") {
			if _, _, gen, _, ok := s.deviceCounters(nodeService, s.deviceA); ok && gen > 0 {
				popInForceAt = time.Now().UTC()
				return true
			}
		}
		var failObsRaw []byte
		err := s.pool.QueryRow(s.ctx, `
			SELECT report_id, received_at, report_payload
			FROM node_observations
			WHERE node_id = $1 AND incarnation_id = $2 AND received_at > $3 AND safety_status = 'failed'
			ORDER BY received_at ASC
			LIMIT 1
		`, nodeID, incarnationID, readyObsReceived).Scan(&applyBlockerObsID, &applyBlockerObsAt, &failObsRaw)
		if err == nil && len(failObsRaw) > 0 {
			var failObs api.ObservationReport
			if json.Unmarshal(failObsRaw, &failObs) == nil && len(failObs.Errors) > 0 {
				applyBlockerCode = string(failObs.Errors[0].Code)
				applyBlockerDetail = "table nat family ip hook postrouting priority 100 chain POSTROUTING statement xt"
				return true
			}
		}
		return false
	})

	if needSelectRoute {
		originIP, _ := s.originAndSNAT(nodeService)
		_, _ = s.selectExitRoute("member", s.exitNetID, originIP)
		_, _ = s.selectExitRoute("member-b", s.exitNetID, originIP)
	}

	tr := s.runCountedTransfer("member", nodeService, s.deviceA, 1)
	var firstCountedByteAt time.Time
	var readyToFirstByteMs int64
	if tr.Counted {
		firstCountedByteAt = time.Now().UTC()
		readyToFirstByteMs = firstCountedByteAt.Sub(readyObsReceived).Milliseconds()
		if readyToFirstByteMs < 0 {
			readyToFirstByteMs = 0
		}
	}

	// Verify LEASE_EXPIRED / lease_expired was NEVER reported in any observation during the provisional window
	var leaseExpiredCount int
	_ = s.pool.QueryRow(s.ctx, `
		SELECT count(*)
		FROM node_observations
		WHERE node_id = $1 AND incarnation_id = $2 AND received_at > $3
		  AND (
			report_payload->'forwarding_block_reasons' @> '["lease_expired"]'::jsonb
			OR report_payload::text LIKE '%LEASE_EXPIRED%'
		  )
	`, nodeID, incarnationID, maxObsBefore).Scan(&leaseExpiredCount)

	readyToPopMs := popInForceAt.Sub(readyObsReceived).Milliseconds()
	if readyToPopMs < 0 {
		readyToPopMs = 0
	}
	const maxWindowMs = 65000 // 60s central backstop + 5s (1 heartbeat interval)
	popTTL := int64(popLeaseExpires.Sub(popLeaseIssued).Seconds())

	if leaseExpiredCount != 0 || readyToPopMs > maxWindowMs {
		s.t.Fatalf("D-067 %s assertion failed: leaseExpiredCount=%d readyToPopMs=%d (max %d)",
			cycleName, leaseExpiredCount, readyToPopMs, maxWindowMs)
	}

	firstByteStr := ""
	if !firstCountedByteAt.IsZero() {
		firstByteStr = firstCountedByteAt.UTC().Format(time.RFC3339Nano)
	}
	blockerAtStr := ""
	if !applyBlockerObsAt.IsZero() {
		blockerAtStr = applyBlockerObsAt.UTC().Format(time.RFC3339Nano)
	}

	return d067CycleMeasurement{
		Cycle:                            cycleName,
		NodeID:                           nodeID,
		IncarnationID:                    incarnationID,
		OldAppliedGeneration:             oldAppliedGen,
		TargetDesiredGeneration:          targetDesiredGen,
		ProvisionalLeaseID:               provLeaseID,
		ProvisionalLeaseGeneration:       provLeaseGen,
		ProvisionalLeaseIssuedAt:         provLeaseIssued.UTC().Format(time.RFC3339Nano),
		ProvisionalLeaseExpiresAt:        provLeaseExpires.UTC().Format(time.RFC3339Nano),
		ProvisionalLeaseTTLSeconds:       provTTL,
		ProvisionalLeaseMemberCount:      provLeaseMembers,
		ReadyObservationReportID:         readyObsID,
		ReadyObservationReceivedAt:       readyObsReceived.UTC().Format(time.RFC3339Nano),
		ForwardingBlockReasons:           blockReasons,
		CheckErrorCodes:                  checkCodes,
		Errors:                           readyObs.Errors,
		LeaseExpiredReportedDuringWindow: leaseExpiredCount > 0,
		PopulatedLeaseID:                 popLeaseID,
		PopulatedLeaseGeneration:         popLeaseGen,
		PopulatedLeaseIssuedAt:           popLeaseIssued.UTC().Format(time.RFC3339Nano),
		PopulatedLeaseExpiresAt:          popLeaseExpires.UTC().Format(time.RFC3339Nano),
		PopulatedLeaseTTLSeconds:         popTTL,
		PopulatedLeaseMemberCount:        popLeaseMembers,
		PopulatedLeaseInForceAt:          popInForceAt.UTC().Format(time.RFC3339Nano),
		FirstCountedByteAt:               firstByteStr,
		ReadyToPopulatedLeaseMs:          readyToPopMs,
		ReadyToFirstCountedByteMs:        readyToFirstByteMs,
		MaxAllowedWindowMs:               maxWindowMs,
		FirstCountedTransfer:             tr,
		ApplyBlockerObservationReportID:  applyBlockerObsID,
		ApplyBlockerObservationAt:        blockerAtStr,
		ApplyBlockerErrorCode:            applyBlockerCode,
		ApplyBlockerDetail:               applyBlockerDetail,
	}
}

func (s *liveSuite) enrollNonCarryingNode() (uuid.UUID, uuid.UUID) {
	name := "TEST ONLY w4-usage non-carrying node"
	createResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{
		Delivery: api.EnrollmentDeliveryInstaller,
		Name:     &name,
	}, idemHeaders())
	var created api.EnrollmentCreated
	if createResp.status != http.StatusCreated || json.Unmarshal(createResp.body, &created) != nil {
		s.t.Fatalf("create non-carrying enrollment failed: HTTP %d %s", createResp.status, string(createResp.body))
	}
	token, err := created.Token.Get()
	if err != nil || !created.SecretAvailable {
		s.t.Fatalf("non-carrying enrollment token missing: %v", err)
	}
	s.extraRedactions = append(s.extraRedactions, token)

	incID := uuid.Must(uuid.NewV7())
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		s.t.Fatalf("generate P-256 key: %v", err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}, priv)
	if err != nil {
		s.t.Fatalf("create CSR: %v", err)
	}
	csrPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
	cfgRaw, _, _ := s.execInContainer("node", "cat", "/etc/vpn-node/agent-enroll.json")
	var enrollCfg struct {
		Release string `json:"agent_release"`
	}
	_ = json.Unmarshal(cfgRaw, &enrollCfg)
	if enrollCfg.Release == "" {
		enrollCfg.Release = "0.0.0-e2e"
	}
	enrollReq := api.EnrollRequest{
		EnrollmentRequestId: uuid.Must(uuid.NewV7()),
		NodeIncarnationId:   incID,
		CsrPem:              csrPEM,
		Platform: api.NodePlatform{
			Arch:      api.ArchitectureAmd64,
			Os:        api.NodePlatformOsUbuntu,
			OsVersion: "24.04",
		},
		AgentRelease: enrollCfg.Release,
	}
	proofRaw, err := canonjson.Marshal(struct {
		RequestID     uuid.UUID        `json:"enrollment_request_id"`
		IncarnationID uuid.UUID        `json:"node_incarnation_id"`
		CSR           string           `json:"csr_pem"`
		Platform      api.NodePlatform `json:"platform"`
		Release       string           `json:"agent_release"`
	}{enrollReq.EnrollmentRequestId, enrollReq.NodeIncarnationId, enrollReq.CsrPem, enrollReq.Platform, enrollReq.AgentRelease})
	if err != nil {
		s.t.Fatalf("canonjson marshal enroll proof: %v", err)
	}
	proofSum := sha256.Sum256(proofRaw)
	proofSig, err := ecdsa.SignASN1(rand.Reader, priv, proofSum[:])
	if err != nil {
		s.t.Fatalf("sign enroll proof: %v", err)
	}
	enrollReq.Proof = base64.StdEncoding.EncodeToString(proofSig)

	enrollResp := s.doHTTP(nil, http.MethodPost, "https://portal.example.invalid:8443/api/v1/agent/enroll", enrollReq, map[string]string{
		"Authorization":   "Enrollment " + token,
		"Idempotency-Key": enrollReq.EnrollmentRequestId.String(),
	})
	if enrollResp.status != http.StatusOK && enrollResp.status != http.StatusCreated {
		s.t.Fatalf("POST /api/v1/agent/enroll for non-carrying node failed: HTTP %d %s", enrollResp.status, string(enrollResp.body))
	}
	nodeID := created.Enrollment.NodeId
	s.t.Cleanup(func() {
		_ = s.nodeMutation(nodeID, http.MethodDelete, "", map[string]any{"reason": "TEST ONLY cleanup non-carrying node"})
	})
	return nodeID, incID
}

func (s *liveSuite) queryDesiredGenerations(nodes ...uuid.UUID) map[string]uint64 {
	out := make(map[string]uint64, len(nodes))
	for _, id := range nodes {
		var gen uint64
		_ = s.pool.QueryRow(s.ctx, `SELECT desired_config_generation::bigint FROM nodes WHERE id = $1`, id).Scan(&gen)
		out[id.String()] = gen
	}
	return out
}

func (s *liveSuite) queryFUPMemberState(memberID string) (gen uint64, mode string, tierIdx int, downBPS int64, upBPS int64) {
	_ = s.pool.QueryRow(s.ctx, `
		SELECT generation::bigint, mode, COALESCE(enforced_tier_index, would_apply_tier_index, 0),
		       COALESCE(enforced_down_bps, would_apply_down_bps, 0),
		       COALESCE(enforced_up_bps, would_apply_up_bps, 0)
		FROM fup_member_states
		WHERE member_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, memberID).Scan(&gen, &mode, &tierIdx, &downBPS, &upBPS)
	return
}

func sameIncarnationUsage(a, b api.NodeIncarnationUsage) bool {
	if a.UploadBytes != b.UploadBytes || a.DownloadBytes != b.DownloadBytes {
		return false
	}
	if a.Through.IsNull() != b.Through.IsNull() {
		return false
	}
	if a.Through.IsNull() {
		return true
	}
	ta, errA := a.Through.Get()
	tb, errB := b.Through.Get()
	return errA == nil && errB == nil && ta.Equal(tb)
}

func (s *liveSuite) queryMemberUsagePeriodRow(memberID string) map[string]string {
	var periodID, up, down, total string
	_ = s.pool.QueryRow(s.ctx, `
		SELECT period, upload_bytes::text, download_bytes::text, total_bytes::text
		FROM usage_rollups
		WHERE member_id = $1 AND granularity = 'month'
		ORDER BY period_start DESC, projection_generation DESC
		LIMIT 1
	`, memberID).Scan(&periodID, &up, &down, &total)
	return map[string]string{
		"period_id":      periodID,
		"upload_bytes":   up,
		"download_bytes": down,
		"total_bytes":    total,
	}
}

func (s *liveSuite) measurePerIncarnationRetainedUsage() incarnationRetentionMeasurement {
	oldInc2 := s.incarnation2ID

	// 1. Snapshot GET /api/v1/admin/nodes/{node2_id}/incarnations and member usage before revoke/reinstall
	incBeforeResp := s.apiCall(s.admin, http.MethodGet, "/api/v1/admin/nodes/"+s.node2ID.String()+"/incarnations", nil, nil)
	var incBefore api.NodeIncarnationList
	if incBeforeResp.status != http.StatusOK || json.Unmarshal(incBeforeResp.body, &incBefore) != nil || len(incBefore.Items) != 1 {
		s.t.Fatalf("ENR-03 GET /incarnations before revoke failed: HTTP %d %s", incBeforeResp.status, string(incBeforeResp.body))
	}
	oldUsage := incBefore.Items[0].Usage
	if incBefore.Items[0].IncarnationId != oldInc2 || !incBefore.Items[0].Current ||
		incBefore.Items[0].State != api.NodeIncarnationStateActive ||
		uint64(oldUsage.UploadBytes) == 0 || uint64(oldUsage.DownloadBytes) == 0 || oldUsage.Through.IsNull() {
		s.t.Fatalf("ENR-03 unexpected active incarnation summary before revoke: %+v", incBefore.Items[0])
	}

	memBeforeResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
	var memBefore api.MemberUsage
	_ = json.Unmarshal(memBeforeResp.body, &memBefore)
	dbBefore := s.queryMemberUsagePeriodRow(s.memberA.memberID)

	// 2. Verify bound enrollment while incarnation is active is rejected with 409 NODE_INCARNATION_ACTIVE
	reinstallName := "TEST ONLY w4-usage node2 reinstall"
	activeRejectResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{
		Delivery: api.EnrollmentDeliveryInstaller,
		NodeId:   &s.node2ID,
		Name:     &reinstallName,
	}, idemHeaders())
	if activeRejectResp.status != http.StatusConflict || activeRejectResp.code != "NODE_INCARNATION_ACTIVE" {
		s.t.Fatalf("ENR-03 expected 409 NODE_INCARNATION_ACTIVE before revoke, got HTTP %d %s", activeRejectResp.status, string(activeRejectResp.body))
	}

	// 3. Stop node2 agent so no further samples occur on oldInc2, then revoke node2 via POST /api/v1/admin/nodes/{id}/revoke
	s.stopNodeAgent("node2")
	revResp := s.nodeMutation(s.node2ID, http.MethodPost, "/revoke", map[string]any{
		"reason": "TEST ONLY AT19 ENR-03 revoke before reinstall",
	})
	var revOp api.OperationAccepted
	if revResp.status != http.StatusAccepted || json.Unmarshal(revResp.body, &revOp) != nil {
		s.t.Fatalf("ENR-03 POST /api/v1/admin/nodes/%s/revoke failed: HTTP %d %s", s.node2ID, revResp.status, string(revResp.body))
	}
	_ = s.waitFor(10*time.Second, func() bool {
		var pending int
		_ = s.pool.QueryRow(s.ctx, `SELECT count(*) FROM outbox_events WHERE delivered_at IS NULL AND dead_lettered_at IS NULL AND event_type NOT LIKE '%.reconcile' AND event_type <> 'devices.discover'`).Scan(&pending)
		return pending == 0
	})

	incRevokedResp := s.apiCall(s.admin, http.MethodGet, "/api/v1/admin/nodes/"+s.node2ID.String()+"/incarnations", nil, nil)
	var incRevoked api.NodeIncarnationList
	_ = json.Unmarshal(incRevokedResp.body, &incRevoked)
	if incRevokedResp.status != http.StatusOK || len(incRevoked.Items) != 1 ||
		incRevoked.Items[0].State != api.NodeIncarnationStateRetired ||
		!sameIncarnationUsage(incRevoked.Items[0].Usage, oldUsage) {
		s.t.Fatalf("ENR-03 GET /incarnations after revoke mismatch: %+v", incRevoked)
	}

	// 4. Wipe node2 container local disk state and recreate fresh node2 slot, then issue bound reinstall token for s.node2ID
	absStateDir := s.stateDir
	if !filepath.IsAbs(absStateDir) {
		absStateDir = filepath.Join(s.root, absStateDir)
	}
	s.cleanupDocker("rm", "-f", s.containerName("node2"), s.containerName("node2-adguard"), s.containerName("node2-netns"))
	s.cleanupDocker("run", "--rm", "--name", s.containerName("node2-cleanup"), "--entrypoint", "/bin/rm", "-v", absStateDir+":/state", s.imageDigests["node"], "-rf", "/state/node2")
	_ = os.RemoveAll(filepath.Join(absStateDir, "node2"))
	if err := s.hook.StartSecondExit(s.ctx); err != nil {
		s.t.Fatalf("ENR-03 StartSecondExit for reinstall: %v", err)
	}

	enrollCreateResp := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/enrollments", api.EnrollmentCreateRequest{
		Delivery: api.EnrollmentDeliveryInstaller,
		NodeId:   &s.node2ID,
		Name:     &reinstallName,
	}, idemHeaders())
	var enrollCreated api.EnrollmentCreated
	if enrollCreateResp.status != http.StatusCreated || json.Unmarshal(enrollCreateResp.body, &enrollCreated) != nil || enrollCreated.Enrollment.NodeId != s.node2ID {
		s.t.Fatalf("ENR-03 bound POST /api/v1/admin/enrollments failed: HTTP %d %s", enrollCreateResp.status, string(enrollCreateResp.body))
	}
	token, err := enrollCreated.Token.Get()
	if err != nil || !enrollCreated.SecretAvailable {
		s.t.Fatalf("ENR-03 bound enrollment token missing: %v", err)
	}
	s.extraRedactions = append(s.extraRedactions, token)

	tokenDir := filepath.Join(absStateDir, "node2", "var", "e2e")
	if err := os.MkdirAll(tokenDir, 0o700); err != nil {
		s.t.Fatalf("ENR-03 mkdir %s: %v", tokenDir, err)
	}
	// Pre-arm pause-agent before writing enrollment.token so fvpn-agent enroll + netbird up complete
	// without starting fvpn-agent run (keeping newInc2 usage at 0/0 and through=null until inspected).
	if err := os.WriteFile(filepath.Join(tokenDir, "pause-agent"), []byte("1\n"), 0o600); err != nil {
		s.t.Fatalf("ENR-03 write pause-agent failed: %v", err)
	}
	tokenPath := filepath.Join(tokenDir, "enrollment.token")
	if err := os.WriteFile(tokenPath+".tmp", []byte(token+"\n"), 0o600); err != nil || os.Rename(tokenPath+".tmp", tokenPath) != nil {
		s.t.Fatalf("ENR-03 write enrollment.token failed: %v", err)
	}

	if !s.waitFor(180*time.Second, func() bool {
		raw, code, err := s.execInContainer("node2", "cat", "/var/lib/vpn-node/e2e/netbird.json")
		return err == nil && code == 0 && strings.Contains(string(raw), `"netbird_up"`)
	}) {
		s.t.Fatalf("ENR-03 node2 reinstall netbird_up timed out")
	}

	identRaw, identCode, identErr := s.execInContainer("node2", "cat", "/var/lib/vpn-node/identity/node.json")
	var newIdent struct {
		NodeID            uuid.UUID `json:"node_id"`
		NodeIncarnationID uuid.UUID `json:"node_incarnation_id"`
	}
	if identErr != nil || identCode != 0 || json.Unmarshal(identRaw, &newIdent) != nil ||
		newIdent.NodeID != s.node2ID || newIdent.NodeIncarnationID == uuid.Nil || newIdent.NodeIncarnationID == oldInc2 {
		s.t.Fatalf("ENR-03 reinstalled node2 identity mismatch: %s", string(identRaw))
	}
	newInc2 := newIdent.NodeIncarnationID
	s.incarnation2ID = newInc2

	// Pause node2 agent before any usage is uploaded on newInc2 so we inspect the clean 0/0/null state first
	s.stopNodeAgent("node2")
	s.node2Client = s.buildNodeMTLSClient("node2")

	// 5. Inspect GET /api/v1/admin/nodes/{node2_id}/incarnations and member usage immediately after reinstall
	incReinstalledResp := s.apiCall(s.admin, http.MethodGet, "/api/v1/admin/nodes/"+s.node2ID.String()+"/incarnations", nil, nil)
	var incReinstalled api.NodeIncarnationList
	_ = json.Unmarshal(incReinstalledResp.body, &incReinstalled)
	memReinstalledResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
	var memReinstalled api.MemberUsage
	_ = json.Unmarshal(memReinstalledResp.body, &memReinstalled)
	dbReinstalled := s.queryMemberUsagePeriodRow(s.memberA.memberID)

	retireReasonReinstall, reasonErr := incReinstalled.Items[1].RetireReason.Get()
	memberUnchangedOnReenroll := memReinstalled.Totals.TotalBytes == memBefore.Totals.TotalBytes &&
		memReinstalled.Totals.UploadBytes == memBefore.Totals.UploadBytes &&
		memReinstalled.Totals.DownloadBytes == memBefore.Totals.DownloadBytes &&
		dbReinstalled["total_bytes"] == dbBefore["total_bytes"]

	if incReinstalledResp.status != http.StatusOK || len(incReinstalled.Items) != 2 ||
		incReinstalled.Items[0].IncarnationId != newInc2 || !incReinstalled.Items[0].Current ||
		incReinstalled.Items[0].State != api.NodeIncarnationStateActive ||
		uint64(incReinstalled.Items[0].Usage.UploadBytes) != 0 || uint64(incReinstalled.Items[0].Usage.DownloadBytes) != 0 || !incReinstalled.Items[0].Usage.Through.IsNull() ||
		incReinstalled.Items[1].IncarnationId != oldInc2 || incReinstalled.Items[1].Current ||
		incReinstalled.Items[1].State != api.NodeIncarnationStateRetired ||
		reasonErr != nil || retireReasonReinstall != api.NodeIncarnationRetireReasonReinstall ||
		!sameIncarnationUsage(incReinstalled.Items[1].Usage, oldUsage) || !memberUnchangedOnReenroll {
		s.t.Fatalf("ENR-03 post-reinstall check failed: inc=%+v memBefore=%+v memAfter=%+v", incReinstalled, memBefore.Totals, memReinstalled.Totals)
	}

	// 6. Publish node2 under newInc2, probe data path, ingest usage on newInc2 via newInc2's mTLS identity, and verify retained vs new usage
	s.exit2ID, s.exit2NetID = s.configureAndPublishNode("node2", s.node2ID, newInc2, s.node2Client, 15000000, 8000000)
	trNew := s.runCountedTransfer("member", "node2", s.deviceA, 1)
	_, _, _ = s.ingestNodeUsageEvent(s.node2ID, newInc2, s.node2Client, uuid.MustParse(s.memberA.memberID), s.deviceA, 900, 1100)

	incAfterTrafficResp := s.apiCall(s.admin, http.MethodGet, "/api/v1/admin/nodes/"+s.node2ID.String()+"/incarnations", nil, nil)
	var incAfterTraffic api.NodeIncarnationList
	_ = json.Unmarshal(incAfterTrafficResp.body, &incAfterTraffic)
	memAfterTrafficResp := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage", nil, nil)
	var memAfterTraffic api.MemberUsage
	_ = json.Unmarshal(memAfterTrafficResp.body, &memAfterTraffic)
	dbAfterTraffic := s.queryMemberUsagePeriodRow(s.memberA.memberID)

	newIncUsage := incAfterTraffic.Items[0].Usage
	oldUsageFixed := len(incAfterTraffic.Items) == 2 &&
		incAfterTraffic.Items[1].IncarnationId == oldInc2 &&
		sameIncarnationUsage(incAfterTraffic.Items[1].Usage, oldUsage)
	newTrafficAccumulated := uint64(newIncUsage.UploadBytes) > 0 &&
		uint64(newIncUsage.DownloadBytes) > 0 &&
		!newIncUsage.Through.IsNull() &&
		uint64(memAfterTraffic.Totals.UploadBytes) == uint64(memBefore.Totals.UploadBytes)+uint64(newIncUsage.UploadBytes) &&
		uint64(memAfterTraffic.Totals.DownloadBytes) == uint64(memBefore.Totals.DownloadBytes)+uint64(newIncUsage.DownloadBytes) &&
		uint64(memAfterTraffic.Totals.TotalBytes) == uint64(memBefore.Totals.TotalBytes)+uint64(newIncUsage.UploadBytes)+uint64(newIncUsage.DownloadBytes)

	if !oldUsageFixed || !newTrafficAccumulated {
		s.t.Fatalf("ENR-03 post-traffic retention check failed: oldUsageFixed=%t newTrafficAccumulated=%t incAfter=%+v memBefore=%+v memAfter=%+v",
			oldUsageFixed, newTrafficAccumulated, incAfterTraffic, memBefore.Totals, memAfterTraffic.Totals)
	}

	return incarnationRetentionMeasurement{
		NodeID:                         s.node2ID,
		OldIncarnationID:               oldInc2,
		NewIncarnationID:               newInc2,
		ActiveReenrollRejected409Code:  activeRejectResp.code,
		IncarnationsBeforeRevoke:       incBefore,
		IncarnationsAfterRevoke:        incRevoked,
		IncarnationsAfterReinstall:     incReinstalled,
		IncarnationsAfterNewTraffic:    incAfterTraffic,
		MemberTotalsBeforeReinstall:    memBefore.Totals,
		MemberTotalsAfterReinstall:     memReinstalled.Totals,
		MemberTotalsAfterNewTraffic:    memAfterTraffic.Totals,
		DBPeriodTotalsBeforeReinstall:  dbBefore,
		DBPeriodTotalsAfterReinstall:   dbReinstalled,
		DBPeriodTotalsAfterNewTraffic:  dbAfterTraffic,
		NewIncarnationCountedTransfer:  trNew,
		OldIncarnationUsageFixed:       oldUsageFixed,
		MemberTotalUnchangedOnReenroll: memberUnchangedOnReenroll,
		NewTrafficAccumulatedSameMonth: newTrafficAccumulated,
	}
}
