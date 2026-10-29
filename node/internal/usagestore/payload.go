package usagestore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// canonicalEvent builds the stored payload (§10.5).
//
// The bytes are the RFC 8785 form of the event including payload_sha256.
// payload_sha256 is the hex SHA-256 of the canonical event with that member
// removed, matching contracts/vectors/usage-event-payload-sha256.json and
// canonjson.HashHex. Times are UTC. Large integers stay decimal strings.
func canonicalEvent(nodeID, incID uuid.UUID, seq uint64, d Draft) (payload []byte, sum string, err error) {
	if !d.End.After(d.Start) {
		return nil, "", fmt.Errorf("usagestore: interval_end must be after interval_start")
	}
	quality := api.UsageQuality(d.Quality)
	if !quality.Valid() {
		return nil, "", fmt.Errorf("usagestore: invalid quality %q", d.Quality)
	}
	ev := api.UsageEvent{
		SchemaVersion:     api.UsageEventSchemaVersionN1,
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		MeterEpochId:      d.EpochID,
		EventSeq:          decstr.U64(seq),
		MeteringVersion:   api.MeteringVersionForwardAdmissionV1,
		IntervalStart:     d.Start.UTC(),
		IntervalEnd:       d.End.UTC(),
		Quality:           quality,
		Records:           make([]api.UsageRecord, 0, len(d.Records)),
	}
	if quality == api.UsageQualityGap {
		if !api.UsageGapReason(d.GapReason).Valid() {
			return nil, "", fmt.Errorf("usagestore: gap event requires a gap reason")
		}
		last := presence.NewNullNullable[time.Time]()
		if d.LastCheckpoint != nil {
			last = presence.NewNullableWithValue(d.LastCheckpoint.UTC())
		}
		ev.Gap = &api.UsageGap{
			Reason:           api.UsageGapReason(d.GapReason),
			LastCheckpointAt: last,
		}
	} else if d.GapReason != "" {
		return nil, "", fmt.Errorf("usagestore: gap reason set on a non-gap event")
	}
	if quality != api.UsageQualityGap && len(d.Records) == 0 {
		return nil, "", fmt.Errorf("usagestore: non-gap event needs at least one record")
	}
	seen := make(map[uuid.UUID]bool, len(d.Records))
	for _, r := range d.Records {
		if r.MemberID == uuid.Nil || r.DeviceID == uuid.Nil {
			return nil, "", fmt.Errorf("usagestore: record is missing member or device")
		}
		if seen[r.DeviceID] {
			return nil, "", fmt.Errorf("usagestore: duplicate device in event")
		}
		seen[r.DeviceID] = true
		if r.BindingRevision == 0 {
			return nil, "", fmt.Errorf("usagestore: binding_revision must be positive")
		}
		if !periodOverlaps(r.PeriodID, ev.IntervalStart, ev.IntervalEnd) {
			return nil, "", fmt.Errorf("usagestore: period_id %s does not overlap the interval", r.PeriodID)
		}
		ev.Records = append(ev.Records, api.UsageRecord{
			MemberId:        r.MemberID,
			DeviceId:        r.DeviceID,
			BindingRevision: decstr.U64(r.BindingRevision),
			PeriodId:        r.PeriodID,
			UploadBytes:     decstr.U64(r.Upload),
			DownloadBytes:   decstr.U64(r.Download),
		})
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		return nil, "", fmt.Errorf("usagestore: marshal event: %w", err)
	}
	without, err := canonjson.CanonicalizeWithout(raw, "payload_sha256")
	if err != nil {
		return nil, "", fmt.Errorf("usagestore: canonicalize event: %w", err)
	}
	digest := sha256.Sum256(without)
	ev.PayloadSha256 = hex.EncodeToString(digest[:])
	raw, err = json.Marshal(ev)
	if err != nil {
		return nil, "", fmt.Errorf("usagestore: marshal event: %w", err)
	}
	full, err := canonjson.Canonicalize(raw)
	if err != nil {
		return nil, "", fmt.Errorf("usagestore: canonicalize event: %w", err)
	}
	return full, ev.PayloadSha256, nil
}

// periodOverlaps reports whether UTC month id overlaps the half-open
// interval [start, end) (§10.4, §10.9).
func periodOverlaps(id string, start, end time.Time) bool {
	if len(id) != 7 || id[4] != '-' {
		return false
	}
	t, err := time.Parse("2006-01", id)
	if err != nil || t.Format("2006-01") != id {
		return false
	}
	ps := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	pe := ps.AddDate(0, 1, 0)
	return ps.Before(end.UTC()) && pe.After(start.UTC())
}

// PeriodID is the UTC calendar month containing t (§10.4). The node sends it
// as a hint. The central projector is the authority for a split (§10.9).
func PeriodID(t time.Time) string {
	t = t.UTC()
	return fmt.Sprintf("%04d-%02d", t.Year(), int(t.Month()))
}
