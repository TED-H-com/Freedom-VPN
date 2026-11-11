package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"reflect"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

type publicKey struct {
	ID           string          `json:"key_id"`
	Purpose      signing.Purpose `json:"purpose"`
	State        string          `json:"state"`
	NotBefore    time.Time       `json:"not_before"`
	NotAfter     time.Time       `json:"not_after"`
	PublicSHA256 string          `json:"public_sha256"`
	RevokedAt    *time.Time      `json:"revoked_at,omitempty"`
}

func public(k *signing.KeyRecord) *publicKey {
	if k == nil {
		return nil
	}
	sum := sha256.Sum256(k.Public)
	return &publicKey{k.ID, k.Purpose, k.State, k.NotBefore, k.NotAfter, hex.EncodeToString(sum[:]), k.RevokedAt}
}

type changeDetails struct {
	OldRevision decstr.U64 `json:"old_revision"`
	NewRevision decstr.U64 `json:"new_revision"`
	Key         *publicKey `json:"key"`
	Previous    *publicKey `json:"previous_key"`
	NodeTrust   string     `json:"node_trust"`
	NoSigner    bool       `json:"purpose_has_no_signer"`
}
type document struct {
	Kind              string         `json:"kind"`
	Outcome           string         `json:"outcome"`
	Message           string         `json:"message"`
	Change            *changeDetails `json:"change,omitempty"`
	UnreferencedKeyID string         `json:"unreferenced_key_id,omitempty"`
	AttemptedKeyID    string         `json:"attempted_key_id,omitempty"`
	ReadBack          string         `json:"read_back,omitempty"`
}

func base(o options) document { return document{Kind: "fvpn-file-vault-" + o.subcommand + "-v1"} }
func confirmed(o options, r signing.ChangeResult, durability bool) (document, int) {
	d := base(o)
	d.Outcome, d.Message = "applied", "New key active."
	if o.subcommand == "revoke" {
		d.Outcome, d.Message = "revoked", "Key revoked permanently."
	}
	code := exitOK
	if r.Outcome == signing.ChangeAlreadyRevoked {
		d.Outcome, d.Message, code = "already_revoked", "Key already revoked; nothing changed.", exitAlreadyRevoked
	}
	if durability {
		d.Outcome, code, d.ReadBack = "durability_not_confirmed", exitDurability, "matched"
		if o.subcommand == "rotate" {
			d.Message = "New key observed active; durability not confirmed. Do not rotate again."
		} else {
			d.Message = "Key observed revoked; durability not confirmed. Do not revoke again."
		}
	}
	trust := "nodes_require_new_bundle"
	if o.purpose == signing.LedgerEpoch {
		trust = "node_bundle_unchanged"
	}
	d.Change = &changeDetails{OldRevision: r.OldRevision, NewRevision: r.NewRevision, Key: public(r.Key), Previous: public(r.Previous), NodeTrust: trust}
	// Only an applied revocation of an active key establishes this fact. An
	// already-revoked key may coexist with a later replacement active signer.
	d.Change.NoSigner = o.subcommand == "revoke" && r.Previous != nil && r.Previous.State == "active"
	return d, code
}
func classify(o options, r signing.ChangeResult) (document, int) {
	d := base(o)
	switch r.Outcome {
	case signing.ChangeApplied, signing.ChangeAlreadyRevoked:
		return confirmed(o, r, false)
	case signing.ChangeRefused:
		d.Outcome, d.Message = "precondition_refused", "Nothing changed. Run fvpn-vault inspect before deciding what to do next."
		return d, exitRefused
	case signing.ChangeNotApplied:
		d.Outcome, d.Message = "not_applied", "Ledger not replaced. Run fvpn-vault inspect; review any unreferenced key file manually."
		d.UnreferencedKeyID = r.CreatedKeyID
		return d, exitNotApplied
	default:
		d.Outcome, d.Message = "unknown", "Replacement may have occurred; read-back did not confirm the expected change. Run fvpn-vault inspect; do not repeat blindly."
		if r.Key != nil {
			d.AttemptedKeyID = r.Key.ID
		}
		return d, exitUnknown
	}
}
func confirms(s signing.FileSnapshot, r signing.ChangeResult) bool {
	if r.Key == nil || r.OldRevision == decstr.Max || r.NewRevision != r.OldRevision+1 || s.Revision != r.NewRevision {
		return false
	}
	for _, k := range s.Keys {
		if k.ID == r.Key.ID && reflect.DeepEqual(k, *r.Key) {
			return true
		}
	}
	return false
}
func writeResult(stdout, stderr io.Writer, d document, code int) int {
	raw, err := json.MarshalIndent(d, "", "  ")
	if err == nil {
		raw = append(raw, '\n')
		var n int
		n, err = stdout.Write(raw)
		if err == nil && n == len(raw) {
			return code
		}
	}
	// outcome is constructed internally from a closed set, never provider text.
	return fail(stderr, exitOutput, "result output failed: "+d.Outcome+"; run fvpn-vault inspect; do not repeat blindly")
}
