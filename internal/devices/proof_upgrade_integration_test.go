//go:build integration

package devices

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

func TestD052_F03UpgradeLegacyHashAfterProofRefresh(t *testing.T) {
	const issuer, subject = "https://issuer.example.invalid", "TEST-upgrade-subject"
	env := newD052Env(t, issuer, 3)
	ctx := t.Context()
	member := env.loginMember(issuer, "portal", subject)
	env.grantMember(member, env.revision, time.Hour)
	env.addUpstreamPeer(subject, "TEST-upgrade-peer", "100.64.0.126")
	if err := env.adapter.Discover(ctx, env.devices); err != nil {
		t.Fatal(err)
	}
	if err := env.devices.SyncMember(ctx, member); err != nil {
		t.Fatal(err)
	}
	d, _, err := env.adapter.device(ctx, "test-deploy", "TEST-upgrade-peer")
	if err != nil {
		t.Fatal(err)
	}
	binding, ok, err := env.adapter.Bindings.Get(ctx, "test-deploy", "users", member.String())
	if err != nil || !ok {
		t.Fatal("synced user binding missing", err)
	}
	immutableHash := binding.DesiredHash
	// Historical input: simulate the exact adapter hash persisted before F-03.
	raw, err := canonjson.Marshal([]string{d.IdentityBindingID, "1", subject})
	if err != nil {
		t.Fatal(err)
	}
	hash, err := canonjson.Hash(raw)
	if err != nil {
		t.Fatal(err)
	}
	binding.DesiredHash = hex.EncodeToString(hash[:])
	if err = env.adapter.Bindings.Put(ctx, binding); err != nil {
		t.Fatal(err)
	}
	if _, err = env.pool.Exec(ctx, `UPDATE netbird_user_bindings SET mapping_proof='TEST ONLY old proof' WHERE member_id=$1`, member); err != nil {
		t.Fatal(err)
	}
	if result, err := env.adapter.bindUser(ctx, subject); err != nil || result != identityUpdated {
		t.Fatal("proof refresh", result, err)
	}
	if err = env.adapter.SetAuthorization(ctx, "test-deploy", "TEST-upgrade-peer", true, []uuid.UUID{env.group}); err != nil {
		t.Fatal("legacy adapter hash bricked authorization", err)
	}
	binding, ok, err = env.adapter.Bindings.Get(ctx, "test-deploy", "users", member.String())
	if err != nil || !ok || binding.DesiredHash != immutableHash {
		t.Fatal("legacy hash not converted", err)
	}
	store := NetBirdBindings{Pool: env.pool}
	if verified, err := store.VerifyMemberIdentity(ctx, "test-deploy", d); err != nil || verified {
		t.Fatal("stale revision accepted for conversion", err)
	}
	d.MappingRevision = "2"
	d.IdentityBindingID = uuid.NewString()
	if verified, err := store.VerifyMemberIdentity(ctx, "test-deploy", d); err != nil || verified {
		t.Fatal("foreign identity accepted for conversion", err)
	}
}

func TestD052_F07IdentityAuditIsAllowlistedAndIdempotent(t *testing.T) {
	const issuer, subject = "https://issuer.example.invalid", "TEST-private-mapping-subject"
	env := newD052Env(t, issuer, 3)
	ctx := t.Context()
	member := env.loginMember(issuer, "portal", subject)
	if result, err := env.adapter.bindUser(ctx, subject); err != nil || result != identityCreated {
		t.Fatal(result, err)
	}
	if result, err := env.adapter.bindUser(ctx, subject); err != nil || result != identityUnchanged {
		t.Fatal(result, err)
	}
	if _, err := env.pool.Exec(ctx, `UPDATE netbird_user_bindings SET mapping_proof='TEST ONLY old proof' WHERE member_id=$1`, member); err != nil {
		t.Fatal(err)
	}
	if result, err := env.adapter.bindUser(ctx, subject); err != nil || result != identityUpdated {
		t.Fatal(result, err)
	}
	if result, err := env.adapter.bindUser(ctx, "TEST-private-unknown-subject"); err != nil || result != identityUnmapped {
		t.Fatal(result, err)
	}
	store := audit.Store{Pool: env.pool}
	page, err := store.List(ctx, api.ListAuditEventsParams{})
	if err != nil || store.AnomalyCount() != 0 {
		t.Fatal("malformed audit rows", err, store.AnomalyCount())
	}
	seen := map[string]int{}
	for _, event := range page.Items {
		if !strings.HasPrefix(event.Action, "identity_binding.") {
			continue
		}
		seen[event.Action]++
		raw, err := json.Marshal(event)
		if err != nil || strings.Contains(string(raw), "TEST-private") || strings.Contains(string(raw), issuer) {
			t.Fatal("identity audit disclosed provider identity", err)
		}
		if event.Action == "identity_binding.rejected" && event.Result != api.AuditEventResultDenied {
			t.Fatal("mapping rejection was not denied")
		}
	}
	for _, action := range []string{"identity_binding.created", "identity_binding.proof_updated", "identity_binding.rejected"} {
		if seen[action] != 1 {
			t.Fatal("missing or duplicate identity audit", action, seen[action])
		}
	}
}
