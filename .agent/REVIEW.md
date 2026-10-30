# Adversarial Security Review — `r12-enforce-limits-review`

Reviewed areas:
1. **Node enforcement verifier and deny list** (`node/enforce/*`, `docs/poc/G2-netlab.md`)
2. **Control rate-limit classes and body limits** (`internal/httpapi/*`, `internal/config/config.go`, `pkg/canonjson/*`, `docs/control/README.md`, `docs/control/AUTH.md`)
3. **D-063 issuer address validation** (`node/updater/*`, `cmd/fvpn-updater/main.go`, `docs/updates/node-updater.md`)
4. **Pre-result heartbeat seeding** (`internal/fleetupdate/serving.go`, `docs/updates/central-lifecycle.md`)

---

## High

### H-1: `VerifyServing` accepts out-of-model forwarding rulesets (`ct state related` / `meta mark` / unconditional `accept` in `forward`, missing download `@allowed_v4` drop, missing `@denied_v4`/`@host_v4` drops and tainted `@resolver_v4` when `Probe == nil`, `prerouting` DNAT chains, `input` accepts, `interval` flags on `@allowed_v4`, and duplicate chain names)

- **Where:** `node/enforce/verify.go:47-91`, `node/enforce/verify.go:150-258`, `node/enforce/verify.go:307-310`, `node/enforce/verify.go:555-572`, `node/enforce/verify.go:643-661`, `node/enforce/parse_nft.go:107-135`, `node/enforce/parse_nft.go:190-213`, `node/enforce/parse_nft.go:545-555`
- **Scenario:**
  `VerifyServing` (`verify.go:32`) is the fail-closed read-back gate after `nft -f` and before `Apply` reports `Verified: true`. Ten distinct classes of ruleset drift or renderer bugs pass `VerifyServing(rs, m)` while forwarding member traffic outside the serving model:
  1. **`ct state related` accept at index 0 of `forward` (`parse_nft.go:545`):** `mentionsEstablished` only checks `v == "established"` and ignores `"related"`. An accept rule matching `ct state related` placed before `ip saddr != @allowed_v4 drop` is neither classified as `established` in `checkForwardOrder` (`verify.go:558`) nor rejected as `other` in serving mode (`verify.go:236` only rejects `other != nil` when `baseline == true`).
  2. **`meta mark` or arbitrary non-probe accept before `@allowed_v4` drop (`verify.go:152-162, 236`):** When `m.Probe == nil`, `VerifyServing` only rejects rules whose counter name is `probe_up`/`probe_down` or that reference `@probe_v4`/`@probe_dst_v4`. For `m.Probe != nil`, `probeScopedAccept` (`verify.go:264`) only classifies rules touching `@probe_v4`, `@probe_dst_v4`, `@resolver_v4`, `probe_up`, or `probe_down`. Any other `accept` rule in `forward` (e.g., `iifname "fv-fh" meta mark 0x1 accept` at index 0) lands in `other`, which is ignored when `baseline == false`.
  3. **Unconditional `accept` right after upload `ip saddr != @allowed_v4 drop` (`verify.go:236, 555`):** Placing `iifname "fv-fh" oifname "fv-ih" accept` immediately after the upload `@allowed_v4` drop satisfies `seenAllowDrop` and bypasses the rogue-DNS drop, `@denied_v4` drop, `@host_v4` drop, and all per-device counters/QoS classifiers.
  4. **Missing download `ip daddr != @allowed_v4 drop` (`verify.go:555-572`):** `checkForwardOrder` uses a single boolean `seenAllowDrop` set by either direction's `@allowed_v4` drop (`r.NotInSet == setAllowedV4`). Once the upload `ip saddr != @allowed_v4 drop` is seen, `seenAllowDrop` is `true`, so removing the download `ip daddr != @allowed_v4 drop` entirely passes `VerifyServing`, allowing `iifname "fv-ih" oifname "fv-fh" ct state established,related accept` to forward download packets to a revoked member whose IP was deleted from `@allowed_v4` when `conntrack -D` fails.
  5. **Missing `@denied_v4` and `@host_v4` drops when `m.Probe == nil` (`verify.go:58-65, 226-230`):** When `m.Probe == nil`, `VerifyServing` validates the elements of `@denied_v4` (`verify.go:60`) but never checks that any rule in `forward` drops `@denied_v4` or `@host_v4` (`hasDeniedDrop` and `hasHostDrop` at `verify.go:226` are only checked inside `if m.Probe != nil`).
  6. **Tainted prefix (`0.0.0.0/0`) in `@resolver_v4` when `m.Probe == nil` (`verify.go:307-310`):** `verifyResolverSet` returns `nil` immediately when `m.Probe == nil` (`if m.Probe == nil { return nil }`). In serving mode without a probe, `@resolver_v4` is still used by the member DNS rules (`ip daddr @resolver_v4 tcp/udp dport 53 accept` before `@denied_v4` drop); adding `0.0.0.0/0` to `@resolver_v4` passes `VerifyServing` and allows members to reach port 53 on any denied/RFC1918/link-local/host destination.
  7. **Extra base chains in `inet fvpn` such as `prerouting` DNAT (`verify.go:36-45`):** `VerifyServing` only checks `expectChain(rs, "forward", ...)` and `hasDropOnIngress(rs, "input", ...)` and ignores extra chains in `inet fvpn`. A `prerouting` chain (`type nat hook prerouting priority -100`) with `iifname "fv-fh" ip daddr 169.254.169.254 dnat ip to 93.184.216.34` (or rewriting a public IP to an internal host after `forward` via routing/NAT) is accepted.
  8. **`accept` rule before `drop` in `inet fvpn input` (`verify.go:643-661`):** `hasDropOnIngress` scans `input` for the presence of `iifname "fv-fh" drop` and `ip saddr 100.64.0.0/10 drop` without checking whether an `accept` rule precedes them, accepting `iifname "fv-fh" tcp dport 22 accept` at the top of `input`.
  9. **`flags interval` on `@allowed_v4` (`parse_nft.go:190-213`):** `parseSet` ignores `m.Set.Flags`, so `@allowed_v4` with `"flags": ["interval", "timeout"]` is accepted by `VerifyServing`.
  10. **Duplicate chain names across families (`parse_nft.go:107-135`):** `ParseRuleset` keys `rs.Chains` by `m.Chain.Name` alone (ignoring `m.Chain.Family` and `m.Chain.Table` when merging rules and overwriting the chain header). A second `chain ip fvpn forward` with `policy drop` overwrites the header of `chain inet fvpn forward` (`policy accept`), masking the `accept` policy from `expectChain`.
- **Proof:** `TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding` in `node/enforce/review_r12_test.go`:
  ```text
  === RUN   TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding
  === RUN   TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding/ct_state_related_accept_before_allowed_v4_drop
      review_r12_test.go:180: TODO(review): VerifyServing accepted out-of-model ruleset (ct_state_related_accept_before_allowed_v4_drop)
  === RUN   TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding/meta_mark_match_accept_before_allowed_v4_drop
      review_r12_test.go:180: TODO(review): VerifyServing accepted out-of-model ruleset (meta_mark_match_accept_before_allowed_v4_drop)
  === RUN   TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding/unconditional_accept_after_upload_allow_drop_bypasses_denied_and_qos
      review_r12_test.go:180: TODO(review): VerifyServing accepted out-of-model ruleset (unconditional_accept_after_upload_allow_drop_bypasses_denied_and_qos)
  === RUN   TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding/missing_download_allowed_v4_drop_before_established_accept
      review_r12_test.go:180: TODO(review): VerifyServing accepted out-of-model ruleset (missing_download_allowed_v4_drop_before_established_accept)
  === RUN   TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding/missing_denied_v4_and_host_v4_drops_when_probe_nil
      review_r12_test.go:180: TODO(review): VerifyServing accepted out-of-model ruleset (missing_denied_v4_and_host_v4_drops_when_probe_nil)
  === RUN   TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding/tainted_prefix_in_resolver_v4_when_probe_nil
      review_r12_test.go:180: TODO(review): VerifyServing accepted out-of-model ruleset (tainted_prefix_in_resolver_v4_when_probe_nil)
  === RUN   TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding/prerouting_dnat_chain_rewrites_destination_before_forward
      review_r12_test.go:180: TODO(review): VerifyServing accepted out-of-model ruleset (prerouting_dnat_chain_rewrites_destination_before_forward)
  === RUN   TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding/input_accept_member_ingress_before_drop
      review_r12_test.go:180: TODO(review): VerifyServing accepted out-of-model ruleset (input_accept_member_ingress_before_drop)
  === RUN   TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding/interval_flag_on_allowed_v4_set
      review_r12_test.go:180: TODO(review): VerifyServing accepted out-of-model ruleset (interval_flag_on_allowed_v4_set)
  === RUN   TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding/duplicate_forward_chain_first_accept_policy_masked_by_second_drop_entry
      review_r12_test.go:180: TODO(review): VerifyServing accepted out-of-model ruleset (duplicate_forward_chain_first_accept_policy_masked_by_second_drop_entry)
  --- FAIL: TestReviewR12_Q1_VerifierAcceptsOutOfModelForwarding (0.01s)
  ```
- **Fix (`netlab`):**
  1. In `VerifyServing` (`node/enforce/verify.go`), require every `accept` rule in `forward` to match either an authorized serving rule (member DNS to `@resolver_v4:53` after upload `@allowed_v4` drop, per-device upload/download accept after `@denied_v4` and `@host_v4` drops, or download `ct state established,related` after download `@allowed_v4` drop) or the verified probe rules when `m.Probe != nil`; reject `other != nil` in both serving and baseline modes.
  2. In `mentionsEstablished` (`node/enforce/parse_nft.go:545`), match both `"established"` and `"related"`.
  3. In `checkForwardOrder` (`node/enforce/verify.go:555`), track upload `ip saddr != @allowed_v4 drop` and download `ip daddr != @allowed_v4 drop` separately, and require both to exist and precede all direction-matching accepts (including `established,related` and `@denied_v4`/`@host_v4` drops before upload device accepts).
  4. Always run `verifyResolverSet(rs, m)`, `hasDeniedDrop`, and `hasHostDrop` in `VerifyServing` even when `m.Probe == nil` (when `len(m.Resolvers) > 0` or `len(m.Bindings) > 0`).
  5. Reject any chain in `inet fvpn` outside `{"forward", "input", "output", "postrouting"}`, reject any `accept` rule on `MemberIngress` or `100.64.0.0/10` before the drops in `input`, reject `"interval"` set flags on `allowed_v4`/`probe_v4`/`probe_dst_v4`/`resolver_v4`, and reject duplicate `(family, table, chain)` definitions in `ParseRuleset`.

---

### H-2: D-063 issuer address validation accepts an empty node interface list, follows symlinks on `agent.json`, and leaves probe omission unrecorded in challenge/attestation/journal artifacts

- **Where:** `node/updater/probe_addresses.go:51-65`, `node/updater/probe_addresses.go:96-116`, `node/updater/probe_addresses.go:181-196`, `node/updater/attest.go:38`
- **Scenario:**
  1. **Empty interface list bypasses `node_source` and `node_destination` checks (`probe_addresses.go:96-116`):** While DNS resolution (`resolve` at `probe_addresses.go:125-127`) fails closed when `len(addrs) == 0`, `probeEnvironment.addresses` (`probe_addresses.go:100-116`) does not check `len(known.node) == 0` after filtering `interfaces()` for `addr.Is4() && addr.IsGlobalUnicast()`. If `interfaces()` returns an empty slice or only loopback/link-local/IPv6 addresses during interface bring-up, `known.node` is empty and `validateProbePeer` accepts a probe whose `SourceV4` or `Destinations` is the node's own IPv4 address.
  2. **Symlinked `agent.json` followed (`probe_addresses.go:51-58`):** `readProbeAgentConfig` calls `os.Open(path)` followed by `f.Stat()`. Because `f.Stat()` (`fstat`) follows symlinks (and `os.Open` blocks indefinitely on a named pipe/FIFO before `f.Stat()` is reached), a symlinked `agent.json` passes `st.Mode().IsRegular()`.
  3. **Probe omission is unrecorded outside transient logs (`probe_addresses.go:181-196`, `attest.go:38`):** When `challengeProbe` refuses a configured `probe_peer` during `issueChallenge` (e.g., because DNS or interface addresses changed at issuance time), it logs once per reason class and returns `nil`. `healthattest.WriteChallenge` serializes `Challenge{Probe: nil}` with `json:"probe,omitempty"`, omitting `"probe"` from `health-challenge.json`. Neither `health-challenge.json`, `health-attestation.json`, `update-journal`, `UpdateResult`, nor `ObservationReport` records that a configured `probe_peer` was omitted, and the agent's `tunnel_probe` check still passes (`overlay_up`) without running the synthetic peer probe.
- **Proof:** `TestReviewR12_Q6_EmptyInterfaceListAndSymlinkAgentConfig` in `node/updater/review_r12_d063_test.go`:
  ```text
  === RUN   TestReviewR12_Q6_EmptyInterfaceListAndSymlinkAgentConfig
  === RUN   TestReviewR12_Q6_EmptyInterfaceListAndSymlinkAgentConfig/empty_interface_list_bypasses_node_address_check
      review_r12_d063_test.go:45: TODO(review): probeEnvironment.addresses succeeded with len(known.node)=0 and validate() accepted probe_peer &{PeerID:probe-private-marker SourceV4:100.64.8.1/32 Destinations:[{AddrV4:1.2.3.4/32 Proto:tcp Port:443}] ResolverDNS:true} without any node IPv4 interface addresses
  === RUN   TestReviewR12_Q6_EmptyInterfaceListAndSymlinkAgentConfig/symlinked_agent_config_accepted
      review_r12_d063_test.go:66: TODO(review): readProbeAgentConfig followed symlink /tmp/TestReviewR12_Q6_EmptyInterfaceListAndSymlinkAgentConfigsymlink3141078783/003/symlink-agent.json -> /tmp/TestReviewR12_Q6_EmptyInterfaceListAndSymlinkAgentConfigsymlink3141078783/002/real-agent.json instead of rejecting non-regular lstat mode
  === RUN   TestReviewR12_Q6_EmptyInterfaceListAndSymlinkAgentConfig/probe_omission_unrecorded_in_challenge_file
      review_r12_d063_test.go:110: TODO(review): configured probe_peer was refused and omitted during rollback challenge issuance, leaving health-challenge.json ({"txn_id":"01a18c27-0bbd-7d62-a3ae-2f9474e6223b","nonce":"2d09098b21bb723f95b0a10e57382811eb5bd6faf47e05721fe3b2636efed02d","release_id":"01a18c27-0bb9-7a18-b1c1-a3e4953904e0","node_id":"01a18c27-0bb7-7bbf-b397-f9c4f44b142a","incarnation_id":"01a18c27-0bb7-7bc0-a333-a5c4da742c1f","boot_id":"01a18c27-0bb7-7bc1-a22f-6ced3a024999","issued_at":"2026-10-28T15:00:00Z","expires_at":"2026-10-28T15:00:45Z","components":{"node_agent":"1.2.0","netbird":"0.76.2","node_updater":"1.2.0","enforcement":"1.2.0"}}) indistinguishable from an unconfigured probe
  --- FAIL: TestReviewR12_Q6_EmptyInterfaceListAndSymlinkAgentConfig (0.01s)
  ```
- **Fix (`updater`):**
  1. In `probeEnvironment.addresses` (`node/updater/probe_addresses.go:116`), return `known, probeRefusal("node_addresses_unavailable")` when `len(known.node) == 0` (and update test fixtures in `probe_test.go:42` and `probe_addresses_test.go:190` to supply a non-colliding node IPv4 address such as `192.0.2.10`).
  2. In `readProbeAgentConfig` (`node/updater/probe_addresses.go:51`), check `os.Lstat(path)` (`!lst.Mode().IsRegular()`) before `os.Open(path)` (or open with `syscall.O_NOFOLLOW|syscall.O_NONBLOCK`) and verify `os.SameFile(lst, st)`.
  3. Record probe refusal/omission in the challenge/journal metadata (or fail closed when a configured `probe_peer` cannot be validated at challenge issuance) so candidate promotion or rollback decisions made without the configured synthetic probe are auditable.

---

### H-3: Pre-result heartbeat seeding in `observeTx` misattributes cross-job audits/seeding to `*n.job`, mutates `current_release_id` when `last_good_release_id` is unexpected, and poisons `update_jobs.attempt_release_id` when a heartbeat arrives before `NodeUpdateIntent`

- **Where:** `internal/fleetupdate/serving.go:121-150`, `internal/fleetupdate/intent.go:168`, `internal/fleetupdate/results.go:178-183`
- **Scenario:**
  Three flaws exist in `observeTx` (`internal/fleetupdate/serving.go:118-159`):
  1. **Cross-job misattribution (`serving.go:121-150`):** `ObserveReportTx` validates `r.Update.JobId` (`jobID`) against `update_jobs` and passes `&jobID` to `observeTx`, but lines 121–150 ignore `jobID` and read `*n.job` (`fleet_node_state.current_job_id`). When `job1` (gen 1, target `l.new`) is superseded by `job2` (gen 2, target `l.old`, so `n.job = &job2`), a heartbeat explicitly bound to `job1` (`r.Update.JobId = job1, gen = 1`) reporting `job1`'s own target `l.new` evaluates `expected` against `job2`'s target (`l.old`), treats `l.new` as unexpected, writes 0 audits for `job1`, and misattributes an `update_observation.unexpected_release` audit `(job2, l.new)` to `job2`. Conversely, if a heartbeat for `job1` reports `job2`'s target, `observeTx` seeds `nodes.current_release_id` from `job2`'s target and confirms `job1`.
  2. **Partial mutation when `last_good_release_id` is unexpected (`serving.go:136-141`):** Line 136 checks `if expected(o.CurrentReleaseID) && !equalID(n.current, o.CurrentReleaseID)` without checking `equalID(n.lastGood, o.LastGoodReleaseID)` or `expected(o.LastGoodReleaseID)`. A heartbeat carrying `CurrentReleaseID = l.new` (target) alongside an unexpected `LastGoodReleaseID = other` mutates `nodes.current_release_id` to `l.new` even while emitting an `update_observation.unexpected_release` audit for `other`, violating the contract that an unexpected release leaves both columns intact.
  3. **Pre-intent target seeding poisons `attempt_release_id` (`serving.go:126`, `intent.go:168`, `results.go:180`):** Line 126 allows `j.releaseID` (`targetRelease`) to seed `nodes.current_release_id` even when `j.issued == nil` (before `NodeUpdateIntent` has ever been called to authorize the job). If a heartbeat seeds `nodes.current_release_id = l.new` while the job is still unissued (`issued_at IS NULL`), the first call to `NodeUpdateIntent` (`intent.go:168`: `attempt_release_id = CASE WHEN issued_at IS NULL THEN $4 ELSE attempt_release_id END`) snapshots `$4 = n.current` (`l.new`) into `attempt_release_id`. Thereafter, `SubmitResult` (`results.go:180`) rejects a legitimate pre-switch failure (`stage = "download", current_release_id = l.old`) with 422 `VALIDATION_FAILED: Pre-health failure requires null health, an error and the unchanged attempt release`.
- **Proof:** `TestReviewR12_Q7_CrossJobAuditAndSeedingMisattribution` and `TestReviewR12_Q7_PreIntentTargetSeedPoisonsAttemptRelease` in `internal/fleetupdate/review_r12_seeding_integration_test.go`:
  ```text
  === RUN   TestReviewR12_Q7_CrossJobAuditAndSeedingMisattribution
      review_r12_seeding_integration_test.go:30: isolated database: fvpn_t6ae46530_36585e62117a4cb4ac820bf79f04480e
      review_r12_seeding_integration_test.go:89: TODO(review): (1) heartbeat for superseded job1 (01a18c2b-47b1-7de6-94d5-e6462640d5e0) reporting job1's target 01a18c2b-47a2-7303-a449-3d354eef9b3f wrote 0 audits for job1 and misattributed 1 unexpected_release audit(s) to job2 (01a18c2b-47bb-757e-887f-c543224a599e); (2) heartbeat with unexpected last_good=01a18c2b-47c2-7ecd-95fa-670b41916f59 still mutated current_release_id=01a18c2b-47a2-7303-a449-3d354eef9b3f (want 01a18c2b-479c-7413-a921-e0ec6c37f863 intact)
  --- FAIL: TestReviewR12_Q7_CrossJobAuditAndSeedingMisattribution (0.42s)
  === RUN   TestReviewR12_Q7_PreIntentTargetSeedPoisonsAttemptRelease
      review_r12_seeding_integration_test.go:107: isolated database: fvpn_t6ae46531_1354da4f4e5d4c6dba31ba77c100a2b6
      review_r12_seeding_integration_test.go:138: TODO(review): pre-intent heartbeat seeded current_release_id=01a18c2b-4952-7a87-8b8a-5e609c6a0b11 before NodeUpdateIntent, poisoning update_jobs.attempt_release_id and causing legitimate pre-switch failure (stage=download, current=01a18c2b-494c-7e63-b803-eb54bef38fa4) to be rejected: VALIDATION_FAILED: Pre-health failure requires null health, an error and the unchanged attempt release
  --- FAIL: TestReviewR12_Q7_PreIntentTargetSeedPoisonsAttemptRelease (0.42s)
  ```
- **Fix (`fleet-update`):**
  1. In `observeTx` (`internal/fleetupdate/serving.go:121-150`), resolve `effectiveJob := jobID` (falling back to `n.job` only when `jobID == nil`); allow `targetRelease = &j.releaseID` only when `n.job != nil && equalID(effectiveJob, n.job) && equalID(j.incarnation, &i.IncarnationID) && j.issued != nil`; and pass `*effectiveJob` to `auditUnexpectedRelease`.
  2. Gate the `UPDATE nodes SET current_release_id=$2` at `serving.go:136` on `expected(o.CurrentReleaseID) && equalID(n.lastGood, o.LastGoodReleaseID) && !equalID(n.current, o.CurrentReleaseID)` so any unexpected or disagreeing `last_good_release_id` leaves both columns intact.

---

## Medium

### M-1: Control body validation above `canonjson.MaxBytes` (1 MiB) accepts duplicate JSON keys, invalid UTF-8 bytes, and lone UTF-16 surrogates, and ignores `json.Unmarshal` errors before `matchingIdentity`

- **Where:** `internal/httpapi/middleware.go:419-436`, `internal/httpapi/middleware.go:505-530`
- **Scenario:**
  Follow-up 16 (SECOPS-15) changed `checkJSONDocument` (`middleware.go:505-530`) so that bodies up to `canonjson.MaxBytes` (`1,048,576` bytes = 1 MiB) are validated by `canonjson.Canonicalize(raw)`, whereas bodies in `(1 MiB, 8 MiB]` on `SubmitUsageBatch` fall back to `encoding/json`'s `json.NewDecoder(bytes.NewReader(raw)).Decode(&doc)` into a `json.RawMessage`. Unlike `canonjson.Canonicalize` (and contrary to `docs/control/README.md:94`, which states that JSON bodies reject duplicate object keys and invalid UTF-8 with `400 MALFORMED_REQUEST`), `json.NewDecoder.Decode(&doc)` into `json.RawMessage` does **not** check for duplicate object keys, invalid UTF-8 bytes (`0xff`), or lone UTF-16 surrogates (`\uDEAD`). Consequently, a usage batch body of `1,048,576` bytes (1 MiB) with duplicate `"node_id"` keys, raw `0xff` bytes, or `\uDEAD` is rejected with `400 MALFORMED_REQUEST`, while the exact same payload padded by 1 space to `1,048,577` bytes (1 MiB + 1) passes `checkJSONDocument` and reaches OpenAPI validation / `json.Unmarshal` (allowing last-key-wins vs first-key-wins parser differentials or silent `U+FFFD` replacement). In addition, `middleware.go:432` ignores the error from `_ = json.Unmarshal(raw, &body)` before calling `matchingIdentity(body, p)` (which defaults to `true` when identity keys are absent).
- **Proof:** `TestReviewR12_Q5_AboveCanonMaxBytesAcceptsDuplicateKeysAndInvalidUTF8` in `internal/httpapi/review_r12_test.go`:
  ```text
  === RUN   TestReviewR12_Q5_AboveCanonMaxBytesAcceptsDuplicateKeysAndInvalidUTF8
  === RUN   TestReviewR12_Q5_AboveCanonMaxBytesAcceptsDuplicateKeysAndInvalidUTF8/duplicate_keys
      review_r12_test.go:243: TODO(review): duplicate_keys rejected with 400 at 1048576 bytes (1 MiB) but returned HTTP 422 at 1048577 bytes (1 MiB + 1): {"error":{"code":"VALIDATION_FAILED","message":"Request does not match the API schema.","request_id":"01a18c23-3bd6-787d-b167-61b4e6c6a902","retryable":false,"details":{}}}
  === RUN   TestReviewR12_Q5_AboveCanonMaxBytesAcceptsDuplicateKeysAndInvalidUTF8/invalid_utf8_byte
      review_r12_test.go:243: TODO(review): invalid_utf8_byte rejected with 400 at 1048576 bytes (1 MiB) but returned HTTP 422 at 1048577 bytes (1 MiB + 1): {"error":{"code":"VALIDATION_FAILED","message":"Request does not match the API schema.","request_id":"01a18c23-3bde-7436-a57d-2f6a9d4ccd87","retryable":false,"details":{}}}
  === RUN   TestReviewR12_Q5_AboveCanonMaxBytesAcceptsDuplicateKeysAndInvalidUTF8/lone_utf16_surrogate
      review_r12_test.go:243: TODO(review): lone_utf16_surrogate rejected with 400 at 1048576 bytes (1 MiB) but returned HTTP 422 at 1048577 bytes (1 MiB + 1): {"error":{"code":"VALIDATION_FAILED","message":"Request does not match the API schema.","request_id":"01a18c23-3be4-7196-9254-0bfcc4c73b11","retryable":false,"details":{}}}
  --- FAIL: TestReviewR12_Q5_AboveCanonMaxBytesAcceptsDuplicateKeysAndInvalidUTF8 (0.04s)
  ```
- **Fix (`control`):**
  In `checkJSONDocument` (`internal/httpapi/middleware.go:505`), validate bodies in `(canonjson.MaxBytes, limit]` with a bounded streaming token/syntax walk (or `canonjson` parser parameterized by `limit`) that enforces `utf8.Valid(raw)`, rejects lone UTF-16 surrogates, rejects duplicate object keys at every object level, and enforces `canonjson.MaxDepth` (256) and integer bounds `±(2^53-1)`; also check the error returned by `json.Unmarshal(raw, &body)` at `middleware.go:432`.

---

### M-2: A single IPv6 `/48` allocation can exhaust the shared `ingress` rate-limit table (`MaxRatePrincipals = 10,000`, 1-hour eviction) on both public and mTLS listeners, locking out all new client addresses for 1 hour

- **Where:** `internal/httpapi/middleware.go:183-211`, `internal/httpapi/middleware.go:255-256`, `internal/httpapi/middleware.go:309`, `internal/httpapi/middleware.go:344-354`
- **Scenario:**
  `limiter` (`middleware.go:183-211`) caps its map at `MaxRatePrincipals = 10,000` entries and only evicts entries older than `1 * time.Hour`; when `len(l.entries) >= l.max` and a new key arrives, `allow` returns `time.Second` (HTTP 429 `RATE_LIMITED`) without inserting the key. Because a single IPv6 `/48` site allocation contains $2^{16} = 65,536$ distinct `/64` prefixes:
  1. **Public/browser listener (`AgentListener: false`):** Unauthenticated `anonymous` routes (`EnrollNode`, `ReceiveProviderWebhook`) insert **two** keys per IP into the shared `ingress` map (`"ingress:anonymous:"+peer` at line 309 and `"anonymous:anonymous:"+peer` at line 353, because `if p.Role == "" { bucket = ingress }`). Sending 1 request from each of **5,000 `/64` prefixes** inside a single `/48` to `/api/v1/agent/enroll` fills all 10,000 slots of `ingress`. For the next 1 hour, every authenticated admin or member connecting from a new IPv4 `/32` or IPv6 `/64` is rejected at line 309 with HTTP 429 `RATE_LIMITED`.
  2. **mTLS listener (`AgentListener: true`):** `ingress.allow("ingress:"+ingressClass+":"+peer, o.Config.IngressRate)` at line 309 runs **before** verifying `r.TLS` / client certificates at line 330. Sending 1 unauthenticated request (no client cert) to each of the 4 mTLS route classes (`heartbeat`, `lease`, `usage`, `agent`) from **2,500 `/64` prefixes** inside one `/48` inserts $2,500 \times 4 = 10,000$ entries into `ingress` and locks out every legitimate mTLS node connecting from a new IP address with HTTP 429 `RATE_LIMITED` for up to 1 hour.
- **Proof:** `TestReviewR12_Q4_IPv6Slash48ExhaustsSharedIngressTable` in `internal/httpapi/review_r12_test.go`:
  ```text
  === RUN   TestReviewR12_Q4_IPv6Slash48ExhaustsSharedIngressTable
  === RUN   TestReviewR12_Q4_IPv6Slash48ExhaustsSharedIngressTable/public_listener_5000_slash64s_on_enroll
      review_r12_test.go:63: TODO(review): 5,000 /64s from one IPv6 /48 on /api/v1/agent/enroll filled shared ingress map (10000 entries) and locked out authenticated admin from 198.51.100.77 with 429: {"error":{"code":"RATE_LIMITED","message":"Authentication request rate exceeded.","request_id":"01a18c23-3b6b-7598-ac2f-eb52d7bef80f","retryable":true,"details":{}}}
  === RUN   TestReviewR12_Q4_IPv6Slash48ExhaustsSharedIngressTable/mtls_listener_2500_slash64s_without_client_cert
      review_r12_test.go:101: TODO(review): 2,500 unauthenticated /64s from one IPv6 /48 filled mTLS ingress map (10000 entries) before client-cert check and locked out mTLS node 198.51.100.77 with 429: {"error":{"code":"RATE_LIMITED","message":"Authentication request rate exceeded.","request_id":"01a18c23-3bb3-7348-8879-ebb4717b6742","retryable":true,"details":{}}}
  --- FAIL: TestReviewR12_Q4_IPv6Slash48ExhaustsSharedIngressTable (0.19s)
  ```
- **Fix (`control`):**
  1. On the mTLS listener (`mtls == true`), verify `r.TLS` / `o.Agents.AuthenticateAgent` (which uses the already-completed TLS handshake certificate in `r.TLS.VerifiedChains`) before inserting into `ingress`, or separate post-handshake mTLS limiters from unauthenticated TCP peers.
  2. Separate the `anonymous` class limiter map from the `browser` `ingress` limiter map (or partition `ingress` per class and evict full-token idle buckets before 1 hour when at capacity).

---

### M-3: `Samples` emits zero-metadata (`MemberID = uuid.Nil, BindingRevision = 0`) samples for revoked devices' lingering nftables counters, and `Apply` drops counter/sample read-back on `ConntrackDeleteError`

- **Where:** `node/enforce/apply.go:120-137`, `node/enforce/apply.go:316-346`, `node/enforce/render_nft.go:62-75`
- **Scenario:**
  1. **Revoked device counters leak into `Samples` (`apply.go:316-346`):** `Render` intentionally keeps named counters (`u_<dev>`, `d_<dev>`, `nu_<dev>`, `nd_<dev>`) in `inet fvpn` across generations so counters remain monotonic (`render_nft.go:62`, `plan.Finals`). When a device is removed from `m.Bindings`, its 4 named counters remain in `rs.Counters`. In `Samples(m, rs.Counters)` (`apply.go:332`), `b := byID[id.String()]` does not check `, ok := byID[id.String()]`. For every revoked device whose named counter still exists in the kernel table, `Samples` appends 4 `Sample` records with `DeviceID = <revoked_device_id>`, `MemberID = uuid.Nil`, `PeerID = ""`, `BindingRevision = 0`, and the revoked device's cumulative `Packets`/`Bytes`.
  2. **`Apply` drops `Counters` and `Samples` on `ConntrackDeleteError` (`apply.go:120-137`):** In `Apply`, `deleteConntrack` (`apply.go:120-127`) runs **before** `verified.Counters = rs.Counters` and `verified.Samples = Samples(m, rs.Counters)` (`apply.go:128-136`). When `conntrack -D` fails for the first address in `m.Revoke`, `Apply` aborts the `m.Revoke` loop without attempting remaining revoked addresses and returns `ApplyResult{Verified: true, Counters: nil, Samples: nil}` alongside `ConntrackDeleteError`, discarding the counter snapshot from the verified ruleset.
- **Proof:** `TestReviewR12_Q2_RevokedDeviceCountersLeakIntoSamplesAndConntrackDropsCounters` in `node/enforce/review_r12_test.go`:
  ```text
  === RUN   TestReviewR12_Q2_RevokedDeviceCountersLeakIntoSamplesAndConntrackDropsCounters
      review_r12_test.go:258: TODO(review): (1) Samples emitted 4 entries for revoked device 019a0000-0000-7000-8000-0000000000a2 with zero MemberID=00000000-0000-0000-0000-000000000000 BindingRevision=0; (2) Apply on ConntrackDeleteError returned Verified=true with empty Counters (0) and Samples (0)
  --- FAIL: TestReviewR12_Q2_RevokedDeviceCountersLeakIntoSamplesAndConntrackDropsCounters (0.00s)
  ```
- **Fix (`netlab`):**
  1. In `Samples` (`node/enforce/apply.go:332`), check `b, ok := byID[id.String()]` and skip counters whose device is no longer in `m.Bindings` (their final readings were already captured in `plan.Finals`).
  2. In `Apply` (`node/enforce/apply.go:120-137`), populate `verified.Counters` and `verified.Samples` before running the `m.Revoke` loop, and attempt `deleteConntrack` for all addresses in `m.Revoke` before returning `errors.Join` / `ConntrackDeleteError`.

---

## Low

### L-1: Per-class concurrency gate (`gates[class]`) in `httpapi.New` returns HTTP 429 `RATE_LIMITED` without incrementing `fvpn_control_rate_limited_total`

- **Where:** `internal/httpapi/middleware.go:364-370`
- **Scenario:**
  When `gates[class]` (`MaxConcurrent`) is full in `httpapi.New` (`middleware.go:364-370`), the middleware writes HTTP 429 `RATE_LIMITED` (`"Request concurrency exceeded."`) with `Retry-After: 1` but does not increment `o.RateLimited` (`fvpn_control_rate_limited_total`), whereas `BrowserRoutesWithMetrics` (`middleware.go:676`) increments `rateLimited.WithLabelValues("login", "concurrency").Inc()` and `docs/control/README.md:84` / `docs/control/AUTH.md:125` state that every 429 increments `fvpn_control_rate_limited_total`.
- **Proof:** `TestReviewR12_Q4_ConcurrencyGate429OmitsRateLimitedMetric` in `internal/httpapi/review_r12_test.go`:
  ```text
  === RUN   TestReviewR12_Q4_ConcurrencyGate429OmitsRateLimitedMetric
      review_r12_test.go:187: TODO(review): concurrency gate returned 429 ({"error":{"code":"RATE_LIMITED","message":"Request concurrency exceeded.","request_id":"01a18c23-3bc3-76ab-84c8-79111e861939","retryable":true,"details":{}}}) with Retry-After="1" but fvpn_control_rate_limited_total remained 0
  --- FAIL: TestReviewR12_Q4_ConcurrencyGate429OmitsRateLimitedMetric (0.02s)
  ```
- **Fix (`control`):**
  In `httpapi.New` (`internal/httpapi/middleware.go:367`), call `if o.RateLimited != nil { o.RateLimited.WithLabelValues(class, "concurrency").Inc() }` before writing the 429 response.

---

### L-2: `Render` only merges `Datapath.SNAT/32` into `@host_v4` when `m.Probe != nil`

- **Where:** `node/enforce/render_nft.go:34-38`, `node/enforce/render_nft.go:283-286`
- **Scenario:**
  In `RenderBaseline` (`render_nft.go:283-286`) and in `Render` when `m.Probe != nil` (`render_nft.go:35-37`), `m.Datapath.SNAT/32` is automatically merged into `host` (`@host_v4`). However, in `Render` when `m.Probe == nil`, `m.Datapath.SNAT/32` is **not** merged into `host`. If the caller leaves `m.HostV4` empty in normal serving mode (`m.Probe == nil`) on a public uplink, `@host_v4` is rendered empty and `m.Datapath.SNAT` is not protected by `ip daddr @host_v4 drop`.
- **Proof:** `TestReviewR12_Q3_ServingWithoutProbeOmitsSNATFromHostV4` in `node/enforce/review_r12_test.go`:
  ```text
  === RUN   TestReviewR12_Q3_ServingWithoutProbeOmitsSNATFromHostV4
      review_r12_test.go:281: TODO(review): Render with Probe=nil and HostV4=nil omitted Datapath.SNAT (192.0.2.37/32) from @host_v4, whereas Probe!=nil includes it
  --- FAIL: TestReviewR12_Q3_ServingWithoutProbeOmitsSNATFromHostV4 (0.00s)
  ```
- **Fix (`netlab`):**
  In `Render` (`node/enforce/render_nft.go:34-38`), unconditionally append `netip.PrefixFrom(m.Datapath.SNAT, 32)` into `host` regardless of whether `m.Probe` is nil.

---

## What I tried that held

### 1. What the verifier does not see (Question 1)

#### Complete table of `accept` rules emitted by `Render` and `RenderBaseline` and their verifier clauses

| Mode | Chain | Emitted `accept` rule (`render_nft.go`) | Verifier clause (`verify.go`) |
|---|---|---|---|
| **Baseline (`Probe == nil`)** | `forward` | None (0 `accept` rules; chain policy `drop`) | `verify.go:53` (`expectChain` `policy drop`), `verify.go:152-162` (rejects `probe_up`/`probe_down`/`@probe_v4`/`@probe_dst_v4`), `verify.go:236` (`len(other) > 0` rejected when `baseline == true`). |
| **Baseline (`Probe != nil`)** | `forward` | 1. *(Optional, if `ResolverDNS && len(Resolvers)>0`)* `iifname <in> oifname <up> ip saddr @probe_v4 ip daddr @resolver_v4 tcp dport 53 counter name "probe_up" accept`<br>2. *(Optional, if `ResolverDNS && len(Resolvers)>0`)* `iifname <in> oifname <up> ip saddr @probe_v4 ip daddr @resolver_v4 udp dport 53 counter name "probe_up" accept`<br>3. `iifname <in> oifname <up> ip saddr @probe_v4 ip daddr . meta l4proto . th dport @probe_dst_v4 counter name "probe_up" accept`<br>4. `iifname <up> oifname <in> ip daddr @probe_v4 ct state established,related counter name "probe_down" accept` | `verify.go:164-258`: `probeScopedAccept` (`verify.go:264-305`) classifies every probe rule into `probeUp`, `probeDown`, `probeDNS`, or `other`; checks exact counts (`len(probeUp)==1`, `len(probeDown)==1`, `len(probeDNS)==2` sorted `["tcp","udp"]` or `0` when `!ResolverDNS`, `len(other)==0`); verifies `hasInvalidDrop`, `hasDeniedDrop`, `hasHostDrop`, `hasBadDNSDrop`, `verifyResolverSet`, `@probe_v4` element + timeout, and `@probe_dst_v4` tuple elements + timeout. |
| **Serving (`Probe == nil`)** | `forward` | 1. *(If `len(Resolvers)>0`)* `iifname <in> ip daddr @resolver_v4 tcp dport 53 counter name "dns_u_<dev>" ... accept` (or `ip daddr @resolver_v4 tcp/udp dport 53` before `ctbadns`)<br>2. `iifname <up> oifname <in> ct state established,related goto fvpn_down` (where `fvpn_down` / `forward` has per-device `ip daddr <ip> counter name "d_<dev>" meta priority set <classid> accept`, or `ip saddr @resolver_v4 tcp/udp sport 53 counter name "dns_d_<dev>"`)<br>3. Per-device upload: `iifname <in> oifname <up> ip saddr <ip> counter name "u_<dev>" meta priority set <classid> accept` | `verify.go:47-91` + `checkForwardOrder` (`verify.go:555-572`): checks `expectChain("forward", "filter", "forward", -150, "drop")`, `hasV6Drop`, `@allowed_v4` host-only elements + timeouts (`verify.go:93-124`), `@denied_v4` covers `m.denyV4()` (`verify.go:60-65`), per-device upload/download rules with counter + `meta priority set` (`verify.go:126-149`), and `seenAllowDrop` before `established` or device accepts (`verify.go:555-572`). *(Gaps documented in **H-1** above.)* |
| **Serving (`Probe != nil`)** | `forward` | All Serving (`Probe == nil`) accepts above **plus** the 2–4 Probe accepts (`probe_up` tuple, optional 2 `probe_up` DNS rules, `probe_down` established/related) | All Serving checks **plus** `verify.go:164-258` (`probeScopedAccept`, `verifyResolverSet`, `@probe_v4`, `@probe_dst_v4`, `hasInvalidDrop`, `hasDeniedDrop`, `hasHostDrop`, `hasBadDNSDrop`) — except `len(other) > 0` at `verify.go:236` is gated on `if baseline`. |
| **Both modes** | `input`, `output` | Chain policy `accept` (`render_nft.go:96-97, 290-291`), with `input` having `iifname <in> counter name "ctin" drop` and `ip saddr 100.64.0.0/10 counter name "ctin" drop` | `verify.go:44` (`hasDropOnIngress(rs, "input", m.Datapath.MemberIngress)` checks the two drop rules exist; gap on preceding `accept` rules in **H-1**). |
| **Both modes** | `postrouting` | Chain policy `accept`, plus `oifname <up> ip saddr @allowed_v4 snat ip to <snat>` and/or `oifname <up> ip saddr @probe_v4 snat ip to <snat>` | `verify.go:196-205` verifies `hasProbeSNAT` when `m.Probe != nil`. |

#### Out-of-table / kernel-level constructs vs. in-table bugs and threat model (`docs/poc/G2-netlab.md`)
- **Outside `inet fvpn` (other tables/families such as `ip`, `ip6`, or a second `inet` table at priority `< -150`, or external flowtable offload):** By AGENTS.md §4 ("Node network ownership: only the nftables table `inet fvpn` and objects we create... Never `flush ruleset`, never touch other tables") and `Apply` (`apply.go:84`: `nft -j list table inet fvpn`), the verifier inspects only `table inet fvpn`. Note that in Linux netfilter, an `accept` verdict in an earlier `forward` hook (e.g., `table ip other` at priority `-200`) does **not** skip `table inet fvpn forward` at priority `-150` (a packet traverses all `NF_INET_FORWARD` hooks until one returns `NF_DROP`), **except** when another table uses `flow add @ft` (nftables flowtable offload, which bypasses classic `forward` hooks at ingress after flow establishment) or `prerouting`/`postrouting` NAT rewrites addresses around `@denied_v4`/`@allowed_v4`. Host-level adversaries creating rogue tables outside `inet fvpn` are outside the threat model (the node agent runs as root on a trusted host), whereas renderer/state drift and parser blind spots **inside** `inet fvpn` (all 10 cases in **H-1**, including `ct state related`, `meta mark`, missing download `@allowed_v4` drop, `prerouting` DNAT inside `inet fvpn`, `interval` flags, and `/0` prefixes in `@resolver_v4` when `Probe == nil`) **are** inside the verifier's own contract.
- Note that a `/0` element in `@probe_dst_v4` (`0.0.0.0/0 . tcp . 443`) **is** caught and rejected by `parseConcatDest` (`parse_nft.go:257-295`), which calls `hasPrefixOrRange(cell)` and `netip.ParseAddr` on the address cell of every concatenation tuple (`TestReviewR12_Q2_Q3_RevocationOrderingAndDenyBoundariesHeld`).
- `docs/poc/G2-netlab.md` documents the `table inet fvpn` ownership boundary, the `-150` filter priority (after conntrack `-200`, before filter `0` and srcnat `100`), and the conntrack revocation model, but does **not** document that `VerifyServing` ignores non-probe `other` accepts in `forward`, `prerouting` chains inside `inet fvpn`, or `@resolver_v4` when `Probe == nil`.

---

### 2. Revocation ordering and lingering state (Question 2)

- **Ordering proof across `testdata/serving.nft`, `testdata/probe-serving.nft`, and `Render` (`TestReviewR12_Q2_Q3_RevocationOrderingAndDenyBoundariesHeld`):**
  In `Render` (`render_nft.go:100-195`) and in both parsed golden rulesets:
  - Rule 0: generation comment rule (`ip saddr 192.0.2.255 ip daddr 192.0.2.255 comment ...`)
  - Rules 1–2: IPv6 drops (`iifname "fv-fh" meta nfproto ipv6 drop`, `oifname "fv-fh" meta nfproto ipv6 drop`)
  - Rules 3–4: invalid conntrack state drops (`iifname "fv-fh" ct state invalid drop`, `oifname "fv-fh" ct state invalid drop`)
  - Rule 5 (in `serving.nft` line 40; in `probe-serving.nft` line 46, preceded only by probe-source `@probe_v4` rules when `Probe != nil`):
    `iifname "fv-fh" ip saddr != @allowed_v4 counter name "ctunauth" drop` (upload allow-set drop)
  - Rule 6 (in `serving.nft` line 41; in `probe-serving.nft` line 47):
    `oifname "fv-fh" ip daddr != @allowed_v4 counter name "ctunauth" drop` (download allow-set drop)
  - All member established/related accepts (`iifname "fv-ih" oifname "fv-fh" ct state established,related counter name "d_<dev>" meta priority set ... accept`), member DNS accepts, and member upload accepts appear strictly **after** both the `ip saddr != @allowed_v4 drop` and `ip daddr != @allowed_v4 drop` rules.
  - When `Probe != nil` (`testdata/probe-serving.nft`), `probe_down` (`iifname "fv-ih" oifname "fv-fh" ip daddr @probe_v4 ct state established,related counter name "probe_down" accept`) is placed at rule index 9 (`probe-serving.nft:45`) ahead of `oifname "fv-fh" ip daddr != @allowed_v4 drop` (`probe-serving.nft:47`) **specifically** so the synthetic probe peer (`@probe_v4`, which is not a member and not in `@allowed_v4`) can receive return traffic; it is strictly guarded by `ip daddr @probe_v4` (and `validateServing` at `validate.go:258-262` forbids `Probe.SourceV4` from overlapping any member binding in `@allowed_v4`). Thus, traffic entering on the uplink (`fv-ih`) toward a revoked member (`ip daddr` not in `@probe_v4` and removed from `@allowed_v4`) misses `probe_down` and is dropped immediately by `oifname "fv-fh" ip daddr != @allowed_v4 drop` before reaching any member download accept.
- **What a revoked member keeps after `Revoke` succeeds vs. fails:**
  1. **Kernel `@allowed_v4` membership:** Removed atomically by `flush set inet fvpn allowed_v4` inside the `nft -f` transaction before `deleteConntrack` runs. Both upload and download packets for the revoked IP are dropped at rules 5–6 regardless of whether `conntrack -D` succeeds or fails.
  2. **tc HTB class (`render_tc.go:20-58`):** If the revoked member has no remaining bindings in `m.QoS.Members`, `Render` emits `class del dev fv-ih classid 1:<minor>` and `class del dev fv-fh classid 1:<minor>` in `plan.TC`, which `Apply` executes before `deleteConntrack` (`apply.go:75`). However, `reserveUnmappedClasses` (`classid.go:47`) preserves the `member_id -> minor` mapping in `plan.Assignments` so a future re-admission reuses the same minor.
  3. **Named nftables counters (`u_<dev>`, `d_<dev>`, `nu_<dev>`, `nd_<dev>`):** Kept in `table inet fvpn` (never deleted by `Render` so counters stay monotonic across transient drops). `Render` emits their last expected delta in `plan.Finals`, **but** as proven in **M-3**, `Samples(m, rs.Counters)` (`apply.go:332`) also continues emitting 4 `Sample` structs per revoked device on every subsequent `Apply` with `MemberID = uuid.Nil, BindingRevision = 0`.
  4. **Conntrack entries (including UDP flows with churning source ports):** `deleteConntrack` (`apply.go:254-275`) runs two commands per revoked IP: `conntrack -D -s <ip>` (matching original tuple source = member overlay IP, which deletes **all** outbound UDP flows regardless of how many source ports the member churned) and `conntrack -D -d <ip>` (matching original tuple destination = member overlay IP). If `conntrack -D` fails, the conntrack entries remain in the kernel table until timeout (and hold SNAT port bindings on `<snat>`), `Apply` returns `ApplyResult{Verified: true}` + `*ConntrackDeleteError` (with `Counters: nil, Samples: nil`, **M-3**), and the ruleset still blocks packet forwarding via `@allowed_v4`.

---

### 3. Deny-list boundaries (Question 3)

- **Exact contents of `SpecDeniedV4()` (`node/enforce/prefixes.go:10-48`):**
  16 merged prefixes covering every IANA IPv4 Special-Purpose Address Registry block whose "Globally Reachable" column is not `True`, plus multicast (`224.0.0.0/4`):
  `0.0.0.0/8`, `10.0.0.0/8`, `100.64.0.0/10`, `127.0.0.0/8`, `169.254.0.0/16`, `172.16.0.0/12`, `192.0.0.0/24`, `192.0.2.0/24`, `192.88.99.0/24`, `192.168.0.0/16`, `198.18.0.0/15`, `198.51.100.0/24`, `203.0.113.0/24`, `224.0.0.0/4`, `240.0.0.0/4`, `255.255.255.255/32`.
  (The three globally reachable IANA blocks `192.31.196.0/24` AS112-v4, `192.52.193.0/24` AMT, and `192.175.48.0/24` Direct Delegation AS112 are excluded by design.)
- **Public LAN behind the node (no RFC 1918 on the uplink):**
  - **Node's own addresses:** Protected by two mechanisms: (a) traffic addressed to a local IP on the node takes the kernel `input` hook (`iifname "fv-fh" drop` and `ip saddr 100.64.0.0/10 drop` at priority `-150`), never entering `forward`; and (b) in `forward`, `@host_v4` drops destinations in `m.HostV4` (and `m.Datapath.SNAT/32` when `m.Probe != nil`, see **L-2**).
  - **Node's public uplink subnet and gateway (`ExtraDeniedV4`):** `@denied_v4` is **not** limited to `SpecDeniedV4()`; `m.denyV4()` (`model.go:217-225`) unions `SpecDeniedV4()` with `m.ExtraDeniedV4` ("Host-detected on-link and reachable prefixes go here, including a private subnet that uses a public prefix (§8.1)"). If the caller populates the public uplink on-link prefix (e.g., `198.51.100.0/24` or a public `/24`) into `m.ExtraDeniedV4`, both the uplink subnet and the upstream gateway IP are in `@denied_v4` and dropped in `forward` (while transit packets routed *through* the gateway's MAC address carry the remote Internet `ip daddr` and are unaffected). If the caller does **not** populate `m.ExtraDeniedV4` for a public on-link subnet, other hosts on that public uplink L2 segment and the gateway IP itself remain reachable by members.
  - **Designated resolver (`m.Resolvers`) on ports other than 53:** In `Render` (`render_nft.go:130-156`), the ruleset emits:
    1. `iifname "fv-fh" ip daddr @resolver_v4 tcp dport 53 goto fvpn_dns_up` (accepts and counts member TCP/53 to `@resolver_v4`)
    2. `iifname "fv-fh" ip daddr @resolver_v4 udp dport 53 goto fvpn_dns_up` (accepts and counts member UDP/53 to `@resolver_v4`)
    3. `iifname "fv-fh" tcp dport 53 counter name "ctbadns" drop` and `udp dport 53 counter name "ctbadns" drop` (drops port 53 to any non-resolver)
    4. `iifname "fv-fh" ip daddr @denied_v4 counter name "ctdeny" drop`
    5. `iifname "fv-fh" ip daddr @host_v4 counter name "cthost" drop`
    There is **no** rule `ip daddr @resolver_v4 drop` for non-53 ports in serving mode (unlike baseline with probe at `render_nft.go:322`, which has `iifname "fv-fh" oifname "fv-ih" ip daddr @resolver_v4 counter name "ctbadns" drop` after the port-53 accepts!). Consequently, in serving mode, a resolver on a **public** IP address (not in `@denied_v4` or `@host_v4`) **is reachable by members on non-53 ports** (e.g., AdGuard admin port 3000/443/80) unless the resolver's IP is also placed in `ExtraDeniedV4` or `HostV4`! (When the resolver IP is in RFC 1918 / `100.64.0.0/10` / `HostV4`, rules 1–2 accept port 53 *before* `@denied_v4`/`@host_v4`, and rules 4–5 drop all non-53 ports to the resolver.)
- **`192.0.0.9/32` (RFC 7723 Port Control Protocol Anycast) and `192.0.0.10/32` (RFC 8155 Traversal Using Relays around NAT Anycast):**
  Both `/32` blocks sit inside the IANA parent block `192.0.0.0/24` (RFC 6890 IPv4 IETF Protocol Assignments, which is `Globally Reachable: False`). Because `SpecDeniedV4()` includes the parent `192.0.0.0/24` (`prefixes.go:24`) and nftables interval sets do not punch `/32` holes out of `192.0.0.0/24`, both `192.0.0.9/32` and `192.0.0.10/32` are denied by the parent `/24`. This is **intended** and explicitly documented in `prefixes.go:10-16` (the parent `192.0.0.0/24` is `Globally Reachable = False`, and fail-closed inclusion of the whole `/24` prevents leaking IETF protocol assignment addresses such as DS-Lite `192.0.0.0/29`).

---

### 4. Limiter keys, placement, memory, and metrics (Question 4)

- **Key source (`connectionRateKey`, `middleware.go:213-244`):**
  The limiter keys strictly on `r.RemoteAddr` (the TCP peer address), never on `X-Forwarded-For`, `Forwarded`, or `X-Real-IP` (`TestAnonymousAddressBurstUsesIPv6Prefix` in `limiter_test.go:190-200`). A client cannot bypass its bucket by forging forwarded headers.
- **Behavior behind lab and production nginx (`deploy/devstack/nginx.conf`):**
  In `deploy/devstack/nginx.conf:53-92`, nginx terminates TLS on `:443` and reverse-proxies `/api/` and `/auth/` to `https://control:8443` via standard HTTP `proxy_pass` (`X-Forwarded-Proto https`). Because standard HTTP `proxy_pass` opens TCP connections from the nginx container's IP to `control:8443`, `r.RemoteAddr` seen by `fvpn-control` is always the nginx proxy IP (`127.0.0.1` or the container IP). Therefore:
  - Every unauthenticated client behind nginx shares **one** `<nginx_ip>` bucket for `login` (20/s, burst 40), `ingress` (30/s, burst 60), and `anonymous` (2/s, burst 10) — creating a shared-bucket self-denial bottleneck behind a reverse proxy unless `fvpn-control` terminates connections directly or uses a trusted-proxy header.
  - Conversely, the mTLS agent listener (`:8444` in devstack) is exposed directly or via TCP passthrough (since mTLS requires the client certificate handshake on `fvpn-control`), so `r.RemoteAddr` is the node's real TCP peer IP.
- **mTLS classes (`heartbeat`, `lease`, `usage`, `agent`) and two nodes behind one NAT IP (`TestReviewR12_Q4_Q5_LimiterAndBodyBoundariesHeld`):**
  Every request passes **two** limiters in `httpapi.New` (`middleware.go:309` and `353`):
  1. Pre-auth `ingress` limiter keyed by `"ingress:" + ingressClass + ":" + peer` (where `peer` is IPv4 `/32` or IPv6 `/64`), using `IngressRate` (`30/s`, burst `60`).
  2. Post-auth class limiter `lim` keyed by `class + ":" + p.Scope()` (`"heartbeat:agent:<node_id>"`, `"lease:agent:<node_id>"`, `"usage:agent:<node_id>"`), using the class rate (`1/s` burst `5` for `heartbeat`/`lease`, `2/s` burst `10` for `usage`).
  When two mTLS nodes (`nodeA`, `nodeB`) share one NAT IPv4 address (`198.51.100.50`), each node gets its own full per-node class burst (5 heartbeats, 5 lease renewals, 10 usage batches), while sharing the per-class `ingress` address bucket (`30/s`, burst `60` per class on that NAT IP). Up to 12 nodes behind one NAT IP can burst 5 heartbeats simultaneously (`12 * 5 = 60`) before hitting the shared `IngressRate` address bucket.
- **Measured bursts for IPv6 `/64` keying (`middleware.go:236-240`):**
  `connectionRateKey` masks every IPv6 address to its `/64` prefix (`netip.PrefixFrom(addr, 64).Masked().Addr().String() + "/64"`).
  - **Client with a `/128` allocation (or multiple `/128` hosts in one `/64`):** All `/128` addresses in `2001:db8:1:2::/64` map to `"2001:db8:1:2::/64"` and share a single bucket: measured burst is **10** on `anonymous` (`EnrollNode`), **40** on `login`, and **60** on `ingress` (`TestReviewR12_Q4_Q5_LimiterAndBodyBoundariesHeld`).
  - **Attacker with a `/48` allocation ($65,536$ `/64` prefixes):** Each `/64` gets an independent bucket (10 burst on `anonymous`, 60 burst on `ingress` per `/64`), up to the `MaxRatePrincipals = 10,000` map cap ($5,000 \times 10 = 50,000$ anonymous requests or $2,500$ unauthenticated mTLS sweeps across 4 classes), at which point the shared `ingress` map fills and locks out all new IPs for 1 hour (**M-2**).
- **Memory and CPU cost of a $10^6$-address sweep (`middleware.go:183-211`):**
  The `limiter` map is strictly bounded at `MaxRatePrincipals = 10,000` entries (`config.go:88`). A `bucketState` value is 16 bytes (`tokens float64`, `updated time.Time` = 24 bytes -> 32 bytes struct) plus a ~45-byte string key and Go map bucket overhead (~120–160 bytes/entry), so each `limiter` instance (`ingress`, `lim`, `login`) uses **at most ~1.6 MiB of heap** at full capacity (`10,000` entries). During a sweep of $10^6$ source addresses:
  - The first `5,000` (on `anonymous`) or `10,000` (on `login`) distinct prefixes fill `l.entries` to `10,000`.
  - For the remaining $990,000+$ addresses within the 1-hour window, `len(l.entries) >= l.max` triggers an $O(N)$ (`N = 10,000`) map scan (`for k, v := range l.entries { if now.Sub(v.updated) > time.Hour { delete(...) } }`) under `l.mu.Lock()` on **every single request**! At $N = 10,000$, each failed eviction scan takes ~80–150 µs under the global mutex, serializing all requests at ~7,000–12,000 req/s CPU-bound on `l.mu` while allocating zero new map entries.
- **`Retry-After` correctness at the boundary (`middleware.go:205-209`, `pkg/apierr/apierr.go`):**
  `delay := time.Duration((1 - b.tokens) / r.PerSecond * float64(time.Second))` with `if delay < time.Second { delay = time.Second }`. `apierr.Write` formats `Retry-After` by ceiling sub-second durations to integer seconds (`int( (d + time.Second - 1) / time.Second )`), so any positive wait `(0, 1s]` emits `Retry-After: 1`, `(1s, 2s]` emits `Retry-After: 2`, etc.
- **Metrics label cardinality (`fvpn_control_rate_limited_total`):**
  Labels are strictly `(class, limiter)` where `class` $\in$ `{"anonymous", "heartbeat", "lease", "usage", "agent", "browser", "login"}` (7 values) and `limiter` $\in$ `{"address", "subject", "concurrency"}` (3 values), giving a bounded cardinality of at most **$7 \times 3 = 21$ time series** (never keyed by IP, node ID, or route path).

---

### 5. Body limits after follow-up 16 (Question 5)

#### Measured exact byte boundaries vs. documented limits (`internal/config/config.go:63-86`, `docs/control/README.md:91-101`, `docs/control/AUTH.md:114-125`)

In `middleware.go:398`: `io.ReadAll(io.LimitReader(r.Body, bodyLimit+1))` rejects `len(raw) > bodyLimit` with HTTP **413 `PAYLOAD_TOO_LARGE`**. Within `bodyLimit`, a schema-valid JSON body padded with spaces returns **200/202** (or `501` when unimplemented in bare router fixtures), malformed JSON returns **400 `MALFORMED_REQUEST`**, and schema-invalid JSON returns **422 `VALIDATION_FAILED`**:

| Class | Operations | Documented Cap (`docs/control/README.md` / `config.go`) | Max Valid (`200`/`202`/`501`) | Min `413 PAYLOAD_TOO_LARGE` | Malformed JSON (`400`) |
|---|---|---:|---:|---:|---:|
| **`browser`** | Admin & Member routes | `64 KiB` (`BrowserBodyBytes = 64 << 10`) | **`65,536` B** | **`65,537` B** | `1 .. 65,536` B |
| **`agent`** | `GetAgentDesiredState`, `RenewNodeCertificate`, etc. | `256 KiB` (`AgentBodyBytes = 256 << 10`) | **`262,144` B** | **`262,145` B** | `1 .. 262,144` B |
| **`heartbeat`** | `SubmitObservation` | `256 KiB` (`AgentBodyBytes = 256 << 10`) | **`262,144` B** | **`262,145` B** | `1 .. 262,144` B |
| **`lease`** | `RenewServingLease` | `256 KiB` (`AgentBodyBytes = 256 << 10`) | **`262,144` B** | **`262,145` B** | `1 .. 262,144` B |
| **`anonymous` (enroll)** | `EnrollNode` | `256 KiB` (`AgentBodyBytes = 256 << 10`) | **`262,144` B** | **`262,145` B** | `1 .. 262,144` B |
| **`anonymous` (webhook)** | `ReceiveProviderWebhook` | `256 KiB` (`WebhookBodyBytes = 256 << 10`) | **`262,144` B** | **`262,145` B** | `1 .. 262,144` B |
| **`usage`** | `SubmitUsageBatch` | `8 MiB` (`UsageBodyBytes = 8 << 20`) | **`8,388,608` B** | **`8,388,609` B** | `1 .. 8,388,608` B |
| **Bootstrap** | `ConsumeAgentBootstrap` | `0 B` (empty body required, `middleware.go:391-396`) | **`0` B** | **`262,145` B** | **`1 .. 262,144` B** (`400`) |

- **Adversarial bodies at the usage cap (`8 MiB`) (`TestReviewR12_Q4_Q5_LimiterAndBodyBoundariesHeld` & `TestReviewR12_Q5_AboveCanonMaxBytesAcceptsDuplicateKeysAndInvalidUTF8`):**
  - **Deep nesting:** At `<= 1 MiB`, `canonjson.Canonicalize` rejects nesting depth `> 256` (`canonjson.MaxDepth = 256`) with `400 MALFORMED_REQUEST`. At `(1 MiB, 8 MiB]`, `encoding/json` enforces its internal `maxNestingDepth = 10000` (returning `400` above 10,000 levels, and `422 VALIDATION_FAILED` for 257–10,000 levels).
  - **Many duplicate keys, invalid UTF-8, lone UTF-16 surrogates:** Rejected with `400 MALFORMED_REQUEST` at `<= 1,048,576` bytes (`canonjson.MaxBytes`), but **accepted by `checkJSONDocument` at `1,048,577 .. 8,388,608` bytes** (**M-1**).
  - **Huge numbers:** At `<= 1 MiB`, `canonjson` rejects integers outside $\pm(2^{53}-1)$ (> 16 digits) with `400`. At `(1 MiB, 8 MiB]`, `json.NewDecoder` into `json.RawMessage` accepts arbitrary-length digit tokens in `checkJSONDocument`, and OpenAPI validation (`kin-openapi`) or `decstr.U64` rejects out-of-schema numbers with `422`/`400`.
  - **NUL byte (`0x00`):** Both `canonjson` (`<= 1 MiB`) and `encoding/json` (`> 1 MiB`) reject an unescaped `0x00` byte anywhere inside or outside a JSON string with `400 MALFORMED_REQUEST`.
  - **Valid JSON whose canonical form exceeds 1 MiB while raw bytes do not:** Mathematically impossible for `canonjson.Canonicalize(raw)`: every valid JCS token (stripped whitespace, unescaped `\uXXXX` / surrogate pairs to UTF-8, control chars `\u00xx` / `\b\t\n\f\r`, integers without leading zeros or exponents) has `len(canonical) <= len(raw)`. Conversely, for `SubmitUsageBatch` in `(1 MiB, 8 MiB]`, `internal/usage/ingest.go` hashes each individual `UsageEvent` with `canonjson.HashHex` (1 MiB cap per event), while a batch of up to `MaxBatchEvents = 500` events can total up to 8 MiB.
  - **Chunked transfer with no `Content-Length` (`r.ContentLength = -1`) and slow body:** `io.ReadAll(io.LimitReader(r.Body, bodyLimit+1))` (`middleware.go:398`) reads from chunked streams regardless of `r.ContentLength` and returns `413` at `bodyLimit + 1` bytes (`8,388,609` B for usage). A slow-loris body is bounded by the server's `ReadTimeoutSeconds` (default `15s`) and `WriteTimeoutSeconds` (default `15s`, `middleware.go:265`), while holding 1 slot in `gates[class]` (`MaxConcurrent = 64`).
  - **CPU and memory per 8 MiB usage request, and when limiter cost is paid:**
    - **Limiter ordering:** Both `ingress.allow` (`middleware.go:309`), `bucket.allow` (`middleware.go:353`), and `gates[class]` (`middleware.go:365`) execute **before** `io.ReadAll` (`middleware.go:398`). The rate-limit token is spent before reading a single byte of the body (`TestReviewR12_Q4_Q5_LimiterAndBodyBoundariesHeld`).
    - **Memory & CPU:** Reading an 8 MiB usage body allocates ~8 MiB for `raw` (`io.ReadAll`), ~8 MiB for `doc` (`json.RawMessage` in `checkJSONDocument`), and ~16–24 MiB for the `kin-openapi` `interface{}` AST + handler `api.UsageBatchRequest` struct (~32–40 MiB peak per in-flight 8 MiB request; bounded by `MaxConcurrent = 64` and `UsageRate = 2/s` burst `10` per node).
  - **Signed-document path (`canonjson.MaxBytes`):** `canonjson.MaxBytes` (`1 << 20 = 1,048,576` bytes) is unchanged (`pkg/canonjson/canonjson.go:48, 193`), and `envelope.Sign` / `envelope.Verify` (`pkg/envelope/envelope.go:206`) call `canonjson.Canonicalize(raw)` first, strictly refusing any signed document above 1 MiB (`TestReviewR12_Q4_Q5_LimiterAndBodyBoundariesHeld`).

---

### 6. D-063 issuer inputs and TOCTOU (Question 6)

- **TOCTOU between `challengeProbe` (`node/updater`) and the agent's install (`node/agent`):**
  1. `fvpn-updater` calls `challengeProbe` (`node/updater/probe_addresses.go:181`) under the update lock during `issueChallenge` (`attest.go:38`), resolving DNS (`control_url`, `result_url`, resolver `admin_url`/`dns_addr`) and reading `net.InterfaceAddrs()`, and writes `health-challenge.json` (TTL `45s`).
  2. During the 45-second window before/while `fvpn-agent` reads `health-challenge.json`, interface addresses, DNS answers for `control_url`, or `agent.json` on disk can change.
  3. **What `fvpn-agent` re-checks (`node/agent/probeexc.go:26-99`):** When the agent loads `health-challenge.json` (`activeProbeException`), `validateChallengeProbe` (`probeexc.go:53-99`) re-verifies:
     - Challenge signature/freshness, `node_id`, `incarnation_id`, `boot_id`, `release_id`, and `ExpiresAt` ($\le 300\text{s}$).
     - `SourceV4` is a valid IPv4 `/32` inside `100.64.0.0/10` and not equal to `cfg.Datapath.SNAT` (`probeexc.go:65-72`).
     - Every destination `AddrV4` is a valid IPv4 `/32`, not equal to `cfg.Datapath.SNAT`, not equal to any configured resolver in `cfg.Resolvers`, and not contained in `enforce.SpecDeniedV4()` or `cfg.ExtraDeniedV4` (`probeexc.go:77-96`).
     - Additionally, `enforce.Render` / `validateProbe` (`node/enforce/validate.go:232-279`) re-checks that `SourceV4` and `Destinations` do not overlap `m.HostV4`, `m.Datapath.SNAT`, `m.Resolvers` (for non-DNS rules), `m.Bindings` (`@allowed_v4`), or `m.denyV4()`.
     - **What `fvpn-agent` does NOT re-check:** The agent does **not** re-resolve `control_url` / `result_url` DNS or re-enumerate all non-SNAT host interface addresses when installing the probe exception from `health-challenge.json`.
- **DNS answer for the control host that omits the control plane's real address (`TestReviewR12_Q6_IssuerValidationHeld`):**
  If DNS for `control_url` returns only `192.0.2.1` while the real control plane is at `1.2.3.4` (or under split-horizon / round-robin DNS that omits `1.2.3.4`), `validateProbePeer` accepts `1.2.3.4/32` as a probe destination. The §13.9 enforcement exception (`Render` / `RenderBaseline`) then installs:
  - `@probe_v4 = { <SourceV4> timeout <remaining_ttl>s }` (at most 45s from `HealthChallengeTTL`, hard-capped at `ProbeMaxSeconds = 300s`)
  - `@probe_dst_v4 = { 1.2.3.4 . <proto> . <port> timeout <remaining_ttl>s }`
  - `iifname "fv-fh" oifname "fv-ih" ip saddr @probe_v4 ip daddr . meta l4proto . th dport @probe_dst_v4 counter name "probe_up" accept`
  - `iifname "fv-ih" oifname "fv-fh" ip daddr @probe_v4 ct state established,related counter name "probe_down" accept`
  - `oifname "fv-ih" ip saddr @probe_v4 snat ip to <SNAT>`
  Thus, during that $\le 45\text{s}$ challenge window, the designated probe peer (`SourceV4`) is allowed to send SNATed traffic from the node's uplink IP to `1.2.3.4` on that exact `<proto>/<port>` (and to `@resolver_v4:53` if `ResolverDNS == true`).
- **Symlinked or oversized `agent.json`, `control_url` with userinfo or port, IPv4-mapped answers, empty interface list (`TestReviewR12_Q6_EmptyInterfaceListAndSymlinkAgentConfig` & `TestReviewR12_Q6_IssuerValidationHeld`):**
  - **Symlinked `agent.json`:** Followed via `os.Open` + `f.Stat()` (**H-2**).
  - **Oversized `agent.json`:** `readProbeAgentConfig` (`probe_addresses.go:60-64`) reads `io.LimitReader(f, (1<<20)+1)` and fails closed with `probeRefusal("agent_config_unavailable")` at `1,048,577` bytes (`1 MiB + 1`).
  - **`control_url` carrying userinfo (`https://user:pass@host/api/v1`):** Refused fail-closed by `probeURLHost` (`probe_addresses.go:175`: `u.User != nil`) with `probeRefusal("control_addresses_unavailable")`.
  - **`control_url` carrying a port (`https://host:8443/api/v1`):** `probeURLHost` extracts `u.Hostname()` (`"host"`), resolves all its DNS answers, and refuses matching probe destinations with `probeRefusal("control_destination")`.
  - **IPv4-mapped IPv6 answers (`::ffff:1.2.3.4`, `::ffff:100.64.8.1`):** Both `probeEnvironment.addresses` (`probe_addresses.go:112`) and `resolve` (`probe_addresses.go:123, 134`) call `addr.Unmap()`, converting IPv4-mapped IPv6 interface and DNS addresses to IPv4 and refusing matching `node_source`, `node_destination`, `control_destination`, and `resolver_destination`.
  - **Empty interface list:** Accepted with `known.node == nil` (**H-2**).
  - **Auditability of probe omission:** Only logged once per reason class (`"health challenge probe omitted"`); omitted from `health-challenge.json`, `health-attestation.json`, `update-journal`, `UpdateResult`, and `ObservationReport` (**H-2**).

---

### 7. Seeding rule edges (Question 7)

All four edge cases were tested against PostgreSQL in `TestReviewR12_Q7_SeedingRuleEdgesHeld`, `TestReviewR12_Q7_CrossJobAuditAndSeedingMisattribution`, and `TestReviewR12_Q7_PreIntentTargetSeedPoisonsAttemptRelease` (`internal/fleetupdate/review_r12_seeding_integration_test.go`):
1. **Job target recorded for a different incarnation (`serving.go:126`):**
   `if equalID(j.incarnation, &i.IncarnationID) { targetRelease = &j.releaseID }` strictly checks that the job's `execution_incarnation_id` equals the authenticated node's `IncarnationID`. A job target belonging to a different incarnation leaves `targetRelease == nil` and cannot seed `current_release_id`. (In addition, `ObserveReportTx` at `serving.go:228-235` rejects any `ObservationReport` whose `r.Update.JobId` has `execution_incarnation_id != i.IncarnationID` with `409 INCARNATION_MISMATCH`.)
2. **Recorded `last-good` that is subsequently revoked (`serving.go:65-75`, `133-141`):**
   `NodeVersionView` (`serving.go:65-75`) computes `RecoveryRequired` from whether `nodes.last_good_release_id` is in `fleet_config.revoked_release_ids` (or when `last_good_release_id IS NULL` and `current_release_id` is revoked). When the recorded `last_good_release_id` (`l.old`) is subsequently revoked via `ApplySupportTx`, `NodeVersionView` immediately reports `RecoveryRequired = true` while pre-result heartbeats reporting `last_good_release_id = l.old` remain allowed (`equalID(id, n.lastGood)` is true) and never clear `RecoveryRequired`.
3. **Two heartbeats alternating `current` between the two recorded values (`l.old` and `l.new`) while trying to move `last-good` to `l.new` (`serving.go:118, 136-141`):**
   Because `observeTx` never executes an `UPDATE` on `nodes.last_good_release_id` (only `SubmitResult` on an accepted `succeeded` / `already_current` result updates `last_good_release_id`), alternating `CurrentReleaseID` between `l.new` and `l.old` only moves `nodes.current_release_id` between `l.new` and `l.old`. `nodes.last_good_release_id` remains `l.old` throughout, and because `o.LastGoodReleaseID` (`l.new`) $\neq$ `n.lastGood` (`l.old`), `agrees` is `false` on every such heartbeat and `last_observed_at` is set to `NULL` (`Unconfirmed = true`).
4. **Heartbeat that names `target` (`l.new`) as `last-good` while `current` stays on the boot release (`l.old`) (`serving.go:118, 133-158`):**
   - `equalID(n.current, o.CurrentReleaseID)` is `true` (`l.old == l.old`), so line 136 (`!equalID(n.current, o.CurrentReleaseID)`) is `false` and `nodes` is not updated (`current = l.old, last_good = l.old`).
   - `agrees` (`serving.go:118`) is `false` (`n.lastGood = l.old != o.LastGoodReleaseID = l.new`), so line 158 sets `update_jobs.last_observed_at = NULL` (`Unconfirmed = true`).
   - Because `expected(id)` (`serving.go:133-135`) returns `true` for `targetRelease` (`l.new`) even when checking `o.LastGoodReleaseID` at line 144, **zero** `update_observation.unexpected_release` audits are written when `o.LastGoodReleaseID == targetRelease` (audits are only written for releases outside `{targetRelease, n.current, n.lastGood}`).
   - *(See **H-3** for the three seeding flaws found around cross-job `n.job` vs `jobID`, partial mutation of `current_release_id` when `last_good_release_id` is unexpected, and pre-intent target seeding poisoning `attempt_release_id`.)*
