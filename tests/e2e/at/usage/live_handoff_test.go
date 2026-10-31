//go:build e2e

package usage

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"
)

// AT20 is this lane's ENR-05 extension, not the DNS acceptance case §19 AT20.
// Read-only ledger exports and public APIs are the evidence sources. The
// historical stream must never be reconstructed under the presenting identity.
type handoffEvent struct {
	Event            api.UsageEvent  `json:"event"`
	Raw              json.RawMessage `json:"-"`
	CentralCount     int             `json:"central_count_by_payload_sha256"`
	PayloadUnchanged bool            `json:"canonical_payload_unchanged"`
	SamplesUnchanged bool            `json:"historical_samples_unchanged"`
}

type handoffBatch struct {
	Key       string          `json:"idempotency_key"`
	Body      json.RawMessage `json:"request"`
	Stored    json.RawMessage `json:"stored_response"`
	Replayed  json.RawMessage `json:"replayed_response"`
	ByteEqual bool            `json:"stored_response_byte_equal"`
}

func (s *liveSuite) memberPeriodUsage(period string) api.MemberUsage {
	r := s.apiCall(s.memberA, http.MethodGet, "/api/v1/member/usage?period_id="+period, nil, nil)
	var view api.MemberUsage
	if r.status != http.StatusOK || json.Unmarshal(r.body, &view) != nil {
		s.t.Fatalf("AT20 member usage: HTTP %d %s", r.status, r.code)
	}
	return view
}

// Read only a quiescent copy, including WAL. Never open the live ledger with a
// second writer or use the old CLI's current-stream watermark after adoption.
func (s *liveSuite) handoffLedgerCopy(slot string) *sql.DB {
	tmp := s.t.TempDir()
	for _, name := range []string{"usage.db", "usage.db-wal"} {
		if name == "usage.db-wal" {
			_, code, err := s.execInContainer(slot, "test", "-f", "/var/lib/vpn-node/state/"+name)
			if err != nil || code != 0 {
				continue
			}
		}
		if _, err := s.hook.ExportContainerFile(s.ctx, slot, "/var/lib/vpn-node/state/"+name, filepath.Join(tmp, name)); err != nil {
			s.t.Fatalf("AT20 quiescent usage export: %v", err)
		}
	}
	db, err := sql.Open("sqlite", "file:"+filepath.Join(tmp, "usage.db")+"?mode=ro&_pragma=query_only(1)")
	if err != nil {
		s.t.Fatal(err)
	}
	s.t.Cleanup(func() { _ = db.Close() })
	return db
}

func (s *liveSuite) heldEvents(db *sql.DB) []handoffEvent {
	rows, err := db.QueryContext(s.ctx, `SELECT payload FROM events WHERE state = 'outbox' ORDER BY seq`)
	if err != nil {
		s.t.Fatal(err)
	}
	defer rows.Close()
	var events []handoffEvent
	for rows.Next() {
		var e handoffEvent
		if err = rows.Scan(&e.Raw); err != nil {
			s.t.Fatal(err)
		}
		if err = json.Unmarshal(e.Raw, &e.Event); err != nil {
			s.t.Fatal(err)
		}
		events = append(events, e)
	}
	if err = rows.Err(); err != nil {
		s.t.Fatal(err)
	}
	return events
}

func heldMemberBytes(events []handoffEvent, member, period string) (string, string) {
	up, down := new(big.Int), new(big.Int)
	for _, ev := range events {
		for _, r := range ev.Event.Records {
			if r.MemberId.String() == member && r.PeriodId == period {
				up.Add(up, new(big.Int).SetUint64(uint64(r.UploadBytes)))
				down.Add(down, new(big.Int).SetUint64(uint64(r.DownloadBytes)))
			}
		}
	}
	return up.String(), down.String()
}

func decimalSum(a, b string) string {
	x, ok := new(big.Int).SetString(a, 10)
	if !ok {
		panic("invalid decimal evidence")
	}
	y, ok := new(big.Int).SetString(b, 10)
	if !ok {
		panic("invalid decimal evidence")
	}
	return x.Add(x, y).String()
}

func (s *liveSuite) d112Configuration(slot string) map[string]any {
	// Config has no sampling override. Decode the running node's actual config
	// and retain its checksum/provenance; ensureUsage passes the meter default.
	raw, code, err := s.execInContainer(slot, "cat", "/etc/vpn-node/agent-lab.json")
	if err != nil || code != 0 {
		s.t.Fatal("AT20 read running node configuration")
	}
	var node map[string]json.RawMessage
	if err = json.Unmarshal(raw, &node); err != nil {
		s.t.Fatal(err)
	}
	for _, k := range []string{"sampling_interval_seconds", "sample_interval_seconds", "meter_interval_seconds"} {
		if _, present := node[k]; present {
			s.t.Fatalf("AT20 unexpected meter configuration %s; inspect effective value", k)
		}
	}
	centralRaw, code, err := s.execInContainer("control", "cat", "/etc/fvpn/usage.json")
	if err != nil || code != 0 {
		s.t.Fatal("AT20 read running central usage configuration")
	}
	path := filepath.Join(s.t.TempDir(), "usage.json")
	if err = os.WriteFile(path, centralRaw, 0600); err != nil {
		s.t.Fatal(err)
	}
	cfg, err := config.LoadUsage(path)
	if err != nil {
		s.t.Fatal(err)
	}
	bound := *cfg.SamplingIntervalBoundSeconds
	const interval = 10 // node/agent/usage.go -> meter.DefaultSampleInterval.
	if interval > bound {
		s.t.Fatalf("D-112 meter interval %ds exceeds bound %ds", interval, bound)
	}
	var explicit map[string]json.RawMessage
	if err = json.Unmarshal(centralRaw, &explicit); err != nil {
		s.t.Fatal(err)
	}
	_, configured := explicit["sampling_interval_bound_seconds"]
	return map[string]any{
		"node_meter_interval_seconds":             interval,
		"central_sampling_interval_bound_seconds": bound,
		"interval_lte_bound":                      interval <= bound,
		"node_config_read":                        "/etc/vpn-node/agent-lab.json",
		"node_interval_source":                    "Engine.ensureUsage passes no interval override; meter.DefaultSampleInterval is 10s in the recorded image source",
		"central_config_read":                     "/etc/fvpn/usage.json",
		"central_bound_explicit":                  configured,
	}
}

func (s *liveSuite) verifyHistoricalEvents(events []handoffEvent) {
	for i := range events {
		ev := &events[i]
		var raw []byte
		var seq, inc, hash string
		if err := s.pool.QueryRow(s.ctx, `SELECT e.event_seq::text,e.incarnation_id::text,encode(e.payload_hash,'hex'),p.payload
            FROM usage_events e JOIN usage_event_payloads p ON p.event_id=e.id
            WHERE e.node_id=$1 AND e.incarnation_id=$2 AND e.event_seq=$3`,
			ev.Event.NodeId, ev.Event.NodeIncarnationId, ev.Event.EventSeq.String()).Scan(&seq, &inc, &hash, &raw); err != nil {
			s.t.Fatal(err)
		}
		ev.PayloadUnchanged = bytes.Equal(raw, ev.Raw) && hash == ev.Event.PayloadSha256 && seq == ev.Event.EventSeq.String() && inc == ev.Event.NodeIncarnationId.String()
		if err := s.pool.QueryRow(s.ctx, `SELECT count(*) FROM usage_events WHERE payload_hash=decode($1,'hex')`, hash).Scan(&ev.CentralCount); err != nil {
			s.t.Fatal(err)
		}
		ev.SamplesUnchanged = true
		for idx, r := range ev.Event.Records {
			var n int
			if err := s.pool.QueryRow(s.ctx, `SELECT count(*) FROM usage_samples s JOIN usage_events e ON e.id=s.event_id
                WHERE e.node_id=$1 AND e.incarnation_id=$2 AND e.event_seq=$3 AND s.record_index=$4
                AND s.member_id=$5 AND s.device_id=$6 AND s.binding_revision=$7
                AND s.upload_bytes=$8 AND s.download_bytes=$9 AND s.meter_epoch_id=$10
                AND to_char(s.interval_start AT TIME ZONE 'UTC','YYYY-MM')=$11`,
				ev.Event.NodeId, ev.Event.NodeIncarnationId, ev.Event.EventSeq.String(), idx, r.MemberId, r.DeviceId, r.BindingRevision.String(),
				r.UploadBytes.String(), r.DownloadBytes.String(), ev.Event.MeterEpochId, r.PeriodId).Scan(&n); err != nil {
				s.t.Fatal(err)
			}
			ev.SamplesUnchanged = ev.SamplesUnchanged && n == 1
		}
		if !ev.PayloadUnchanged || !ev.SamplesUnchanged || ev.CentralCount != 1 {
			s.t.Fatalf("AT20 historical event %s: payload=%t samples=%t count=%d", seq, ev.PayloadUnchanged, ev.SamplesUnchanged, ev.CentralCount)
		}
	}
}

func (s *liveSuite) centralStreamACK(node, inc uuid.UUID) string {
	var ack string
	if err := s.pool.QueryRow(s.ctx, `SELECT highest_contiguous_seq::text FROM usage_checkpoints WHERE node_id=$1 AND incarnation_id=$2`, node, inc).Scan(&ack); err != nil {
		s.t.Fatal(err)
	}
	return ack
}

func (s *liveSuite) originCLI(slot string) e2e.UsageSnapshot {
	snap, err := s.hook.UsageLedgerSnapshot(s.ctx, slot)
	if err != nil {
		s.t.Fatal(err)
	}
	return snap
}

func (s *liveSuite) addHandoffCheck(rec *Record, name string, ok bool, detail string) {
	status := StatusPass
	if !ok {
		status = StatusFail
		rec.Status = StatusFail
		rec.Blockers = append(rec.Blockers, name+": "+detail)
	}
	rec.MechanismChecks = append(rec.MechanismChecks, MechanismCheck{Name: name, SpecRef: "ENR-05, D-105, D-112, §10.5–§10.8", Status: status, Summary: detail})
}

func (s *liveSuite) installerToken(node uuid.UUID) string {
	name := "TEST ONLY AT20 surviving usage ledger"
	req := api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, Name: &name}
	if node != uuid.Nil {
		req.NodeId = &node
	}
	r := s.apiCall(s.admin, http.MethodPost, "/api/v1/admin/enrollments", req, idemHeaders())
	var out api.EnrollmentCreated
	if r.status != http.StatusCreated || json.Unmarshal(r.body, &out) != nil {
		s.t.Fatalf("AT20 enrollment: HTTP %d %s", r.status, r.code)
	}
	token, err := out.Token.Get()
	if err != nil || !out.SecretAvailable {
		s.t.Fatal("AT20 enrollment token unavailable")
	}
	s.extraRedactions = append(s.extraRedactions, token)
	return token
}

func (s *liveSuite) stopInstaller() {
	// D-102 follow-up 18: systemd consumers have no pause-agent supervisor.
	for _, unit := range []string{"fvpn-updater.timer", "fvpn-updater.service", "fvpn-watchdog.service", "fvpn-agent.service"} {
		if _, code, err := s.execInContainer("installer-node", "systemctl", "stop", unit); err != nil || code != 0 {
			s.t.Fatalf("AT20 stop %s", unit)
		}
	}
	states, err := s.hook.InstallerServices(s.ctx, "installer-node")
	if err != nil {
		s.t.Fatal(err)
	}
	for unit, state := range states {
		if state == "active" || state == "activating" || state == "deactivating" {
			s.t.Fatalf("AT20 %s not quiescent: %s", unit, state)
		}
	}
}

func (s *liveSuite) incarnationView(node uuid.UUID) api.NodeIncarnationList {
	r := s.apiCall(s.admin, http.MethodGet, "/api/v1/admin/nodes/"+node.String()+"/incarnations", nil, nil)
	var view api.NodeIncarnationList
	if r.status != http.StatusOK || json.Unmarshal(r.body, &view) != nil {
		s.t.Fatalf("AT20 incarnations: HTTP %d %s", r.status, r.code)
	}
	return view
}

func (s *liveSuite) evalAT20(t *testing.T) (rec Record) {
	rec = s.baseRecord("AT20", "ENR-05 retired-stream hand-off (usage lane extension)", "§6.2–§6.3, §10.5–§10.8; ENR-05, D-105, D-112, D-115, D-118",
		"The current certificate delivers the surviving old stream unchanged and once only; member before + held equals after, origin ACK is independent, late events are audited and retired certificates refused.", []string{"ENR03", "MTR02", "MTR03", "MTR04", "API01", "DATA01"})
	rec.Status = StatusFail
	rec.Actual = "AT20 did not complete; completed steps and failure stage are retained below."
	defer func() {
		if err := WriteEvidence(s.evidence, rec, s.extraRedactions...); err != nil {
			t.Error(err)
		}
	}()
	stage := func(name string) { rec.Observations["last_stage"] = name; t.Log("AT20 " + name) }
	must := func(err error) {
		t.Helper()
		if err != nil {
			rec.Blockers = append(rec.Blockers, fmt.Sprint(err))
			t.Fatal(err)
		}
	}
	stage("prepare fresh installer serving node")
	s.stopNodeAgent("node")
	s.stopNodeAgent("node2")
	must(s.hook.PrepareInstallerNode(s.ctx))
	releases, err := s.hook.RegisterInstallerReleases(s.ctx, s.adminHook())
	must(err)
	ids := []uuid.UUID{s.baseline.Boot, releases[0].ID, releases[1].ID}
	_, _, err = s.hook.ActivatePolicy(s.ctx, s.adminHook(), "release_support", "at20-installer-"+s.runID, map[string]any{
		"approved_release_ids": ids, "revoked_release_ids": []string{}, "minimum_serving_version": map[string]any{"serving_release_ids": ids}})
	must(err)
	rec.Observations["installer_release_ids"] = ids
	must(s.hook.HoldInstallerFirstPoll(s.ctx, true))
	must(s.hook.StageInstallerToken(s.ctx, "installer-node", s.installerToken(uuid.Nil)))
	installed, err := s.hook.RerunInstaller(s.ctx, "installer-node")
	must(err)
	rec.Observations["initial_install"] = installed
	if installed.Exit != 0 {
		t.Fatalf("AT20 initial installer: %s", installed.Status)
	}
	node, err := uuid.Parse(installed.NodeID)
	must(err)
	origin, err := uuid.Parse(installed.Incarnation)
	must(err)
	rec.NodeIncarnation = origin.String()
	rec.Observations["handoff_node_id"] = node
	rec.Observations["origin_incarnation_id"] = origin
	retiredClient := s.buildNodeMTLSClient("installer-node")
	must(s.hook.ConfigureInstallerServing(s.ctx))
	must(s.hook.HoldInstallerFirstPoll(s.ctx, false))
	must(s.hook.RestoreInstallerServices(s.ctx, "installer-node", installed.Services))
	imageID, _, err := s.execInContainer("installer-node", "sha256sum", "/opt/vpn-node/current/fvpn-agent")
	must(err)
	rec.Observations["installed_agent_binary_sha256"] = strings.Fields(string(imageID))[0]
	rec.Observations["d112_configuration"] = s.d112Configuration("installer-node")
	patch := s.nodeMutation(node, http.MethodPatch, "", map[string]any{
		"capacity": map[string]any{"down_bps": 1000000000, "up_bps": 1000000000, "member_down_bps_cap": 60000000, "member_up_bps_cap": 30000000}})
	if patch.status != http.StatusAccepted && patch.status != http.StatusOK {
		t.Fatalf("AT20 capacity: %d %s", patch.status, string(patch.body))
	}
	if !s.waitFor(90*time.Second, func() bool {
		var ready bool
		_ = s.pool.QueryRow(s.ctx, `SELECT provisioning_state='ready' AND safety_status='passed' FROM nodes WHERE id=$1`, node).Scan(&ready)
		return ready
	}) {
		t.Fatal("AT20 installer did not reach ready safety baseline")
	}
	pub := s.nodeMutation(node, http.MethodPut, "/publication", map[string]any{"publication_intent": "published", "group_ids": []string{s.groupID.String()}, "reason": "TEST ONLY AT20 serving"})
	if pub.status != http.StatusAccepted {
		t.Fatalf("AT20 publication: %d %s", pub.status, string(pub.body))
	}
	var exit uuid.UUID
	if !s.waitFor(90*time.Second, func() bool {
		return s.pool.QueryRow(s.ctx, `SELECT e.id FROM exits e JOIN nodes n ON n.id=e.node_id WHERE n.id=$1 AND n.provisioning_state='ready'`, node).Scan(&exit) == nil
	}) {
		t.Fatal("AT20 installer exit unavailable")
	}
	selected, err := s.hook.MemberTransportStack(s.containerName("member"), "installer-node")
	must(err)
	originIP, _ := s.originAndSNAT("installer-node")
	if _, ok := s.selectExitRoute("member", exitNetworkID(exit), originIP); !ok {
		t.Fatal("AT20 select installer exit")
	}
	var baseline transferResult
	if !s.waitFor(90*time.Second, func() bool {
		baseline = s.runCountedTransfer("member", "installer-node", s.deviceA, 1)
		return baseline.Counted
	}) {
		t.Fatalf("AT20 no counted installer traffic: %+v", baseline)
	}
	rec.Observations["installer_serving_transfer"] = baseline
	// Establish committed bytes under the origin before holding additional traffic.
	s.waitForUploadedTraffic("installer-node", node, origin, s.deviceA)
	stage("hold usage POST, produce real traffic and quiesce original ledger")
	must(s.hook.HoldUsageAcknowledgements(s.ctx, "installer-node", true))
	countersBefore, err := selected.CountedMemberTraffic(s.ctx, s.deviceA)
	must(err)
	heldTransfer := s.runCountedTransfer("member", "installer-node", s.deviceA, 3)
	if !heldTransfer.Counted {
		t.Fatalf("AT20 held traffic did not count: %+v", heldTransfer)
	}
	var held e2e.UsageSnapshot
	if !s.waitFor(40*time.Second, func() bool {
		held, err = s.hook.UsageLedgerSnapshot(s.ctx, "installer-node")
		if err != nil {
			return false
		}
		for _, ev := range held.UnackedEvents {
			for _, r := range ev.Records {
				if r.MemberID == s.memberA.memberID && r.UploadBytes != "0" && r.DownloadBytes != "0" {
					return true
				}
			}
		}
		return false
	}) {
		t.Fatal("AT20 no nonzero held usage")
	}
	countersAfter, err := selected.CountedMemberTraffic(s.ctx, s.deviceA)
	must(err)
	states, err := s.hook.InstallerServices(s.ctx, "installer-node")
	must(err)
	s.stopInstaller()
	first, err := s.hook.UsageLedgerSnapshotBytes(s.ctx, "installer-node")
	must(err)
	second, err := s.hook.UsageLedgerSnapshotBytes(s.ctx, "installer-node")
	must(err)
	if !bytes.Equal(first, second) {
		t.Fatal("AT20 stopped snapshots differ")
	}
	must(json.Unmarshal(first, &held))
	preDB := s.handoffLedgerCopy("installer-node")
	events := s.heldEvents(preDB)
	if len(events) == 0 {
		t.Fatal("AT20 held stream empty")
	}
	period := time.Now().UTC().Format("2006-01")
	up, down := heldMemberBytes(events, s.memberA.memberID, period)
	if up == "0" || down == "0" {
		t.Fatal("AT20 held member bytes empty")
	}
	for _, ev := range events {
		var n int
		must(s.pool.QueryRow(s.ctx, `SELECT count(*) FROM usage_events WHERE node_id=$1 AND incarnation_id=$2 AND event_seq=$3`, node, origin, ev.Event.EventSeq.String()).Scan(&n))
		if n != 0 {
			t.Fatalf("AT20 held seq %s already central", ev.Event.EventSeq)
		}
	}
	before := s.memberPeriodUsage(period)
	beforeInc := s.incarnationView(node)
	rec.Observations["origin_cli_before_reenrollment"] = held
	rec.Observations["quiescent_cli_byte_equal"] = true
	rec.Observations["held_transfer"] = heldTransfer
	rec.Observations["held_device_counters_before"] = countersBefore
	rec.Observations["held_device_counters_after"] = countersAfter
	rec.Observations["member_period_before"] = before
	rec.Observations["incarnations_before"] = beforeInc
	rec.Observations["held_upload_bytes"] = up
	rec.Observations["held_download_bytes"] = down
	rec.Observations["held_total_bytes"] = decimalSum(up, down)
	rec.Observations["services_before_stop"] = states
	stage("API retirement and controlled same-node reinstall")
	revoked := s.nodeMutation(node, http.MethodPost, "/revoke", map[string]any{"reason": "TEST ONLY AT20 controlled reinstall"})
	if revoked.status != http.StatusAccepted {
		t.Fatalf("AT20 revoke: %d %s", revoked.status, revoked.code)
	}
	rec.Observations["revoke_operation"] = json.RawMessage(revoked.body)
	var retiredAt time.Time
	if !s.waitFor(30*time.Second, func() bool {
		view := s.incarnationView(node)
		for _, v := range view.Items {
			if v.IncarnationId == origin && v.State == api.NodeIncarnationStateRetired {
				retiredAt, _ = v.RetiredAt.Get()
				return !retiredAt.IsZero()
			}
		}
		return false
	}) {
		t.Fatal("AT20 origin did not retire")
	}
	var accepted api.OperationAccepted
	must(json.Unmarshal(revoked.body, &accepted))
	// Wait for all API revocation worker effects; do not infer them from 202.
	if !s.waitFor(45*time.Second, func() bool {
		var pending int
		_ = s.pool.QueryRow(s.ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id=$1 AND delivered_at IS NULL AND dead_lettered_at IS NULL`, node).Scan(&pending)
		return pending == 0
	}) {
		t.Fatal("AT20 revocation worker effects pending")
	}
	must(s.hook.HoldInstallerFirstPoll(s.ctx, true))
	replacement, err := s.hook.ControlledReinstall(s.ctx, "installer-node", s.installerToken(node))
	must(err)
	rec.Observations["controlled_reinstall"] = replacement
	if replacement.Exit != 0 || replacement.Status != "reenrolled" || replacement.NodeID != node.String() || replacement.Incarnation == origin.String() {
		t.Fatalf("AT20 controlled reinstall: %s/%d", replacement.Status, replacement.Exit)
	}
	if replacement.Before.UsageDB != replacement.Hashes.UsageDB || replacement.Before.UsageWAL != replacement.Hashes.UsageWAL {
		t.Fatal("AT20 reinstall changed usage DB/WAL")
	}
	current, err := uuid.Parse(replacement.Incarnation)
	must(err)
	rec.Observations["presenting_incarnation_id"] = current
	rec.Observations["central_retired_at"] = retiredAt
	view := s.incarnationView(node)
	rec.Observations["incarnations_after_reenrollment"] = view
	if len(view.Items) != 2 || view.Items[0].IncarnationId != current || !view.Items[0].Current || !view.Items[0].CreatedAt.After(retiredAt) {
		t.Fatal("AT20 current incarnation ordering")
	}
	refused := s.agentCallWithClient(retiredClient, http.MethodGet, "/api/v1/agent/usage-checkpoint", nil, nil)
	rec.Observations["retired_certificate_refusal"] = map[string]any{"http_status": refused.status, "code": refused.code, "request_id": refused.headers.Get("X-Request-ID")}
	if refused.status != http.StatusForbidden || refused.code != "IDENTITY_REVOKED" {
		t.Fatalf("AT20 retired certificate: %d %s", refused.status, refused.code)
	}
	presenter := s.buildNodeMTLSClient("installer-node")
	stage("production D-118 adoption and autonomous origin upload")
	must(s.hook.ConfigureInstallerServing(s.ctx))
	must(s.hook.HoldInstallerFirstPoll(s.ctx, false))
	must(s.hook.RestoreInstallerServices(s.ctx, "installer-node", states))
	must(s.hook.HoldUsageAcknowledgements(s.ctx, "installer-node", false))
	if !s.waitFor(150*time.Second, func() bool {
		var ack string
		err := s.pool.QueryRow(s.ctx, `SELECT highest_contiguous_seq::text FROM usage_checkpoints WHERE node_id=$1 AND incarnation_id=$2`, node, origin).Scan(&ack)
		return err == nil && ack == held.LastEventSeq
	}) {
		t.Fatal("AT20 autonomous hand-off did not advance origin ACK")
	}
	s.stopInstaller()
	newCLI := s.originCLI("installer-node")
	if newCLI.NodeIncarnationID != current.String() {
		t.Fatal("AT20 new CLI did not name current ledger")
	}
	postDB := s.handoffLedgerCopy("installer-node")
	var oldACK, last, oldEpoch, finalCode string
	must(postDB.QueryRowContext(s.ctx, `SELECT CAST(acked_through AS TEXT),CAST(last_seq AS TEXT),ledger_epoch,final_code FROM retired_streams WHERE incarnation_id=?`, origin.String()).Scan(&oldACK, &last, &oldEpoch, &finalCode))
	if oldACK != held.LastEventSeq || last != held.LastEventSeq || finalCode != "" {
		t.Fatalf("AT20 retired local ACK %s/%s %s", oldACK, last, finalCode)
	}
	rec.Observations["current_cli_after_handoff"] = newCLI
	rec.Observations["retired_local_stream"] = map[string]string{"node_incarnation_id": origin.String(), "acked_through": oldACK, "last_seq": last, "ledger_epoch": oldEpoch, "final_code": finalCode}
	s.verifyHistoricalEvents(events)
	rec.Observations["handed_off_events"] = events
	after := s.memberPeriodUsage(period)
	rec.Observations["member_period_after"] = after
	afterInc := s.incarnationView(node)
	rec.Observations["incarnations_after_handoff"] = afterInc
	equality := after.Totals.UploadBytes.String() == decimalSum(before.Totals.UploadBytes.String(), up) && after.Totals.DownloadBytes.String() == decimalSum(before.Totals.DownloadBytes.String(), down) && after.Totals.TotalBytes.String() == decimalSum(before.Totals.TotalBytes.String(), decimalSum(up, down))
	if !equality {
		t.Fatalf("AT20 before+held equality: before=%s held=%s after=%s", before.Totals.TotalBytes, decimalSum(up, down), after.Totals.TotalBytes)
	}
	rec.Observations["before_plus_held_equals_after"] = true
	rec.Observations["member_view_scope"] = "GET member/usage exposes aggregate period totals, not incarnation rows; origin attribution is proven by immutable event/sample joins and admin nodes/{id}/incarnations."
	stage("same-key agent batch replay and independent ACKs")
	currentBefore := s.centralStreamACK(node, current)
	rows, err := postDB.QueryContext(s.ctx, `SELECT idempotency_key,body FROM upload_batches ORDER BY created_at`)
	must(err)
	var batches []handoffBatch
	for rows.Next() {
		var b handoffBatch
		must(rows.Scan(&b.Key, &b.Body))
		var req api.UsageBatchRequest
		must(json.Unmarshal(b.Body, &req))
		if req.OriginIncarnationId == nil || *req.OriginIncarnationId != origin {
			continue
		}
		if req.NodeIncarnationId != current || req.NodeId != node {
			t.Fatal("AT20 handoff presenter mismatch")
		}
		keyHash := sha256.Sum256([]byte(b.Key))
		must(s.pool.QueryRow(s.ctx, `SELECT b.response_body FROM usage_batch_replays b JOIN idempotency_requests i ON i.id=b.id WHERE i.key_hash=$1`, keyHash[:]).Scan(&b.Stored))
		response := s.agentCallWithClient(presenter, http.MethodPost, "/api/v1/agent/usage-batches", []byte(b.Body), map[string]string{"Idempotency-Key": b.Key})
		b.Replayed = response.body
		b.ByteEqual = bytes.Equal(b.Stored, b.Replayed)
		var ack api.UsageBatchResult
		must(json.Unmarshal(b.Stored, &ack))
		if response.status != http.StatusOK || !b.ByteEqual || ack.NodeIncarnationId != origin || ack.NodeId != node {
			t.Fatalf("AT20 stored replay/ACK mismatch: HTTP %d", response.status)
		}
		batches = append(batches, b)
	}
	must(rows.Err())
	must(rows.Close())
	if len(batches) == 0 {
		t.Fatal("AT20 missing agent saved origin batch")
	}
	rec.Observations["agent_batches_and_stored_replays"] = batches
	currentAfter := s.centralStreamACK(node, current)
	if currentBefore != currentAfter {
		t.Fatal("AT20 origin replay advanced current ACK")
	}
	s.verifyHistoricalEvents(events)
	replayView := s.memberPeriodUsage(period)
	if replayView.Totals != after.Totals {
		t.Fatal("AT20 replay changed member totals")
	}
	rec.Observations["current_ack_before_origin_replay"] = currentBefore
	rec.Observations["current_ack_after_origin_replay"] = currentAfter
	rec.Observations["origin_central_ack"] = s.centralStreamACK(node, origin)
	rec.Observations["member_period_after_replay"] = replayView
	stage("live over-window rejection and audit")
	cfg := rec.Observations["d112_configuration"].(map[string]any)
	bound := cfg["central_sampling_interval_bound_seconds"].(int)
	lateEnd := retiredAt.Add(time.Duration(bound+1) * time.Second)
	if time.Now().Before(lateEnd) {
		if !s.waitFor(20*time.Second, func() bool { return time.Now().After(lateEnd) }) {
			t.Fatal("AT20 late-event clock wait")
		}
	}
	lateSeq := uint64(events[len(events)-1].Event.EventSeq) + 1
	late := makeSignedUsageEvent(t, node, origin, events[0].Event.MeterEpochId, lateSeq, lateEnd.Add(-time.Second), lateEnd, api.UsageQualityComplete, nil, []api.UsageRecord{})
	lateResp := s.agentCallWithClient(presenter, http.MethodPost, "/api/v1/agent/usage-batches", api.UsageBatchRequest{
		NodeId: node, NodeIncarnationId: current, OriginIncarnationId: &origin, LedgerEpoch: presence.NewNullableWithValue(uuid.MustParse(oldEpoch)), Events: []api.UsageEvent{late},
	}, idemHeaders())
	var lateACK api.UsageBatchResult
	must(json.Unmarshal(lateResp.body, &lateACK))
	rec.Observations["late_event_probe"] = map[string]any{"source": "TEST ONLY deliberately late mTLS protocol event; not a sampler event; no clock or ledger edits", "event": late, "http_status": lateResp.status, "result": lateACK}
	if lateResp.status != http.StatusOK || len(lateACK.Results) != 1 || lateACK.Results[0].Status != api.UsageEventResultStatusRejected || lateACK.NodeIncarnationId != origin {
		t.Fatalf("AT20 late event: %d %+v", lateResp.status, lateACK)
	}
	var audit json.RawMessage
	must(s.pool.QueryRow(s.ctx, `SELECT jsonb_build_object('id',id,'action',action,'result',result,'metadata',after_diff,'request_id',request_id) FROM audit_events WHERE action='usage.ingest_rejected' AND after_diff->>'origin_incarnation_id'=$1 AND after_diff->>'reason'='retired_window' ORDER BY occurred_at DESC LIMIT 1`, origin.String()).Scan(&audit))
	rec.Observations["late_event_audit"] = audit
	var acceptedLate int
	must(s.pool.QueryRow(s.ctx, `SELECT count(*) FROM usage_events WHERE node_id=$1 AND incarnation_id=$2 AND event_seq=$3`, node, origin, decstr.U64(lateSeq).String()).Scan(&acceptedLate))
	if acceptedLate != 0 || s.centralStreamACK(node, origin) != oldACK {
		t.Fatal("AT20 rejected event mutated accepted ledger/ACK")
	}
	rec.Status = StatusPass
	for _, check := range []string{"BeforePlusHeldByteEquality", "OriginalPayloadAndHistoricalAttribution", "AgentSameKeyStoredResultReplay", "OriginACKAndIndependentCurrentACK", "D112ConfiguredIntervalWithinBound", "LiveLateEventRejectedAndAudited", "RetiredCertificateRefused"} {
		s.addHandoffCheck(&rec, check, true, "Live assertion completed; deciding API, ledger and byte evidence retained in observations.")
	}
	rec.Actual = fmt.Sprintf("Same node %s reenrolled %s -> %s through ControlledReinstall. Member period %s: before %s + held %s = after %s bytes; %d original events retained once, %d agent batches replayed byte-identically, origin ACK %s, independent current ACK %s. Meter interval 10s <= central bound %ds. Live late event rejected/audited and retired certificate refused with 403 IDENTITY_REVOKED. Member API provides period totals; incarnation attribution is read through central samples and admin incarnation history.", node, origin, current, period, before.Totals.TotalBytes, decimalSum(up, down), after.Totals.TotalBytes, len(events), len(batches), oldACK, currentAfter, bound)
	stage("complete")
	return rec
}
