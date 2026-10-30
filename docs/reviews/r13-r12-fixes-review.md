# Adversarial Security Review — `r13-r12-fixes-review`

- **Branch:** `r13-r12-fixes-review`
- **Commit:** `12496fd0d057632919d8f1153b92a9a660fdc520`
- **Date:** `2026-10-30 (2026-10-30 14:00:32 -0400)`
- **Reviewing model:** `agy/gemini-4-argon-high` (recorded in commit `12496fd` `Agent:` trailer)
- **Source report:** `.agent/REVIEW.md` (git blob `e66393db6a16b7ce94472564e526a7254720114a`, SHA-256 `13bc71172ff626baded35dee6993448ca6d0df842e08c1bd766f8085e3f13134`)
- **Review branch proof tests:** `r13-r12-fixes-review:node/enforce/review_r13_enforce_test.go`, `r13-r12-fixes-review:internal/httpapi/review_r13_limits_test.go`, `r13-r12-fixes-review:internal/fleetupdate/review_r13_seeding_integration_test.go`, `r13-r12-fixes-review:node/agent/review_r13_agent_test.go`, `r13-r12-fixes-review:node/updater/review_r13_updater_test.go`

## Scope

Adversarial security review of the `r12` remediation wave (`#166`–`#174`):

1. **Node enforcement verifier and non-53 resolver drop** (`node/enforce/*` in `#172`)
2. **Revocation conntrack retry and sample filtering** (`node/agent/revocation.go`, `node/agent/engine.go`, `node/agent/decide.go` in `#166`; `node/enforce/apply.go` in `#172`)
3. **Control rate-limit tables, `trusted_proxies`, and body validation** (`internal/httpapi/middleware.go`, `internal/httpapi/jsondoc.go`, `internal/config/proxies.go` in `#169`, `#174`)
4. **D-063 probe omission and issuer address guards** (`node/updater/*`, `node/internal/updatejournal/*`, `node/internal/healthattest/*`, `node/agent/probeexc.go`, `node/agent/attest.go` in `#168`, `#171`)
5. **Pre-result heartbeat seeding** (`internal/fleetupdate/serving.go`, `internal/fleetupdate/retry.go`, `internal/fleetupdate/results.go` in `#167`)

## Findings

| Severity | Finding | Status | Closing PR |
|---|---|---|---|
| High | H-1: `VerifyServing` and `VerifyBaseline` (`node/enforce/parse_nft.go`, `node/enforce/verify.go`) accept out-of-model forwarding rulesets (`jump`/`goto` into unchecked `output`, `vmap` verdict maps, `!=` negated `dport`/`sport`/`iifname`/`oifname`, anonymous sets, `meta l4proto`/`meta mark` narrowing on mandatory drops, `flow add`/`FLOWOFFLOAD`, and unverified `meta priority` class IDs; `TestReviewR13_Q1_VerifierAcceptsOutOfModelRulesetsAndQoS`). | open | pending (netlab follow-up) |
| High | H-2: `parseForwardedIP` and `splitUnbracketedIPv6` (`internal/httpapi/middleware.go`) strip a trailing `:0..9` hextet on IPv6 addresses containing `::` as if it were a port, shifting `/64` (`2001:db8:1::2:3:4:5:6` in `2001:db8:1:2::/64` -> `2001:db8:1::/64`) into `trusted_proxies` and allowing spoofed `X-Forwarded-For` hops (`TestReviewR13_Q3_XForwardedForIPv6HextetStrippedAsPortBypassingTrustedProxy`). | open | pending (control follow-up) |
| High | H-3: `AuthenticateAgent` (`internal/httpapi/middleware.go`, `internal/pki/service.go`) runs uncached PostgreSQL queries (`CheckFence` + 3-table `JOIN`) before `ingress[class]`, `subjects`, and `gates[class]` on the agent mTLS listener and returns on auth failure before spending any limiter token (`TestReviewR13_Q3_AgentAuthRunsBeforeRateLimiterAndConcurrencyGate`). | open | pending (control follow-up) |
| High | H-4: Before any `UpdateResult` is accepted for an incarnation (`!accepted`), `observeTx` (`internal/fleetupdate/serving.go`) lets a heartbeat for an issued job `J1` overwrite `nodes.current_release_id` to `target` without an accepted result, poisoning `attempt_release_id` when `J1` is superseded by `J2` and causing `SubmitResult` (`internal/fleetupdate/results.go`) to reject `J2`'s legitimate pre-health failure with `422 VALIDATION_FAILED` (`TestReviewR13_Q6_IssuedJobHeartbeatMovesCurrentReleaseWithoutAcceptedResultAndPoisonsSupersedingBatch`). | open | pending (fleet-update follow-up) |
| Medium | M-1: `Service.Retry` (`internal/fleetupdate/retry.go`) creates Attempt 2 with the same `generation` as superseded Attempt 1, and `ObserveReportTx` (`internal/fleetupdate/serving.go`) checks only `j.generation == generation` (not `jobID == n.job`), so a late heartbeat carrying Attempt 1's `job_id` advances `fleet_node_state.last_observed_at` and flips confirmed Attempt 2 back to `Unconfirmed = true` (`TestReviewR13_Q6_SupersededAttemptHeartbeatSameGenerationInvalidatesCurrentAttemptConfirmation`). | open | pending (fleet-update follow-up) |
| Medium | M-2: `Engine.addresses` (`node/agent/engine.go`) tracks only flat IPv4 addresses without `DeviceID`/`MemberID` (skipping `conntrack -D` on a same-IP device swap and charging old-flow download bytes to the new device's counter), and `mergeRevocations` (`node/agent/revocation.go`) never prunes re-admitted IPv4s from `pendingRevoke` (`TestReviewR13_Q2_SameIPDeviceSwapSkipsConntrackDelete`, `TestReviewR13_Q2_PendingRevokeNotPrunedOnReAdmission`). | open | pending (agent follow-up) |
| Medium | M-3: When `conntrack` is removed or permanently fails after `Boot` preflight, `Engine.apply` (`node/agent/engine.go`, `node/agent/decide.go`) retries once per tick forever while keeping `MemberForwardingEnabled = true`, `SafetyStatus = passed`, and all `Checks = passed`, and `SubmitObservation` (`internal/nodes/observations.go`) still marks `member.revoke` / `member_device.revoke` `"nodes"` steps `succeeded` (`TestReviewR13_Q2_PermanentConntrackFailureNeverEscalates`). | open | pending (agent / control follow-up) |
| Medium | M-4: `seal` (`node/updater/apply.go`, `node/updater/result.go`) maps `StageRecoveryRequired` via `journalStageAPI` to `"report"`, overwriting `txn.ErrorStage` (`"rollback"` or `"migrate"`), and when `toRecovery` is entered before health checks (`txn.Health == nil`), seals `stage = "report"` with `health = null`, which central `SubmitResult` (`internal/fleetupdate/results.go`) rejects with `422 VALIDATION_FAILED` (`TestReviewR13_Q5_ToRecoveryOverwritesErrorStageWithReportAndSealsNullHealth`). | open | pending (updater follow-up) |
| Medium | M-5: `evictIdle` (`internal/httpapi/middleware.go`) inspects only `l.tail` (causing head-of-line blocking under mixed rates when faster buckets ahead of `l.tail` are 100% full), and an IPv6 `/48` sweep of `anonymousLim` denies legitimate `POST /api/v1/agent/enroll` requests with `429` (`TestReviewR13_Q3_EvictIdleTailOnlyHeadOfLineBlockingAndAnonSweep`). | open | pending (control follow-up) |
| Low | L-1: `ParseQdiscJSON` and `parseClassText` (`node/enforce/parse_tc.go`) ignore `fq_codel`'s parent major handle (`99:<minor>` vs `1:<minor>`) and allow inline `# rate ... ceil ...` comments and `\` continuations to mask rate/ceil mismatches (`TestReviewR13_Q1_VerifierAcceptsOutOfModelRulesetsAndQoS`). | open | pending (netlab follow-up) |
| Low | L-2: After `walkJSONDocument` streams a `(1 MiB, 8 MiB]` body, `openapi3filter.ValidateRequest` and `json.Unmarshal` + `matchingIdentity` (`internal/httpapi/middleware.go`) unmarshal the full multi-MiB body into `any` DOMs twice before the handler unmarshals it a third time (`TestReviewR13_Q4_MatchingIdentityUnmarshalsFullMultiMiBBodyIntoAny`). | open | pending (control follow-up) |
| Low | L-3: `RecordProbeOmission` (`node/internal/updatejournal/probe.go`) writes `probe_omission/<txn_id>/<nonce>` keys into SQLite `meta`, and neither `cleanTxnNotes` (`node/updater/apply.go`) nor `Retain` (`node/updater/retain.go`) prunes them (`TestReviewR13_Q5_ProbeOmissionJournalMetaRowsNeverPruned`). | open | pending (updater follow-up) |

## What Held

1. **Streaming large-body JSON lexer (`internal/httpapi/jsondoc.go`):** `walkJSONDocument` enforces valid UTF-8, rejects BOM and unescaped control characters (`0x00`–`0x1F`), rejects lone UTF-16 surrogates, canonicalizes escaped keys before duplicate-key comparison at every object depth, caps nesting at `canonjson.MaxDepth` (`256`), rejects fractions/exponents/leading zeros/`-0` and integers outside `±(2^53-1)` (`±9007199254740991`), and rejects trailing non-whitespace tokens in a single streaming pass over `bytes.NewReader(raw)`.
2. **D-063 regular-file and non-symlink check on `agent.json` (`node/updater/probe_addresses.go`):** `readProbeAgentConfig` checks `os.Lstat` regular mode before opening with `syscall.O_NOFOLLOW|syscall.O_NONBLOCK`, verifies `os.SameFile(lst, st)` and `st.Mode().IsRegular()`, and caps reads at `1 MiB` (`1,048,577` bytes refused with `agent_config_unavailable`).
3. **D-063 empty node IPv4 interface refusal and bounded observation detail (`node/updater/probe_addresses.go`, `node/agent/probeexc.go`, `node/agent/attest.go`):** When `interfaces()` returns zero global-unicast IPv4 addresses, `probeEnvironment.addresses` fails closed with `node_addresses_unavailable`, and `fvpn-agent` sanitizes `ObservationReport.checks[name=tunnel_probe].detail` via `knownOmissionReason` (`node_addresses_unavailable`, `agent_config_unavailable`, `control_addresses_unavailable`, `peer_address_conflict`, `peer_destination_invalid`, or `unspecified`) without leaking `peer_id` or IP addresses, while keeping `MemberForwardingEnabled = false`.
4. **Control `ParseTrustedProxies` minimum prefix validation (`internal/config/proxies.go`):** `ParseTrustedProxies` rejects prefix lengths shorter than `/8` for IPv4 and `/32` for IPv6, IPv4-in-IPv6 prefixes, and empty CIDR tokens at config load time.
5. **Pre-intent seeding guard and `last_good_release_id` agreement (`internal/fleetupdate/serving.go`):** Requiring `j.issued != nil` and `equalID(n.lastGood, o.LastGoodReleaseID)` before pre-result seeding prevented pre-intent heartbeats from poisoning `attempt_release_id` and prevented partial mutation of `current_release_id` when `last_good_release_id` disagreed.

---

## Detailed Review Notes

### High Findings

#### `H-1` — `VerifyServing` and `VerifyBaseline` accept out-of-model forwarding rulesets (`open`)

- **Where:** `node/enforce/parse_nft.go:270-360`, `node/enforce/parse_nft.go:404-482`, `node/enforce/verify.go:53-91`, `node/enforce/verify.go:196-399`, `node/enforce/verify.go:592-625`
- **Problem:** Eight classes of out-of-model nftables rules and QoS drifts pass `VerifyServing(rs, m)` (and several also pass `VerifyBaseline(rs, m.Datapath)`):
  1. `jump` / `goto` into `chain inet fvpn output` (which `verifyBaseChains` allows in `allowedChains` without inspecting its rules, where an `accept` returns a terminal verdict to `forward`) or into unverified regular chains.
  2. Verdict maps (`vmap`, e.g., `ip daddr vmap { 0.0.0.0/0 : accept }`) ignored by `parseRule`.
  3. Negated `!=` on `dport`/`sport` (`udp dport != 53`) and `iifname`/`oifname` (`parseMatch` ignores `m.Op` for ports and interfaces, letting `udp dport != 53 ip daddr @resolver_v4 accept` pass `isMemberAllowDNS` and `iifname != "fv-fh" ip saddr != @allowed_v4 drop` satisfy `seenUploadAllowDrop`).
  4. Anonymous sets (`udp dport { 53, 443 }`, where `extractPort` returns `53` from the first element).
  5. Extra narrowing predicates (`meta l4proto icmp`, `meta mark 0xdead`, `tcp dport 1`) on mandatory drop rules (`@allowed_v4`, `@denied_v4`, `@host_v4`, non-resolver port 53, and `@resolver_v4` non-53).
  6. `flow add @f` / `xt target "FLOWOFFLOAD"` bypassing later rules and counters via the netfilter flowtable fastpath.
  7. `VerifyBaseline` checks only `len(fwd.Rules) > 0` without rejecting `accept`/`jump`/`goto`/`vmap` rules in `forward`.
  8. `isMemberDeviceRule` checks `r.HasPriority` (`true`) without verifying `r.Priority == tcHandle(d.ClassMinor)`.
- **Proof:** `TestReviewR13_Q1_VerifierAcceptsOutOfModelRulesetsAndQoS` in `r13-r12-fixes-review:node/enforce/review_r13_enforce_test.go`.

#### `H-2` — `parseForwardedIP` and `splitUnbracketedIPv6` strip a trailing `:0..9` hextet on compressed IPv6 addresses, bypassing `trusted_proxies` (`open`)

- **Where:** `internal/httpapi/middleware.go:793-832`
- **Problem:** When an unbracketed IPv6 address in `X-Forwarded-For` contains `::` (`strings.Count(raw, ":") < 7`) and ends in a single decimal digit `:0` through `:9` (for example, `2001:db8:1::2:3:4:5:6` in untrusted `/64` `2001:db8:1:2::/64`), `splitUnbracketedIPv6` strips `:6` as a port because `2001:db8:1::2:3:4:5` is also a syntactically valid compressed IPv6 address—whose first four hextets shift from `2001:0db8:0001:0002` to `2001:0db8:0001:0000` (`2001:db8:1::/64`). When `2001:db8:1::/64` is in `trusted_proxies`, `effectiveClientAddr` treats the untrusted client's hop as a trusted proxy and walks further left in `X-Forwarded-For`, allowing `X-Forwarded-For` spoofing across rate-limit buckets.
- **Proof:** `TestReviewR13_Q3_XForwardedForIPv6HextetStrippedAsPortBypassingTrustedProxy` in `r13-r12-fixes-review:internal/httpapi/review_r13_limits_test.go`.

#### `H-3` — `AuthenticateAgent` runs uncached PostgreSQL queries before rate limiters and concurrency gates (`open`)

- **Where:** `internal/httpapi/middleware.go:275-351`, `internal/pki/service.go:411-458`
- **Problem:** In `httpapi.New`, `authn.AuthenticateAgent` runs before `ingress[class].allow(addrKey)`, `subjects.allow(subjKey)`, and `gates[class].acquire()`. For every request carrying any leaf certificate on the mTLS listener (`heartbeat`, `lease`, `usage`, `agent`), `AuthenticateAgent` executes two synchronous PostgreSQL queries (`CheckFence` + 3-table `JOIN` across `nodes`, `node_incarnations`, and `node_certificates`), and when `VerifyPeerCertificate` fails (expired/revoked cert, fenced node, superseded incarnation, or unknown CA), returns `401`/`403` before `ingress[class].allow` or `gates[class].acquire()` ever runs.
- **Proof:** `TestReviewR13_Q3_AgentAuthRunsBeforeRateLimiterAndConcurrencyGate` in `r13-r12-fixes-review:internal/httpapi/review_r13_limits_test.go`.

#### `H-4` — Pre-result heartbeat seeding for an issued job moves `nodes.current_release_id` without an accepted `UpdateResult` and poisons superseding batch `attempt_release_id` (`open`)

- **Where:** `internal/fleetupdate/serving.go:93-155`, `internal/fleetupdate/intent.go:168`, `internal/fleetupdate/results.go:178-183`
- **Problem:** While `!accepted` holds for an incarnation, `observeTx` includes `targetRelease = &j.releaseID` in `expected(...)` whenever `j.issued != nil` and writes `nodes.current_release_id = *o.CurrentReleaseID` when `equalID(n.lastGood, o.LastGoodReleaseID)` holds. If a node receives `NodeUpdateIntent` for job `J1` (`release-2.0.0`, setting `j1.issued != nil`) and sends a heartbeat with `current_release_id = release-2.0.0, last_good_release_id = release-1.0.0` before any `SubmitResult` is accepted, `observeTx` updates `nodes.current_release_id` to `release-2.0.0`. If `J1` is then superseded by `J2` (`release-2.0.1`), `J2`'s first `NodeUpdateIntent` snapshots `attempt_release_id = release-2.0.0` into `J2`'s `observed_evidence`, causing `SubmitResult` to reject `J2`'s legitimate pre-health failure (`current_release_id = release-1.0.0`) with `422 VALIDATION_FAILED`.
- **Proof:** `TestReviewR13_Q6_IssuedJobHeartbeatMovesCurrentReleaseWithoutAcceptedResultAndPoisonsSupersedingBatch` in `r13-r12-fixes-review:internal/fleetupdate/review_r13_seeding_integration_test.go`.

---

### Medium Findings

#### `M-1` — `Service.Retry` reuses the superseded attempt's `generation`, allowing a late Attempt-1 heartbeat to flip confirmed Attempt 2 back to `Unconfirmed = true` (`open`)

- **Where:** `internal/fleetupdate/retry.go:130-156`, `internal/fleetupdate/serving.go:53-119`, `internal/fleetupdate/serving.go:255-261`
- **Problem:** `Service.Retry` creates Attempt 2 (`J2`) with `generation = b.Generation` (identical to superseded Attempt 1 `J1`), while `ObserveReportTx` checks `j.generation != nil && *j.generation == generation` without checking `equalID(jobID, n.job)`. After `J2` succeeds at `T1`, a delayed heartbeat generated before `T1` carrying `job_id = J1` and `applied_config_generation = G` passes the generation check and advances `fleet_node_state.last_observed_at` to `T2 > T1` while `nodes.current_release_id` matches `J1`/`J2`'s `release_id`, causing `NodeVersionView` to flip `J2` back to `Unconfirmed = true` and `State = "failed"`.
- **Proof:** `TestReviewR13_Q6_SupersededAttemptHeartbeatSameGenerationInvalidatesCurrentAttemptConfirmation` in `r13-r12-fixes-review:internal/fleetupdate/review_r13_seeding_integration_test.go`.

#### `M-2` — Same-IP device swap skips `conntrack -D`, and re-admitted IPv4s remain in `pendingRevoke` (`open`)

- **Where:** `node/agent/engine.go:34`, `node/agent/engine.go:171-195`, `node/agent/revocation.go:16-67`, `node/enforce/render_nft.go:171-180`
- **Problem:** `Engine.addresses` tracks only `map[netip.Addr]struct{}` without `DeviceID` or `MemberID`, so when `D1` is replaced by `D2` with the same `OverlayIPv4` in one compilation, `removedAddresses` returns `nil`, `conntrack -D` never runs, and `D1`'s established conntrack entries continue forwarding while charging download bytes to `D2`'s `dl_<D2>` counter. In addition, `mergeRevocations` never removes newly admitted `current` IPv4s from `pendingRevoke`, so if `conntrack -D` fails transiently on `T1` and the IP is re-assigned on `T2`, `T2`'s retry flushes the new device's live conntrack flows.
- **Proof:** `TestReviewR13_Q2_SameIPDeviceSwapSkipsConntrackDelete` and `TestReviewR13_Q2_PendingRevokeNotPrunedOnReAdmission` in `r13-r12-fixes-review:node/agent/review_r13_agent_test.go`.

#### `M-3` — Permanent post-boot `conntrack -D` failure never disables member forwarding or blocks central revocation completion (`open`)

- **Where:** `node/agent/engine.go:180-195`, `node/agent/decide.go:39-108`, `internal/nodes/observations.go:258-323`
- **Problem:** If `conntrack` is removed or permanently fails after `Boot` preflight, `Engine.apply` catches `enforce.ConntrackDeleteError`, retains the IP in `pendingRevoke`, and returns `nil`, while `Decide` still sets `MemberForwardingEnabled = true`, `SafetyStatus = passed`, and all 6 `Checks = passed` (emitting only an informational problem code `ISOLATION_POLICY_INVALID`), and central `SubmitObservation` (`completeNodeOperationsTx`) still marks `member.revoke` / `member_device.revoke` `"nodes"` steps `succeeded`.
- **Proof:** `TestReviewR13_Q2_PermanentConntrackFailureNeverEscalates` in `r13-r12-fixes-review:node/agent/review_r13_agent_test.go`.

#### `M-4` — `toRecovery` overwrites `txn.ErrorStage` with `"report"` and seals `health: null` when entered before health checks (`open`)

- **Where:** `node/updater/apply.go:473-496`, `node/updater/result.go:175-218`, `internal/fleetupdate/results.go:223-227`
- **Problem:** `toRecovery` transitions the journal to `StageRecoveryRequired` and calls `seal(ctx, txn, code, msg)`, where `buildPayload` overwrites `stage` with `journalStageAPI(txn.Stage)` (`"report"` for `StageRecoveryRequired`) instead of preserving `txn.ErrorStage` (`"rollback"` or `"migrate"`). When `toRecovery` is entered before `StageHealthCheck` (`txn.Health == nil`), `buildPayload` seals `stage = "report"` with `health = null`, which central `SubmitResult` (`requireHealthEvidence`) rejects with `422 VALIDATION_FAILED`.
- **Proof:** `TestReviewR13_Q5_ToRecoveryOverwritesErrorStageWithReportAndSealsNullHealth` in `r13-r12-fixes-review:node/updater/review_r13_updater_test.go`.

#### `M-5` — `evictIdle` checks only `l.tail` (head-of-line blocking under mixed rates), and an IPv6 `/48` sweep of `anonymousLim` denies `POST /api/v1/agent/enroll` (`open`)

- **Where:** `internal/httpapi/middleware.go:254-305`, `internal/httpapi/middleware.go:577-637`
- **Problem:** `evictIdle` checks only `l.tail` and stops if `tokensAt(l.tail, now) < l.burst`, so on a multi-rate table such as `subjects` (`admin` `10/s` burst `20` vs. `member` `10/min` burst `10`), a single non-full `member` entry at `l.tail` prevents eviction of 9,999 `100%`-full `admin` entries ahead of it for up to `60 s`. Separately, `/api/v1/agent/enroll` (`agent` class) checks `anonymousLim` (`10,000` capacity, `60 s` eviction) before token lookup, so an unauthenticated sweep from `10,000` `/64` prefixes inside one IPv6 `/48` fills `anonymousLim` and causes legitimate `POST /api/v1/agent/enroll` calls from new addresses to fail with `429 RATE_LIMITED`.
- **Proof:** `TestReviewR13_Q3_EvictIdleTailOnlyHeadOfLineBlockingAndAnonSweep` in `r13-r12-fixes-review:internal/httpapi/review_r13_limits_test.go`.

---

### Low Findings

#### `L-1` — `ParseQdiscJSON` and `parseClassText` ignore `fq_codel` parent major handle and allow `#` comments / `\` continuations in `tc class show` (`open`)

- **Where:** `node/enforce/parse_tc.go:57-77`, `node/enforce/parse_tc.go:107-170`, `node/enforce/verify.go:504-552`
- **Problem:** `ParseQdiscJSON` parses only the `<minor>` suffix of `parent "<major>:<minor>"` (`parent 99:10` parses as `ParentMinor: 10`), and `parseClassText` joins `\`-continued lines before stripping `#` comments and scans tokens past inline `# rate ... ceil ...` comments.
- **Proof:** `TestReviewR13_Q1_VerifierAcceptsOutOfModelRulesetsAndQoS` in `r13-r12-fixes-review:node/enforce/review_r13_enforce_test.go`.

#### `L-2` — Multi-MiB `(1 MiB, 8 MiB]` usage bodies are unmarshaled into `any` DOMs twice in middleware after `walkJSONDocument` (`open`)

- **Where:** `internal/httpapi/middleware.go:404-417`, `internal/httpapi/middleware.go:505-530`
- **Problem:** After `walkJSONDocument` streams a `(1 MiB, 8 MiB]` `SubmitUsageBatch` body, `openapi3filter.ValidateRequest` and `json.Unmarshal(raw, &body)` before `matchingIdentity` allocate two full `map[string]any` / `[]any` DOM trees before the handler decodes the body a third time.
- **Proof:** `TestReviewR13_Q4_MatchingIdentityUnmarshalsFullMultiMiBBodyIntoAny` in `r13-r12-fixes-review:internal/httpapi/review_r13_limits_test.go`.

#### `L-3` — `probe_omission/<txn_id>/<nonce>` rows in SQLite `meta` are never pruned (`open`)

- **Where:** `node/internal/updatejournal/probe.go:23-69`, `node/updater/apply.go:639-646`, `node/updater/retain.go:118`
- **Problem:** `RecordProbeOmission` inserts `probe_omission/<txn_id>/<nonce>` keys into SQLite `meta`, and neither `cleanTxnNotes` nor `Retain` (`PruneHistory`) deletes them.
- **Proof:** `TestReviewR13_Q5_ProbeOmissionJournalMetaRowsNeverPruned` in `r13-r12-fixes-review:node/updater/review_r13_updater_test.go`.
