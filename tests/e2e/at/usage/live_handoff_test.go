//go:build e2e

package usage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
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
	if _, err := s.runHostCmd("docker", "cp", s.containerName(slot)+":/var/lib/vpn-node/state/.", tmp); err != nil {
		s.t.Fatalf("AT20 quiescent usage export: %v", err)
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
	raw, code, err := s.execInContainer(slot, "cat", "/etc/vpn-node/agent-run.json")
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
		"node_config_read":                        "/etc/vpn-node/agent-run.json",
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

var _ = fmt.Sprintf
var _ = time.Second
