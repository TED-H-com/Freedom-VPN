//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

type IdentityFingerprints struct {
	MachineID        string `json:"machine_id_sha256"`
	AgentSPKI        string `json:"agent_spki_sha256"`
	NetBirdPublicKey string `json:"netbird_public_key_sha256"`
}

// MachineIdentityFingerprints hashes canonical machine-id text, DER Agent SPKI,
// and raw X25519 NetBird public key, inside the node; no key material is exported.
func (s *Stack) MachineIdentityFingerprints(ctx context.Context, slot string) (IdentityFingerprints, error) {
	var out IdentityFingerprints
	err := s.nodeHelper(ctx, slot, "identity-fingerprints", nil, &out)
	return out, err
}

type ClonedIdentityFixture struct {
	Source, Clone string
	Before        IdentityFingerprints
	stack         *Stack
	incarnation   uuid.UUID
}
type CloneObservation struct {
	Source, Clone           IdentityFingerprints
	SameIdentity            bool `json:"same_identity"`
	EndpointRows            int  `json:"endpoint_rows"`
	Conflicts               int  `json:"conflicts"`
	MappingFrozen           bool `json:"mapping_frozen"`
	SourceStage, CloneStage string
}

// StartClonedIdentityFixture stops the source for a consistent private snapshot,
// copies durable identity/NetBird state before node-clone boots, then resumes source.
// It requires unused node-clone; use a separate fresh stack for clean-second-exit tests.
// This is a destructive fault for the disposable stack: never publish the clone.
func (s *Stack) StartClonedIdentityFixture(ctx context.Context, slot string) (*ClonedIdentityFixture, error) {
	if slot != "node" {
		return nil, fmt.Errorf("clone fixture requires primary source and unused node-clone")
	}
	before, err := s.MachineIdentityFingerprints(ctx, slot)
	if err != nil {
		return nil, err
	}
	raw, err := s.lab.docker(s.lab.node, "cat", "/var/lib/vpn-node/identity/node.json")
	var id struct {
		Inc uuid.UUID `json:"node_incarnation_id"`
	}
	if err != nil || json.Unmarshal(raw, &id) != nil || id.Inc == uuid.Nil {
		return nil, fmt.Errorf("clone source identity unavailable")
	}
	if err = s.startSecondExit(ctx, true); err != nil {
		return nil, err
	}
	return &ClonedIdentityFixture{Source: slot, Clone: "node-clone", Before: before, stack: s, incarnation: id.Inc}, nil
}

// Observe is read-only. Conflict/freeze values are actual product rows, not an
// inferred safety verdict from sharing an identity or a zero peer count.
func (f *ClonedIdentityFixture) Observe(ctx context.Context) (CloneObservation, error) {
	var out CloneObservation
	var err error
	if f == nil || f.stack == nil {
		return out, fmt.Errorf("clone fixture not initialized")
	}
	out.Source, err = f.stack.MachineIdentityFingerprints(ctx, f.Source)
	if err != nil {
		return out, err
	}
	out.Clone, err = f.stack.MachineIdentityFingerprints(ctx, f.Clone)
	if err != nil {
		return out, err
	}
	out.SameIdentity = out.Source == out.Clone && out.Source == f.Before
	out.SourceStage = f.stack.lab.stage(f.stack.lab.prefix+"-"+f.Source, "/var/lib/vpn-node/e2e/status.json")
	out.CloneStage = f.stack.lab.stage(f.stack.lab.prefix+"-"+f.Clone, "/var/lib/vpn-node/e2e/status.json")
	db, err := f.stack.lab.db(ctx)
	if err != nil {
		return out, err
	}
	defer db.Close(ctx)
	err = db.QueryRow(ctx, `SELECT (SELECT count(*) FROM node_identity_endpoints WHERE incarnation_id=$1), (SELECT count(*) FROM node_identity_conflicts WHERE incarnation_id=$1), EXISTS(SELECT 1 FROM node_identity_endpoints WHERE incarnation_id=$1 AND mapping_frozen)`, f.incarnation).Scan(&out.EndpointRows, &out.Conflicts, &out.MappingFrozen)
	return out, err
}
