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
// instead of an unrelated wall-clock phase. For issue time A, D=floor(A+55s),
// so the unchanged eligible issue window (D-35s,D-5s] contains A+[27s,33s].
// It also contains the renewal after an extra A+10.5s renewal, including a
// one-second step delay. No lease or existing grant is changed or extended.
func PlanMemberExpiry(issued, expires, observed time.Time, accepted bool) (MemberExpirySchedule, bool) {
	p := MemberExpirySchedule{Name: "fresh-renewal anchor", Observed: observed, Issued: issued, Expires: expires}
	if !accepted || issued.IsZero() || expires.Sub(issued) != 60*time.Second || observed.Before(issued) || observed.Sub(issued) > 5*time.Second {
		return p, false
	}
	p.NextExpected = issued.Add(30 * time.Second)
	p.Deadline = issued.Add(55 * time.Second).UTC().Truncate(time.Second)
	return p, true
}
