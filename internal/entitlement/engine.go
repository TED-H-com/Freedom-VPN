// Package entitlement computes finite VPN rights independently of payment adapters
// and node leases (Project Spec §9.2, §15.2, §16.3).
package entitlement

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

var ErrInvalid = errors.New("invalid entitlement input")

const DefaultDeviceLimit = 3

type Plan struct {
	ID          uuid.UUID   `json:"plan_revision_id"`
	DeviceLimit int         `json:"device_limit"`
	ExitGroups  []uuid.UUID `json:"exit_group_ids"`
}
type Period struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
type Subscription struct {
	ID                    uuid.UUID
	ProviderStatus        string
	PaidFrom, PaidThrough *time.Time
	// ServicePeriods excludes revoked full-refund periods; billing facts stay intact.
	ServicePeriods        []Period
	CancelAtPeriodEnd     bool
	GraceFrom, GraceUntil *time.Time
	GraceReason           string
}
type Grant struct {
	ID             uuid.UUID
	Plan           Plan
	Source, Reason string
	Start, End     time.Time
	Revoked        bool
}
type Config struct {
	GraceDuration      time.Duration
	GraceReason        string
	MaxGrantDuration   time.Duration
	MaxTrialDuration   time.Duration
	DefaultDeviceLimit int
}

// TestConfig is TEST ONLY. Grace/trial/refund choices remain 待決策 (§21).
func TestConfig() Config {
	return Config{GraceDuration: 72 * time.Hour, GraceReason: "TEST ONLY payment grace", MaxGrantDuration: 366 * 24 * time.Hour, MaxTrialDuration: 7 * 24 * time.Hour, DefaultDeviceLimit: DefaultDeviceLimit}
}
func (c Config) Validate() error {
	if c.GraceDuration < 0 || c.GraceDuration > 366*24*time.Hour || c.GraceDuration > 0 && strings.TrimSpace(c.GraceReason) == "" || c.MaxGrantDuration <= 0 || c.MaxGrantDuration > 366*24*time.Hour || c.MaxTrialDuration < 0 || c.MaxTrialDuration > c.MaxGrantDuration || c.DefaultDeviceLimit < 1 || c.DefaultDeviceLimit > 100 {
		return ErrInvalid
	}
	return nil
}

type Input struct {
	Now            time.Time
	MemberStatus   string
	AdminSuspended bool
	Plan           *Plan
	Subscription   *Subscription
	Grants         []Grant
	MemberGroups   []uuid.UUID
	Config         Config
}
type Result struct {
	State               string      `json:"state"`
	Plan                *Plan       `json:"plan"`
	DeviceLimit         int         `json:"device_limit"`
	ExitGroups          []uuid.UUID `json:"exit_group_ids"`
	MemberGroups        []uuid.UUID `json:"member_group_ids"`
	ValidFrom           *time.Time  `json:"valid_from"`
	EffectiveServiceEnd *time.Time  `json:"effective_service_end"`
	PaidThrough         *time.Time  `json:"paid_through"`
	GraceUntil          *time.Time  `json:"grace_until"`
	CancelAtPeriodEnd   bool        `json:"cancel_at_period_end"`
	AdminSuspended      bool        `json:"admin_suspended"`
	SourceRefs          []uuid.UUID `json:"source_refs"`
}
type Projection struct {
	Result
	MemberID   uuid.UUID  `json:"member_id"`
	Revision   decstr.U64 `json:"revision"`
	ComputedAt time.Time  `json:"computed_at"`
	Changed    bool       `json:"-"`
}

func instant(t time.Time) bool { return !t.IsZero() && t.Year() >= 1970 && t.Year() <= 9999 }
func ValidPeriod(start, end time.Time, maximum time.Duration) bool {
	return instant(start) && instant(end) && start.Before(end) && (maximum == 0 || end.Sub(start) <= maximum)
}
func utc(t time.Time) *time.Time { t = t.UTC().Truncate(time.Microsecond); return &t }
func ids(in []uuid.UUID) []uuid.UUID {
	out := append([]uuid.UUID{}, in...)
	slices.SortFunc(out, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	return slices.Compact(out)
}
func validPlan(p Plan) bool {
	return p.ID != uuid.Nil && p.DeviceLimit >= 1 && p.DeviceLimit <= 100 && len(p.ExitGroups) > 0 && len(p.ExitGroups) <= 100 && !slices.Contains(p.ExitGroups, uuid.Nil)
}
func (r Result) Allows(now time.Time) bool {
	return (r.State == "active" || r.State == "cancel_at_period_end" || r.State == "grace") && !r.AdminSuspended && r.Plan != nil && r.ValidFrom != nil && r.EffectiveServiceEnd != nil && !now.Before(*r.ValidFrom) && now.Before(*r.EffectiveServiceEnd)
}

// Compute implements every row of §9.2. It never reads a clock or a provider.
// Disconnected future periods cannot bridge a gap in paid service.
func Compute(in Input) (Result, error) {
	r := Result{State: "pending_payment", AdminSuspended: in.AdminSuspended, ExitGroups: []uuid.UUID{}, MemberGroups: ids(in.MemberGroups), SourceRefs: []uuid.UUID{}}
	if !instant(in.Now) || in.Config.Validate() != nil {
		return r, ErrInvalid
	}
	if !slices.Contains([]string{"pending_payment", "active", "expired", "suspended", "closed"}, in.MemberStatus) {
		return r, ErrInvalid
	}
	type interval struct {
		Period
		ref   uuid.UUID
		plan  *Plan
		grace bool
	}
	var spans []interval
	if in.Plan != nil {
		if !validPlan(*in.Plan) {
			return r, ErrInvalid
		}
		p := *in.Plan
		p.ExitGroups = ids(p.ExitGroups)
		r.Plan = &p
	}
	if s := in.Subscription; s != nil {
		if s.ID == uuid.Nil || (s.PaidFrom == nil) != (s.PaidThrough == nil) {
			return r, ErrInvalid
		}
		if s.PaidThrough != nil {
			if !ValidPeriod(*s.PaidFrom, *s.PaidThrough, 0) {
				return r, ErrInvalid
			}
			r.PaidThrough = utc(*s.PaidThrough)
			r.State = "expired"
		}
		r.CancelAtPeriodEnd = s.CancelAtPeriodEnd
		for _, p := range s.ServicePeriods {
			if !ValidPeriod(p.Start, p.End, 0) || s.PaidThrough == nil || p.End.After(*s.PaidThrough) {
				return r, ErrInvalid
			}
			spans = append(spans, interval{p, s.ID, in.Plan, false})
		}
		if (s.GraceFrom == nil) != (s.GraceUntil == nil) {
			return r, ErrInvalid
		}
		if s.GraceUntil != nil {
			if s.PaidThrough == nil || s.GraceReason == "" || !s.GraceFrom.Equal(*s.PaidThrough) || !ValidPeriod(*s.GraceFrom, *s.GraceUntil, in.Config.GraceDuration) || in.Config.GraceDuration == 0 {
				return r, ErrInvalid
			}
			r.GraceUntil = utc(*s.GraceUntil)
			if s.ProviderStatus == "past_due" && !s.CancelAtPeriodEnd {
				spans = append(spans, interval{Period{*s.GraceFrom, *s.GraceUntil}, s.ID, in.Plan, true})
			}
		}
		if s.ProviderStatus == "past_due" {
			r.State = "past_due"
		}
	}
	for _, g := range in.Grants {
		if g.ID == uuid.Nil || strings.TrimSpace(g.Reason) == "" || !validPlan(g.Plan) || !ValidPeriod(g.Start, g.End, in.Config.MaxGrantDuration) {
			return r, ErrInvalid
		}
		if !slices.Contains([]string{"admin_grant", "trial", "compensation", "migration"}, g.Source) {
			return r, ErrInvalid
		}
		if g.Source == "trial" && (in.Config.MaxTrialDuration == 0 || g.End.Sub(g.Start) > in.Config.MaxTrialDuration) {
			return r, ErrInvalid
		}
		if !g.Revoked {
			p := g.Plan
			spans = append(spans, interval{Period{g.Start, g.End}, g.ID, &p, false})
		}
	}
	slices.SortFunc(spans, func(a, b interval) int { return a.Start.Compare(b.Start) })
	// Only the connected component containing now confers rights. Retain a past
	// interval for diagnostics after expiry without authorizing it.
	var start, end *time.Time
	var refs []uuid.UUID
	var chosen *Plan
	graceOnly := true
	for _, s := range spans {
		if in.Now.Before(s.Start) || !in.Now.Before(s.End) {
			continue
		}
		if chosen != nil && s.plan != nil && chosen.ID != s.plan.ID {
			return r, ErrInvalid
		}
		chosen = s.plan
		refs = append(refs, s.ref)
		graceOnly = graceOnly && s.grace
		if start == nil || s.Start.Before(*start) {
			start = utc(s.Start)
		}
		if end == nil || s.End.After(*end) {
			end = utc(s.End)
		}
	}
	if end != nil {
		for _, s := range spans {
			if s.End.After(*end) && !s.Start.After(*end) && s.plan != nil && chosen != nil && s.plan.ID == chosen.ID {
				end = utc(s.End)
				refs = append(refs, s.ref)
			}
		}
		r.ValidFrom = start
		r.EffectiveServiceEnd = end
		r.SourceRefs = ids(refs)
		r.Plan = chosen
		r.State = "active"
		if graceOnly {
			r.State = "grace"
		} else if r.CancelAtPeriodEnd {
			r.State = "cancel_at_period_end"
		}
	} else {
		for _, s := range spans {
			if !s.End.After(in.Now) && (r.EffectiveServiceEnd == nil || s.End.After(*r.EffectiveServiceEnd)) {
				r.ValidFrom = utc(s.Start)
				r.EffectiveServiceEnd = utc(s.End)
				r.SourceRefs = []uuid.UUID{s.ref}
				r.State = "expired"
			}
		}
	}
	if r.Plan != nil {
		p := *r.Plan
		p.ExitGroups = ids(p.ExitGroups)
		r.Plan = &p
		r.DeviceLimit = p.DeviceLimit
		r.ExitGroups = p.ExitGroups
	}
	if r.Plan == nil && (r.State == "active" || r.State == "grace" || r.State == "cancel_at_period_end") {
		return r, ErrInvalid
	}
	if in.AdminSuspended || in.MemberStatus == "suspended" {
		r.State = "suspended"
		r.AdminSuspended = true
	}
	if in.MemberStatus == "closed" {
		r.State = "closed"
	}
	return r, nil
}
