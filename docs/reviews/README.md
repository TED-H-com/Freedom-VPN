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
| `r24-hitless-guard-review` | [`r24-hitless-guard-review.md`](r24-hitless-guard-review.md) | 2026-11-03 | `agy/gemini-4-argon-high` | Netlab 31 design at 52a7e59b; review ba5d3004, 08:19:51 EST; G5/G6 owner trade | 8 (0 Critical, 4 High, 3 Medium, 1 Low); #423 enforcement fixes, C1 revision documents only, H-1/H-3 returned to agent-core 41; [#426 additive L-1 / M-3 corrections](../poc/G2-followup33-design/corrections.md) | [`node/enforce/review_r24_test.go`](../../node/enforce/review_r24_test.go) adopted by #423; copied M-3 dropped; agent/watchdog proofs review-branch only |
| `r25-retain-review` | [r25-retain-review.md](r25-retain-review.md) | 2026-11-03 | `agy/gemini-4-argon-high`, target `8cc27609` | Agent-core 40/41, carrying 37–41; unmerged | 0 Critical / 1 High / 1 Medium / 0 Low; RETURNED to agent-core 42 before merge, not yet judged | None; 680-line node/agent/review_r25_test.go on `1afecb29` only; exact archive only. |
| `r26-expiry-fixes-review` | [`r26-expiry-fixes-review.md`](r26-expiry-fixes-review.md) | 2026-11-03 | `agy/gemini-4-argon-high` | Agent-core 42 at 8dec4100 and 37–42 together; expiry, retention and jitter | 0 Critical, 0 High, 0 Medium, 0 Low; reviewer recommends merge 37–42; orchestrator merged 37–43 as #429 under D-150 | [`node/agent/review_r26_test.go`](../../node/agent/review_r26_test.go), six TestR26_Held_* proofs imported by 43; [cut dispositions and errata](#r26-dispositions-at-the-follow-up-39-cutoff-2026-11-03-est) |
| `r27-members-failures-review` | [r27-members-failures-review.md](r27-members-failures-review.md) | 2026-11-03 | `agy/gemini-4-argon-high` | Members 18 run 2; proofs 62146ff1, report committed by orchestrator in dfde8d25 | 0 Critical, 1 High, 3 Medium, 1 Low; [current dispositions](#r27-and-r28-dispositions-at-the-follow-up-43-cutoff-2026-11-04-est) | [Members proofs](../../tests/e2e/at/members/review_r27_test.go) in #445; [agent](../../node/agent/review_r27_test.go) / [enforce](../../node/enforce/review_r27_test.go) proofs in #447 |
| `r28-expiry-probe-review` | [r28-expiry-probe-review.md](r28-expiry-probe-review.md) | 2026-11-03 | `agy/gemini-4-argon-high` | Isolation 20 delivered head 9be409f8; review 34a8d61f, F20-01 | 0 Critical, 1 High, 1 Medium; [current dispositions](#follow-up-44-dispositions) | [Agent proofs](../../node/agent/review_r28_test.go) in #447; [member-expiry proofs](../../tests/e2e/at/isolation/member_expiry_test.go) in #452 |
| `r29-update-recover-review` | [r29-update-recover-review.md](r29-update-recover-review.md) | 2026-11-03 | `agy/gemini-4-argon-high` | F30-01 design diagnosis; base dd052c60, report and proofs committed by the reviewer at 5b1ca726 | 0 Critical, 1 High, 1 Medium, 2 Low; [cut dispositions](#r29-dispositions-at-the-follow-up-43-cutoff-2026-11-04-est) | [Updater proofs](../../node/updater/review_r29_test.go) in #446 and [agent proofs](../../node/agent/review_r29_test.go) in #454, byte-identical |
| `r30-update-fixes-review` | [r30-update-fixes-review.md](r30-update-fixes-review.md) | 2026-11-04 | `agy/gemini-4-argon-high` | Post-merge D-155–D-158 review (#446, #454, #461, #463); head da2560a9, base 9b9546c9; committed by Ted H | 0 Critical, 2 High, 3 Medium, 0 Low; [cut dispositions](#r30-dispositions-at-the-follow-up-46-cutoff-2026-11-04-est) | Agent/watchdog proofs adopted by #470; updater proofs by #471; all five findings fixed, r31 running at cutoff |
| `r31-recovery-lock-fixes-review` | [r31-recovery-lock-fixes-review.md](r31-recovery-lock-fixes-review.md) | 2026-11-04 | `agy/gemini-4-argon-high` | Post-merge agent-core 51 #470 / updater 26 #471; head 0e33b9fe, base fd0f2494; committed by Ted H | 0 Critical, 0 High, 1 Medium, 0 Low; [cut dispositions](#r31-dispositions-at-the-follow-up-47-cutoff-2026-11-04-est) | Both proofs adopted by #477; M-1 fixed; updater needs nothing; no owner question |
| `r32-dedup-records-review` | [r32-dedup-records-review.md](r32-dedup-records-review.md) | 2026-11-04 | `agy/gemini-4-argon-high` | Post-merge agent-core 52 #477 (`r31` M-1 & observation records); head 21e59029, base 1c39a0e7; committed by Ted H | 0 Critical, 0 High, 0 Medium, 0 Low; [cut dispositions](#r32-dispositions-at-the-follow-up-48-cutoff-2026-11-04-est) | None on `main` (0 findings); 4 `TestR32_Held_*` proofs on review branch `21e59029` ([frozen copy](../../tests/handover/testdata/history/follow-up-48/review-node_agent_review_r32_test.go.txt)); no owner question |

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
  not exist. The follow-up 31 line L-1 cites names no lock path.
  current-guarantees.md:27 begins the guard's timing definition, “Let `R` be
  the final monotonic render sample, …”. No file in the follow-up 31 design
  names a lock location; at reviewed tree 52a7e59b, no file under docs/poc/
  names update.lock or /var/lib/vpn-node either. L-1's lock-path premise has
  no source, and its replacement contradicts main. An additive
  correction to the netlab revision is requested; its immutable files and the
  review archive stay unchanged. This does not undo the empty-set correction.

## Follow-up 37 additive r24 errata (2026-11-03 EST)

At #426, [netlab 33's corrections](../poc/G2-followup33-design/corrections.md)
supersede the L-1 judgment and B1 characterization adopted by netlab 32.
The #423 dispositions and original archive above remain historical and unchanged.

- **L-1: confirmed in part.** The lock-location half is rejected: the cited
  follow-up 31 line names no lock path, node/updater/config.go does not exist,
  and every main default uses /var/lib/vpn-node/update.lock. The empty eligible-set
  half remains specified by follow-up 32, documents only. Delayed fallback
  fencing depends on a shared lock and reaping timed-out children, not the path's
  spelling. This answers follow-up 36's additive-correction request.
- **M-3: B1 citation correction.** Neither TCP_RTO_MIN nor socket pacing appears
  anywhere at 52a7e59b. The cited d133-comparison.md:65–75 supplies stop/admission
  rules. B1 is the first candidate arm of A1,B1,A2,B2, including follow-up 27's
  failed F27-01 read. Follow-up 32 adopted r24's unsupported characterization;
  the constraint against transport/socket changes in any C1 acceptance run or
  comparison arm stays. M-3's tooling-prerequisite status is otherwise unchanged.

The correction's table has 16 rows: 14 Verified and two Corrected. Both netlab
32 trees and the raw r24 archive remain byte-identical. Agent-core H-1/H-3
stay OPEN on main — RETURNED; #425 records fixes on unmerged agent-core 41.
At briefing r25 was running on 8cc27609, status only; its archive belongs to
a later cut. The owner's G5/G6 trade stays undecided; production keeps the
current guard. See [follow-up 37](../handover/follow-up-37.md).

## r25 dispositions at the follow-up 38 cutoff (2026-11-03 EST)

**agy `gemini-4-argon-high`** reviewed agent-core follow-ups **40 and 41** at
**8cc2760965abbbb1af3340976c6c4adf1be3dc37**, 41's delivered tip carrying
37–41, **not on main**. Source: `r25-retain-review`, commit
**1afecb29c82bcc8cea6f39a16fa724f0acc7a5d5**, **11:30:07 EST**.
The [raw archive](r25-retain-review.md) is byte-for-byte that commit's
`.agent/REVIEW.md`: **31,868 bytes**, SHA-256
**7514218895b014d62d09693b5de38193b3a412c5fa65c2aeb3b44d720d414286**.
Inspection found no host path, username or secret. Product paths are retained.
Proofs: **node/agent/review_r25_test.go**, **680 lines** on 1afecb29, not on
main and not copied by handover. Agent-core 42's brief imports that file alone
first. No proof or finding is adopted on main at this cut.

Findings: **0 Critical, 1 High, 1 Medium, 0 Low**. Recommendation:
**“Merge after named fixes”** — H-1 and M-1 before follow-ups 37–41 merge.
This is **the reviewer's recommendation, not a decision**. Both findings are
**RETURNED — not yet judged; agent-core 42 before merge** under D-141.

| Finding | Reviewer's claim | Disposition at #427 |
|---|---|---|
| H-1 | One binding's deadline, with a sibling still valid, withdraws the whole node to baseline before the control/probe pass. In verifyCurrentServing or the retained branch, the early return leaves survivors unserved, a `-` heartbeat reason and a contradictory observation report. r25 attributes this to follow-up 41 interacting with expireServing, which it dates to follow-up 23. | RETURNED — not yet judged; agent-core 42 before merge |
| M-1 | canRetainServing requires exact intent/open-epoch equality; after a same-interval epoch rotation, the next unchanged tick forces a redundant Apply. r25 attributes this to follow-up 41 interacting with follow-up 40. | RETURNED — not yet judged; agent-core 42 before merge |

**Open question, unanswered:** which part of H-1, if any, already exists on
main? The candidate is main's whole-node withdrawal when the earliest admitted
binding's deadline passes. The orchestrator asked agent-core 42 to judge it
under D-141. No main defect disposition or operator warning is added here.
Agent-core 42 was running since 11:36 EST at briefing, also handling Netlab
F32-01. Agent-core 37–41 were delivered and held. Their judgments are outside
this cut; r20–r24's agent-core rows stay as follow-up 37 recorded them.

**What held — the reviewer's five answered questions, not acceptance:**

| Question | Reviewer's claim (subject to the additive errata below) |
|---|---|
| Q1 — stale or unsafe retention | Whole-model equality, pending/failure gates, node lock, latch reread, live kernel/member-path verification, clock checks and heartbeat protect skipped Apply ticks; real control can return an unchanged 304. |
| Q2 — per-device deadlines | Member-only caps and per-device ValidUntil preserve independent timeouts in either binding order, including distant/zero/inactive bindings. H-1 is the sibling-expiry exception. The claimed future-ValidFrom wake is corrected below. |
| Q3 — r23 fixes | Counting-interval continuity, retained intent on transient marker reads, closed-epoch preparation and handoff persistence/observation gates hold. M-1 concerns the separate live-retention path. |
| Q4 — edited-test coverage | Apply-fault fixtures force an actual model change, retained-tick live-drift coverage remains, and the reviewer claims no weakened assertions. Its stated file inventory and description of all edits are corrected below. |
| Q5 — fixture changes | Pinning the fake signer clock produces actual 304 ticks; separate renewal coverage remains. Waiting for a counted UDP burst is justified by the upload counter and accept sharing one rule, followed by the ledger/counter equality assertion. |

The reviewer inspected the netlab scenario statically and did not execute it.
Question 6 reports the two findings; it is not a sixth “held” assurance.
Every historical failure stays failed. See [follow-up 38](../handover/follow-up-38.md).

**Additive citation errata to r25 (archive unchanged).** Code and existing
agent tests below were checked at **8cc27609**; the new review proofs at
**1afecb29**. Neither commit is on main. These are source corrections, not
agent-core 42's judgments. The separate frozen handover audit retains the
source readings and test-name checks.

- **M-1's switchEpoch citation and mechanism:** usage.go:349–364 is the end
  of settlement and start of readMeter, not switchEpoch; node/agent/reset.go
  does not exist. The implementation is node/agent/meterreset.go:311–332.
  It consumes a pending kernel nudge through `e.meter.Sample(ctx)`; there is
  no NotifyRulesetSwitch call in the reviewed implementation. The strict
  epoch equality at retain.go:53 is present. This correction does not judge M-1.
- **Q1 call site and presence margin:** canRetainServing is called at
  engine.go:1178, not 1181 (a comment). allowed_presence.go:81–88 skips the
  presence requirement when a binding's computed remaining authorization is
  **at most 101 ms**; above that it requires the prefix to be present. It does
  not itself assert a live element's remaining timeout is at least 101 ms.
  The full serving verifier supplies the other checks. No new bound is inferred.
- **Q2 binding clock and wake:** no bindingActive helper exists in the cited
  code. decide.go:623 and 646–650 use signed lease time plus monotonic elapsed
  time and inline ValidFrom/ValidUntil checks, not bindingActive(b, in.Now)
  at line 615. nextStepDelay is engine.go:630–646; it caps by policy expiry
  and grant-less renewal, with no future-ValidFrom wake calculation. The
  cited 649–659 span signaled and the Step declaration. Future activation is
  evaluated on a subsequent tick; the claimed special wake is unsupported.
- **Q1/Q3/Q4 proof names:** two cited names are absent at 8cc27609:
  TestAT13_F41_LiveDriftOnSkippedTickRepairsOrWithdraws and
  TestR23_H2_PostFinalSampleCheckpointFailureStillPreparesNewEpoch.
  The actual declarations are TestISO01_F41_SkippedTicksStillDetectLiveDrift
  in node/agent/followup41_drift_test.go:12 and
  TestR23_H2_InterruptedCheckpointPreparesNextEpoch in
  node/agent/followup40_test.go:158. The drift file has 73 lines, so the cited
  47–105 range runs past EOF. The six named r25 proof declarations and their
  three quoted failure sites match 1afecb29; none is a main test here.
- **Q3 L-1 / Q4 inventory:** in the stated b17c8c17..8cc27609 range,
  node/agent/engine_test.go and node/agent/review_r23_test.go are unchanged;
  node/agent/review_r24_test.go is added. Across that exact range there are
  **14 modified and ten added test files**, including merged main changes,
  not the report's list of 12 edited files. Within node/agent there are
  **11 modified test files**. In particular followup23_expiry_test.go changes
  deadline assertions to rendered per-device timeouts; its edits are not
  solely an Apply-trigger helper or lease-refresh flag. This corrects the
  inventory and edit description without issuing a coverage judgment.
- **Q5 log and rule quote:** the exact log message “usage checkpoint applied”
  is emitted at node/internal/usageupload/upload.go:518; engine.go:758–761
  calls restoreUsage before enforce at 767. render_nft.go:179's upload rule
  contains the counter, `meta priority set` and `accept`; the archive's
  quoted `ip daddr != @denied_v4` clause is absent there. Denied/host drops
  precede that rule. The counter/accept relationship remains visible, and
  scenario_at_usage_test.go:68 checks nonzero bytes and equality to
  nftAfter. The raw log quote matches; its direct emitter and rule quotation
  are corrected here.

## r26 dispositions at the follow-up 39 cutoff (2026-11-03 EST)

The [r26 archive](r26-expiry-fixes-review.md) is unchanged from
`.agent/REVIEW.md` on `r26-expiry-fixes-review` at
**5c6d6abe6af7ebba4707ef64c56c157362f74b7b**, dated **12:28:35 EST**:
**19,545 bytes**, SHA-256
**8cd7cc8be881389d5f50e70abe2e18eb13136b2281d66a261d9ad5820a3f0d28**.
Inspection found no host path, username or secret; product paths are retained.
**agy `gemini-4-argon-high`** reviewed agent-core 42 at
**8dec4100ce9fba96ea8b0436c41ed8d2b0c75545**, and 37–42 together
(**77626af4..8dec4100**). Findings: **0 Critical, 0 High, 0 Medium, 0 Low**.

**“Merge 37 to 42 as they are” is the reviewer's recommendation.** The
orchestrator subsequently merged **37–43 as #429 under provisional D-150**,
after 43's two timing windows qualified no leg. No comparative performance
conclusion follows. Agent-core 44 measures after the merge; its results are
outside this cut. Both r25's and r26's proof files are now on main. Agent-core
43 imported r26's file byte-identically in fe256ba9; no proof is copied by
this handover.

The following are the **reviewer's seven answered questions**, with citation
corrections below; they are not new acceptance measurements:

| Question | What the reviewer says held |
|---|---|
| AQ1 | Earliest wake/retention and last-binding whole-node deadlines are separate; per-binding kernel timeouts still cut off the earlier device, with all member/node caps and saturation checks. Watchdog behavior is unchanged. |
| AQ2 | Sibling expiry proceeds to same-tick survivor Apply without intermediate baseline. Early returns verify actual retained serving and report truthfully; last-binding/node expiry still closes. |
| AQ3 | The expired-only path requires the exact unchanged surviving subset and expired removed bindings; live verification and a checkpoint precede narrower intent/Apply. Counter and restart accounting preserve retired bytes. |
| AQ4 | Both reset-epoch producers retain only through proven SameCountingInterval continuity. Intent refresh is durable before memory; persistence failure withdraws, real counting boundaries reject retention. |
| AQ5 | Renewal jitter is uniform in integer milliseconds, drawn once per lease, persisted atomically, restored and clamped. Default lead is 108–132 s over 24,001 points; random failure and legacy rows use zero offset. |
| AQ6 | The enforcement change is only the DeadlineMono comment; dated 42 records match the production/validation boundary. |
| AQ7 | No pre-existing tests were modified by 42's own changes, r25's import is exact, and the reviewer reports earlier review/package invariants held. Main-merge test edits are excluded from that statement. |

**Not examined by r26:** (1) privileged Docker netlab execution and D-133
paired timing; (2) physical linux/arm64 and live external NetBird / AdGuard /
PostgreSQL services; (3) central control/worker and portal packages outside
the review scope. [43's outcome](../node/followup43-outcome-2026-11-03.md)
supplies the ordinary B lab pass (all 13 scenarios, D087/D104 included), two
unqualified timing attempts and plain whole-module Go tests, which include
central Go packages. It supplies arm64 cross-builds, not physical arm64 or
live external service acceptance, and no portal validation. Configured test
PostgreSQL integration is not a claim of live external-service acceptance.
Neither timing attempt completes expiry/renewal packet predicates. The ordinary
B gate is functional evidence, not a qualified timing leg.

The six Go proof names now on main in `node/agent/review_r26_test.go` are:

- `TestR26_Held_AQ1_ServingDeadlineAndKernelCutOff`
- `TestR26_Held_AQ2_SurvivorsApplyAndEarlyReturnTruthfulness`
- `TestR26_Held_AQ3_ExpiredOnlyRemovalExactnessAndAccounting`
- `TestR26_Held_AQ4_SameCountingIntervalRetentionAndIntentPersistence`
- `TestR26_Held_AQ5_RenewalJitterBoundsPersistenceAndClamping`
- `TestR26_Held_AQ6_AQ7_ValidationRecordAndZeroDecisionContract`

**Additive citation errata to r26 (archive unchanged).** Code and existing
proofs were checked at **8dec4100**, new r26 proofs at **5c6d6abe**. The frozen
source audit retains the cited excerpts and declaration checks. These correct
citations and descriptions, without adding a finding or changing the review's
recommendation:

- AQ1's heartbeat assignment is engine.go:1537–1540 and
  boot_heartbeat.go:25, not engine.go:1390 / boot_heartbeat.go:17.
  The latter two lines are respectively an unrelated serving reset and the
  beginning of the Beat literal.
- There is no deviceDeadlines helper at render_nft.go:560–597. Actual
  per-binding timeout arithmetic is node/enforce/deadline.go:35–61;
  validate.go:123–132 calls it, and render_nft.go:576–586 renders elements.
  The 100 ms install-budget check is apply.go:200–206. The cited
  releaseDeadlineGuard at deadline_install.go:49–50 only deletes the guard;
  it does not perform that elapsed-budget check. Equality to 100 ms is
  accepted by the `>` check; the archive's later `< 100ms` wording is strict
  where the code permits equality. This does not extend a binding deadline.
- validateServing at validate.go:130 does not reject a future binding as a
  validation error: memberTimeout returns non-admission and the renderer
  omits its allow element while retaining its counter. buildModel's future
  binding exclusion at decide.go:649–650 is correctly cited.
- There is no watchdog Check function at watch.go:116–157. Once starts at
  119 and calls baselineDecision (170–215), including its second read before
  Baseline. A non-serving heartbeat with a verified baseline already present
  need not trigger another Baseline. The rules are unchanged; the archive's
  unconditional `!hb.Serving` description is too broad.
- AQ3's delta.go:98–125 is usageCounters, not planContinuity; the latter
  starts at 171. attribute starts at 432, and the cited 457–504 range is
  inside it; the zero-delta skip at 492–493 precedes its mapping check.
- AQ4's switchEpoch is meterreset.go:311–332, not 255–263 (rearm). It consumes
  a pending nudge through the meter sample; the cited planSwitch and
  SameCountingInterval mechanisms are present.
- Proof declaration lines: r26 AQ1 **53** and AQ4 **424** match; AQ2 is
  **206**, not 196; AQ3 **281**, not 259; AQ5 **470**, not 462; AQ6/AQ7
  **544**, not 533. At 8dec4100, r25 H-1b is **149**, not 144; r25 M-1
  **242**, not 254; agentstore's lease/jitter transaction proof **11**, not
  12. All cited test names exist. The other declaration citations match.
- AQ7's inventory excludes the a4b76e09 main merge as the archive says.
  The full 8cc27609..8dec4100 diff also modifies three test files from main
  (isolation's producer_timing_test.go and scheduled_test.go, and handover's
  handover_test.go). No pre-existing node proof is changed by 42.


**Follow-up 40 additive correction — 2026-11-03 EST (#430 request).**
At **8dec4100ce9fba96ea8b0436c41ed8d2b0c75545**, the watchdog's
`baselineDecision` spans **node/watchdog/watch.go:170–209**; `mono()` starts
at **211**. The earlier erratum's **170–215** range includes the next function.
Its function name and first line are correct. The archive and that erratum
remain unchanged; this correction changes **no finding and no disposition**.
The [frozen source audit](../../tests/handover/testdata/history/follow-up-40/source-audit.json)
pins the source bytes and exact excerpt at that commit.

## r20 to r25 dispositions at the follow-up 39 cutoff (2026-11-03 EST)

Agent-core 37–43 merged as **#429**. These are the delivered D-141 judgments,
including the additive corrections within their dated records. Earlier cutoff
sections remain historical. “FIXED” applies only to confirmed defects backed
by the named Go tests now on main; rejected sub-claims stay rejected. Test
names containing an old failure describe regression tests, not current failures.
The records and every declaration below are pinned by TestHandover_Followup39Claims.

| Finding | Disposition | Judgment, correction and implemented fix | Proofs on main |
|---|---|---|---|
| r20 H-1 | FIXED — #429 | Confirmed refused-adoption pending-byte loss. 37 restores/settles retained usage; its claim that an Apply-before-retained-write crash could only gap was false. 38 corrects provenance using pre-refresh desired/live generation, closes settlement before refresh/Apply, clears retired bindings and opens an unbound epoch. Later 39/40 refinements remain in the merged branch. Unknown attribution still gaps; no invented owner. Record: [followup37-r20-adoption-review-2026-11-03.md](../node/followup37-r20-adoption-review-2026-11-03.md). | `TestR20_H1_RefusedAdoptionAttributesPendingDeltaOnce`; `TestR21_H1_MixedBindingsCrashWindows` |
| r20 M-1 | FIXED — #429 | Confirmed TERM could leave live rules under a false heartbeat. Local baseline completion is bounded; 38 corrects 37’s false 13 s claim to one 10 s Boot budget and 3 s Close, preserves nonzero Boot failures, and stops Run entering another Step after TERM. Successful completed baseline plus TERM still exits zero. Record: [followup37-r20-adoption-review-2026-11-03.md](../node/followup37-r20-adoption-review-2026-11-03.md). | `TestR20_M1_TermDuringBootInstallsBaselineAndExitsZero`; `TestR21_M2_CloseWaitsForTimedOutBootWithinStopBudget`; `TestR21_M2_TermCompletedBootDoesNotStartAnotherStepBudget` |
| r20 M-2 | FIXED — #429 | Confirmed delayed refusal. Generation, clock and set checks precede lock/probe and sets are checked again after evaluation. An already-held adoption node lock is not mistaken for an updater lock, avoiding a self-conflicting probe baseline. Record: [followup37-r20-adoption-review-2026-11-03.md](../node/followup37-r20-adoption-review-2026-11-03.md). | `TestR20_M2_RefusalPrecedesLockAndProbe`; `TestR20_M2_AdoptionLockIsNotUpdaterLock` |
| r20 M-3 | FIXED — #429 | Confirmed heartbeat and stale-error defects; original two-hour monotonic rewind proof rejected and replaced. Withdrawing handoff writes its heartbeat; a verified Boot baseline clears refused-adoption error, quarantine keeps it. 38 corrects orphan member-path diagnostics without discarding topology evidence. Record: [followup37-r20-adoption-review-2026-11-03.md](../node/followup37-r20-adoption-review-2026-11-03.md). | `TestR20_M3_ShutdownHandoffRecordsWithdrawal`; `TestR20_M3_RefusedAdoptionErrorDoesNotBlockNonServingAck`; `TestR20_M3_AdoptionQuarantineKeepsError`; `TestR21_L1_RefusedAdoptionMemberPathLeavesOrphanMemberPathPending` |
| r20 L-1 | CLOSED — rejected; guards in #429 | Empty installed DeniedV4 is unreachable because enforce rejects it. Retained generation ahead of ACK is harmless: persisted desired already makes adoption refuse. Explicit equality guards remain. 38 replaces the retired-binding assertion with closed-epoch/empty bindings because that assertion encoded 37’s regression. Record: [followup37-r20-adoption-review-2026-11-03.md](../node/followup37-r20-adoption-review-2026-11-03.md). | `TestR20_L1_EmptyDeniedV4NeverEqual`; `TestR20_L1_RetainedAheadOfAckRefusesAdoption` |
| r21 H-1 | FIXED — #429 | Confirmed 37 regression: crash after counting Apply before retained persistence can charge the old member. Boot captures desired before refresh and live generation, initially intersects plausible identities; 39’s appended correction replaces over-restrictive/unscoped intersection with durable boot/epoch intent. Unknown identities gap individually; unchanged devices retain records. Record: [followup38-r21-attribution-and-boot-2026-11-03.md](../node/followup38-r21-attribution-and-boot-2026-11-03.md). | `TestR21_H1_CrashBetweenApplyAndRetainedAppliedMisattributesToPreviousMember`; `TestR21_H1_MixedBindingsCrashWindows` |
| r21 H-2 | FIXED — #429 | Confirmed pre-existing failed-settlement/rebinding mischarge; 37 narrowed the restart window. Pending final settlement holds refresh and Apply, closes the interval and prepares an unbound epoch. A persistent source/storage failure keeps forwarding off. 39 extends the pending gate to runtime handoff as the appended correction states. Record: [followup38-r21-attribution-and-boot-2026-11-03.md](../node/followup38-r21-attribution-and-boot-2026-11-03.md). | `TestR21_H2_FailedSettlementThenRebindingApplyMisattributesToNewMember`; `TestR22_H1B_FailedHandoffSettlementAndSampling` |
| r21 M-1 | FIXED — #429 | Both confirmed: open-head rebinding gap pre-existed; restoring retired identities into a closed epoch was a 37 regression. Close the old interval, clear identities and prepare an unbound baseline before new traffic. Original proof’s pre-Apply traffic and nil-binding close were corrected; the old r20 retained-ahead assertion was corrected too. Record: [followup38-r21-attribution-and-boot-2026-11-03.md](../node/followup38-r21-attribution-and-boot-2026-11-03.md). | `TestR21_M1_RefusedAdoptionLeavesOpenRetainedBindingCausingFalseConflictOnRebind`; `TestR21_M1_FinalClockGapClosesEpoch` |
| r21 M-2 | FIXED — #429 | All four confirmed: uploader cancellation pre-existed (37 exposed entry-canceled/lock wait); canceled recovery latch pre-existed; stacked phases were 37’s design with older cleanup/Run-to-Step paths; false exit zero/stale failure heartbeat pre-existed. Caller-bound upload/lock, shared 10 s Boot worker plus 3 s Close, no spurious recovery latch, nonzero ErrBoot and failure receipts; 39’s correction publishes conservative receipts before blocking work. Record: [followup38-r21-attribution-and-boot-2026-11-03.md](../node/followup38-r21-attribution-and-boot-2026-11-03.md). | `TestR21_M2_BootUnderSIGTERMUncanceledUploadAndRecoverBlocked`; `TestR21_M2_OneBootBudgetAllStages`; `TestR21_M2_CloseWaitsForTimedOutBootWithinStopBudget`; `TestR21_M2_RunCanceledBootExitStatusMatrix` |
| r21 L-1 | FIXED — #429 | Confirmed 37 orphan reason; earlier main retained the actual error. Keep pending topology evidence, but heartbeat override requires a current member-path error. Non-serving decision/heartbeat agree; later serving attempts still verify and eventual repair clears it. Record: [followup38-r21-attribution-and-boot-2026-11-03.md](../node/followup38-r21-attribution-and-boot-2026-11-03.md). | `TestR21_L1_RefusedAdoptionMemberPathLeavesOrphanMemberPathPending` |
| r22 H-1 | FIXED — #429 | A/B confirmed pre-existing: same-pass lease refresh can rebind before a sample commits; failed runtime handoff bypassed settlement. Withdraw, settle outgoing identities, prepare unbound epoch and persist scoped intent before identity change; all failed handoffs gate refresh/Apply. B’s permanent paused-sampling explanation is rejected: successful Apply already calls resumeSampling unconditionally; retain ErrHold/source-failure recovery and periodic sampling guards. Record: [followup39-r22-judgments-2026-11-03.md](../node/followup39-r22-judgments-2026-11-03.md). | `TestR22_H1_UnsettledGenerationTransitionAndHandoffMisattributeUsage`; `TestR22_H1B_FailedHandoffSettlementAndSampling` |
| r22 H-2 | FIXED — #429 | All five confirmed with corrected traces: .1 live marker excludes unapplied desired; .2 unpublished desired cannot admit; .3 omitted devices have no counting rule, unique identities recover but older reuse needs provenance; .4 failed retained write triggers Baseline, then crash before settlement; .5 real first Apply counts before retained persistence, replacing artificial deletion. 38 introduced the over-restrictive/unscoped intersection/nil-model return; older restart loss and 37’s stale model existed. Pre-Apply boot/epoch intent plus qualified legacy recovery fixes false gaps; unknown older ownership still gaps. 40’s appended reset-continuity correction is retained. Record: [followup39-r22-judgments-2026-11-03.md](../node/followup39-r22-judgments-2026-11-03.md). | `TestR22_H2_CaptureRestartBindingsFalseGapsAcrossCrashWindows`; `TestR22_H2_RemovedThenReaddedCounterKeepsOlderBytesOutOfNewBinding` |
| r22 M-1 | FIXED — #429 | Confirmed pre-existing early initialization latch, made persistently wedged by 38’s pending gate. Publish initialization only after success, release partially initialized resources and retry meter/store construction. Record: [followup39-r22-judgments-2026-11-03.md](../node/followup39-r22-judgments-2026-11-03.md). | `TestR22_M1_TransientBootMeterOpenFailureWedgesRetainedPendingSettlement` |
| r22 M-2 | FIXED — #429 | Pre-existing irreversible adoption ordering confirmed: prepare meter read-only with an existing continuous open epoch, defer bindings/activation/upload until final gates. Original false-gap trace rejected: nil-bound FinalSample created its gap before Boot and closed while forwarding remained live. Reachable open-epoch late-abort proof replaces it; no pre-decision mutation/upload. Record: [followup39-r22-judgments-2026-11-03.md](../node/followup39-r22-judgments-2026-11-03.md). | `TestR22_M2_AdoptionPreparationHasNoUsageEffectsBeforeLateAbort` |
| r22 M-3 | FIXED — #429 | A/B confirmed: 38’s bounded worker exposed older late/failure publication gaps; publish conservative startup before blocking stages and verified baseline before ledger work, with capped writes inside the shared budget. C confirmed pre-existing: failed probe clear falsely burned nonce and left stale receipt; keep uncleared, retry without reinstall/extension and report failure. DatapathValue is production-unreachable after LoadConfig validation, retained only as ErrBoot/heartbeat guard. Existing kernel expiry still bounds failed-clear exposure. Record: [followup39-r22-judgments-2026-11-03.md](../node/followup39-r22-judgments-2026-11-03.md). | `TestR22_M3_FailedOrTimedOutBootAndProbeClearLeaveUntruthfulHeartbeat`; `TestR22_M3_DatapathValidationGuard`; `TestR22_M3_ProbeClearRetriesWithoutBurningNonceOrExtendingLifetime` |
| r23 H-1 | FIXED confirmed sub-traces — #429; H-1b guard retained | H-1a confirmed extra mapping gap, rejecting zero-gap/reset-byte demand: 8192 attributed, one legitimate counter_discontinuity gap, not 9216. H-1b is a confirmed provenance limitation; absent-idle-counter fixture is a guard, real zero-counter trace already passes 39, so no healthy-renderer outage proven. H-1c confirmed 39 transient-marker gate losing intent. Accept only proven same-boot reset continuity, load durable intent independently, reject contradictory marker and settled boundaries; exercise real repair/nudge. Switching is incident-driven, no measured frequency. Record: [followup40-r23-judgments-2026-11-03.md](../node/followup40-r23-judgments-2026-11-03.md). | `TestR23_H1a_EpochSwitchDuringServing_InvalidatesUsageIntentAndCreatesFalseConflictOnRestart`; `TestR23_H1b_GuardMissingIdleCounter_EpochSwitchKeepsAdoption`; `TestR23_H1c_MarkerReadFailureKeepsIntentAcrossRestart`; `TestR23_H1b_IdleZeroCounterSurvivesSwitchAndRestart`; `TestR23_H1b_RepairApplySwitchEpochKeepsIntent`; `TestR23_H1_IntentCannotCrossSettledBoundaryOrFailedIntentWrite` |
| r23 H-2 | FIXED — #429 | Confirmed interrupted barrier/checkpoint publication after a committed close. Dual-purpose flag pre-existed; 39 added the false admission error. Separate epochClosed from finalDone/barrier completion, always prepare next epoch before admission after interruption; main’s passing serving assertion did not prove preparation. Record: [followup40-r23-judgments-2026-11-03.md](../node/followup40-r23-judgments-2026-11-03.md). | `TestR23_H2_HandoffBarrierRaceLeavesFinalDoneFalse_NextServeFailsAdmission`; `TestR23_H2_InterruptedCheckpointPreparesNextEpoch` |
| r23 H-3 | FIXED — #429 | a/b/b′ confirmed: stale decision after settlement failure; unverified serving after failed Baseline; checkpoint-removal failure bypassing withdrawal. 39 added pre-Apply exposure, underlying ordering pre-existed. Latch/clear serving before fallible work, baseline before invalidation, recompute reports including early returns and keep retry. A failed kernel command is not verified closure. Healing-final-sample reason guard preserves the withdrawal cause; its receipt is after initial 40 edits, not an unchanged-39/main introduction proof. Record: [followup40-r23-judgments-2026-11-03.md](../node/followup40-r23-judgments-2026-11-03.md). | `TestR23_H3a_PreApplyHandoffSettleFailure_EmitsCorruptObservationAndDashHeartbeat`; `TestR23_H3b_PreApplyHandoffBaselineFailure_LeavesServingTrueWithFailedSafety`; `TestR23_H3_M1_FailedHandoffReportsRecoversAndRestarts`; `TestR23_H3a_HealingFinalSampleKeepsWithdrawalReasonUntilNextPass` |
| r23 M-1 | FIXED — #429 | Confirmed pre-existing missed settlement latch (38’s pending state omitted this return): first Baseline fails, second succeeds. Latch before withdrawal and preserve second fail-closed attempt; restart/recovery must not depend on a later incidental sample. Record: [followup40-r23-judgments-2026-11-03.md](../node/followup40-r23-judgments-2026-11-03.md). | `TestR23_M1_WithdrawSecondBaselineSuccess_LeavesOpenEpochUnsettledWithRetainedPendingFalse`; `TestR23_H3_M1_FailedHandoffReportsRecoversAndRestarts` |
| r23 L-1 | CLARIFIED — #429 | Ordering clarified additively in 39 records: latch removes orphan checkpoint before adoption and under node lock; read-only ExistingEpochOnly preparation precedes final gates; bindings and activateUsage wait. Reviewer’s quoted sentence and loadUsageIntent/prepareAdoptionUsage/resumeAdoptionUsage names are absent. No standalone L-1 proof exists; the named M-2 guard supports ordering, not a newly invented L-1 test. Record: [followup40-r23-judgments-2026-11-03.md](../node/followup40-r23-judgments-2026-11-03.md). | `TestR22_M2_AdoptionPreparationHasNoUsageEffectsBeforeLateAbort` |
| r24 H-1 | FIXED — #429 | Confirmed pre-existing unchanged-poll Apply; timeout-only retention dates to 28. Compare complete renderer inputs and pending work under node lock, freshly verify serving/member path and allow-element completeness. Healthy genuine 304 ticks retain enforcement; renewals, changed models or required repair still Apply. Original fake-signing fixture was actually 200 with a real 1 ms expiry change; orchestrator approved setup-only genuine-304 correction. Missing-element live drift was an additional confirmed verifier dependency. Record: [followup41-r24-judgments-2026-11-03.md](../node/followup41-r24-judgments-2026-11-03.md). | `TestR24_H1_SteadyStateNoOpStepReappliesEveryPollTick`; `TestAT10_F41_Actual304RetainsFullModel`; `TestISO01_F41_SkippedTicksStillDetectLiveDrift`; `TestISO01_F41_RendererChangesAlwaysApply`; `TestAT13_F41_LeaseRenewalApplies` |
| r24 H-3 | FIXED — #429 | Deadline collapse confirmed pre-existing in member aggregation (25), with earlier binding window. Correct mechanism: renderer already caps ValidUntil; agent collapsed every binding into shared DeadlineMono and first binding supplied wall ends. Keep member-only caps plus independent device timeout in either order. Retained model after withdrawal/reboot and Apply value-copy clock are true C1 design warnings, not new production authorization defects; no C1 receipt/eligibility. Distant deadline arithmetic caps before anchor addition. Record: [followup41-r24-judgments-2026-11-03.md](../node/followup41-r24-judgments-2026-11-03.md). | `TestR24_H3_AppliedModelRetainedAcrossWithdrawAndMultiDeviceDeadlineCollapse`; `TestAT13_F41_DeviceDeadlinesInEitherOrder`; `TestAT13_F41_DistantDeviceDeadlineUsesMemberCap` |
| r25 H-1 | FIXED — #429 | Confirmed device/member siblings at Step entry, verify and retained read. Correct attribution: verify-path whole-node withdrawal/contradictory report and watchdog earliest-deadline closure pre-existed on main; 41 added entry/retained variants. Earliest wake stays separate from last-installed-binding watchdog cap; re-decide and Apply survivors same tick, exact expired-only removal checkpoints old bytes without intermediate baseline. Last expiry/node caps and errors fail closed; current reasons replace false dash. Follow-up 38’s main-overlap question is closed. Record: [followup42-r25-judgments-2026-11-03.md](../node/followup42-r25-judgments-2026-11-03.md). | `TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass`; `TestR25_H1b_SiblingExpiryDuringVerifyOrRetainLeavesSurvivingSiblingInBaselineWithDashHeartbeat`; `TestAT13_F42_WatchdogAcrossSiblingAndLastDeadline`; `TestMTR03_F42_ExpiredRemovalSettlesBytesWithoutClosingInterval`; `TestMTR03_F42_ExpiredSettlementFailureKeepsIntentAndCloses`; `TestAT13_F42_ReportingRefreshesExistingCause` |
| r25 M-1 | FIXED — #429 | Confirmed 41 strict retention equality for both post-Apply switch and sampler discontinuity. SameCountingInterval proves continuity; persist refreshed intent before memory. Real final-sample/boot/closed-epoch boundaries reject, and failed durable refresh withdraws. Restart can interpret either side of the write. Record: [followup42-r25-judgments-2026-11-03.md](../node/followup42-r25-judgments-2026-11-03.md). | `TestR25_M1_EpochSwitchDuringServingLeavesIntentEpochStaleAndForcesRedundantApplyOnNext304Tick`; `TestMTR03_F42_RetainedEpochIntentIsDurable`; `TestMTR03_F42_RetentionRejectsCountingBoundaries`; `TestMTR03_F42_IntentRefreshFailureWithdraws` |
| Netlab F32-01 | FIXED — #429 | Confirmed pre-existing two-byte nanosecond modulus effectively fixed at minus 12 s and re-drawn on checks. Uniform integer-millisecond jitter is drawn once per lease and saved atomically: default lead 108–132 s, 24,001 points; zero offset on random failure/legacy row, restart reuse and configuration clamp. Half-window cap permits at most 0.55 of window, subject to existing 1 s base floor. Grant-less D-067 schedule unchanged. Record: [followup42-outcome-2026-11-03.md](../node/followup42-outcome-2026-11-03.md). | `TestAT13_F32_01_UniformMillisecondRenewalJitter`; `TestAT13_F32_01_RenewalDrawOncePerLease`; `TestAT13_F32_01_FailedRandomReadKeepsSafeLead`; `TestAT13_F32_01_LeaseReplayAndRestartKeepDraw`; `TestAT13_F42_LeaseAndRenewalJitterCommitTogether` |

**Scope differences from the briefing table:** no requested row is missing.
Follow-up 41 additionally judges **r24 H-2 fair-share change** as expected
behavior, not an agent defect: removing an independent member changes remaining
guaranteed shares. `TestR24_H2_MemberRevocationMutatesSiblingQoSRateInDecide`
passes unchanged. This does not change netlab 32's separate H-2 tc-order fix
or documents-only eligibility revision. Follow-up 42's outcome additionally
judges the stale **DeadlineMono comment**, corrected with the approved
comment-only change; no enforcement runtime change. Its judgments also discuss
F32-01, which its outcome explicitly records. 41's retained-model and value-copy
clock observations are C1 design warnings, not added runtime findings.

**Follow-up 38 question closed:** 42 judges r25 H-1's verify-path early
withdrawal/contradictory report and watchdog whole-node close at the earliest
binding deadline as pre-existing on main; 41 introduced Step-entry and retained
read variants. This is the lane's corrected attribution, not r25's original
attribution adopted wholesale. The earlier open question remains historical.

**r25 proof file now on main:** agent-core 42 imported it unchanged from
1afecb29. Its three finding and three held declarations are
`TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass`,
`TestR25_H1b_SiblingExpiryDuringVerifyOrRetainLeavesSurvivingSiblingInBaselineWithDashHeartbeat`,
`TestR25_M1_EpochSwitchDuringServingLeavesIntentEpochStaleAndForcesRedundantApplyOnNext304Tick`,
`TestR25_Held_AQ1_RetainedTickVerifiesUnderLockAndBlocksOnAnyPendingGateOrClockSkew`,
`TestR25_Held_AQ2_PerDeviceDeadlinesAndEdgeCases`, and
`TestR25_Held_AQ3_SameCountingIntervalAndHandoffRecoveryInvariants`.

## r27 dispositions at the follow-up 42 cutoff (2026-11-03 EST)

The [raw r27 archive](r27-members-failures-review.md) is byte-for-byte from
`r27-members-failures-review` at **dfde8d25**: **221 lines**, SHA-256
`3f4cc753495fe24edb1bc48a33ab8748231c14414d3056fcc7c33e4ca30abb10`.
The reviewer committed its proofs in **62146ff1**, but not the report. The
orchestrator committed the report byte-for-byte in **dfde8d25**. These are
separate provenance facts; the archive is now a D-098 input.

Findings: **0 Critical, 1 High, 3 Medium, 1 Low**. The archive's original
recommendations stay intact; the dispositions below apply at #445.

| Finding | Disposition at #445 | Boundary |
|---|---|---|
| H-1 | OPEN — agent-core 48, running at the cutoff | Missing interface at restart falsely triggers TC_FOREIGN_FILTER. Members' new negative subcheck fails on run 2; no product fix here. |
| M-1 | OPEN — agent-core 48, running at the cutoff | Closed recovery and provisional-timer race delay the populated lease. Deny-all Apply 584 remains inside the recovery bracket. |
| M-2 | ANSWERED — members 20 #445 | Restart boundary and each observation's held lease; no grant-less lease means D-067 blocked / not exercised, never pass. |
| M-3 | ANSWERED — members 20 #445 | Confirmed D-135/D-136 brackets, all losses counted, verified sibling-serving recovery required. Offline replay is a proof, not a verdict. |
| L-1 | OPEN — agent-core 48, running at the cutoff | Repeated quarantine/baseline writes within one Step. |
| Question to the owner | ANSWERED — confirmed D-135 and D-136, applied by the orchestrator | No new decision; no outage ceiling or zero-loss criterion. |

Members imported all four proofs first, unchanged, in **e0844864**: the two
TODO proofs failed and the two Held proofs passed. M-2 now calls the repaired
predicate with its original reports/held leases; M-3 already called
`siblingRecovery` and now supplies the verified Apply/written heartbeat facts
its scenario states. The Held proofs stay byte-identical. Main carries
`TestR27_M2_AT09_RestartProvisionalPredicateAndTimelineScope`,
`TestR27_M3_AT02_SiblingRecoveryBracketAndQuarantinePredicates`,
`TestR27_Held_AT09_RestartRecoveryConjunctsAndTimelineTrace`, and
`TestR27_Held_AT02_CapturedRecoveryPacketAndBracketAccounting` in
[the members proof file](../../tests/e2e/at/members/review_r27_test.go).
Agent/enforce proofs remain on the review branch at this cutoff. Every
historical run stays failed; the next assembled members run waits for
agent-core 48.

## r28 dispositions at the follow-up 42 cutoff (2026-11-03 EST)

The [raw r28 archive](r28-expiry-probe-review.md) is byte-for-byte from
`r28-expiry-probe-review` at **34a8d61f**: **231 lines**, SHA-256
`0af3db0ca803a0e37497618d9dca852df667d2e9a5a643f72a56676b07e593d3`.
Its base is isolation 20's delivered head **9be409f8**. This archive is now a
D-098 input. Findings: **0 Critical, 1 High, 1 Medium**.

| Finding | Disposition at #445 | Boundary |
|---|---|---|
| H-1 | OPEN — product defect; agent-core 48, running | The hung desired-state request spends the Step's control budget; the tunnel probe receives that expired context and fails without running. This diagnoses F20-01. See [operator behaviour](../handover/operator-guide.md#r28-h-1). |
| M-1 | OPEN — isolation, after agent-core 48 | memberExpiry passed in c52bea6c while H-1 had withdrawn all forwarding. The recorded pass stays unchanged; it is not evidence for AT13 per-member expiry under a serving node. Isolation's next run after agent-core 48 decides it. F20-01's failure stays failed. |
| Optional NetBird management-loss note | RECORDED — no action now | Readable status with management disconnected is a separate question; no new decision or expanded F20-01 fix is assigned. |

**Orchestrator verification, on main at dd052c60**, as supplied in the
follow-up 42 brief: H-1's TODO proof fails and the three Held proofs pass.
Swapping the probe's context alone turns the TODO proof green, but fails
`TestR21_M2_OneBootBudgetAllStages/adoption_probe`. Agent-core 48 is briefed
to keep both. This is the orchestrator's reported verification, not a new
handover execution or an implemented fix.

The review-branch proof names are
`TestR28_ControlOutageDoesNotFailTunnelProbeBeforeLeaseExpiry`,
`TestR28_WhatHeld_RefreshProbeAndNetbirdProberContextPropagation`,
`TestR28_WhatHeld_FastControlRefusalPreservesServingUntilDeadline`, and
`TestR28_WhatHeld_PostDeadlineReasonOrderingInDecideAndBaseline`.
They are pinned through the review archive, not claimed as declarations on
main. The existing R21 budget proof is on main. No merged fix or date is
promised. **r29 is running, status only, and is not archived in this cut.**

## r27 and r28 dispositions at the follow-up 43 cutoff (2026-11-04 EST)

The raw archives and the #445 dispositions above remain historical. This cut
ends at #456 / **04d31ea5**. The [frozen agent-core 48 note](../../tests/handover/testdata/history/follow-up-43/docs_node_followup48-2026-11-03.md.txt)
and [PR #447](https://github.com/TED-H-com/Freedom-VPN/pull/447) record all four
imported TODO proofs failing before and passing after, and six Held/WhatHeld
proofs passing. The deterministic R23 heartbeat proof also fails before and
passes after; Boot's R21 `adoption_probe` budget stays protected.

| Finding | Disposition at #456 | Evidence and remaining boundary |
|---|---|---|
| r27 H-1 | FIXED — #447; holds live in #455 | Independent successful link inventory distinguishes interface absence from a foreign filter. Members 21 AT09 index 27 passes; no TC_FOREIGN_FILTER after restart. `TestR27_H1_AuditBlockedMissingInterfaceAtBootLatchesFalseTCForeignFilter`. |
| r27 M-1 | FIXED — #447 | Grant-less deadline schedules Step; freshly verified deny-all recovery model is retained. `TestR27_M1_ProvisionalTimerRaceAndClosedRecoveryDelayPopulatedLease`. Members 21 has no re-baseline after closed recovery, but its remaining 4.968 s / 241 losses waiting for the slot is a separate availability item under D-157. |
| r27 M-2 / M-3 | ANSWERED — #445; exercised in #455 | AT09 index 25 remains blocked / not exercised, 26 passes. AT02 accounting and sibling recovery pass D-135/D-136. Every historical failed run stays failed. |
| r27 L-1 | FIXED — #447 | One containment attempt per local pass, verified baseline reused until rules mutate. `TestR27_L1_ForeignTCQuarantineStepRedundantBaselinesAndReQuarantine`. |
| r28 H-1 / Isolation F20-01 | FIXED — #447; qualified live short whole-node expiry in #452 | Independent local context preserves the probe under hung control requests. `TestR28_ControlOutageDoesNotFailTunnelProbeBeforeLeaseExpiry`; the original F20-01 run stays failed. |
| r28 M-1 | CHECK MERGED — #452; live coverage still open | Member expiry now requires a serving node. Run 20 evidence replays as fail; its historical pass remains recorded but is not serving-node evidence. Isolation F21-01 never reached the probe window; isolation 22 is running at cutoff. |
| R23 boot heartbeat flake | FIXED — #447 | Remove the independent 100 ms child deadline while retaining publication expiry checks; `TestF48_R23_BootHeartbeatUsesOneBudget`. |

Agent-core 48's lab records **12 pass, 1 fail, 1 optional skip**. The
orchestrator diagnoses that D087/D104 failure as **F48-01**, a pre-existing
same-boot startup/watchdog race; 48 found it, did not cause it. Agent-core 49
#454 fixes it under D-156; its lab passes **13/13 on one attempt**. See the
[release rule](../handover/operator-guide.md#d156-release-rule).

## r29 dispositions at the follow-up 43 cutoff (2026-11-04 EST)

The [raw r29 archive](r29-update-recover-review.md) is byte-for-byte from
`r29-update-recover-review` at **5b1ca726c2515967824f4294823aaf1f2ac5f8a7**,
base **dd052c6012f10fbde147f61931a50c00bec1f98b**: **180 lines**, SHA-256
`5c54bfb22cc9f407f779f8f2f401530ffe5238711b2adc10401d69abee20b7af`.
It is a **design diagnosis of F30-01**, by **agy/gemini-4-argon-high**. The
reviewer committed the report and proofs at 5b1ca726; that commit carries
`Agent: agy/gemini-4-argon-high`. Unlike r27, this report was committed by the
reviewer. This raw archive is now a **D-098 input**, not a rewritten disposition.

Findings: **0 Critical, 1 High, 1 Medium, 2 Low**.

| Finding | Disposition at #456 | Boundary |
|---|---|---|
| H-1 | FIXED — updater half #446, agent half #454 under D-155 | Exact live-lock ExitBusy 3; deferred candidate boot recovery and later Step retry. Terminal failures stay terminal. F30-01 acceptance closure remains pending. |
| M-1 | FIXED — #454 | One tracked recovery child with a 250 ms first wait; later Steps poll without waiting. Real crashed blocked/switched/checking stages keep the agent running; stop cancels and joins. |
| L-1 | FIXED — #446 | Six lock-taking entry points check update.lock before journal access; Retain never opens a journal. |
| L-2 | OPEN — devstack 34, running at the cutoff | Nightly quarantine's stale citations and ordering note. Entry removal requires the passing lease-expiry acceptance result. |
| Question to the owner | ANSWERED — D-155 | The orchestrator's provisional Option B decision; the owner may overturn. Rejected A, C, D1 and D2 are recorded in the decision table. |

Updater's import **03e24246** and agent's import **facfd361** preserve their
entire r29 proof files byte-identically. Each side's two TODO proofs fail
before and pass after; each side's two Held proofs pass throughout. Main now
carries:

- Updater: `TestR29_H1_UpdaterExitCodeDistinguishesLiveUpdateBusy`,
  `TestR29_L1_ContendedRecoverChecksUpdateLockBeforeOpeningJournal`,
  `TestR29_Held_ContendedRecoverDuringStepStartAndCheckDoesNotMutateTxnOrGate`,
  `TestR29_Held_CrashedSwitchedAndCheckingTxnsRecoveredUnderFreeLock`.
- Agent: `TestR29_H1_CandidateBootUnderHeldUpdateLockDefersRecoverAndServesAfterCommit`,
  `TestR29_M1_RecoverOnceDoesNotSelfDeadlockOrKillOpenTransactionRecovery`,
  `TestR29_Held_D053AndD064PassWhileRecoverBlockedUnderHeldUpdateLock`,
  `TestR29_Held_TerminalRecoveryRequiredExit2KeepsForwardingOffAfterLockRelease`.

The [cut](../handover/follow-up-43.md) pins the imported proof identities and
producer evidence. Handover did not rerun the review proofs or acceptance.
**F30-01 stays open** until an acceptance run passes `lease-expiry` with the
nightly quarantine entry removed. Devstack 34 is running for that at cutoff.


<a id="follow-up-44-dispositions"></a>

## Follow-up 44 dispositions (2026-11-04 EST)

The earlier dated tables describe their original cuts. At #463:

| Finding | Current disposition | Frozen evidence |
|---|---|---|
| r29 L-2 | ANSWERED — #458; stale citations left with the removed lease-expiry entry. member-ipv6 remains. | [Devstack 34](../../tests/handover/testdata/history/follow-up-44/docs_e2e_runs_2026-11-04_followup34_analysis.md.txt) |
| r28 M-1 / Isolation F21-01 | ANSWERED — #459, first live pass under a serving node, all five requests time out with a 26.3 s node-lease extension. | [Isolation 22](../../tests/handover/testdata/history/follow-up-44/docs_e2e_at_isolation_followup22-2026-11-04.md.txt) |

F30-01 remains open despite lease-expiry's pass: devstack 34 never started a
candidate. Updater 25 #463 adds D-158's bounded live-lock wait; closure now
requires update-good to start the candidate, deferred boot recovery under its
parent's lock to clear, candidate member service, then a passing lease-expiry.
Devstack 35 runs for it at cutoff. Every historical failure stays failed; raw
review archives and imported proofs are unchanged. See [the cut](../handover/follow-up-44.md).


## r30 dispositions at the follow-up 45 cutoff (2026-11-04 EST)

The [raw r30 archive](r30-update-fixes-review.md) is byte-for-byte from
`r30-update-fixes-review`, head **da2560a906a9db4fd3ed48708724f664a46a2053**,
base **9b9546c9502fe21239dde26b156f38dfbb0c1a58**: **216 lines**, SHA-256
`b7615f2de08ca6914fb85bd957c00ff7b903dcfa1ac03d229711d1eb38efb933`.
This is a **post-merge code review of D-155 through D-158** (#446, #454, #461,
#463), by **agy/gemini-4-argon-high**, committed by **Ted H** as the branch
records. The archive is now a **D-098 input**. Findings: **0 Critical, 2 High,
3 Medium, 0 Low**.

| Finding | Disposition at #468 | Boundary |
|---|---|---|
| H-1 | OPEN — agent-core 51, running at cutoff | Canceled Boot context can skip or wedge recovery through the engine API. The orchestrator's reading: not reachable from fvpn-agent today; run and once modes pass one context to Boot and Step. Latent fail-open default in the engine API, routed for repair anyway. |
| H-2 | OPEN — updater 26, running at cutoff | Stale-then-live lock answers terminate early. D-158 as clarified requires five consecutive stale answers, reset by a live answer, and the live budget from the first attempt. |
| M-1 | OPEN — updater 26, running at cutoff | Non-busy first-attempt error emits a spurious wait record. D-158 as clarified emits it only after the first attempt misses on ErrBusy. |
| M-2 | OPEN — agent-core 51, running at cutoff | Pre-enforce lease attempt can suppress post-recovery renewal after the applied generation advances. |
| M-3 | OPEN — agent-core 51, running at cutoff | Pending-outbox deduplication can prevent a new closed recovery from binding its own report. |
| Question to the owner | None | The review asks none. The orchestrator's D-158 clarification is provisional; the owner may overturn it. |

The review branch's `r30-update-fixes-review:node/agent/review_r30_test.go`,
`r30-update-fixes-review:node/updater/review_r30_test.go` and
`r30-update-fixes-review:node/watchdog/review_r30_test.go` are
**routed, not adopted**: none is on main at this cutoff. Their branch identities
are retained in the [provenance manifest](../../tests/handover/testdata/history/follow-up-45/review-provenance.json).
No review proof is represented as a merged or rerun proof here.

D-158's clarified contract is in the [decision row](../handover/README.md#follow-up-45-decisions-register-2026-11-04-est).
Main still counts five attempts from the first stale answer; updater 26 is
running for the consecutive-stale rule and wait-record correction. Its results
belong to a later cut. The [follow-up 45 cut](../handover/follow-up-45.md) records
F30-01's separate closure in devstack 35: the candidate deferred recovery,
cleared and served, then lease-expiry passed. That run met no contended
acquisition, so D-158's wait remains unit-proven, not measured live.


## r30 dispositions at the follow-up 46 cutoff (2026-11-04 EST)

At **#473**, all five r30 findings are fixed on main. The raw archive and
its earlier dated dispositions remain unchanged. The
[frozen agent-core 51 note](../../tests/handover/testdata/history/follow-up-46/docs_node_followup51-2026-11-04.md.txt)
and [frozen updater 26 note](../../tests/handover/testdata/history/follow-up-46/docs_updates_node-followup26-2026-11-04.md.txt)
record diagnosis, unchanged-code failures and passing repaired proofs.

| Finding | Disposition at #473 | Boundary |
|---|---|---|
| H-1 | FIXED — #470 | Only Run owns the recover child's run context; outside Run no per-call context owns it. Close cancels and joins. The orchestrator's reading stays: neither CLI mode reaches H-1. |
| H-2 | FIXED — #471, clarified D-158 | Five consecutive stale answers; every live answer resets the count. No exit 3 before the first-attempt 10 s budget; the first live answer at or past it gives exit 3. |
| M-1 | FIXED — #471, clarified D-158 | Wait record requires first-attempt ErrBusy, including later cancellation. Non-busy first error and cancellation before any attempt write none. |
| M-2 | FIXED — #470 | One catch-up request for a newly available ACK despite an earlier same-Step attempt; retry bound and Retry-After remain. |
| M-3 | FIXED — #470 | A closed recovery waiting to bind its report bypasses dedup, after the 32-entry capacity guard. |

Main carries the agent, updater and watchdog r30 proofs. Every assertion is
retained; the agent's two manual-clock corrections preserve D-155's retry
floor and the fake 503's Retry-After. Agent-core 51's ordinary lab passes
13/13 on one attempt, functional evidence only. D-155, D-156, D-157, D-067,
D-135 and D-136 remain; forwarding needs a populated lease and verified Apply.
Updater's recover/standalone Retain prompt window, error lines, exit table
and nodelock remain. **D-158's wait remains unmeasured live.**

**Review r31 is running at cutoff**, a post-merge code review of agent-core 51
and updater 26 by **agy/gemini-4-argon-high**. Devstack 36 is running on those
fixes with D-162's TEST ONLY fixture change and consumer/If-Match audit.
These are status only; no delivered or merged PR was present at the initial
check. Later results are **after the cutoff**. See [follow-up 46](../handover/follow-up-46.md).

## r31 dispositions at the follow-up 47 cutoff (2026-11-04 EST)

The [raw r31 archive](r31-recovery-lock-fixes-review.md) is byte-for-byte from
`r31-recovery-lock-fixes-review`, head **0e33b9fe3ef62244aab7a35eb1119ef6a7036621**,
base **fd0f249418926e875928d4e6799ac2fc05c6ba9f**: **302 lines**, SHA-256
`46300735f2a56cfcab37d22b862153efd2f2895e6e25c244f7af938ca5ec7aaa`.
This is a **post-merge code review of agent-core 51 (#470) and updater 26
(#471)**, by **agy/gemini-4-argon-high**, committed by **Ted H** as the branch
records. It is now a **D-098 input**. Findings: **0 Critical, 0 High,
1 Medium, 0 Low**. The raw report retains its original source descriptions;
the [frozen agent-core 52 diagnosis](../../tests/handover/testdata/history/follow-up-47/docs_node_followup52-2026-11-04.md.txt)
records the actual current call sequence and reachability before the fix.

| Finding | Disposition at #477 | Boundary |
|---|---|---|
| r31 M-1 | FIXED — agent-core 52 #477 | The shared `canBindRecoveryReport` predicate is extracted exactly from existing binding rules. At unchanged state with a report pending, only an armed, unbound recovery that can bind its own report bypasses dedup. Case A's isolation latch persists through Steps; B's armed failed-meter state is constructed, not reached by normal Steps; C's stale lease can last one enforcement pass. |
| r30 H-1, H-2, M-1, M-2 | HELD in r31 | The fidelity table marks all four implemented, along with the four clarified D-158 clauses. Updater needs nothing. |
| r30 M-3 | Exception incomplete in r31; FIXED — #477 | r31 M-1 is the incomplete guard in #470's exception. Ordinary dedup is restored; unchanged-state binding remains covered. |
| Question to the owner | None | The review asks none. |

Both r31 proof files are on main through **#477**:
`node/agent/review_r31_test.go` and `node/updater/review_r31_test.go`.
The updater is byte-identical to the review branch, SHA-256
`00414c8f141bab43caed86eb683d28c0baa50fb3ee27d833711316a6c4e7b132`.
The agent differs only as its dated note records: three comment labels change
from `TODO` to `Regression`; case C crosses the existing **30 s stale-lease
alarm** and asserts that real transition before taking its unchanged-state
baseline. **Every imported assertion is kept**; all three M-1 subtests fail
on unfixed code. Its held proofs pass before and after. The four main proof
names are `TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox`,
`TestR31_Held_H1_RecoverLifetimesAndFailClosedGate`,
`TestR31_Held_M2_M3_InteractionSingleEarlyRenewalAfterAcceptedReport` and
`TestR31_Held_D158_AnswerSequencesAndRecordInvariants`.

The 32-entry outbox, report body, cadence, retry, backoff, Retry-After,
agent-core 51's unchanged-state binding, D-067, D-135/D-136, D-155/D-156 and
D-157 remain. Agent-core 52's ordinary lab passes **13/13**; no assembled live
claim is made. Devstack 36 **#476** separately records D-158's first live wait:
**446 ms, acquired**, with stale-then-live contention unobserved. The two r30
dispositions sections above stay byte-identical. **Review r32**, a post-merge
review of agent-core 52 by **agy/gemini-4-argon-high**, is running at briefing,
status only; no delivered or merged PR at the initial check. See the
[follow-up 47 cut](../handover/follow-up-47.md).

## r32 dispositions at the follow-up 48 cutoff (2026-11-04 EST)

The [raw r32 archive](r32-dedup-records-review.md) is byte-for-byte from
`r32-dedup-records-review`, head **21e59029934a34f7125805e116a2ffaaf61b60cb**,
base **1c39a0e73825dbcf911dfd2bec6e75aed64c979b**: **201 lines**, **31,981 bytes**,
SHA-256 `5b7ec63eecbe64e278e16bf24ef3a8e9ea3e625d6bde701d6d0c0e5735e1960f`.
This is a **post-merge code review of agent-core 52 (#477, commit
`341d099a6ebeeac5221275d6db748a6b3bfdc94d` merged as
`1c39a0e73825dbcf911dfd2bec6e75aed64c979b`)**, by **agy/gemini-4-argon-high**,
committed by **Ted H** as the branch records. It is now a **D-098 input**.
Findings: **0 Critical, 0 High, 0 Medium, 0 Low**.

| Item | Disposition at #481 | Boundary |
|---|---|---|
| `r31 M-1` (`canBindRecoveryReport(rep)`) | HELD in r32 (`implemented`) | `noteRecoveryReport` (`node/agent/recovery_renewal.go:27-31`) delegates directly to `canBindRecoveryReport(rep)` (`node/agent/recovery_renewal.go:35-67`), preserving every pre-`#477` binding check line-for-line while keeping `recoveryRenewalEligible()` as the coarse engine precondition (`node/agent/review_r31_test.go:80`). Pending-outbox dedup bypass in `node/agent/observe.go:46-57` is gated on `canBindRecoveryReport(rep)` across Cases A, B, and C. |
| `observation enqueued` & `observation submission` records | HELD in r32 (`implemented`) | `node/agent/observe.go:74-77, 313-320, 341-343, 359-405`, `node/agent/observation_log.go:18-100`, and `docs/node/README.md:151-220` match runtime values. `submitObservation` preserves exact `flush` control flow (`202` accepted, `IdentityFrozen`, `403 IDENTITY_REVOKED`, `409`/`422` conflict continuation, retryable backoff with jitter and `Retry-After`). `observationAttempts` is bounded by the 32-entry outbox cap. |
| What held (Invariants 1–6 & Questions 1–7) | HELD in r32 | Invariant 1 (D-157's renewal), Invariant 2 (Dedup across Cases A/B/C), Invariant 3 (No new authority), Invariant 4 (Delivery unchanged), Invariant 5 (Records leak nothing: finite `observationErrorClass` and validated `api.ErrorCode` tokens), and Invariant 6 (Records are exact). Under Q2, when `!e.canBindRecoveryReport(rep)` returns early before updating `e.lastObs`, `e.report` runs twice per `Step` after the first elapsed interval while armed and unbindable with pending items, performing only transient in-memory construction plus bounded read-only SQLite snapshots (`usage.db` 2 s, `journal.db` 1 s) without mutating any state observable by Central or later Steps. Under Q7, later state catch-up binds on the next due `observe` and dispatches at most one early renewal. |
| Proof file (`node/agent/review_r32_test.go`) | Review branch only (`present_on_main: false`) | Because r32 found 0 defects and routed 0 owner actions, `node/agent/review_r32_test.go` (`20,346` bytes, SHA-256 `8625fecccc518a49c10551bbd63d26be2807dc60b558816855182cb5d7ad999a`, [frozen copy](../../tests/handover/testdata/history/follow-up-48/review-node_agent_review_r32_test.go.txt)) remains on `21e59029934a34f7125805e116a2ffaaf61b60cb` and is not on `main`. Its four top-level tests are `TestR32_Held_Q1_Q2_DedupExceptionAndRebuildCost`, `TestR32_Held_Q3_Q4_SubmitObservationBranchesAndDiagnosticMap`, `TestR32_Held_Q5_Q6_ObservationRecordFieldsAndErrorClasses`, and `TestR32_Held_Q7_RecoveryInteractionLateBindAndSingleEarlyRenewal`. |
| Question to the owner | None | The review asks none (`0` findings, `0` owner questions). |

In **#481**, [members follow-up 24](../../tests/handover/testdata/history/follow-up-48/docs_e2e_at_members_followup24-2026-11-04.md.txt)
subsequently consumed the `observation enqueued` and `observation submission`
records in live run `dee8492f` under `D-162`: **18 stored observations**,
**18 `observation enqueued` records** and **18 `observation submission`
attempts** across all nine native MTU windows, joined by report ID with zero
missing joins, plus closed recovery report
`01a1a782-6b90-74a8-bcd6-c9a06650a2ec` (`due_reason: "closed_recovery_binding"`)
accepted on attempt 1 in **8 ms**. The r31 dispositions section above stays
byte-identical. See the [follow-up 48 cut](../handover/follow-up-48.md).
