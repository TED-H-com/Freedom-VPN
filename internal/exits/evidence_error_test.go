package exits

import (
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"strings"
	"testing"
)

func TestSECOPS11_PublicationFailureCauseIsBounded(t *testing.T) {
	for _, tc := range []struct {
		err         error
		phase, want string
	}{
		{&netbird.Error{Kind: netbird.Upstream, Status: 503}, "publish", "publish_rejected:503"},
		{&netbird.Error{Kind: netbird.Unconfirmed}, "readback", "readback_failed:unconfirmed"},
		{PublicationEvidenceError("no_distribution_groups"), "publish", "no_distribution_groups"},
		{PublicationEvidenceError("route_missing"), "readback", "route_missing"},
		{PublicationEvidenceError("route_disabled"), "readback", "route_disabled"},
		{errors.New(strings.Repeat("TEST ONLY sensitive upstream response ", 100)), "publish", "publish_failed:unavailable"},
		{PublicationEvidenceError("TEST ONLY untrusted reason"), "readback", "readback_failed:unavailable"},
	} {
		got := PublicationFailureCause(tc.err, tc.phase)
		if got != tc.want || len(got) > 64 {
			t.Fatal("unclassified/unbounded failure", got)
		}
	}
}
