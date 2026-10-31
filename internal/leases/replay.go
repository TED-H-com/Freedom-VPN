package leases

import (
	"context"
	"encoding/json"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// ReadRenewal returns the immutable envelope for an authenticated identity's renewal without signing or extending it.
// D-088: its signed expiry can exceed the clamped serving_leases.expires_at;
// replay is issuance evidence, not a check of the authoritative central deadline.
func (s *Service) ReadRenewal(ctx context.Context, nodeID, incarnationID, renewalID uuid.UUID) (api.ServingLeaseEnvelope, error) {
	var raw []byte
	if err := s.pool.QueryRow(ctx, `SELECT envelope FROM serving_leases WHERE node_id=$1 AND incarnation_id=$2 AND renewal_request_id=$3`, nodeID, incarnationID, renewalID).Scan(&raw); err != nil {
		return api.ServingLeaseEnvelope{}, err
	}
	var out api.ServingLeaseEnvelope
	err := json.Unmarshal(raw, &out)
	return out, err
}
