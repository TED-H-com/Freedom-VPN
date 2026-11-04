package isolation

import "time"

// MemberExpirySchedule records the pre-grant, read-only renewal anchor. These
// are TEST ONLY schedule values for this fixture's 60-second node lease.
type MemberExpirySchedule struct {
	Name         string    `json:"name"`
	Observed     time.Time `json:"observed_at"`
	Issued       time.Time `json:"anchor_issued_at"`
	Expires      time.Time `json:"anchor_expires_at"`
	NextExpected time.Time `json:"next_expected_renewal"`
	Deadline     time.Time `json:"member_deadline"`
}

// PlanMemberExpiry anchors a single new grant to a fresh accepted renewal,
// instead of an unrelated wall-clock phase. For issue time A, D=floor(A+44s).
// The unchanged predicate needs issue > D-35s and acceptance seen < D-5s.
// This admits a binding-bearing renewal at A+[10s,34s], seen within 2s,
// even if grant/peer setup and the counted control take until A+34s:
// D-35s <= A+9s, while D-5s > A+38s. Thus there is at least 1s of
// lease-extension margin and more than 2s of observation margin. This covers
// the retained A+10.460s, A+17.149s and runner A+19.133s renewals, as well
// as ordinary A+[27s,33s] renewal plus a 1s step delay. Earlier grant-only
// renewals can be ignored; waiting for their successors at A+55s put the
// binding-bearing renewal just outside the eligible window on the runner.
// These are TEST ONLY replay assumptions, not a production latency bound.
// Slower setup/acceptance still blocks under the unchanged live predicate.
// No lease or existing grant is changed or extended.
func PlanMemberExpiry(issued, expires, observed time.Time, accepted bool) (MemberExpirySchedule, bool) {
	issued, expires, observed = issued.UTC(), expires.UTC(), observed.UTC()
	p := MemberExpirySchedule{Name: "fresh-renewal anchor", Observed: observed, Issued: issued, Expires: expires}
	if !accepted || issued.IsZero() || expires.Sub(issued) != 60*time.Second || observed.Before(issued) || observed.Sub(issued) > 5*time.Second {
		return p, false
	}
	p.NextExpected = issued.Add(30 * time.Second)
	p.Deadline = issued.Add(44 * time.Second).UTC().Truncate(time.Second)
	return p, true
}
