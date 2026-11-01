package usage

import (
	"testing"
	"time"

	centralusage "github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestAT15AT17AT20_LivePeriodsAtUTCBoundaries(t *testing.T) {
	member := uuid.MustParse("0195f000-0000-7000-8000-000000000017")
	for _, tc := range []struct {
		name, readAt, summary, held, monthStart, monthEnd string
		intervals                                         [3]string
	}{
		{"month_start", "2026-11-01T00:00:00Z", "2026-10", "2026-10", "2026-10-01", "2026-11-01", [3]string{"2026-10", "2026-10", "2026-10"}},
		{"first_seconds", "2026-11-01T00:00:03Z", "2026-11", "2026-10", "2026-11-01", "2026-12-01", [3]string{"2026-10", "2026-10", "2026-10"}},
		{"offsets_cross_month", "2026-11-01T00:00:17Z", "2026-11", "2026-11", "2026-11-01", "2026-12-01", [3]string{"2026-10", "2026-11", "2026-11"}},
		{"mid_month", "2026-11-15T12:00:00Z", "2026-11", "2026-11", "2026-11-01", "2026-12-01", [3]string{"2026-11", "2026-11", "2026-11"}},
		{"year_end", "2026-12-31T23:59:59Z", "2026-12", "2026-12", "2026-12-01", "2027-01-01", [3]string{"2026-12", "2026-12", "2026-12"}},
		{"year_start", "2027-01-01T00:00:00Z", "2026-12", "2026-12", "2026-12-01", "2027-01-01", [3]string{"2026-12", "2026-12", "2026-12"}},
		{"offsets_cross_year", "2027-01-01T00:00:17Z", "2027-01", "2027-01", "2027-01-01", "2027-02-01", [3]string{"2026-12", "2027-01", "2027-01"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readAt, err := time.Parse(time.RFC3339, tc.readAt)
			if err != nil {
				t.Fatal(err)
			}
			// AT17: the summary was fetched a second before the daily request.
			start, _ := time.Parse(time.DateOnly, tc.monthStart)
			end, _ := time.Parse(time.DateOnly, tc.monthEnd)
			summary := api.MemberUsage{PeriodId: intervalPeriod(readAt.Add(-time.Second)), PeriodStart: start, PeriodEnd: end}
			p, err := summaryPeriod(summary)
			if err != nil || p.ID != tc.summary || !p.Start.Equal(start) || !p.End.Equal(end) {
				t.Fatalf("summary period=%+v error=%v", p, err)
			}
			// AT15: the actual live offsets, including ev3 and the unbound ev5.
			base := readAt.Add(-20 * time.Second)
			for i, offset := range []time.Duration{0, 4 * time.Second, 8 * time.Second} {
				from := base.Add(offset)
				id := intervalPeriod(from.In(time.FixedZone("TEST ONLY +08", 8*3600)))
				p, err := centralusage.ParseMonth(id)
				if err != nil || id != tc.intervals[i] || !p.Start.Before(from.Add(2*time.Second)) || !p.End.After(from) {
					t.Fatalf("record %d: period=%s does not overlap [%s,%s): %v", i, id, from, from.Add(2*time.Second), err)
				}
			}
			// AT20: quiescent events were captured before the read-time clock.
			events := []handoffEvent{{Event: api.UsageEvent{Records: []api.UsageRecord{{MemberId: member, PeriodId: intervalPeriod(readAt.Add(-5 * time.Second)), UploadBytes: 3, DownloadBytes: 7}}}}}
			held, err := heldMemberPeriod(events, member.String())
			up, down := heldMemberBytes(events, member.String(), held)
			if err != nil || held != tc.held || up != "3" || down != "7" {
				t.Fatalf("held=%s bytes=%s/%s error=%v", held, up, down, err)
			}
		})
	}
}

func TestAT17_SummaryPeriodRejectsInvalidMonthAndBounds(t *testing.T) {
	for _, view := range []api.MemberUsage{
		{PeriodId: "2026-13"},
		{PeriodId: "2026-11", PeriodStart: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)},
	} {
		if _, err := summaryPeriod(view); err == nil {
			t.Fatalf("accepted invalid summary %+v", view)
		}
	}
}

func TestAT20_HeldPeriodUsesMemberTrafficAcrossMonths(t *testing.T) {
	member := uuid.MustParse("0195f000-0000-7000-8000-000000000020")
	other := uuid.MustParse("0195f000-0000-7000-8000-000000000021")
	record := func(id uuid.UUID, period string, up, down uint64) handoffEvent {
		return handoffEvent{Event: api.UsageEvent{Records: []api.UsageRecord{{MemberId: id, PeriodId: period, UploadBytes: decstr.U64(up), DownloadBytes: decstr.U64(down)}}}}
	}
	for _, tc := range []struct {
		name, want string
		events     []handoffEvent
	}{
		{"gap_only", "", []handoffEvent{{}}},
		{"other_member", "", []handoffEvent{record(other, "2027-01", 1, 2)}},
		{"zero_bytes", "", []handoffEvent{record(member, "2027-01", 0, 0)}},
		{"invalid_period", "", []handoffEvent{record(member, "2026-13", 1, 2)}},
		{"ignore_empty_and_other", "2026-12", []handoffEvent{record(member, "2026-12", 3, 7), record(member, "2027-01", 0, 0), record(other, "2027-01", 1, 2)}},
		{"latest_nonzero", "2027-01", []handoffEvent{record(member, "2027-01", 5, 11), record(member, "2026-12", 3, 7)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := heldMemberPeriod(tc.events, member.String())
			if p != tc.want || (err != nil) != (tc.want == "") {
				t.Fatalf("period=%s error=%v", p, err)
			}
			if tc.name == "latest_nonzero" {
				up, down := heldMemberBytes(tc.events, member.String(), "")
				if up != "8" || down != "18" {
					t.Fatalf("incarnation-wide bytes=%s/%s", up, down)
				}
			}
		})
	}
}
