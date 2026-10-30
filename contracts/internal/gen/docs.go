package gen

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// TestNow is the instant at which the signed examples and vectors are
// verified. Test keys are valid in [TestKeyNotBefore, TestKeyNotAfter).
const (
	TestNow          = "2026-10-28T15:00:00Z"
	TestKeyNotBefore = "2026-01-01T00:00:00Z"
	TestKeyNotAfter  = "2028-01-01T00:00:00Z"
)

// TestOnlyLabel marks every committed seed.
const TestOnlyLabel = "TEST ONLY - NOT A SECRET"

// TestKey is a deterministic Ed25519 key for vectors and examples. The seed
// is SHA-256 of "TEST ONLY - NOT A SECRET: fvpn contracts <label> test key".
type TestKey struct {
	Label   string `json:"label"`
	KeyID   string `json:"key_id"`
	Purpose string `json:"purpose"`
	Trusted bool   `json:"trusted"`
	SeedHex string `json:"seed_hex"`
	PubB64  string `json:"public_key_b64"`
	priv    ed25519.PrivateKey
}

// Private returns the private key (tests only).
func (k TestKey) Private() ed25519.PrivateKey { return k.priv }

func newTestKey(label, keyID, purpose string, trusted bool) TestKey {
	seed := sha256.Sum256([]byte(TestOnlyLabel + ": fvpn contracts " + label + " test key"))
	priv := ed25519.NewKeyFromSeed(seed[:])
	return TestKey{
		Label: label, KeyID: keyID, Purpose: purpose, Trusted: trusted,
		SeedHex: hex.EncodeToString(seed[:]),
		PubB64:  base64.StdEncoding.EncodeToString(priv.Public().(ed25519.PublicKey)),
		priv:    priv,
	}
}

// TestKeys returns the contract test keys. Each purpose has its own key, as
// in production (AGENTS.md §3); "untrusted" is never in the trust store.
func TestKeys() map[string]TestKey {
	ks := []TestKey{
		newTestKey("release", "release-signing-test-2026", "release", true),
		newTestKey("desired-state", "policy-signing-test-2026", "desired-state", true),
		newTestKey("lease", "lease-signing-test-2026", "lease", true),
		newTestKey("recovery", "recovery-signing-test-2026", "recovery", true),
		newTestKey("untrusted", "untrusted-test-key", "release", false),
	}
	m := map[string]TestKey{}
	for _, k := range ks {
		m[k.Label] = k
	}
	return m
}

// Fixed example IDs. The node, incarnation, enrollment request, release,
// release-support, operation, request, ledger epoch and recovery record IDs
// are the UUIDs used in the spec's own examples (§14.7, §14.8, §17.3).
const (
	idNode           = "93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb"
	idIncarnation    = "d1c5d642-f6ac-4ca4-bcd6-5460d491a2e0"
	idReleaseR42     = "6d2797d0-b9a5-4724-96af-9b3039d351b1"
	idReleaseR41     = "cdb25ffa-5395-400c-ad3e-b32899f14bb8"
	idReleaseSupport = "8e210b2c-f844-4ad9-88f6-7e6fa0cbbe68"
	idLedgerEpoch    = "7612b7d9-1e70-4390-b86b-7c3456e4df27"
	idRecoveryRecord = "578bbd2d-b93c-42c5-84c6-5f899e388208"
	idMemberA        = "dc01a74b-9e6f-41ac-81cd-03291d0a88c0"
	idDeviceA1       = "2df6c7df-e7e8-4bd4-9f2c-a5836dd0d8ea"
	idDeviceA2       = "18fa8abd-79d4-43a5-8403-886930f651f9"
	idMemberB        = "4e8ca24f-b523-45ef-a782-068cc1c9f809"
	idDeviceB1       = "de484d49-0949-45b1-b38a-40709eb79a7c"
	idMeterEpoch14   = "b91a511c-0af5-48af-85cb-eb69c4a0070c"
	idMeterEpoch15   = "909f68b7-a5b4-44da-afd5-e1683e42c6fd"
	idBatch          = "85ef720a-31a0-4250-90d2-4b92c58695bc"
	idTarget         = "7648dcdb-c82e-4611-8608-06895697e0a9"
	idJob            = "73986232-fd84-4add-b639-9f2cc23476a0"
	idResult         = "0af1eb76-0bcd-4378-8442-eb0a2663a35d"
	idBoot           = "dc4f6bc5-954b-46e8-825d-57fdf335bee5"
	idLease          = "c4506654-d908-46c6-95e5-e9fa6b30f9ed"
	idResult2        = "4e953e98-be1b-4e89-95c8-927e06e632e8"
	idJob2           = "02d164bb-b3bf-44ba-9470-5dba6122c3e0"
)

// Signed and hashed example documents. They follow the spec examples
// (§10.5, §13.3, §13.10.2, §13.10.3, §14.7, §17.3) with the `demo`
// placeholders replaced by UUIDs.

var releaseManifest = `{
  "manifest_schema_version": 1,
  "release_id": "` + idReleaseR42 + `",
  "product_version": "1.1.0",
  "release_sequence": "42",
  "created_at": "2026-10-28T14:00:00Z",
  "source_commit": "1111111111111111111111111111111111111111",
  "build_id": "build-test-42",
  "components": {"node_agent": "1.1.0", "node_updater": "1.1.0", "netbird": "0.76.2", "enforcement": "1.1.0"},
  "compatibility": {
    "control_api_major": 1,
    "policy_schema_versions": [1],
    "usage_event_schema_versions": [1],
    "minimum_updater_version": "1.0.0",
    "state_schema_from": [1],
    "state_schema_to": 1,
    "migration_id": null,
    "requires_service_restart": true
  },
  "artifacts": [
    {"os_id": "ubuntu", "os_versions": ["24.04"], "architecture": "amd64", "format": "tar.zst",
     "url": "https://releases.example.invalid/r42/amd64.tar.zst", "size_bytes": "83886080",
     "installed_size_bytes": "209715200", "sha256": "2222222222222222222222222222222222222222222222222222222222222222"},
    {"os_id": "ubuntu", "os_versions": ["24.04"], "architecture": "arm64", "format": "tar.zst",
     "url": "https://releases.example.invalid/r42/arm64.tar.zst", "size_bytes": "78643200",
     "installed_size_bytes": "199229440", "sha256": "3333333333333333333333333333333333333333333333333333333333333333"}
  ],
  "rollback": {
    "allowed_release_ids": ["` + idReleaseR41 + `"],
    "state_readable_by_previous": true,
    "required_free_bytes": "419430400",
    "health_profile": "vpn-node-standard-v1"
  },
  "evidence": {
    "sbom_url": "https://releases.example.invalid/r42/sbom.json",
    "sbom_sha256": "4444444444444444444444444444444444444444444444444444444444444444",
    "notices_url": "https://releases.example.invalid/r42/NOTICE.txt",
    "source_url": "https://releases.example.invalid/r42/source.tar.zst"
  }
}`

var desiredState = `{
  "schema_version": 1,
  "node_id": "` + idNode + `",
  "node_incarnation_id": "` + idIncarnation + `",
  "config_generation": "128",
  "publication_intent": "published",
  "safety_profile": "internet-only-v1",
  "policy_refs": {"authorization_generation": "52", "fup_generation": "9", "dns_generation": "12"},
  "binding_revision": "407",
  "desired_release_id": "` + idReleaseR42 + `",
  "supported_release_policy_id": "` + idReleaseSupport + `",
  "serving_release_ids": ["` + idReleaseR41 + `", "` + idReleaseR42 + `"],
  "lease_renewal_endpoint": "/api/v1/agent/lease-renewals",
  "accounting_period": {"period_id": "2026-10", "period_start": "2026-10-01T00:00:00Z", "period_end": "2026-11-01T00:00:00Z"},
  "device_bindings": [
    {"member_id": "` + idMemberA + `", "device_id": "` + idDeviceA1 + `", "peer_id": "peer-test-a1",
     "allowed_ipv4": ["100.64.10.11"], "allowed_ipv6": [], "binding_revision": "405",
     "valid_from": "2026-10-20T00:00:00Z", "valid_until": "2026-11-20T00:00:00Z"},
    {"member_id": "` + idMemberA + `", "device_id": "` + idDeviceA2 + `", "peer_id": "peer-test-a2",
     "allowed_ipv4": ["100.64.10.12"], "allowed_ipv6": [], "binding_revision": "406",
     "valid_from": "2026-10-21T00:00:00Z", "valid_until": "2026-11-20T00:00:00Z"},
    {"member_id": "` + idMemberB + `", "device_id": "` + idDeviceB1 + `", "peer_id": "peer-test-b1",
     "allowed_ipv4": ["100.64.10.21"], "allowed_ipv6": [], "binding_revision": "407",
     "valid_from": "2026-10-22T00:00:00Z", "valid_until": "2026-11-20T00:00:00Z"}
  ],
  "member_rate_policies": [
    {"member_id": "` + idMemberA + `", "generation": "9", "mode": "enforce", "tier_index": 0,
     "down_bps": null, "up_bps": null, "priority": 0, "reason": "normal", "period_id": "2026-10",
     "valid_from": "2026-10-01T00:00:00Z", "valid_until": "2026-11-01T00:00:00Z"},
    {"member_id": "` + idMemberB + `", "generation": "9", "mode": "enforce", "tier_index": 2,
     "down_bps": 20000000, "up_bps": 10000000, "priority": 2, "reason": "fair_use_tier", "period_id": "2026-10",
     "valid_from": "2026-10-25T08:00:00Z", "valid_until": "2026-11-01T00:00:00Z"}
  ],
  "node_capacity": {"down_bps": 1000000000, "up_bps": 1000000000, "member_down_bps_cap": 200000000, "member_up_bps_cap": 100000000},
  "dns": {"resolver_ipv4": ["100.64.0.53", "100.64.1.53"], "resolver_ipv6": []},
  "update": {
    "node_id": "` + idNode + `",
    "incarnation_id": "` + idIncarnation + `",
    "batch_id": "` + idBatch + `",
    "target_id": "` + idTarget + `",
    "job_id": "` + idJob + `",
    "attempt": 1,
    "generation": "7",
    "release_id": "` + idReleaseR42 + `",
    "manifest_url": "https://releases.example.invalid/r42/manifest.json",
    "manifest_sha256": "%MANIFEST_SHA256%",
    "issued_at": "2026-10-28T14:05:00Z",
    "authorization_valid_until": "2026-10-28T15:05:00Z",
    "publication_intent": "published"
  },
  "issued_at": "2026-10-28T15:00:00Z",
  "expires_at": "2026-10-29T15:00:00Z"
}`

var servingLease = `{
  "schema_version": 1,
  "lease_id": "` + idLease + `",
  "node_id": "` + idNode + `",
  "node_incarnation_id": "` + idIncarnation + `",
  "authorization_generation": "52",
  "config_generation": "128",
  "issued_at": "2026-10-28T15:00:00Z",
  "expires_at": "2026-10-29T15:00:00Z",
  "max_offline_seconds": 86400,
  "members": [
    {"member_id": "` + idMemberA + `", "entitlement_revision": "31",
     "effective_service_end": "2026-11-20T00:00:00Z", "lease_end": "2026-10-29T15:00:00Z"},
    {"member_id": "` + idMemberB + `", "entitlement_revision": "17",
     "effective_service_end": "2026-10-29T03:00:00Z", "lease_end": "2026-10-29T03:00:00Z"}
  ]
}`

var usageCheckpoint = `{
  "ledger_epoch": "` + idLedgerEpoch + `",
  "node_id": "` + idNode + `",
  "node_incarnation_id": "` + idIncarnation + `",
  "highest_contiguous_seq": "400",
  "max_seen_seq": "417",
  "missing_ranges": [{"start_seq": "401", "end_seq": "409"}],
  "recovery_record_id": "` + idRecoveryRecord + `",
  "issued_at": "2026-10-28T18:00:00Z"
}`

var usageEvent = `{
  "schema_version": 1,
  "node_id": "` + idNode + `",
  "node_incarnation_id": "` + idIncarnation + `",
  "meter_epoch_id": "` + idMeterEpoch14 + `",
  "event_seq": "1234",
  "metering_version": "forward_admission_v1",
  "interval_start": "2026-10-28T14:00:00Z",
  "interval_end": "2026-10-28T14:00:10Z",
  "quality": "complete",
  "records": [
    {"member_id": "` + idMemberA + `", "device_id": "` + idDeviceA1 + `", "binding_revision": "42",
     "period_id": "2026-10", "upload_bytes": "123456", "download_bytes": "987654"},
    {"member_id": "` + idMemberB + `", "device_id": "` + idDeviceB1 + `", "binding_revision": "43",
     "period_id": "2026-10", "upload_bytes": "9007199254740993", "download_bytes": "0"}
  ]
}`

var usageEventGap = `{
  "schema_version": 1,
  "node_id": "` + idNode + `",
  "node_incarnation_id": "` + idIncarnation + `",
  "meter_epoch_id": "` + idMeterEpoch15 + `",
  "event_seq": "1235",
  "metering_version": "forward_admission_v1",
  "interval_start": "2026-10-28T14:00:10Z",
  "interval_end": "2026-10-28T14:03:40Z",
  "quality": "gap",
  "records": [],
  "gap": {"reason": "power_loss", "last_checkpoint_at": "2026-10-28T14:00:10Z"}
}`

var updateResult = `{
  "result_id": "` + idResult + `",
  "node_id": "` + idNode + `",
  "incarnation_id": "` + idIncarnation + `",
  "boot_id": "` + idBoot + `",
  "batch_id": "` + idBatch + `",
  "target_id": "` + idTarget + `",
  "job_id": "` + idJob + `",
  "attempt": 1,
  "generation": "7",
  "state": "succeeded",
  "stage": "report",
  "requested_release_id": "` + idReleaseR42 + `",
  "current_release_id": "` + idReleaseR42 + `",
  "previous_release_id": "` + idReleaseR41 + `",
  "manifest_sha256": "%MANIFEST_SHA256%",
  "policy_generation": "18",
  "state_schema_version": 1,
  "usage_checkpoint": {
    "closed_meter_epoch_id": "` + idMeterEpoch14 + `",
    "last_durable_event_seq": "12045",
    "central_acked_through": "12040",
    "new_meter_epoch_id": "` + idMeterEpoch15 + `"
  },
  "health": {"processes": "passed", "identity_and_state": "passed", "internet_only": "passed", "dns_policy": "passed", "tunnel_probe": "passed"},
  "publication_intent": "disabled",
  "member_forwarding_enabled": false,
  "error_code": null,
  "recovery_required": false,
  "started_at": "2026-10-28T14:06:00Z",
  "finished_at": "2026-10-28T14:06:45Z"
}`

var updateResultRolledBack = `{
  "result_id": "` + idResult2 + `",
  "node_id": "` + idNode + `",
  "incarnation_id": "` + idIncarnation + `",
  "boot_id": "` + idBoot + `",
  "batch_id": "` + idBatch + `",
  "target_id": "` + idTarget + `",
  "job_id": "` + idJob2 + `",
  "attempt": 2,
  "generation": "7",
  "state": "rolled_back",
  "stage": "check",
  "requested_release_id": "` + idReleaseR42 + `",
  "current_release_id": "` + idReleaseR41 + `",
  "previous_release_id": "` + idReleaseR42 + `",
  "manifest_sha256": "%MANIFEST_SHA256%",
  "policy_generation": "18",
  "state_schema_version": 1,
  "usage_checkpoint": {
    "closed_meter_epoch_id": "` + idMeterEpoch14 + `",
    "last_durable_event_seq": "12051",
    "central_acked_through": "12049",
    "new_meter_epoch_id": "` + idMeterEpoch15 + `"
  },
  "health": {"processes": "passed", "identity_and_state": "passed", "internet_only": "passed", "dns_policy": "passed", "tunnel_probe": "passed"},
  "candidate_health": {"processes": "passed", "identity_and_state": "passed", "internet_only": "passed", "dns_policy": "passed", "tunnel_probe": "failed"},
  "publication_intent": "published",
  "member_forwarding_enabled": true,
  "error_code": "HEALTH_CHECK_FAILED",
  "recovery_required": false,
  "started_at": "2026-10-28T15:10:00Z",
  "finished_at": "2026-10-28T15:12:30Z"
}`

func mustCanon(doc string) []byte {
	c, err := canonicalize([]byte(doc))
	if err != nil {
		panic(fmt.Sprintf("gen: invalid built-in document: %v", err))
	}
	return c
}

func hexSHA(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func rawJSON(b []byte) json.RawMessage { return json.RawMessage(b) }
