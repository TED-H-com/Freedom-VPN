# Adversarial Security Reviews Archive (`r3`–`r17`)

- **Date:** 2026-10-29 to 2026-10-31 (closure status)
- **Reviewing model:** `grok/grok-4.7`, `agy/gemini-4-argon-high`, `antigravity/gemini`, `antigravity/gemini-4-argon` (see per-review metadata below; `r5` and `r8` commits carried no `Agent:` trailer)
- **Decision / scope reviewed:** Wave 3 and Wave 4 adversarial security reviews (`r3-d055-review` through `r17-rev14-d104-d117-review`) across management peer discovery (`D-055`), DNS threat feeds (`D-054`), PostgreSQL pgBackRest backups and PITR drill (`D-025`, `AT30`), challenge-bound update health attestation (`D-053`), member identity mapping and device slot accounting (`D-052`, `D-029`), exit-group fencing and resolver-document delivery (`D-056`, `D-057`, `D-061`), update result evidence and rate limiting (`D-062`, `D-060`), synthetic-probe exception bounds and node agent hardening (`D-063`, `D-064`), zero-binding publication gating (`SECOPS-11`), node enforcement verifier, control rate-limit tables/large-body validation, `probe_omitted` evidence, and pre-result heartbeat seeding (`#167`–`#172`), the `r12` remediation verification wave (`#166`–`#174`), and the `r13` remediation and `D-067`–`D-072` verification wave (`#189`–`#202`). Earlier Wave 1 and Wave 2 reviews (`r1-db-review`, `r1-contracts-review`, `r2-control-review`, `r2-updater-review`, `r2-ledger-review`) are recorded in [`docs/ROADMAP.md`](../ROADMAP.md).

## Archive Index

| Review Branch | Archive File | Date | Reviewing Model | Scope / Decisions | Findings | Adopted Proof Files on `main` |
|---|---|---|---|---|---|---|
| `r3-d055-review` | [`r3-d055-review.md`](r3-d055-review.md) | 2026-10-29 | `grok/grok-4.7` (commit `f710843` trailer) | `D-055` management peer discovery via one-off enrollment correlation groups (`§6.1`, `§6.2`, `§6.4`, `§7.2`, `§14.5`, `§15.3`) | 9 (5 High, 3 Medium, 1 Low) — all closed in `#109` | [`internal/netbird/discovery_review_test.go`](../../internal/netbird/discovery_review_test.go), [`apps/worker/jobs_nodes_discovery_review_integration_test.go`](../../apps/worker/jobs_nodes_discovery_review_integration_test.go) |
| `r4-dnsfeeds-review` | [`r4-dnsfeeds-review.md`](r4-dnsfeeds-review.md) | 2026-10-29 | `agy/gemini-4-argon-high` (commit `e0a5756` trailer) | `D-054` threat-feed content pipeline, SSRF guard, Layer-3 compilation, `threat_feed_status` (`§12.1`–`§12.5`, `§14.5`, `DNS01`, `DNS02`, `SEC02`) | 7 (4 High, 2 Medium, 1 Low) — all closed in `#93`, `#101` | [`internal/dnsfeeds/review_test.go`](../../internal/dnsfeeds/review_test.go), [`internal/dnsfeeds/review_integration_test.go`](../../internal/dnsfeeds/review_integration_test.go), [`internal/dnspolicy/review_test.go`](../../internal/dnspolicy/review_test.go), [`apps/worker/review_integration_test.go`](../../apps/worker/review_integration_test.go) |
| `r5-backups-review` | [`r5-backups-review.md`](r5-backups-review.md) | 2026-10-29 | Not stated in report or commit `bc79093` | Central PostgreSQL backups, pgBackRest WAL archiving, `restore-pitr.sh`, `verify.sh`, `AT30` PITR drill (`§15.5`, `§17.2`, `§19.2`, `SECOPS-05`) | 11 (3 High, 6 Medium, 2 Low) — all closed in `#94`, `#100`, `#104`, `#107`, `#111` | [`deploy/backup/test/review_proof.sh`](../../deploy/backup/test/review_proof.sh) |
| `r6-d053-review` | [`r6-d053-review.md`](r6-d053-review.md) | 2026-10-29 | `antigravity/gemini` (commit `307deb7` trailer) | `D-053` challenge-bound update health attestation (`§13.6`, `§13.9`, `UPD03`–`UPD05`, `AT23`, `AT25`, `AT27`) | 7 (2 High, 4 Medium, 1 Low) — all closed in `#96`, `#99` | [`node/internal/healthattest/review_d053_test.go`](../../node/internal/healthattest/review_d053_test.go), [`node/updater/review_d053_test.go`](../../node/updater/review_d053_test.go), [`node/agent/review_d053_test.go`](../../node/agent/review_d053_test.go) |
| `r7-d052-review` | [`r7-d052-review.md`](r7-d052-review.md) | 2026-10-29 | `antigravity/gemini` (commit `0180e81` trailer) | `D-052` member identity mapping, device slot accounting, suspension/expiry withdrawal, OIDC `azp`, `D-029` audits (`§5`–`§7`, `§9`, `§14`–`§16`) | 11 (5 High, 6 Medium) — all closed in `#106`, `#109`, `#114`, `#122`, `#125`, `#128`, `#134`, `#137`, `#139` | [`internal/auth/review_d052_test.go`](../../internal/auth/review_d052_test.go), [`internal/auth/review_d052_integration_test.go`](../../internal/auth/review_d052_integration_test.go), [`internal/devices/review_d052_integration_test.go`](../../internal/devices/review_d052_integration_test.go), [`internal/members/httpadapter/review_d052_integration_test.go`](../../internal/members/httpadapter/review_d052_integration_test.go), [`apps/worker/review_d052_integration_test.go`](../../apps/worker/review_d052_integration_test.go) |
| `r8-d057-review` | [`r8-d057-review.md`](r8-d057-review.md) | 2026-10-30 | Not stated in report or commit `c6ea8e7` | `D-056` resolver-document delivery, `D-057` exit groups and publication membership, `D-061` resolver admin access (`§7`, `§12`, `§13`) | 4 (1 High, 2 Medium, 1 Low) — all closed in `#109`, `#120`, `#121`, `#126`, `#127` | [`internal/nodes/httpwire/review_d056_test.go`](../../internal/nodes/httpwire/review_d056_test.go), [`internal/nodes/httpwire/review_d057_test.go`](../../internal/nodes/httpwire/review_d057_test.go), [`node/agent/review_d056_test.go`](../../node/agent/review_d056_test.go), [`node/agent/review_d061_test.go`](../../node/agent/review_d061_test.go), [`node/dns/review_d061_test.go`](../../node/dns/review_d061_test.go) |
| `r9-d062-review` | [`r9-d062-review.md`](r9-d062-review.md) | 2026-10-30 | `grok/grok-4.7` (commit `fa34463` trailer) | `D-062` update result evidence and observation authority, `D-060` login rate limiter, `D-052` canonical issuer identity (`§9.1`, `§13.6`, `§13.10`, `§14.6`) | 6 (1 High, 3 Medium, 2 Low) — all closed in `#137`, `#139`, `#140`, `#141` | [`internal/fleetupdate/review_r9_d062_integration_test.go`](../../internal/fleetupdate/review_r9_d062_integration_test.go), [`internal/httpapi/review_r9_d060_test.go`](../../internal/httpapi/review_r9_d060_test.go), [`internal/members/issuer/issuer_test.go`](../../internal/members/issuer/issuer_test.go) (`TestReviewR9_CanonicalIssuerTable`) |
| `r10-d063-review` | [`r10-d063-review.md`](r10-d063-review.md) | 2026-10-30 | `antigravity/gemini-4-argon` (commit `42299c3` trailer) | `D-063` `§13.9` synthetic-probe exception bounds, health attestation, updater journal snapshot, `D-061` resolver credentials (`UPD02`–`UPD05`, `ISO01`, `MTR01`–`MTR05`, `DNS01`–`DNS04`) | 10 (4 High, 5 Medium, 1 Low) — all closed in `#140`, `#144`, `#150` | [`node/enforce/review_d063_test.go`](../../node/enforce/review_d063_test.go), [`node/agent/review_d063_test.go`](../../node/agent/review_d063_test.go) |
| `r11-d064-review` | [`r11-d064-review.md`](r11-d064-review.md) | 2026-10-30 | `grok/grok-4.7` (commit `8f1a49c` trailer) | `D-064` probe exception vs. `update.lock` & durable issuance (`#150`), `D-063` `probe_peer` address & `SpecDeniedV4` bounds (`#142`, `#144`), pre-result heartbeat seeding (`#140`), zero-binding publication gate (`#148`), lease `Retry-After` (`#146`) | 8 (1 High, 6 Medium, 1 Low) — all closed in `#154`, `#155`, `#156`, `#160`, `#162` | [`node/agent/review_r11_d064_test.go`](../../node/agent/review_r11_d064_test.go), [`node/internal/controlclient/review_r11_lease_test.go`](../../node/internal/controlclient/review_r11_lease_test.go), [`node/updater/review_r11_d063_test.go`](../../node/updater/review_r11_d063_test.go), [`internal/fleetupdate/review_r11_d064_integration_test.go`](../../internal/fleetupdate/review_r11_d064_integration_test.go), [`internal/nodes/review_r11_publication_test.go`](../../internal/nodes/review_r11_publication_test.go) |
| `r12-enforce-limits-review` | [`r12-enforce-limits-review.md`](r12-enforce-limits-review.md) | 2026-10-30 | `agy/gemini-4-argon-high` (commit `844b59a` trailer) | Node enforcement verifier & deny list (`node/enforce`), control rate-limit tables, `trusted_proxies`, & large-body validation (`internal/httpapi`), `D-063` issuer address validation & `probe_omitted` (`node/updater`, `node/agent`), pre-result heartbeat seeding (`internal/fleetupdate`) | 8 (3 High, 3 Medium, 2 Low) — all closed in `#167`, `#168`, `#169`, `#171`, `#172`, `#174` | [`node/enforce/followup7_review_test.go`](../../node/enforce/followup7_review_test.go), [`node/updater/followup11_review_test.go`](../../node/updater/followup11_review_test.go), [`node/agent/probe_omission_test.go`](../../node/agent/probe_omission_test.go), [`internal/fleetupdate/followup8_review_integration_test.go`](../../internal/fleetupdate/followup8_review_integration_test.go), [`internal/httpapi/followup17_review_test.go`](../../internal/httpapi/followup17_review_test.go), [`internal/httpapi/followup18_test.go`](../../internal/httpapi/followup18_test.go) |
| `r13-r12-fixes-review` | [`r13-r12-fixes-review.md`](r13-r12-fixes-review.md) | 2026-10-30 | `agy/gemini-4-argon-high` (commit `12496fd` trailer) | `r12` remediation verification across node enforcement verifier & QoS (`node/enforce`), revocation conntrack & sample filtering (`node/agent`, `node/enforce`), control rate limiters, `trusted_proxies`, & large-body validation (`internal/httpapi`, `internal/config`), `D-063` probe omission & recovery sealing (`node/updater`, `node/agent`), and pre-result heartbeat seeding & retry confirmation (`internal/fleetupdate`) | 12 (4 High, 5 Medium, 3 Low) — all closed in `#189`, `#190`, `#192`, `#193`, `#194`, `#196` | [`node/enforce/review_r13_enforce_test.go`](../../node/enforce/review_r13_enforce_test.go), [`internal/httpapi/review_r13_limits_test.go`](../../internal/httpapi/review_r13_limits_test.go), [`internal/fleetupdate/review_r13_seeding_integration_test.go`](../../internal/fleetupdate/review_r13_seeding_integration_test.go), [`node/agent/review_r13_agent_test.go`](../../node/agent/review_r13_agent_test.go), [`node/updater/review_r13_updater_test.go`](../../node/updater/review_r13_updater_test.go) |
| `r14-d067-d070-review` | [`r14-d067-d070-review.md`](r14-d067-d070-review.md) | 2026-10-30 | `grok/grok-4.7` (commit `70efbde` trailer) | D-067/D-068/D-070 and r13 fixes: revoke fence, migrated addresses, stale readiness, foreign nft rules, node ingress, eviction and anonymous cap | 7 (1 High, 4 Medium, 2 Low); closed #215/#216/#217/#220 | `internal/nodes/observation_readiness_integration_test.go`, `node/agent/review_r14_m1_test.go`, `node/agent/followup15_test.go`, `node/enforce/followup10_review_test.go`, `tests/netlab/scenario_foreign_test.go`, `internal/httpapi/followup20_review_test.go` |
| `r15-d072-d075-review` | [`r15-d072-d075-review.md`](r15-d072-d075-review.md) | 2026-10-30 | `grok/grok-4.7` (commit `dfafae0` trailer) | D-072–D-075 clone detection, lock order, revoke fence, gated publication, Fair-Use ACK, tc bypass and anonymous reserve | 8 (1 High, 6 Medium, 1 Low) — owner fixes adopted in #254/#259/#263/#256/#262/#252; packet-stop and agent quarantine integration remain qualified | [Adopted main proofs](r15-d072-d075-review.md#adopted-proof-files-on-main) |
| `r16-rev13-d089-d099-review` | [`r16-rev13-d089-d099-review.md`](r16-rev13-d089-d099-review.md) | 2026-10-31 | `grok/grok-4.7` | Revision 13, D-089 / D-097 installer results, D-088 freeze, D-090 / D-096 containment, D-095 / D-099 restore hold; base `5575470` | 4 (1 High, 2 Medium, 1 Low); H-1 fixed #291, M-2/L-1 fixed #300, M-1 pending; two known HTTP defects fixed #277 | Owners adopt `TestReviewR16_*`; only the report is archived here. |
| `r17-rev14-d104-d117-review` | [r17-rev14-d104-d117-review.md](r17-rev14-d104-d117-review.md) | 2026-10-31 | `grok/grok-4.7`, base `62f24e6d` | Revision 14, D-104–D-118 | 1 Medium: M-1 → fleet-update 12 (#322), D-127; r16 H-1 / M-1 / M-2 / L-1 re-run and held | [installer correction proof](../../internal/fleetupdate/review_r17_installer_integration_test.go) |
| `r18-generation-change-review` | [r18-generation-change-review.md](r18-generation-change-review.md) | 2026-11-01 | `grok/grok-4.7`, base `5bd75102` | D-135 / corrected D-136, agent-core 31 shutdown and D-138 TEST ONLY observation | 5 OPEN: 3 Medium / 2 Low; M-1/M-2/M-3/L-2 → agent-core 32; L-1 → netlab | Only the report is archived; proof files remain on the review branch. |
| `r19-intersection-design-review` | [r19-intersection-design-review.md](r19-intersection-design-review.md) | 2026-11-01 | `grok/grok-4.7`, base `6eda25d0` | Agent-core 33 option (b) intersection-serving design, D-139 | H-1 → netlab 24; M-2/L-2 → agent-core 34: routed; in progress. Eleven design-only findings → D-141 reconsideration conditions. | Only the report is archived; proof files remain on the review branch. |
| `r20-adoption-signal-review` | [r20-adoption-signal-review.md](r20-adoption-signal-review.md) | 2026-11-03 | `agy/gemini-4-argon-high`, target `7d34c885` | Agent-core 36 same-boot adoption refusal on changed deny / host sets, signal lifetime through Close, durability and lab proofs | 6 routed, not yet judged: 2 High / 3 Medium / 1 Low; H-1, M-1, M-2, M-3 (pre-existing), L-1 → agent-core 37; H-2 → netlab 27 | Only the report is archived; proof files remain on the review branch (`639eda6b`). |
| `r21-adoption-fixes-review` | [r21-adoption-fixes-review.md](r21-adoption-fixes-review.md) | 2026-11-03 | `agy/gemini-4-argon-high`, target `7b3d7bf8` | Agent-core 37 fixes for r20 H-1, M-1, M-2, M-3 and L-1; D-141 | 0 Critical / 2 High / 2 Medium / 1 Low; returned before merge, none confirmed | Proof branch `47f75c04`; none adopted on main at this cut |
| `r22-attribution-fixes-review` | [r22-attribution-fixes-review.md](r22-attribution-fixes-review.md) | 2026-11-03 | `agy/gemini-4-argon-high`, target `31862b3a` | Agent-core 38 carrying 37: r20 fixes and r21 H-1, H-2, M-1, M-2, L-1 fixes; D-141 | 0 Critical / 2 High / 3 Medium / 0 Low; returned to agent-core 39 before merge, none confirmed | Proof branch `ef78b9b7`; none adopted on main at this cut |
| `r23-intent-fixes-review` | [r23-intent-fixes-review.md](r23-intent-fixes-review.md) | 2026-11-03 | `agy/gemini-4-argon-high`, target `99f93112` | Agent-core 37–39, r20/r21/r22 fixes | 5 (0 Critical, 3 High, 1 Medium, 1 Low); returned to agent-core 40 before merge, none confirmed | None; proofs on `26ffc36c` for four findings, L-1 code trace; exact archive only. |
| `r24-hitless-guard-review` | [`r24-hitless-guard-review.md`](r24-hitless-guard-review.md) | 2026-11-03 | `agy/gemini-4-argon-high` | Netlab 31 design at 52a7e59b; review ba5d3004, 08:19:51 EST; G5/G6 owner trade | 8 (0 Critical, 4 High, 3 Medium, 1 Low); #423 enforcement fixes, C1 revision documents only, H-1/H-3 returned to agent-core 41 | [`node/enforce/review_r24_test.go`](../../node/enforce/review_r24_test.go) adopted by #423; copied M-3 dropped; agent/watchdog proofs review-branch only |

## What Held Across the Reviews

- **Cryptographic and database guards:** JCS (`pkg/canonjson`) and Ed25519 envelope verification (`pkg/envelope`), PKI mTLS incarnation binding (`internal/pki`), PostgreSQL append-only and immutability triggers (`db/migrations`), and SQLite WAL journal atomicity (`node/internal/updatejournal`) held across the reviewed paths recorded below.
- **Cross-member isolation (`§7`, `ISO01`):** Member reads (`GET /api/v1/member/me`, `GET /api/v1/member/devices`, `GET /api/v1/member/exits`, `GET /api/v1/member/operations/{operation_id}`) and `DELETE /api/v1/member/devices/{device_id}` never leaked another member's UUIDs, subjects, peer IDs, or exit-group identities (`r7`, `r8`).
- **Diagnostic counter isolation (`D-063`):** Synthetic-probe counters (`probe_up`, `probe_down`) were isolated from `enforce.Samples()` and member usage uploads (`r10`, `r11`, `r12`, `r13`, `r14`).
- **Secret and credential redaction (`SEC01`, `SEC02`):** Threat-feed credentials (`r4`), OIDC tokens and subjects (`r7`), resolver admin credentials and file paths (`r8`, `r10`, `r11`, `r12`, `r13`, `r14`), and journal file paths (`r10`) never leaked into structured logs, HTTP error envelopes, observations, or `audit_events`.

## Consolidated High-Severity Findings Summary

Every individual finding (High, Medium, and Low) is tabulated in full in its respective review file. The table below summarizes representative High-severity findings across the archived reviews:

| Severity | Finding | Status | Closing PR |
|---|---|---|---|
| High | `r3-d055-review`: Paired setup-key and correlation-group reads in `internal/netbird/discovery.go` and `apps/worker/jobs_nodes.go` could freeze a live enrollment, bind a peer after a second peer joined, or audit withdrawal before deleting the correlated peer in `internal/nodes/httpwire/revocation_adapter.go`. | closed | #109 |
| High | `r4-dnsfeeds-review`: Rolling `lastGood` delta baseline, SSRF address/port gaps, unbounded pre-parse buffering, and untruthful `policy.activate` operations in `internal/dnsfeeds/service.go`, `internal/dnsfeeds/fetch.go`, `internal/dnspolicy/sources.go`, and `apps/worker/jobs_policies.go`. | closed | #101 |
| High | `r5-backups-review`: `tests/backup/drill_integration_test.go` and `deploy/backup/restore-pitr.sh` passed without replaying post-backup WAL, wiped non-mountpoint target directories, and reused fixed Docker object names across runs. | closed | #94, #100 |
| High | `r6-d053-review`: Revocation rollback in `node/updater/revoke.go` omitted D-053 challenge issuance and `node/internal/healthattest/io.go` accepted symlinked or shared-writable parent journal directories. | closed | #96 |
| High | `r7-d052-review`: Plan downgrade and group removal kept excess devices or stale node bindings in `internal/entitlement/store.go` and `internal/desiredstate/compiler.go`, `client_scope`-scoped `identity_bindings` in `internal/auth/store.go` created duplicate members, and `/api/v1/auth/callback` in `internal/auth/browser.go` did not check `azp`. | closed | #106, #109, #114, #125, #134, #137 |
| High | `r8-d057-review`: Published DNS resolver documents were not wired into `internal/desiredstate/compiler.go` (`CompileTx`) via `internal/nodes/httpwire/document_inputs.go` and `internal/nodes/resolver_documents.go`. | closed | #109, #121, #127 |
| High | `r9-d062-review`: Generation-bearing heartbeats in `internal/fleetupdate/serving.go` overwrote the accepted serving release and cleared `recovery_required`. | closed | #140 |
| High | `r10-d063-review`: Post-probe `VerifyBaseline` failures, latched stale serving decisions, and mismatched DNS read-back digests in `node/agent/probeexc.go` and `node/agent/resolver.go` attested passed checks, and `node/enforce/parse_nft.go` accepted non-`/32` or prefix/range elements in probe sets. | closed | #144, #150 |
| High | `r11-d064-review`: `installProbeException` and `applyProbeModel` (`node/agent/probeexc.go`) applied the captured serving model via `ApplyProbeModel` while another process held `/var/lib/vpn-node/update.lock`, re-enabling member forwarding over the updater's baseline (`TestReviewR11_ServingProbeIgnoresUpdateLock`). | closed | #162 |
| High | `r12-enforce-limits-review`: `VerifyServing` in `node/enforce/verify.go` accepted out-of-model forwarding rulesets, `node/updater/probe_addresses.go` accepted an empty node IPv4 list or symlinked `agent.json` and left probe omission unrecorded, and `observeTx` in `internal/fleetupdate/serving.go` misattributed cross-job audits/seeding and allowed pre-intent heartbeats to poison `attempt_release_id`. | closed | #167, #168, #171, #172 |
| High | `r13-r12-fixes-review`: `VerifyServing` and `VerifyBaseline` in `node/enforce/parse_nft.go` and `node/enforce/verify.go` accept out-of-model rulesets (`jump`/`goto` into `output`, `vmap`, `!=` negated ports/interfaces, anonymous sets, narrowed drops, `flow add`), `splitUnbracketedIPv6` in `internal/httpapi/middleware.go` strips a `:0..9` hextet on compressed IPv6 addresses shifting `/64` into `trusted_proxies`, `AuthenticateAgent` runs uncached PostgreSQL queries before rate limiters/concurrency gates, and pre-result heartbeat seeding for an issued job in `internal/fleetupdate/serving.go` moves `nodes.current_release_id` without an accepted result and poisons superseding batch `attempt_release_id`. | closed | #189, #190, #196 |
| High | `r14-d067-d070-review`: `ObserveRevokedDeviceCutoffTx` in `internal/nodes/observations.go` scanned every non-revoked/non-decommissioned node without scoping to serving exit groups or bounding by heartbeat freshness (`DefaultStaleTimeout = 90 s`), allowing a single stale, offline, or ungrouped node with a pending `ISOLATION_POLICY_INVALID` apply error to hold `member.revoke` and `member_device.revoke` `"nodes"` steps `pending` indefinitely (`TestReviewR14_Q1_UnrelatedOfflineNodeHoldsRevokePendingIndefinitely`). | closed | #217 |
| High | `r15-d072-d075-review` H-1: the renewal that freezes a cloned incarnation can still issue a member-grant lease at the fleet offline cap; the next failed observation does not shorten that lease. | closed | #254 / #259; server clamp only, see packet-stop limits |

Status checked on main `7e044c71`, 2026-10-31 EDT: r15 owner fixes and proofs adopted; see its archive for the server-clamp and quarantine limits. r16 delivered; dispositions below.

## r16 dispositions at the follow-up 17 cutoff

The report is byte-identical to `.agent/REVIEW.md` on
`origin/r16-rev13-d089-d099-review`, commit
`e672d5d` (14 files: the report plus proof files), reviewed base `5575470`.
SHA-256: `aa1e3fd95905d85975a43ecf304673d993d5fbbfc9817110635fc183c5f081fb`.
Its original wording and ownership labels describe that review's base. The
orchestrator's current routing is below; no proof files were adopted here.

| Finding | Current disposition |
|---|---|
| H-1: fenced device still bound and leased | Fixed by wire-nodes follow-up 31 with the desired-state owner (#291): fenced devices excluded from signed bindings and lease device lists; worker recompiles affected signed envelopes before acknowledging the detecting sync job. Live propagation timing remains unmeasured. |
| M-1: failed identity-freeze persist | Fixed by agent-core follow-up 24 (#305): durable freeze intent, locked-store restart recovery, Step retries, withheld acceptance and lost-response replay guard. Failed-persistence regression retained; #312 reruns the r16 proofs. |
| M-2: installer rollback version write | Fixed by fleet-update follow-up 11 (#300): accepted installer rollback restores current and last-good versions. |
| L-1: installer row cardinality / ETag churn | Fixed by fleet-update follow-up 11 (#300), D-110 deterministic job identity and visible-change-only ETag updates; updater follow-up 15 (#307) adopts the catalog-bound ID and D-114 legacy replacement. |
| Known, reconfirmed: generation-only cache advance | Fixed after the review base by wire-nodes follow-up 29 (#277), [state_cache.go](../../internal/nodes/httpwire/state_cache.go) and its runtime eligibility fence. Accepted D-099 ordinary-proposal semantics apply; live SECOPS-21 remains open. |
| Known, reconfirmed: keyed installer replay 500 | Fixed after the review base by #277, [replay.go](../../internal/nodes/httpwire/replay.go), `TestD089_HTTPInstallerResultKeyedReplay`. |

D-110 / D-111 and their delivery boundaries are in the [operator decisions
register](../handover/README.md#follow-up-17-decisions-register-2026-10-31-edt).
Owners adopt the `TestReviewR16_*` proofs in their follow-ups.

## r17 disposition at the follow-up 18 cutoff

The archive is byte-identical to `.agent/REVIEW.md` at
`origin/r17-rev14-d104-d117-review`, commit
`dcabc41e7d0c12725b1a7090425e2831faa3a12e`, reviewed from main `62f24e6d` by
`grok/grok-4.7`. SHA-256:
`3e2078550ddf24d365881dcee1399da474ae642bf76899f484a8c8ec60d787f7`.

One finding: **M-1 (Medium)**, a later eligible installer outcome could not
correct current/last-good on the same incarnation. Fleet-update 12 (#322)
implements D-127's one-correction bound, immutable first receipt, stored ACK
replay, generation fence and migration 00192. This is harness-proved on
PostgreSQL tests; live install → seed → success and success → rollback captures
remain devstack / updates work not yet run.

r16 H-1 / M-1 / M-2 / L-1 were re-run and held. The review's “Not examined” list
notes D-118 register drift. Follow-up 18 now pins activation in #312, lifetime
locking in #320 and live usage acceptance in #324; pending assertions in the
older claims tests are explicitly historical. The archive itself is unchanged.
SECOPS-21 closed on the assembled restore harness in #316; the open status in
the dated r16/follow-up 17 dispositions above is historical.

## Follow-up 19 acceptance boundary

At #337, r17 and r16 remain exact historical archives. D-127 now has the
updates 10 installer extension (23/23) with a labeled TEST ONLY HTTP rollback
correction. Regular AT23–AT27 are blocked; physical rollback is not inferred.
The [current handover](../handover/follow-up-19.md) and
[findings register](../handover/README.md#findings-register-at-the-follow-up-19-cutoff)
record ENR-07 / F9-01 closure and ISO-F6-03 OPEN. Agent-core 28 is in flight,
not on main at #337. D-129–D-132 do not change the archived review verdicts.

## r18 dispositions at the follow-up 25 cutoff

Exact source: `origin/r18-generation-change-review:.agent/REVIEW.md`, commit
`1a65598fe6ebc5ba394284857ece59d0a572458b`, reviewed base #369 / `5bd75102`.
The [raw archive](r18-generation-change-review.md) is byte-for-byte, SHA-256
`f266a0705292975f5416bda5a082a3f7e779f45e82cb9b9ce54843ce0cb8e33d`.
No proof files were adopted. At #373, all five findings remain OPEN.

| Finding | Owner / status (2026-11-01) |
|---|---|
| M-1: control observation POST after SIGTERM | Agent-core 32 running; deferred context does not carry the cancellable control budget. |
| M-2: stacked shutdown budgets exceed 10 s | Agent-core 32 running; review measured 14.021 s. The unit has no TimeoutStopSec; installer sets the value after agent-core 32 establishes the budget. |
| M-3: replayed lease holds node on baseline without cap or distinct alarm | Agent-core 32 running; wire-nodes handles central's stale-hold reading after agent-core 32. |
| L-1: TEST ONLY sample ignores install deadline | Netlab; disabled production path unaffected in the review. |
| L-2: D-133 rule omits stricter ceiling comparison | Agent-core 32 running; member/node ceilings must be considered before any intersection-serving implementation. |

What held is the review's statement, not an acceptance result: no removed grant
forwarded on any reviewed stale path, observation keys out of production, torn
checkpoints fail closed. No live lab or PostgreSQL suite ran in this review;
the 80-transition population, physical arm64 and live mid-write process death
were not verified. M-1 / M-2 qualify #369's shutdown claims; its historical
cancellation fix remains recorded. D-135 / D-136 / D-137 / D-138 remain
provisional; intersection serving is not implemented by this handover.

## r18 dispositions at the follow-up 26 cutoff

At #382, the raw r18 archive remains byte-identical. The previous #373 table
is historical. [Follow-up 26](../handover/follow-up-26.md) pins these fixes:

| Finding | Disposition |
|---|---|
| M-1 | Fixed #376: no observation POST after caller cancellation. |
| M-2 | Fixed on both sides #376 + #378: one Step budget, 13 s cooperative shutdown, 15 s agent unit stop budget, watchdog ordering and bounded replacement stop. |
| M-3 | Fixed on both sides #376 + #382: distinct hold, node alarm, central own-lease recognition independent of ACK, incident retention and one escalation. Alarm and central escalation not yet exercised in an assembled run. |
| L-1 | Fixed #377: install-phase sample skipped; adopted proof asserts corrected behavior. |
| L-2 | Fixed #376: any higher or missing member/node ceiling fails the D-133 leg. |

D-139 is a provisional option (b) design track, with r19 running at task start.
#379's reference passing is not an acceptance result. A recorded representation
decision follows review before any implementation; option (c) stays in production.

## r19 dispositions at the follow-up 27 cutoff

Exact source: `origin/r19-intersection-design-review:.agent/REVIEW.md`, commit
`6048e099984ee03b69fec961309cd3eabe0e2b65`, delivered 2026-11-01 07:47 EST.
The [raw archive](r19-intersection-design-review.md) is byte-for-byte:
**41,050 bytes**, SHA-256
`14caf0193b256d6f090e8d53ff77e0d5a5c9dab4d025794a6812e15e83228913`.
No proof files were adopted. Findings and What held are review evidence, not
assembled acceptance. [Follow-up 27](../handover/follow-up-27.md) pins this cut.

| Finding | Disposition at #385 |
|---|---|
| H-1 | Routed; in progress — netlab 24. Established download from a newly denied_v4 destination keeps flowing. |
| M-2 | Routed; in progress — agent-core 34. Live desired refs stamped on the applied-generation ACK; production reachability unknown. |
| L-2 | Routed; in progress — agent-core 34. Conntrack attempt persisted after the delete. |
| H-2, M-1, M-3–M-8, L-1, L-3, L-4 | Design-only: D-141 reconsideration conditions, not fixed or implemented. |

**D-141 closes D-139**: option (b) is not implemented on this evidence; option
(c) stays in production. 0 / 80 merged restrictive changes qualify, promotion's
baseline is the same order as 173.8 / 195.6 ms p50 / p99 baseline-to-Apply,
and the representation lacks a signed intersection witness, v2 heartbeat,
partial-service cause, baseline-on-422 and nft-only revocation transaction.
Reconsideration requires a desired-state intersection witness, a measured
qualifying share above zero whose promotion costs less than (c), and the
design-only findings closed. Agent-core 33's note and test-only reference stay
on main as history. D-135 / D-136 / D-137 / D-138 remain provisional; historical
failures stay failed. Earlier review disposition tables retain their cutoff.

## r19 dispositions at the follow-up 28 cutoff (2026-11-01 EST)

[Follow-up 28](../handover/follow-up-28.md) verifies the fixes through #390.
The original report and the #385 disposition table above retain their bytes.

| Finding | Disposition at #390 |
|---|---|
| H-1 | Fixed — #389: source denials precede generic established / related returns, with exact resolver DNS replies retained. ctdenyd is diagnostic, never usage. |
| M-2 | Fixed — #388; production-reachable with due renewal h and busy Apply retaining g. ACK and minimal applied snapshot commit together; only matching refs reported. |
| L-2 | Fixed — #388: reserve conntrack delete batch before Apply. About 1.1 ms, p99 1.59 ms before restricting install with queued deletes; accepted under D-138, cost explicit. |
| H-2, M-1, M-3–M-8, L-1, L-3, L-4 | Design-only findings stay recorded under provisional D-141; no implementation or closure claimed. |

RELATED ICMP from a spec-denied outer source is a new open owner question,
not a defect or a decision. Historical failures stay failed.

## r20 dispositions at the follow-up 30 cutoff (2026-11-03 EST)

Exact source: the `r20-adoption-signal-review` worktree's `.agent/REVIEW.md`,
delivered with proof commit `639eda6b21bee726695c6a51bcf12637587e1433` on top of
agent-core 36's `7d34c885`. The [raw archive](r20-adoption-signal-review.md) is
byte-for-byte: **32,207 bytes**, SHA-256
`af15fb7c42e86eb7269fd9e8bd1396a730cb5ba51643e52e7f6f1c34738a0f1a`. It carries no
host path, username or secret. Each finding has a failing proof test on the
review branch; no proof files were adopted. r20 recommended fixing before the
merge. [Follow-up 30](../handover/follow-up-30.md) pins this cut.

**Merged with routed fixes.** The orchestrator merged agent-core 36 (#401)
anyway: it closes a security gap on main, and what r20 found that 36 introduced
fails closed or is recorded, never silent. The #401 roadmap row records this.
Agent-core 37 judges each finding before fixing it, so none is recorded as
confirmed.

| Finding | Disposition at #403 |
|---|---|
| H-1 | Routed, not yet judged — agent-core 37. A refused same-boot adoption can record member bytes counted since the last sample as an unattributed `mapping conflict` usage gap. |
| H-2 | Routed, not yet judged — netlab 27, with four concrete requirements: interval normalisation, SNAT /32 in host_v4, denied_v4 / host_v4 set flags, and baseline bare vs probe sets with allowed_v4 empty. |
| M-1 | Routed, not yet judged — agent-core 37. SIGTERM during Boot and the heartbeat. |
| M-2 | Routed, not yet judged — agent-core 37. Comparison order inside adoption. |
| M-3 | Routed, not yet judged — agent-core 37; pre-existing. Latched apply error and the shutdown heartbeat. |
| L-1 | Routed, not yet judged — agent-core 37. Empty non-nil DeniedV4 and the generation check. |

What held is the review's statement, not an acceptance result. The lab scenario
was reviewed statically, and the A,B,A,B timing runs were out of scope.
Historical failures stay failed.

## r21 dispositions at the follow-up 31 cutoff (2026-11-03 EST)

Exact source: `r21-adoption-fixes-review` worktree's `.agent/REVIEW.md`, with
proof commit `47f75c04` reviewing agent-core 37's delivered `7b3d7bf8`.
Reviewer: **agy `gemini-4-argon-high`**. The [raw archive](r21-adoption-fixes-review.md)
is byte-for-byte: **29,439 bytes**, SHA-256
`0fcf888bbb59e8e3032abf041f4d600fcb661525a3c390e15850c61e4df2edc0`.
It carries no host path, username or secret. Findings: **0 Critical, 2 High,
2 Medium, 1 Low**, each with a proof test on the review branch. None adopted
onto main. Recommendation: **DO NOT MERGE — Return to agent-core (H-1, H-2,
M-1, M-2, L-1)**.

**Returned to agent-core 38 before merge.** Agent-core 37 is not on main at
#407; its fixes travel with 38 after 38 is reviewed. D-141 requires owner
judgment before fixing; none of r21's findings is recorded as confirmed.

| Finding | Disposition at #407 |
|---|---|
| H-1 | Routed, not yet judged — agent-core 38; review reports old-member billing after an Apply/persistence crash. |
| H-2 | Routed, not yet judged — agent-core 38; review reports new-member billing after failed settlement then rebinding. |
| M-1 | Routed, not yet judged — agent-core 38; review reports a false mapping-conflict gap after rebinding. |
| M-2 | Routed, not yet judged — agent-core 38; review reports canceled-boot upload, recovery latch, budgets and exit-status problems. |
| L-1 | Routed, not yet judged — agent-core 38; review reports an orphan member-path pending flag. |

r20 H-1, M-1, M-2, M-3 and L-1 therefore stay open on main. Separately,
**r20 H-2 and live drift are closed by netlab 27 (#405)**, including periodic
withdrawal without another Apply and baseline after an unrecorded Apply.
The r20 disposition above remains the historical #403 cut. What held in r21
is the review's statement, not new acceptance evidence. [Follow-up 31](../handover/follow-up-31.md)
pins this disposition; historical failures stay failed.

## r22 dispositions at the follow-up 33 cutoff (2026-11-03 EST)

Exact source: `r22-attribution-fixes-review` worktree's `.agent/REVIEW.md`.
Reviewer **agy `gemini-4-argon-high`** reviewed agent-core 38's delivered
**31862b3a** before merge. That tip carries agent-core 37's r20 fixes and
38's fixes for r21 H-1, H-2, M-1, M-2 and L-1. The [raw archive](r22-attribution-fixes-review.md)
is byte-for-byte: **37,600 bytes**, SHA-256
**05e7806a1f5dfec8388e4fe2cbe0e5e1ed11ec1d4c291f3bb5236d7c0fe43c02**.
It carries no host path, username or secret. Findings: **0 Critical, 2 High,
3 Medium, 0 Low**, each with a proof test on the review branch
**r22-attribution-fixes-review**, **ef78b9b7**. Recommendation:
**Hold for fixes: H-1, H-2, M-1, M-2, M-3.**

**Returned to agent-core 39 before merge.** Agent-core 37 and 38 are not on
main at #415; they merge together with 39 after 39 is reviewed. D-141 requires
39 to judge each finding before fixing it; **none is confirmed**. No proof
files are adopted by this handover.

| Finding | Disposition at #415 |
|---|---|
| H-1 | Returned, not yet judged — agent-core 39; review reports unsettled transitions and failed handoff FinalSample allowing wrong-member/revision billing on rebinding Apply. |
| H-2 | Returned, not yet judged — agent-core 39; review reports false mapping-conflict gaps across five crash windows. |
| M-1 | Returned, not yet judged — agent-core 39; review reports transient Boot meter-open failure permanently wedging retained settlement. |
| M-2 | Returned, not yet judged — agent-core 39; review reports late-aborted adoption bypassing PreserveClosedEpoch and contaminating the epoch baseline. |
| M-3 | Returned, not yet judged — agent-core 39; review reports untruthful failed/timed-out Boot heartbeats and probe_clear failure consuming the challenge nonce. |

r20 H-1, M-1, M-2, M-3 and L-1, and r21's five findings, stay open on main.
Earlier disposition tables retain their cutoffs. What held in r22 is the
review's statement, not acceptance evidence. [Follow-up 33](../handover/follow-up-33.md)
pins this disposition; every historical failure stays failed.

**Erratum to What held, item 6 (archive unchanged).** The report cites
“deploy/systemd/fvpn-agent.service” with **TimeoutStopSec=20s** and a
**defaultStopTimeout** in main.go. Neither exists on main. The shipped unit is
`deploy/node/systemd/fvpn-agent.service`, with **TimeoutStopSec=15s**, pinned by
`TestPKG01_AT07ShippedAgentStopBudget` in `packaging/installer/followup9_test.go`.
The budget assertion in the review-branch held test
`TestR22_Held_Q4_Q5_Q6_Q7_FinalSampleCloseAndStopBudgetAndMemberPathEvidence`
checks only **bootLocalBudget + closeLocalBudget == 13s**; those constants
are on agent-core's unmerged branch. It does not check the shipped unit or
a defaultStopTimeout. This prose correction adds no acceptance claim and
changes no byte of the review archive.

## Follow-up 34 review boundary (2026-11-03 EST)

At #418, agent-core 39 (carrying 37 and 38) is held for r23 before merge.
Its judgments are not on main. All five r22 dispositions above remain
returned to agent-core 39, none confirmed on main; the raw archive and
stop-budget erratum are unchanged. r20 H-1, M-1, M-2, M-3 and L-1 and
r21's five findings stay open on main. [Follow-up 34](../handover/follow-up-34.md)
imports no result from r23. Every historical failure stays failed.

## r23 dispositions at the follow-up 35 cutoff (2026-11-03 EST)

Exact source: `r23-intent-fixes-review` worktree's `.agent/REVIEW.md`.
Reviewer **agy `gemini-4-argon-high`** reviewed agent-core 39's delivered
**99f93112** before merge, carrying 37's r20 fixes, 38's r21 fixes and
39's r22 fixes. The [raw archive](r23-intent-fixes-review.md) is byte-for-byte:
**41,791 bytes**, SHA-256
**23f899f8e0b834994785e8c8a1f774b5aeb1a18309bf35439ba1a4f3261b25d0**.
It carries no host path, username or secret. Findings: **0 Critical, 3 High,
1 Medium, 1 Low**. Proof branch **r23-intent-fixes-review**, **26ffc36c**.
H-1, H-2, H-3 and M-1 have executable proof tests. **L-1 is a code trace,
not a named proof test** in the delivered commit; the brief's statement that
all five have proof tests is referred back for clarification.

Recommendation: **Do not merge follow-ups 37, 38, and 39 as they are; merge after
`agent-core` fixes `H-1` (`R23-H-1`), `H-2` (`R23-H-2`), `H-3` (`R23-H-3`),
`M-1` (`R23-M-1`), and `L-1` (`R23-L-1`).**

**Returned to agent-core 40 before merge**, following delivery at **08:07 EST**.
Agent-core 37–39 are not on main at #421; they merge together with 40 after
its fixes. D-141 requires 40 to judge each finding before fixing it;
**none is confirmed**. No proof files are adopted by this handover.

| Finding | Disposition at #421 |
|---|---|
| H-1 | RETURNED — not yet judged; agent-core 40 before merge. Review reports epoch changes leaving usage_apply_intent on a superseded epoch, losing active bindings or producing false mapping_conflict gaps. |
| H-2 | RETURNED — not yet judged; agent-core 40 before merge. Review reports handoff resetting finalDone after epoch closure, preventing preparation before next admission. |
| H-3 | RETURNED — not yet judged; agent-core 40 before merge. Review reports pre-Apply handoff failures leaving serving/decision inconsistent and observations untruthful. |
| M-1 | RETURNED — not yet judged; agent-core 40 before merge. Review reports a successful second baseline with an unsettled epoch and retainedPending false. |
| L-1 | RETURNED — not yet judged; agent-core 40 before merge. Review reports inaccurate checkpoint-invalidation ordering in the follow-up 39 document; code-trace evidence. |

r20 H-1, M-1, M-2, M-3 and L-1 and r21's five remain open on main.
r22's five stay as follow-up 33 recorded them, returned to 39 and none
confirmed on main. Those older routing labels are historical. Agent-core
39/40 judgments are excluded; What held is review evidence, not acceptance.
Review r24 delivered after the cutoff; its results are outside this cut.
[Follow-up 35](../handover/follow-up-35.md) pins these boundaries.
Every historical failure stays failed.

**Errata to r23's What held (archive unchanged), verified on main at #421:**

- **Renderer:** the cited node/enforce/nft.go and node/enforce/nft_test.go do
  not exist. The renderer is [render_nft.go](../../node/enforce/render_nft.go).
- **Generation marker:** the forward chain carries the marker rule
  `fvpn_generation=<G> fvpn_metering=<V>` in that renderer, read by
  `forwardGeneration` in [parse_nft.go](../../node/enforce/parse_nft.go).
  There is no fvpn_meta_v1 comment. The cited TestRenderApply_Golden and
  TestNftPlane_RestartInspection do not exist. The candidate and marker are
  rendered into the same nft batch; the invented table-comment description
  supplies no evidence for that property.
- **Counters:** a render never deletes counters. `TestRender_SecondPassKeepsCounters`
  in [render_test.go](../../node/enforce/render_test.go) rejects a render
  emitting `delete counter` or `delete table`. The archive's claim that a
  device's counters are deleted when it leaves the binding set is false,
  as is its baseline table-deletion claim. The removed/re-added-device
  argument rests only on the epoch baseline, which agent-core 40 checks.
- **Stop budget:** `TestPKG01_AT07ShippedAgentStopBudget` is in
  [packaging/installer/followup9_test.go](../../packaging/installer/followup9_test.go),
  not under packaging/Linux/. The shipped agent unit still sets 15 s.
- **Proof names:** several r20–r22 proof names cited in What held do not
  exist on reviewed tip 99f93112. No absent names are reproduced here.
  Passing proofs and owner judgments on held branches are not main's results.

## r24 dispositions at the follow-up 36 cutoff (2026-11-03 EST)

**agy `gemini-4-argon-high`** reviewed netlab 31's design as delivered at
**52a7e59b**, the design tree #421 merged. Review branch
**r24-hitless-guard-review**, commit **ba5d3004**, dated **08:19:51 EST**.
The [raw archive](r24-hitless-guard-review.md) is byte-for-byte identical to
the supplied review file and committed review: **46,001 bytes**, SHA-256
**7ed69f029bb12b526130e26608474f1b1cb15ac6346c3bdab6a7cc875af437f8**.
Inspection found no host path, username or secret; /var/lib/vpn-node paths are
product paths. Findings: **0 Critical, 4 High, 3 Medium, 1 Low**.

Evidence includes review_r24_test.go under node/agent, node/enforce and
node/watchdog on the review branch. **M-1 is a counterexample trace; L-1 is a
code inspection**. Several proofs cited under agent/watchdog are on the review
branch only, not main; absent names are not reproduced here. #423 imported the
enforcement proof file alone at **cdb384ed**, recorded failures before changes,
and judged the findings under D-141. The copied M-3 proof was subsequently dropped.

The review's recommendation is **“Implement C1 after named changes (`H-1`,
`H-2`, `H-3`, `H-4`, `M-1`, `M-2`, `M-3`, `L-1`)”**. Its owner section says
**“Accept the trade and implement C1 after fixing `H-1` through `M-3`”**.
These are **the reviewer's recommendations**, not an owner decision. The trade
belongs to the owner; the orchestrator does not decide it. Production keeps
the current guard and no C1 implementation is approved by this handover.

| Finding | Disposition at #423 |
|---|---|
| H-1 | OPEN on main — RETURNED, not yet judged; agent-core 41 before merge. Netlab 32 reads it as “confirmed by code”; this is not agent-core 41's D-141 judgment. |
| H-2 | tc order confirmed and FIXED by #423. Eligibility confirmed in part, revised for rate-only fair-share changes with stable classes/non-tightening ceilings, documents only. |
| H-3 | Agent-core part OPEN on main — RETURNED, not yet judged; agent-core 41 before merge. Netlab 32 confirms the outcome with a corrected explanation. Per-device deadlines and a separate verified-serving record remain prerequisites; the record waits for the owner. |
| H-4 | Confirmed and FIXED by #423: state-independent baseline batch; cancellation-independent baseline/quarantine cleanup with five-second budget, counters retained. |
| M-1 | Confirmed design correction in the C1 revision, documents only: old bank deleted at release, fallback installs fenced by lock and child reaping. |
| M-2 | Confirmed in part: post-shape rollback FIXED by #423; fatal Verified:true return rejected as currently unreachable, with regression. |
| M-3 | Confirmed by tooling inspection; copied proof dropped. Real classifier/decorator tests and attribution controls are C1 prerequisites, documents only; no tooling change. |
| L-1 | Netlab 32 judges it confirmed, documents only. Empty eligibility specified; the claimed lock correction disagrees with main, as recorded below and returned for an additive correction. |

Agent-core 37–41's judgments are not on main at this cut. r20–r23 register rows
stay as follow-up 35 recorded them. The r23 L-1 proof-class request is answered
in #422: a code trace is correct, the brief's all-findings-have-tests claim was
wrong, no proof is added, and agent-core 40 judges L-1 with the other four.
The older r23 section and register wording remain historical. Every historical
failure stays failed. See [follow-up 36](../handover/follow-up-36.md).

**Errata to r24 (archive unchanged), verified on main at #423:**

- **H-3 explanation:** [memberTimeout](../../node/enforce/deadline.go) does cap
  each binding by its own ValidUntil. The outcome stands: [Decide](../../node/agent/decide.go)
  folds every binding's ValidUntil into the member's single DeadlineMono and
  takes the member's service end and node window from the first binding.
- **M-3 evidence:** the copied test reproduces the classifier logic of
  [trace.go](../../tests/netlab/cmd/fvpn-netlab/trace.go); it does not call that
  tooling. The finding rests on reading trace.go. Netlab 32 dropped the copy;
  its revision requires real classifier and decorator tests before C1.
- **Loss figures:** “22 UDP losses, 28 TCP retransmissions, and a 256.8 ms TCP
  stall across 100 reloads” has no matching source in netlab 30's record.
  Its [outcome](../poc/G2-followup30-evidence/guard-outcome.md), over **101
  Applies**, records **one D-120 UDP boundary loss, seven matched TCP guard
  drops, seven retransmissions, and a 416.739885 ms maximum read**. Historical
  B1's failed read stays unexplained, and netlab 30 remains host-unqualified.
- **Guard cost:** **28.5 ms median and 94.7 ms maximum** are netlab 30's outer
  command brackets, not exact blackout durations. They give no future bound.
- **Share of time:** at 28.5 ms per 10 s, the illustrative complement is
  **99.715%, not 99.9%**. It is not a measured fleet availability fraction.
- **Watchdog:** **30–40 s is a conditional expectation, not a bound**.
  [Watch.Once](../../node/watchdog/watch.go) uses a try-lock and bounded context;
  lock contention, scheduling, failure or watchdog death defeat a fixed bound.
- **Renewal cadence:** “every ~23h58m” and “every 13m” are nominal
  window-minus-lead illustrations. [leadJitter](../../node/agent/sync.go) takes
  its modulus in nanoseconds with only two random bytes: default jitter is
  −12 s .. −12 s + 65,535 ns. Time-triggered renewal starts at about **108 s
  remaining**, not ~120 s. Non-time-triggered renewals use the full residual
  old deadline. Netlab F32-01 is open and routed to agent-core after 41.
- **Spec reference:** QoS and the host reserve are in **§11**, not §8.3
  (IPv6/NAT).
- **G labels:** r24's What held section uses its own numbering. The
  [follow-up 31 G1–G10 ledger](../poc/G2-followup31-design/current-guarantees.md)
  remains the numbering authority; the owner's trade stays **G5/G6**.
- **Test names:** several agent/watchdog proofs cited in the archive are on
  the review branch only, not main. Their names are not listed as main tests.
- **Additional lock-path discrepancy:** L-1 and netlab 32's
  [protocol](../poc/G2-followup32-design/protocol.md) claim
  `/var/lib/vpn-node/update-journal/update.lock`. Main's
  [agent config](../../node/agent/config.go), [shared lock](../../node/nodelock/lock.go),
  [updater layout](../../node/updater/layout.go) and
  [installer layout](../../packaging/installer/layout.go) instead default to
  `/var/lib/vpn-node/update.lock`. The archive's node/updater/config.go does
  not exist. The original follow-up 31 lock path matches main. An additive
  correction to the netlab revision is requested; its immutable files and the
  review archive stay unchanged. This does not undo the empty-set correction.
