//go:build e2e

package usage

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"

	_ "modernc.org/sqlite"
)

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

func (s *liveSuite) waitNodeServingAndPublished(nodeService string, nodeID uuid.UUID, deviceID uuid.UUID) bool {
	restartedForLease := false
	return s.waitFor(120*time.Second, func() bool {
		var desired, applied, pubObs, provState string
		err := s.pool.QueryRow(s.ctx, `
			SELECT n.desired_config_generation::text, n.applied_config_generation::text, COALESCE(e.publication_observed, ''), n.provisioning_state
			FROM nodes n
			LEFT JOIN exits e ON e.node_id = n.id
			WHERE n.id = $1
		`, nodeID).Scan(&desired, &applied, &pubObs, &provState)
		if err != nil || desired == "0" || desired != applied || pubObs != "active" {
			return false
		}
		hb, code, _ := s.execInContainer(nodeService, "cat", "/run/vpn-node/heartbeat")
		hbStr := string(hb)
		if code != 0 || !strings.Contains(hbStr, "serving 1") {
			if !restartedForLease && provState == "ready" && strings.Contains(hbStr, "reason lease_expired") {
				restartedForLease = true
				s.stopNodeAgent(nodeService)
				s.startNodeAgent(nodeService)
			}
			return false
		}
		if deviceID != uuid.Nil {
			_, _, gen, _, ok := s.deviceCounters(nodeService, deviceID)
			if !ok || gen == 0 {
				return false
			}
		}
		return true
	})
}

func (s *liveSuite) selectExitRoute(memberService string, wantNetID string, originIP string) (string, bool) {
	var listOut string
	var selectedID string
	re := regexp.MustCompile(`ID:\s+(fvpn-exit-[a-z0-9]+)\s+Network:\s+0\.0\.0\.0/0`)
	found := s.waitFor(60*time.Second, func() bool {
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
	routed := s.waitFor(25*time.Second, func() bool {
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
		out, code, err := s.execInContainer(memberService, "wget", "--server-response", "--timeout=5", "--tries=1",
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

func (s *liveSuite) flushNodeUsageUpload(nodeService string, nodeID uuid.UUID) {
	// Wait at least one 10s sample tick so the running sampler records any counter delta,
	// then pause and resume via pause-agent so entrypoint-node.sh runs agent-service.sh stop/start
	// without racing (handoff() runs finalSample() and startup upload runs immediately).
	time.Sleep(11 * time.Second)
	s.stopNodeAgent(nodeService)
	s.startNodeAgent(nodeService)
	_ = s.waitFor(30*time.Second, func() bool {
		hb, code, _ := s.execInContainer(nodeService, "cat", "/run/vpn-node/heartbeat")
		if code != 0 || !strings.Contains(string(hb), "serving 1") {
			return false
		}
		_, _, gen, _, ok := s.deviceCounters(nodeService, s.deviceA)
		if !ok || gen == 0 {
			return false
		}
		snap := s.inspectNodeUsageSQLite(nodeService, uuid.Nil)
		return snap.EventsCount > 0 && snap.OutboxCount == 0 && snap.AckedThrough >= uint64(snap.EventsCount)
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
