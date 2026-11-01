package usage

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	centralusage "github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// intervalPeriod anchors a live protocol fixture to its own interval, never
// to the suite's wall clock or another event's start.
func intervalPeriod(start time.Time) string {
	return start.UTC().Format("2006-01")
}

// summaryPeriod pins subsequent daily reads to the month actually returned by
// the summary, even when the next request starts after UTC midnight.
func summaryPeriod(summary api.MemberUsage) (centralusage.Period, error) {
	p, err := centralusage.ParseMonth(summary.PeriodId)
	if err != nil {
		return p, err
	}
	if !summary.PeriodStart.Equal(p.Start) || !summary.PeriodEnd.Equal(p.End) {
		return p, fmt.Errorf("AT17 summary bounds do not match period %s", p.ID)
	}
	return p, nil
}

type handoffEvent struct {
	Event            api.UsageEvent  `json:"event"`
	Raw              json.RawMessage `json:"-"`
	CentralCount     int             `json:"central_count_by_payload_sha256"`
	PayloadUnchanged bool            `json:"canonical_payload_unchanged"`
	SamplesUnchanged bool            `json:"historical_samples_unchanged"`
}

// heldMemberPeriod selects the latest held period with nonzero traffic in both
// directions. Empty gaps and other members cannot select a month. There is no
// read-time clock dependency; a snapshot can legitimately span two months.
func heldMemberPeriod(events []handoffEvent, member string) (string, error) {
	periods := make(map[string]bool)
	for _, ev := range events {
		for _, r := range ev.Event.Records {
			if r.MemberId.String() == member {
				if _, err := centralusage.ParseMonth(r.PeriodId); err != nil {
					return "", err
				}
				periods[r.PeriodId] = true
			}
		}
	}
	latest := ""
	for p := range periods {
		up, down := heldMemberBytes(events, member, p)
		if up != "0" && down != "0" && p > latest {
			latest = p
		}
	}
	if latest == "" {
		return "", fmt.Errorf("AT20 held member bytes empty in every period")
	}
	return latest, nil
}

// An empty period requests all held months for the incarnation-wide comparison.
func heldMemberBytes(events []handoffEvent, member, period string) (string, string) {
	up, down := new(big.Int), new(big.Int)
	for _, ev := range events {
		for _, r := range ev.Event.Records {
			if r.MemberId.String() == member && (period == "" || r.PeriodId == period) {
				up.Add(up, new(big.Int).SetUint64(uint64(r.UploadBytes)))
				down.Add(down, new(big.Int).SetUint64(uint64(r.DownloadBytes)))
			}
		}
	}
	return up.String(), down.String()
}
