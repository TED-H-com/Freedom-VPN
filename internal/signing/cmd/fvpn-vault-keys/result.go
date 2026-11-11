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

type releaseKeyChange struct {
	Key      *publicKey `json:"key"`
	Previous *publicKey `json:"previous_key"`
}

type changeDetails struct {
	OldRevision        decstr.U64         `json:"old_revision"`
	NewRevision        decstr.U64         `json:"new_revision"`
	OldReleaseRevision decstr.U64         `json:"old_release_revision,omitempty"`
	NewReleaseRevision decstr.U64         `json:"new_release_revision,omitempty"`
	Key                *publicKey         `json:"key,omitempty"`
	Previous           *publicKey         `json:"previous_key"`
	Keys               []releaseKeyChange `json:"keys,omitempty"`
	UsableRootKeyIDs   []string           `json:"usable_root_key_ids,omitempty"`
	NodeTrust          string             `json:"node_trust"`
	NoSigner           bool               `json:"purpose_has_no_signer,omitempty"`
}

func (c changeDetails) MarshalJSON() ([]byte, error) {
	if c.Keys != nil {
		roots := c.UsableRootKeyIDs
		if roots == nil {
			roots = []string{}
		}
		return json.Marshal(struct {
			OldRevision        decstr.U64         `json:"old_revision"`
			NewRevision        decstr.U64         `json:"new_revision"`
			OldReleaseRevision decstr.U64         `json:"old_release_revision"`
			NewReleaseRevision decstr.U64         `json:"new_release_revision"`
			Keys               []releaseKeyChange `json:"keys"`
			UsableRootKeyIDs   []string           `json:"usable_root_key_ids"`
			NodeTrust          string             `json:"node_trust"`
		}{
			OldRevision:        c.OldRevision,
			NewRevision:        c.NewRevision,
			OldReleaseRevision: c.OldReleaseRevision,
			NewReleaseRevision: c.NewReleaseRevision,
			Keys:               c.Keys,
			UsableRootKeyIDs:   roots,
			NodeTrust:          c.NodeTrust,
		})
	}
	return json.Marshal(struct {
		OldRevision decstr.U64 `json:"old_revision"`
		NewRevision decstr.U64 `json:"new_revision"`
		Key         *publicKey `json:"key"`
		Previous    *publicKey `json:"previous_key"`
		NodeTrust   string     `json:"node_trust"`
		NoSigner    bool       `json:"purpose_has_no_signer,omitempty"`
	}{
		OldRevision: c.OldRevision,
		NewRevision: c.NewRevision,
		Key:         c.Key,
		Previous:    c.Previous,
		NodeTrust:   c.NodeTrust,
		NoSigner:    c.NoSigner,
	})
}

type document struct {
	Kind              string         `json:"kind"`
	Outcome           string         `json:"outcome"`
	Message           string         `json:"message"`
	UpdateSHA256      string         `json:"update_sha256,omitempty"`
	Reason            string         `json:"reason,omitempty"`
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
func rejectReason(r signing.ReleaseRejectReason) string {
	switch r {
	case signing.RejectSignature:
		return "signature"
	case signing.RejectPayload:
		return "payload"
	case signing.RejectRevision:
		return "revision"
	case signing.RejectRecord:
		return "record"
	case signing.RejectNoUsableRoot:
		return "no_usable_root"
	default:
		return "record"
	}
}
func confirmedRelease(o options, updateSHA256 string, r signing.ReleaseUpdateResult, durability bool) (document, int) {
	d := base(o)
	d.UpdateSHA256 = updateSHA256
	d.Outcome, d.Message = "applied", "Release trust update applied. Export and deliver the updated node trust bundle and confirm adoption before the next handover step."
	code := exitOK
	if r.Outcome == signing.ChangeAlreadyApplied {
		d.Outcome, d.Message, code = "already_applied", "Release trust update already applied; nothing changed.", exitAlreadyApplied
	}
	if durability {
		d.Outcome, code, d.ReadBack = "durability_not_confirmed", exitDurability, "matched"
		d.Message = "Release trust update observed in ledger; durability not confirmed. Do not repeat blindly."
	}
	keys := make([]releaseKeyChange, len(r.Keys))
	for i, k := range r.Keys {
		after := k.After
		keys[i] = releaseKeyChange{Key: public(&after), Previous: public(k.Before)}
	}
	roots := append([]string{}, r.UsableRootIDs...)
	d.Change = &changeDetails{
		OldRevision:        r.OldRevision,
		NewRevision:        r.NewRevision,
		OldReleaseRevision: r.OldReleaseRevision,
		NewReleaseRevision: r.NewReleaseRevision,
		Keys:               keys,
		UsableRootKeyIDs:   roots,
		NodeTrust:          "nodes_require_new_bundle",
	}
	return d, code
}
func classifyRelease(o options, updateSHA256 string, r signing.ReleaseUpdateResult) (document, int) {
	d := base(o)
	d.UpdateSHA256 = updateSHA256
	switch r.Outcome {
	case signing.ChangeApplied, signing.ChangeAlreadyApplied:
		return confirmedRelease(o, updateSHA256, r, false)
	case signing.ChangeRefused:
		d.Outcome, d.Message = "precondition_refused", "Nothing changed. Run fvpn-vault inspect, check the update with fvpn-release trust-verify against a fresh export, and decide what to do next."
		return d, exitRefused
	case signing.ChangeRejected:
		d.Outcome, d.Message = "rejected", "Release trust update rejected; nothing changed."
		d.Reason = rejectReason(r.RejectReason)
		return d, exitRejected
	case signing.ChangeNotApplied:
		d.Outcome, d.Message = "not_applied", "Ledger not replaced. Run fvpn-vault inspect before deciding what to do next."
		return d, exitNotApplied
	default:
		d.Outcome, d.Message = "unknown", "Replacement may have occurred; read-back did not confirm the expected change. Run fvpn-vault inspect; do not repeat blindly."
		return d, exitUnknown
	}
}
func confirmsRelease(s signing.FileSnapshot, r signing.ReleaseUpdateResult) bool {
	if r.OldRevision == decstr.Max || r.NewRevision != r.OldRevision+1 ||
		s.Revision != r.NewRevision || r.NewReleaseRevision <= r.OldReleaseRevision ||
		s.ReleaseRevision != r.NewReleaseRevision {
		return false
	}
	for _, change := range r.Keys {
		matched := false
		for _, k := range s.Keys {
			if k.ID == change.After.ID {
				matched = reflect.DeepEqual(k, change.After)
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
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
