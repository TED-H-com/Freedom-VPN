package fakecontrol

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// usageLedger is one node incarnation stream. highest is the contiguous
// prefix. maxSeen is the largest stored sequence, which may sit past a gap.
type usageLedger struct {
	hash    map[uint64]string
	highest uint64
	maxSeen uint64
}

func (l *usageLedger) clone() *usageLedger {
	out := &usageLedger{hash: make(map[uint64]string, len(l.hash)), highest: l.highest, maxSeen: l.maxSeen}
	for seq, hash := range l.hash {
		out.hash[seq] = hash
	}
	return out
}

// usageForce overrides the signed checkpoint. The batch ledger still tracks
// the events that were actually stored. "same" keeps the live epoch so a
// lower watermark is refused. "new" replaces the epoch so a lower watermark
// can be accepted.
type usageForce struct {
	epoch   uuid.UUID
	highest uint64
	maxSeen uint64
}

// UsageNodeView is one stream in UsageSnapshot.
type UsageNodeView struct {
	Key     string `json:"key"`
	Highest uint64 `json:"highest"`
	MaxSeen uint64 `json:"max_seen"`
	Events  int    `json:"events"`
}

// UsageSnapshot is the admin view of the fake ledger (TEST ONLY).
type UsageSnapshot struct {
	Epoch        string          `json:"epoch"`
	Forced       bool            `json:"forced"`
	ForceHighest uint64          `json:"force_highest"`
	Unavailable  int             `json:"unavailable"`
	RetryLeft    int             `json:"retry_left"`
	Nodes        []UsageNodeView `json:"nodes"`
}

// SetUsageUnavailable makes the next n usage-batch posts return 503.
// The checkpoint GET is not affected. n <= 0 clears the knob.
func (s *Server) SetUsageUnavailable(n int) {
	if n < 0 {
		n = 0
	}
	s.mu.Lock()
	s.usageUnavail = n
	s.mu.Unlock()
}

// SetUsageRetryAfter makes the next n usage-batch posts return 429 with
// Retry-After of seconds. The checkpoint GET is not affected.
func (s *Server) SetUsageRetryAfter(n int, seconds int) {
	if n < 0 {
		n = 0
	}
	if seconds < 0 {
		seconds = 0
	}
	s.mu.Lock()
	s.usageRetryN = n
	s.usageRetryFor = time.Duration(seconds) * time.Second
	s.mu.Unlock()
}

// SetUsageConflict makes the next n usage-batch posts return 409
// USAGE_EVENT_CONFLICT before the body is read, and stores nothing.
// A payload hash mismatch on the default path is HTTP 200 with a per-event
// conflict result. This knob keeps the whole-request 409 path for tests.
func (s *Server) SetUsageConflict(n int) {
	if n < 0 {
		n = 0
	}
	s.mu.Lock()
	s.usageConflict = n
	s.mu.Unlock()
}

// SetUsageCheckpoint selects the watermark the next signed checkpoint
// reports. mode is "same" (live epoch, caller-supplied highest), "new"
// (a new epoch, so a lower watermark is a new baseline) or "live" (forget
// the override and report the contiguous ACK).
func (s *Server) SetUsageCheckpoint(mode string, highest uint64) (uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch mode {
	case "live":
		s.usageForce = nil
		return s.usageEpoch, nil
	case "same":
		s.usageForce = &usageForce{epoch: s.usageEpoch, highest: highest, maxSeen: highest}
		return s.usageEpoch, nil
	case "new":
		id, err := uuid.NewV7()
		if err != nil {
			return uuid.Nil, err
		}
		s.usageEpoch = id
		s.usageForce = &usageForce{epoch: id, highest: highest, maxSeen: highest}
		return id, nil
	default:
		return uuid.Nil, errInvalid
	}
}

// UsageReport returns the current ledger view for the lab admin.
func (s *Server) UsageReport() UsageSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := UsageSnapshot{
		Epoch:       s.currentEpochLocked().String(),
		Unavailable: s.usageUnavail,
		RetryLeft:   s.usageRetryN,
		Nodes:       []UsageNodeView{},
	}
	if s.usageForce != nil {
		out.Forced = true
		out.ForceHighest = s.usageForce.highest
	}
	for key, led := range s.usageNodes {
		out.Nodes = append(out.Nodes, UsageNodeView{
			Key: key, Highest: led.highest, MaxSeen: led.maxSeen, Events: len(led.hash),
		})
	}
	return out
}

func (s *Server) currentEpochLocked() uuid.UUID {
	if s.usageForce != nil && s.usageForce.epoch != uuid.Nil {
		return s.usageForce.epoch
	}
	return s.usageEpoch
}

func streamKey(node, inc uuid.UUID) string {
	return node.String() + "|" + inc.String()
}

type inboundBatch struct {
	Events              []json.RawMessage `json:"events"`
	LedgerEpoch         json.RawMessage   `json:"ledger_epoch"`
	NodeID              uuid.UUID         `json:"node_id"`
	NodeIncarnationID   uuid.UUID         `json:"node_incarnation_id"`
	OriginIncarnationID *uuid.UUID        `json:"origin_incarnation_id,omitempty"`
}

type parsedUsage struct {
	seq     uint64
	hash    string
	badHash bool
}

func (s *Server) usage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierr.Write(w, r, apierr.New(apierr.MethodNotAllowed, "Method not allowed."))
		return
	}
	node, inc, ok := s.requireIdentity(w, r)
	if !ok {
		return
	}
	s.mu.Lock()
	if s.usageUnavail > 0 {
		s.usageUnavail--
		s.mu.Unlock()
		apierr.Write(w, r, apierr.New(apierr.DependencyUnavailable, "Usage ledger is unavailable."))
		return
	}
	if s.usageRetryN > 0 {
		s.usageRetryN--
		wait := s.usageRetryFor
		s.mu.Unlock()
		if wait <= 0 {
			wait = time.Second
		}
		apierr.Write(w, r, apierr.New(apierr.RateLimited, "Rate limit exceeded.").WithRetryAfter(wait))
		return
	}
	if s.usageConflict > 0 {
		s.usageConflict--
		s.mu.Unlock()
		apierr.Write(w, r, apierr.New(apierr.UsageEventConflict, "Usage event conflict."))
		return
	}
	s.mu.Unlock()

	body, err := io.ReadAll(io.LimitReader(r.Body, canonjson.MaxBytes+1))
	if err != nil || len(body) > canonjson.MaxBytes {
		apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Invalid usage batch."))
		return
	}
	var batch inboundBatch
	if err := json.Unmarshal(body, &batch); err != nil {
		apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Invalid usage batch."))
		return
	}
	if batch.NodeID != node || batch.NodeIncarnationID != inc {
		apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Identity does not match the certificate."))
		return
	}
	epoch, present, err := parseBatchEpoch(batch.LedgerEpoch)
	if err != nil {
		apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Invalid ledger epoch."))
		return
	}
	parsed, bad := classifyUsage(batch.Events)
	if bad {
		apierr.Write(w, r, apierr.New(apierr.ValidationFailed, "Invalid usage event."))
		return
	}

	s.mu.Lock()
	current := s.currentEpochLocked()
	if present && epoch != current {
		s.mu.Unlock()
		apierr.Write(w, r, apierr.New(apierr.LedgerEpochMismatch, "Ledger epoch does not match.").WithDetail("current_ledger_epoch", current.String()))
		return
	}
	if batch.OriginIncarnationID != nil {
		origin := *batch.OriginIncarnationID
		if origin == inc || s.retiredIncarnations[origin] != node {
			s.mu.Unlock()
			apierr.Write(w, r, apierr.New(apierr.StateConflict, "Origin is not retired for this node."))
			return
		}
		inc = origin
	}
	key := streamKey(node, inc)
	led := s.usageNodes[key]
	if led == nil {
		led = &usageLedger{hash: map[uint64]string{}}
	}
	work := led.clone()
	results := work.accept(parsed)
	s.usageNodes[key] = work
	attempt := UsageAttempt{IdempotencyKey: r.Header.Get("Idempotency-Key"), Body: append([]byte(nil), body...)}
	for _, result := range results {
		switch result.status {
		case string(api.UsageEventResultStatusAccepted):
			attempt.Accepted++
		case string(api.UsageEventResultStatusDuplicate):
			attempt.Duplicates++
		}
	}
	s.usageAttempts = append(s.usageAttempts, attempt)
	hold := s.usageResponseHold
	s.usageResponseHold = nil
	s.mu.Unlock()
	if hold != nil {
		hold(r.Context())
	}
	writeUsageResult(w, node, inc, current, work.highest, results)
}

func parseBatchEpoch(raw json.RawMessage) (uuid.UUID, bool, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return uuid.Nil, false, nil
	}
	var text string
	if err := json.Unmarshal(trimmed, &text); err != nil {
		return uuid.Nil, false, err
	}
	id, err := uuid.Parse(text)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, false, errInvalid
	}
	return id, true, nil
}

// classifyUsage checks each event's payload_sha256. A hash mismatch is a
// per-event conflict, not an unreadable body. A sequence of 0 or a body
// that is not an event is invalid.
func classifyUsage(raws []json.RawMessage) ([]parsedUsage, bool) {
	out := make([]parsedUsage, 0, len(raws))
	for _, raw := range raws {
		sum, err := canonjson.HashHex(raw, "payload_sha256")
		if err != nil {
			return nil, true
		}
		var ev struct {
			EventSeq decstr.U64 `json:"event_seq"`
			Hash     string     `json:"payload_sha256"`
		}
		if err := json.Unmarshal(raw, &ev); err != nil || uint64(ev.EventSeq) == 0 {
			return nil, true
		}
		out = append(out, parsedUsage{
			seq: uint64(ev.EventSeq), hash: ev.Hash, badHash: ev.Hash != sum,
		})
	}
	return out, false
}

type usageItem struct {
	seq    uint64
	status string
	code   string
	msg    string
}

// accept stores events up to the first payload-hash conflict. The conflicting
// event is not stored, and neither is any later sequence, so the contiguous
// prefix cannot jump the hole. The caller passes a clone and commits it.
func (l *usageLedger) accept(events []parsedUsage) []usageItem {
	first := map[uint64]string{}
	conflictAt := map[uint64]bool{}
	var blocked uint64
	hasBlocked := false
	note := func(seq uint64) {
		conflictAt[seq] = true
		if !hasBlocked || seq < blocked {
			blocked = seq
			hasBlocked = true
		}
	}
	for _, ev := range events {
		if ev.badHash {
			note(ev.seq)
			continue
		}
		if prev, ok := l.hash[ev.seq]; ok && prev != ev.hash {
			note(ev.seq)
			continue
		}
		if prev, ok := first[ev.seq]; ok && prev != ev.hash {
			note(ev.seq)
			continue
		}
		if _, ok := first[ev.seq]; !ok {
			first[ev.seq] = ev.hash
		}
	}
	items := make([]usageItem, 0, len(events))
	for _, ev := range events {
		if conflictAt[ev.seq] {
			items = append(items, usageItem{
				seq: ev.seq, status: string(api.UsageEventResultStatusConflict),
				code: string(api.ErrorCodeUSAGEEVENTCONFLICT),
				msg:  "Usage event hash does not match.",
			})
			continue
		}
		if hasBlocked && ev.seq > blocked {
			if _, ok := l.hash[ev.seq]; ok {
				items = append(items, usageItem{seq: ev.seq, status: string(api.UsageEventResultStatusDuplicate)})
				continue
			}
			items = append(items, usageItem{
				seq: ev.seq, status: string(api.UsageEventResultStatusRejected),
				code: string(api.ErrorCodeUSAGEEVENTCONFLICT),
				msg:  "Event was not stored past a conflicting sequence.",
			})
			continue
		}
		if _, ok := l.hash[ev.seq]; ok {
			items = append(items, usageItem{seq: ev.seq, status: string(api.UsageEventResultStatusDuplicate)})
			continue
		}
		l.hash[ev.seq] = ev.hash
		if ev.seq > l.maxSeen {
			l.maxSeen = ev.seq
		}
		items = append(items, usageItem{seq: ev.seq, status: string(api.UsageEventResultStatusAccepted)})
	}
	highest := uint64(0)
	for {
		if _, ok := l.hash[highest+1]; !ok {
			break
		}
		highest++
	}
	l.highest = highest
	return items
}

func writeUsageResult(w http.ResponseWriter, node, inc, epoch uuid.UUID, highest uint64, items []usageItem) {
	type row struct {
		Error    any    `json:"error"`
		EventSeq string `json:"event_seq"`
		Status   string `json:"status"`
	}
	var blocked any
	var lowest uint64
	hasBlocked := false
	rows := make([]row, 0, len(items))
	for _, it := range items {
		var errObj any
		if it.code != "" {
			errObj = map[string]any{
				"code": it.code, "message": it.msg, "fields": []any{},
			}
		}
		if it.status == string(api.UsageEventResultStatusConflict) || it.status == string(api.UsageEventResultStatusRejected) {
			if !hasBlocked || it.seq < lowest {
				lowest = it.seq
				hasBlocked = true
			}
		}
		rows = append(rows, row{Error: errObj, EventSeq: strconv.FormatUint(it.seq, 10), Status: it.status})
	}
	if hasBlocked {
		blocked = strconv.FormatUint(lowest, 10)
	}
	body := struct {
		BlockedSeq           any       `json:"blocked_seq"`
		HighestContiguousSeq string    `json:"highest_contiguous_seq"`
		LedgerEpoch          uuid.UUID `json:"ledger_epoch"`
		NodeID               uuid.UUID `json:"node_id"`
		NodeIncarnationID    uuid.UUID `json:"node_incarnation_id"`
		ReceivedAt           time.Time `json:"received_at"`
		Results              []row     `json:"results"`
	}{
		BlockedSeq:           blocked,
		HighestContiguousSeq: strconv.FormatUint(highest, 10),
		LedgerEpoch:          epoch,
		NodeID:               node,
		NodeIncarnationID:    inc,
		ReceivedAt:           time.Now().UTC(),
		Results:              rows,
	}
	writeJSON(w, http.StatusOK, body)
}

func (s *Server) usageCheckpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apierr.Write(w, r, apierr.New(apierr.MethodNotAllowed, "Method not allowed."))
		return
	}
	node, inc, ok := s.requireIdentity(w, r)
	if !ok {
		return
	}
	issued := s.now()
	s.mu.Lock()
	epoch := s.currentEpochLocked()
	highest, maxSeen := uint64(0), uint64(0)
	if s.usageForce != nil {
		highest = s.usageForce.highest
		maxSeen = s.usageForce.maxSeen
	} else if led := s.usageNodes[streamKey(node, inc)]; led != nil {
		highest = led.highest
		maxSeen = led.maxSeen
	}
	if maxSeen < highest {
		maxSeen = highest
	}
	corrupt := s.corrupt
	priv, keyID := s.recPriv, s.recID
	s.mu.Unlock()

	ranges := []api.SeqRange{}
	if maxSeen > highest {
		ranges = []api.SeqRange{{
			StartSeq: decstr.U64(highest + 1),
			EndSeq:   decstr.U64(maxSeen),
		}}
	}
	cp := api.UsageCheckpoint{
		HighestContiguousSeq: decstr.U64(highest),
		IssuedAt:             issued,
		LedgerEpoch:          epoch,
		MaxSeenSeq:           decstr.U64(maxSeen),
		MissingRanges:        ranges,
		NodeId:               node,
		NodeIncarnationId:    inc,
		RecoveryRecordId:     presence.NewNullNullable[uuid.UUID](),
	}
	raw, _, err := s.sign(cp, priv, keyID, corrupt)
	if err != nil {
		apierr.Write(w, r, apierr.New(apierr.InternalError, "Internal error."))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}
