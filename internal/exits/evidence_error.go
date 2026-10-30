package exits

import (
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"strconv"
)

// PublicationEvidenceError carries a classified adapter failure, never response text.
type PublicationEvidenceError string

func (e PublicationEvidenceError) Error() string { return PublicationFailureCause(e, "readback") }

// PublicationFailureCause bounds public step evidence to a fixed vocabulary.
func PublicationFailureCause(err error, phase string) string {
	var classified PublicationEvidenceError
	if errors.As(err, &classified) {
		switch string(classified) {
		case "route_missing", "route_disabled", "route_ambiguous", "route_contract_mismatch",
			"route_peer_mismatch", "route_groups_mismatch", "group_evidence_unavailable",
			"no_distribution_groups", "identity_unconfirmed", "adapter_unavailable":
			return string(classified)
		}
	}
	if phase != "publish" {
		phase = "readback"
	}
	var upstream *netbird.Error
	if errors.As(err, &upstream) {
		if phase == "publish" && upstream.Status >= 400 && upstream.Status <= 599 {
			return "publish_rejected:" + strconv.Itoa(upstream.Status)
		}
		switch upstream.Kind {
		case netbird.NotFound, netbird.Conflict, netbird.RateLimited, netbird.Unauthorized,
			netbird.Upstream, netbird.Unavailable, netbird.Invalid, netbird.Unconfirmed:
			return phase + "_failed:" + string(upstream.Kind)
		}
	}
	return phase + "_failed:unavailable"
}
