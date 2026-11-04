package isolation

import (
	"fmt"
	"strings"
	"time"
)

// MemberExpiryServingRead is a direct local read immediately before the five
// post-deadline probes or immediately after one of them, while control is down.
type MemberExpiryServingRead struct {
	At           time.Time `json:"read_start"`
	End          time.Time `json:"read_end"`
	Heartbeat    string    `json:"heartbeat"`
	ForwardChain string    `json:"forward_chain"`
	ReadError    string    `json:"read_error,omitempty"`
}

// MemberExpiryStatus adds the serving-node requirement to the unchanged member
// denial verdict. Whole-node withdrawal cannot prove per-member expiry (§19 AT13).
// Missing or contradictory local evidence fails this requirement, never passes.
func MemberExpiryStatus(denialStatus string, reads []MemberExpiryServingRead) (string, string) {
	if len(reads) == 0 {
		return StatusFail, "member_expiry_serving_reads: no local serving evidence"
	}
	for i, read := range reads {
		serving, reason, count := "", "", 0
		for _, line := range strings.Split(read.Heartbeat, "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "serving" {
				serving = fields[1]
				count++
			}
			if len(fields) == 2 && fields[0] == "reason" {
				reason = fields[1]
			}
		}
		view := ParseForwardChain(read.ForwardChain)
		if read.ReadError != "" || count != 1 || serving != "1" || !view.Present || view.Baseline || view.MemberAccept == 0 {
			return StatusFail, fmt.Sprintf("member_expiry_serving_reads[%d]: heartbeat serving=%q reason=%q; forward_chain present=%t baseline=%t member_accept=%d; read_error=%q", i, serving, reason, view.Present, view.Baseline, view.MemberAccept, read.ReadError)
		}
	}
	return denialStatus, fmt.Sprintf("member_expiry_serving_reads: all %d local reads show heartbeat serving 1 and a non-baseline member forward chain", len(reads))
}
