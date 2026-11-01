// Package leasehold defines content-free receipts for the TEST ONLY proxy seam.
package leasehold

import "time"

const Directory = "/var/lib/vpn-node/e2e/matching-lease-hold"

type Arm struct {
	ID             string    `json:"id"`
	BaseGeneration string    `json:"base_generation"`
	HoldSeconds    int       `json:"hold_seconds"`
	ArmedAt        time.Time `json:"armed_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type Event struct {
	Kind             string    `json:"kind"`
	At               time.Time `json:"at"`
	Generation       string    `json:"generation,omitempty"`
	LeaseID          string    `json:"lease_id,omitempty"`
	SHA256           string    `json:"body_sha256,omitempty"`
	Bytes            int       `json:"bytes,omitempty"`
	UpstreamStatus   int       `json:"upstream_status,omitempty"`
	DownstreamStatus int       `json:"downstream_status,omitempty"`
	Reason           string    `json:"reason,omitempty"`
}

type Evidence struct {
	Arm              Arm        `json:"arm"`
	TargetGeneration string     `json:"target_generation,omitempty"`
	HoldStartedAt    *time.Time `json:"hold_started_at,omitempty"`
	HoldUntil        *time.Time `json:"hold_until,omitempty"`
	ReleasedAt       *time.Time `json:"released_at,omitempty"`
	Events           []Event    `json:"events"`
}
