//go:build e2e

package secops

import (
	"fmt"
	"strconv"
	"time"
)

func decimal18(n int64) string { return strconv.FormatInt(n, 10) }

func faultDeadline18(lease nodeLease16, accepted bool, heartbeat map[string]string, now int64, grantEnd time.Time) (int64, error) {
	duration := lease.Expires.Sub(lease.Issued)
	hb := heartbeat["policy_expires_mono_ns"]
	fail := func(reason string) (int64, error) {
		return 0, fmt.Errorf("lease %s generation=%s issued=%s cap=%ds lifetime=%s D=%d heartbeat_D=%s accepted=%t read_mono=%d: %s", lease.ID, lease.Generation, lease.Issued.Format(time.RFC3339Nano), lease.Max, duration, lease.Deadline, hb, accepted, now, reason)
	}
	if !accepted || lease.ID == "" || lease.Sent <= 0 {
		return fail("not a locally accepted lease")
	}
	if lease.Max != 60 || duration <= 0 || duration > 60*time.Second {
		return fail("not signed under the 60s cap")
	}
	if lease.Deadline != lease.Sent+int64(duration) || hb != decimal18(lease.Deadline) || heartbeat["serving"] != "1" {
		return fail("accepted lease does not match serving heartbeat deadline")
	}
	if now < lease.Sent || lease.Deadline <= now {
		return fail("accepted deadline is not in the future at the read")
	}
	if grantEnd.IsZero() {
		return fail("missing grant end")
	}
	d := lease.Deadline
	if grantEnd.Before(lease.Expires) {
		d = lease.Sent + int64(grantEnd.Sub(lease.Issued))
	}
	if d <= now {
		return fail("earlier grant deadline is not in the future")
	}
	return d, nil
}
