# Roadmap

Milestones, gates and work packages come from the Project Plan v1.0. Durations are planning aids, not commitments.

## Milestones

| ID | Goal | Exit condition |
|---|---|---|
| M0 | Baseline ready | Scope, open decisions and P0 cases are consistent |
| M1 | Technical feasibility | Core risks have measured evidence (WP02) |
| M2 | Basic service loop | Enrollment, classification, publication, login, devices and revocation work with real apps |
| M3 | Billing and policy | Sandbox billing and policy acceptance pass |
| M4 | Fleet update | One action updates every node and tracks every result |
| M5 | Pilot | P0 cases pass; service limits and parameters are explicit |
| M6 | Delivery | Delivery inventory and acceptance report are complete |

## Evidence gates

| Gate | Question | Evidence |
|---|---|---|
| G1 | Can the official apps on all four platforms log in to the self-hosted service and pick a given exit? | NetBird pinning and API spike, then real-device runs |
| G2 | Does one package block internal networks while nftables and tc really see forwarded packets? | Privileged network lab, then real amd64 and arm64 nodes |
| G3 | Can billing state revoke access, and are replays and offline periods never double counted? | Ledger and entitlement suites |
| G4 | Do all nodes update, catch up after being offline, and keep a disabled state? | Updater fault-injection suite |
| G5 | Do all P0 cases pass, with production parameters and known limits explicit? | Acceptance report |

## Task board

| Task | Work package | Status |
|---|---|---|
| w0-bootstrap: repositories, contract, spec | WP01 | done |
| w1-contracts: OpenAPI, schemas, vectors, shared leaf packages | WP03 | done (#4) |
| w1-db: PostgreSQL schema, DB foundation, CI | WP03 | done (#1) |
| w1-db follow-up 1 | isolated test databases named fvpn_t<unix-hex>_<32hex>; dbtest.Sweep drops idle leaked databases older than a bound (legacy fvpn_<hex> only with FVPN_TEST_DB_SWEEP_LEGACY=1) in one zero-connection-checked batch; FVPN_TEST_DB_SWEEP=1 sweeps once per process; fvpn-db-sweep command and db-test-sweep target | done (#97) |
| w1-netlab: G2 isolation, metering and shaping lab | WP02, WP05, WP07 | done (#16) |
| w1-netbird: NetBird pinning, adapter, dev deployment (G1 server side) | WP02, WP04 | done (#6), G1 open |
| w1-portal: Admin and Member portal on a mock API | WP10 | done (#2) |
| w1-deps-bump: current pgx, goose and transitive module pins | WP03 | done (#3) |
| w1-dnspolicy: domain rules, bundle compiler, AdGuard Home backend | WP08 | done (#5) |
| r1-db-review: adversarial review of the merged schema (read-only) | WP03 | done (3 blockers, 8 majors → w2-db-fixes) |
| w2-db-fixes: review fixes F-01..F-15 in migrations 00001–00005, regressions, service invariants doc | WP03 | done (#7) |
| r1-contracts-review: adversarial review of OpenAPI, schemas, canonjson, envelope (read-only) | WP02 | done (5 majors, 7 minors → w2-contracts-r2) |
| w2-contracts-r2: review fixes, agent bootstrap endpoint, reinstall target, UpdateTarget flags, ledger-epoch purpose, consumer adaptation (41 endpoints, presence types) | WP02 | done (#20) |
| w2-control-core: HTTP framework, auth, idempotency, ETag, operations, audit, worker | WP04, WP09 | done (#13) |
| w2-pki-enroll: node CA, enrollment, CSR issuance, revocation, signing service | WP04 | done (#11) |
| w2-members-billing: members, entitlement, devices, billing adapter and webhooks | WP06 | done (#9) |
| w2-nodes-desiredstate: node registry, desired state compiler, serving leases, exits | WP04, WP05 | done (#17) |
| w2-nodes-desiredstate follow-up 2: review fixes (compile without release target, fair-use state, dns_profile, lease locking, signed generation) | WP05 | done (#27) |
| w2-usage-central: usage ledger ingest, checkpoints, projection, rollups | WP07 | done (#15) |
| w2-usage-central follow-up: checkpoint initialisation race under concurrent first events (MTR03 flake) | WP07 | done (#21) |
| w2-usage-central follow-up 3: review fixes (freeze fence, gap and replay scoping, overflow, ledger-epoch purpose) | WP07 | done (#66) |
| w2-policies-fup: plan, fair-use, DNS and release-support policies, FUP tiers | WP07, WP08 | done (#12) |
| w2-fleet-update: releases, manifests, update batches, serving-version policy | WP09 | done (#10) |
| w2-updater: node updater, update journal, release packaging | WP09 | done (#18) |
| w2-agent-core: node agent, watchdog, control client, time anchor | WP05 | done (#25) |
| w2-usage-agent: node meter, usage store, upload with contiguous ACK | WP07 | done (#24) |
| w2-db-fixes-2: isolated migration CLI test, contract-conformant retention audit rows, one-time incarnation binding | WP03 | done (#14) |
| w3-toolchain: Go 1.27.3 in go.mod and every CI pin | WP01 | done (#19) |
| r2-control-review: adversarial review of the merged control plane (read-only) | WP04, WP06 | done (7 blockers, 7 majors, 2 minors → owner follow-ups) |
| r2-updater-review: adversarial review of updater, enforcement and packaging (read-only) | WP05, WP09 | done (4 blockers, 9 majors, 3 minors → owner follow-ups) |
| r2-ledger-review: adversarial review of ledger, desired state, policies and DNS (read-only) | WP05, WP07, WP08 | done (1 blocker, 9 majors → owner follow-ups) |
| w3-wire-members: members, devices, entitlements, billing behind control and worker; NetBird devices adapter; composition seam (D-026) | WP06 | done (#22) |
| w3-wire-nodes: PKI authenticator, enrollment and bootstrap, desired state, leases, observations, exits, releases, update batches | WP04, WP05, WP09 | done (#32) |
| w3-wire-policies-usage: policies and Fair-Use operations, fup worker, usage mount with PKI signer, audit reader | WP07, WP08 | done (#40) |
| w3-dns-publish: DNS policy compiler, node applier with evidence, lab scenario | WP08 | done (#30) |
| w3-wire-nodes follow-up 1: adopt the composition seam | WP04, WP05, WP09 | done (#32) |
| w3-dns-publish follow-up 1: adopt the worker seam, split document types from the store | WP08 | done (#30) |
| w2-members-billing follow-up 1: review fixes (operations and audit conformance, exit-group eligibility reader, synchronous checkout, patch status) | WP06 | done (#29) |
| w2-control-core follow-up 1: review fixes (conformance helpers, role cookies, limiter split, agent receipts) | WP04 | done (#42) |
| w2-updater follow-up 1: review fixes (symlink escape, forwarding gate, persistent-data check, nodelock, downloads) | WP09 | done (#33) |
| w1-netlab follow-up 1: review fixes (HTB class cleanup, deadline fail-closed, reset detection, IPv4 validation, evidence dir) | WP02, WP05 | done (#35) |
| w1-dnspolicy follow-up 1: compiled bundle round-trip, source fetch SSRF | WP08 | done (#26) |
| w3-wire-policies-usage follow-up 1: adopt the composition seam over the ledger fixes, single-format audit reader, FUP cap removal | WP07, WP08 | done (#40) |
| w2-db-fixes-2 follow-up 1: canonical purge_retained audit row (D-035), accepted domain relaxations documented (D-033) | WP03 | done (#28) |
| w2-agent-core follow-up 1: usage ledger integration, required host reserve, fake control usage endpoints | WP05, WP07 | done (#36) |
| w2-control-core follow-up 2: members fixture on role cookies, module-wide integration suite | WP02 | done (#42) |
| w3-wire-members follow-up 1: retire legacy session job, composition guide refresh | WP02 | done (#31) |
| w3-contracts-r3: additive enums (node.enroll, device_authorize, grant_revoke, admin_identity), node-reported codes, revoked_release_ids, policy_generation, checkout 201-only, role selector, policy schema promotion | WP02 | done (#34) |
| w2-nodes-desiredstate follow-up 3: entitlement serving readers (D-030), dns_generation from publication heads, RenewTx, audit conformance | WP04, WP05 | done (#39) |
| w3-wire-nodes follow-up 2: D-039 fixture isolation (nodestest), dual-stack resolver configuration | WP04, WP05 | done (#32) |
| w2-control-core follow-up 3: terminal dead-lettering with fleet singleton fallback (D-040); follow-up 4: members dead-letter test on D-029; follow-up 5: revision 3 adoption (admin_identity, report-only codes, strict audit.Append, nil-tx guard) | WP04 | done (#42, together with fleet-update follow-up 2) |
| w1-netlab follow-up 2: idle counters are not meter-epoch changes (D-041), exported DetectMeterResets | WP02, WP05 | done (#35) |
| w1-netlab follow-up 3 | D-063 §13.9 synthetic-probe exception in node/enforce: /32 test peer with exact destination tuples, optional resolver DNS on port 53 only, mandatory expiry with a 300 s ceiling enforced by kernel set timeouts, rendered in both the full and the fail-closed baseline rulesets behind the deny/host drops, own probe counters outside member usage, Verify/VerifyBaseline accept exactly the declared rules; lab proves the positive flow and the sibling/destination/port/admin-port/expiry negatives | done (#110) |
| w2-fleet-update follow-up 1: review fixes (SubmitResult generation compare, signed intent issued_at, node_name, Register validation order, applied policy_generation) | WP09 | done (#37) |
| w2-pki-enroll follow-ups 1–2: review fixes (bootstrap verdict, node_id binding, trust bundle filter), migration 00111 guarded reinstall, node.enroll operations, core audit builders | WP04 | done (#43) |
| w3-wire-members follow-up 2: adopt revision 3 kinds (member.device_authorize, member.grant_revoke), D-029 audit conformance | WP06 | done (#38) |
| w3-wire-members follow-up 3: production `netbird_user_bindings` writer (D-052) — shared-issuer public-subject discovery, read-only NetBird user list, native peer admission through the product | WP06 | done (#76) |
| w2-updater follow-up 2: enforce signed revoked_release_ids, ARTIFACT_INVALID reporting, applied policy_generation (D-043) | WP09 | done (#41) |
| w3-wire-policies-usage follow-up 2: fail-closed signing and explicit policies/usage settings (D-045), fixture helpers | WP07, WP08 | done (#40) |
| w2-agent-core follow-up 2: revision 3 node codes, external meter resets via DetectMeterResets (D-041), per-event usage conflicts | WP05, WP07 | done (#44) |
| w2-agent-core follow-up 3: baseline safety separated from serving, unpublished generations acknowledged, data-path probe (publication deadlock, agent side) | WP05, WP07 | done (#75) |
| w2-agent-core follow-up 4: health attestation producer — the running candidate agent answers the updater challenge from its own checks (D-053) | WP05, WP07 | done (#84) |
| w2-agent-core follow-up 5 | Resolver-document consumer with digest/size verification and restart cache, dns_policy_evidence in observations (D-056); signed egress probe overriding the local target and sanitized check detail (D-058); fails closed without an injected DNS applier | done (#92) |
| w2-agent-core follow-up 6 | D-053 producer fixes (review MED-3): desired ETag only after signature and validity-window verification, no health answer before a verified desired state is held, transient first-sync errors no longer attest dns_policy failed; D-061 resolver admin wiring (agent.json dns.resolvers[] + root-only 0600 credentials file, reloaded on SIGHUP, host-local admin origins only) | done (#99) |
| w2-agent-core follow-up 7 | D-062e: updater journal snapshot on every observation (update object, release pointers from the journal, bounded uncached read, classed error log) | done (#115) |
| w1-deps-bump follow-up 1: golang.org/x/crypto GO-2026-5932 | WP01 | done (no change: GO-2026-5932 is the unused x/crypto/openpgp packages, no fixed release exists, govulncheck symbol scan clean) |
| w3-portal-live: portal on the real control plane, Playwright end-to-end | WP10 | done (#56; all screens on live endpoints, Playwright e2e against real control/worker in CI, dev stack seeds inputs and awaits product projection per D-050) |
| w3-portal-live follow-up 3: dev-stack node readiness through the agent observation path (fixes main Portal CI after #62) | WP10 | done (#67) |
| w3-portal-live follow-up 4: pending-apply banner derived from the PATCH response (Portal CI race), DOC12 device-id label, update-target policy evidence | WP10 | done (#74) |
| w3-portal-live follow-up 5: dev-stack member device discovery through the production identity writer (D-052); Portal CI on main green again | WP10 | done (#82) |
| w3-portal-live follow-up 6 | Update-target policy evidence on batch targets, target detail and node detail (D-048); portal dev stack drives update batches, desired state, update results and observations through the product APIs instead of hand-written rows (D-050); two documented fixture deviations remain | done (#91) |
| w3-portal-live follow-up 7 | member groups admin and exit attachment with publication intent (D-057; members never see group identity, unit + Playwright proof), NodeCheck.detail and threat_feed_status surfaces with update_interval_seconds (D-058), Playwright dev stack without raw SQL seeds: groups via API, second node enrolled with its own identity, publication through the endpoint with fake-agent observation (D-050); 7 e2e tests pass three times | done (#108) |
| r2-ledger-review confirmation pass on main 3224345: probes re-run read-only, 11 of 12 findings closed; finding 9 (uint64 clamp) still open in internal/fup → policies follow-up | WP05, WP07, WP08 | done (read-only) |
| w3-devstack-e2e: compose dev stack and first end-to-end acceptance run | WP11 | done (#50; 6 scenarios pass, lab gaps → follow-up 2) |
| w3-devstack-e2e follow-up 2: policies/usage settings in the lab, side-by-side stacks (prefix + port base), NetBird signal via gateway, plan catalog, forwarding baseline check | WP11 | done (#59; 7/12 scenarios pass, 5 blocked on the serving-node and device-binding preconditions) |
| w3-devstack-e2e follow-up 3: publish step in the scenarios, member admission as a native user peer, usage replay with the real device | WP04, WP05, WP09 | done (#68): publish and native member join driven through the product; publication deadlock and missing NetBird user mapping found |
| w3-devstack-e2e follow-up 4: one embedded Dex issuer for the portal and NetBird clients (D-052), publication blockers captured, boot release allowlist aligned, redacted evidence | WP04, WP05, WP09 | done (#81) |
| w3-devstack-e2e follow-up 5: real candidate releases through the D-053 health path (update-good passes), central threat-feed scenario, neutral image cache, re-run on main (9 pass / 6 blocked) | WP04, WP05, WP09 | done (#88) |
| w3-devstack-e2e follow-up 6 | Lab adopts the pgBackRest Postgres image with a physical PITR scenario, authenticated payment-truth hooks (fvpn-e2e-payment snapshots/events), exported PrepareServingBaseline for harness owners (real signed releases, API-activated serving policy, no SQL seeds), served built Portal with a browser-verified billing return, bounded control metrics probe; four runs 12 pass / 8 blocked on the D-057 group API (wire-nodes follow-up 11 landed after these runs) / 0 fail | done (#102) |
| w3-devstack-e2e follow-up 7 | real exit interfaces, persistent recovery config, API publication groups, host-local resolver, native admission, 2×2 scale smoke; 16 pass / 5 blocked / 0 fail ×3; gates: agent lease renewal on generation change, resolver_document reference in signed desired state | done (#118) |
| w2-policies-fup follow-up: reconcile embedded policy schemas with contracts/schemas/policies (D-044), fup uint64 overflow instead of clamp, D-029 audit conformance for policies and fup writers | WP07 | done (#46) |
| r2-updater-review re-run: confirm the updater follow-ups 1–2 against main (read-only) | WP09 | done (16/16 findings closed; regressions R-01..R-05 → updater follow-up 3) |
| w2-updater follow-up 3: review re-run regressions (revoked-current rescue path, artifact origin allowlist D-046, extract caps and ./ root D-047, resume semantics, nodelock start time) | WP09 | done (#47) |
| r2-updater-review confirmation: R-01..R-05 against main after updater follow-up 3 (read-only) | WP09 | done (5/5 closed; 2 minor → updater follow-up 4) |
| w2-updater follow-up 4: reject internal `..` segments at extract, one origin-allowlist rule for IP literals and DNS names, ForceQuery/`#` entries | WP09 | done (#54) |
| w2-updater follow-up 5: challenge-bound health attestation from the candidate agent for isolation, DNS and tunnel checks (D-053); first health report retained in the journal | WP09 | done (#78) |
| w2-updater follow-up 6 | D-053 adversarial review fixes: revocation rollback issues and binds a health challenge under the production adapter and reports previous_release_id truthfully; LastGood/state_schema committed in the stage-advance transaction (no crash-window lies); one private-directory validator for challenge/attestation files (symlink, shared-writable, owner, 0600, same-file, bounded reads); poll keeps the latest valid answer until the deadline; future GeneratedAt rejected; fixed check-reason vocabulary | done (#96) |
| w2-updater follow-up 7 | D-062 result evidence on the node side: pre-health failures carry health=null with the error code (D-062a), rolled-back results carry the restored release checks with the candidate as previous_release_id and the candidate checks in journal schema v2 candidate_health (D-062b), permanent 4xx rejections are recorded with status and API code while later results keep flushing in order (D-062d), read-only updatejournal.Snapshot for the agent (D-062e), D-034e empty allowlist pinned; fvpn-updater status shows the journal | done (#105) |
| w2-updater follow-up 8 | D-062b: candidate_health on rolled-back results (explicit null before any candidate check, omitted elsewhere); pending v2 rollbacks enriched atomically before flush; status shows the wire field | done (#116) |
| w2-fleet-update follow-up 2: core audit builders for every fleet/releases writer, system singleton actor, zero-anomaly read-back | WP09 | done (#42) |
| w2-fleet-update follow-up 3 | D-062 central acceptance: terminal evidence a–d, immutable rejection receipts, nullable observations (e), reconciled with wire-nodes f12 by one merge commit (union of stricter rules, both D-062 suites kept) | done (#117) |
| r2-control-review re-run: confirm control-core follow-ups 1–5, pki and fleet-update against main (read-only) | WP04, WP09 | done (16/16 closed; 2 wiring findings → nodes httpwire follow-up 3, usage replay follow-up 3) |
| w3-wire-nodes follow-up 3: real wiring behind the httpwire 503 gates (member policies in desired state, keyed agent receipts, transactional lease renewal) | WP04, WP05 | done (#49; publish gate remains until the exits seam lands) |
| w2-nodes-desiredstate follow-up 4: SetPublicationTx seam for all publication intents, one member rate algorithm (fup builder) | WP05, WP07 | done (#53) |
| w3-wire-nodes follow-up 4: route SetNodePublication through the exits seam, remove the last placeholder 503 | WP04, WP05 | done (#57) |
| w3-contracts-r4: UpdateTarget applied-policy evidence members (D-048), positive DomainRule.revision (D-049), writer survey re-run | WP02 | done (#58) |
| w3-contracts-r5: contracts revision 5 — resolver-document delivery (D-056), member groups and publication membership (D-057), signed egress probe, check detail, feed status, feed cadence default (D-058); 44 operations, info 1.1.0 | WP02 | done (#85) |
| w3-contracts-r6: contracts revision 5.1 — member_group audit resource type (D-059), API 1.1.1, stale operation counts corrected | WP02 | done (#87) |
| w3-contracts-r7: contracts revision 6 (API 1.2.0, additive, 44 operations) — typed NodeDetail.dns_policy_evidence (D-056), UpdateResult health/rolled_back/candidate_health evidence (D-062), ObservationReport.update from durable updater state (D-062e), rollback.allowed_release_ids empty = no restriction (D-034e), ResolverDocument example as canonical bytes, generated code regenerated (clears control-drift on main) | WP02 | done (#103) |
| w3-wire-nodes follow-up 5: worker execution of node.publication_changed via exits.ExecuteOperation, applied-policy evidence on update targets (D-048) | WP04, WP05, WP09 | done (#60; worker executes node.publication_changed through the exits executor with a real NetBird adapter; UpdateTarget policy evidence populated) |
| w3-wire-nodes follow-up 6: terminal `failed` for publication operations once the dispatcher's retry budget is exhausted | WP04, WP05 | done (#61) |
| w3-wire-nodes follow-up 7: worker members-domain jobs receive the desired-state compiler (member changes recompile desired state) | WP04, WP05 | done (#62) |
| w3-wire-nodes follow-up 8: retire the legacy withdraw handlers; node.revoke/node.retire through the publication executor | WP04, WP05, WP09 | done (#70) |
| w3-wire-nodes follow-up 9: baseline readiness without a management peer id (central side of the publication deadlock) | WP04, WP05, WP09 | done (#77) |
| w3-wire-nodes follow-up 10: management peer discovery through one-off enrollment correlation groups (D-055), fenced worker discovery with terminal conflicts | WP04, WP05, WP09 | done (#83) |
| w3-wire-nodes follow-up 11 | D-057 immutable exit groups (/admin/groups) and atomic publication membership with replay-safe hashes; D-056 reported DNS evidence projection and authorized resolver document reads over the DNS owner reader; D-058 sanitized check detail and egress_probe configuration; D-055 adapter documentation; the three composition stubs replaced by real handlers. Compiler seam for signed resolver references/probes prepared, not yet in the production compiler (follow-up 12) | done (#98) |
| w3-wire-nodes follow-up 12 | D-055 adversarial review fixes (13 proofs pass unchanged: fresh key/group reads, isolation, bounded retries, rollback of tentative bindings, durable withdrawal receipts), revocation before recordPeer withdraws the correlated peer with read-back and truthful audit, resolver-document issuance in the compiler transaction (D-056, closes r8 High), publication requires agent forwarding evidence (SECOPS-11, member_forwarding_unconfirmed), removed-group recompile (D-052 F-02), D-062 central acceptance | done (#109) |
| w4-at-members: acceptance tests for members, devices, entitlements and billing against the lab | WP10 | harness merged (#64): 75 live subchecks pass; AT01–AT22 blocked on member admission (setup-key peer has no NetBird user), the serving node and operator runs |
| w4-at-members follow-up 1 | Members harness re-run on main: native admission through the shared issuer (D-052), last-slot race, device isolation, usage receipts, billing permutations, D-054 feed readback; ten full ATs stay blocked on serving, forwarding and resolver-document delivery | done (#89) |
| w4-at-security-ops: acceptance tests for sessions, PKI, leases, recovery and operations against the lab | WP10 | harness merged (#63): AT28 passed live; AT29–AT32 blocked on the serving node, WAL archiving, the IdP identity fixture and operator runs |
| w4-at-security-ops follow-up 1 | Secops harness D-050 rerun on main: grants and native admission through product inputs, 149 live checks and 342 supporting tests pass, AT28 passed; AT29/AT31 blocked on a counted serving flow (watchdog recovery_required, zero accept rules; 20-way sign-in burst all 429), AT30 on physical PITR, AT32 on physical runs | done (#90) |
| w4-at-security-ops follow-up 2 | AT30 physical PITR on the lab stack (API-derived source, pgBackRest backup, restore-pitr.sh into a disposable copy, ACK 7→10 replay with duplicates, grant/audit boundary verified; recovery window 1.04 s, restore 5.3 s), AT28 + 19 supporting suites pass; AT29/AT31/AT32 blocked with owners named (serving flow → devstack f7, D-060 re-measure → next follow-up, operator runs) | done (#112) |
| w4-at-isolation: acceptance tests for publication, forwarding, isolation and leases against the lab (netlab) | WP10 | harness merged (#65): netlab mechanism cases pass for AT10/AT13; assembled-stack AT10–AT13 blocked (publication 422: provisioning applying, safety failed); gate semantics and AT12 capture in follow-up 1 |
| w4-at-isolation follow-up 1: non-zero exit while blocked; AT12 netlab capture; re-run on main | WP10 | done (#69) |
| w4-at-updates: acceptance tests for releases, update batches, rollback and recovery against the lab | WP10 | harness merged (#73): AT23–AT27 blocked on the serving baseline and on the updater health check that fails for a not-yet-serving node (no last-good); arm64 needs an operator run |
| w4-at-updates follow-up 1 | AT23–AT27 re-run against real signed 0.1.0/1.0.1/1.0.2 candidates through the D-053 health path: local update, tamper rejection, same-transaction recovery and last-good rollback proven; all five blocked on central reporting (observations carry null update fields → unconfirmed; failure results with health=null rejected 422 → target stays queued, needs_attention=false) | done (#95) |
| w4-at-usage: acceptance tests for usage metering, fair use and billing against the lab | WP10 | harness merged (#80): AT14–AT19 blocked on the serving baseline and on exit-group bootstrap (no admin path creates member groups or attaches exits; D-057 in contracts revision 5) |
| w4-handover: production deployment runbook, AT30 restore drill record, calibration guide, operator and member guides | WP12 | done (#55) |
| w4-handover follow-up 1: security-ops review documentation mismatches DOC01–DOC11 and DOC13 resolved (member guide, README, references) | WP12 | done (#72) |
| w4-handover follow-up 2 | self-verifying handover docs: tests/handover + mk/handover.mk check Markdown links, repo paths, make targets, fvpn-* binaries, SQL blocks executed against a migrated database, and a migration inventory derived from db/migrations (no literal counts in the owned docs); drift fixed against current main (endpoints and fields from OpenAPI 1.2.0, D-056/D-057/D-061 behaviour, PostgreSQL 17 casts) | done (#107) |
| w4-ops-backups: central PostgreSQL backups — pgBackRest WAL archiving, base backups, point-in-time restore drill in CI, post-restore checklist and runbook (SECOPS-05, AT30) | WP12 | done (#86) |
| w4-ops-backups follow-up 1 | PITR drill reentrant and race-free: per-run FVPN_BACKUP_DRILL_PREFIX (CI gha<run_id>-<attempt>; bare family name and fvpn-pg rejected), prefix-scoped cleanup, one readiness helper on the final ready line; two concurrent drills pass | done (#94) |
| w4-ops-backups follow-up 2 | D-025 review r5 fixes: the drill proves WAL replay past the backup stop (A,M restored, B excluded), restore refuses non-empty targets, live PGDATA and its descendants, full/diff backups gated on pgBackRest verify, verify reads real primary/repository archive ages and never passes without a primary, handover SQL executed against the restored schema, evidence writes opt-in (FVPN_BACKUP_DRILL_RECORD=1), adversarial proof script runs in CI | done (#100) |
| w4-ops-backups follow-up 3 | review_proof.sh passes shellcheck 0.10 (SC2317/SC2086/SC2012) so the Backup & PITR Drill workflow is green again; the orchestrator gate now runs shellcheck locally | done (#104) |
| w4-ops-backups follow-up 4 | restore-drill and ops runbooks verified against current main: migration inventory stated as the invariant the drill checks (no literal counts), real table/column names, real endpoints with contract headers and bodies, worker job names; review_proof.sh drops its fixed-migration-id predicate | done (#111) |
| w3-wire-policies-usage follow-up 3: durable idempotent replay for usage batches (stored first response, no in-memory map) | WP08 | done (#48) |
| w2-control-core follow-up 6: httpapi command-record accessor for keyed handlers, system actor constant moved to internal/audit | WP04 | done (#51) |
| w2-control-core follow-up 7 (hotfix): migrate the DNS publish fixture to audit.SystemActor and drop the deprecated releases alias (dns-lint SA1019 on main) | WP04 | done (#52) |
| w2-control-core follow-up 8: control loopback metrics listener; config example with all five domain files | WP04, WP09 | done (#71) |
| w2-control-core follow-up 10 | D-060 login burst limiter: browser sign-in routes no longer share the per-address anonymous burst (SECOPS-06 20 concurrent sign-ins from one address complete), dedicated login limiter with its own metric, r7 F-05/F-11 auth fixes with the review proofs, resolver-document GET exempt from the cache rule, config example loader test | done (#106) |
| w2-control-core follow-up 11 | D-052 F-04: one member per canonical (issuer, subject); client_scope is diagnostic; advisory lock + savepoint retry schema-independent; admin identities aligned | done (#114) |
| w3-dns-publish follow-up 2: D-029 audit conformance for publication and withdrawal writers | WP08 | done (#45) |
| w3-dns-publish follow-up 3: threat-feed content pipeline (D-054): bootstrap pin, cadence refresh with last-good, failure observations, Layer 3 rules published to the resolver document | WP08 | done (#79) |
| w3-dns-publish follow-up 4 | Feed cadence default and null attribution (D-058), threat_feed_status projected from real refresh observations, resolver-document delivery reads for the nodes owner (D-056), revision 5 evidence names | done (#93) |
| w3-dns-publish follow-up 5 | D-025 review r4 fixes (F-01–F-07): removal deltas checked against last-good and the approved snapshot with bounded additions and public-suffix wildcard boundaries; SSRF guard (TCP 443 only, special/zoned IPv6, redirects classified ssrf_refused); streaming bounded feed parser; activation operations carry dns_feed_refresh/dns_document steps and never succeed on a failed refresh or publish; lineage fan-out with generation fencing; atomic observations and outbox receipts; migration 00182; proofs kept in CI | done (#101) |
| w3-dns-publish follow-up 6 | contract revision 6 alignment: attribution null accepted and only non-null blanks rejected consistently across source metadata, policy validation and refresh (B9); canonical ResolverDocument vector pinned by recomputing Compile bytes and digest (ResolverDocumentRef and DnsPolicyEvidence examples bound to it) | done (#113) |
| w2-portal-wire: portal on generated contract types, auth session, CSRF, contract-example mocks | WP10 | done (#8) |

Later waves cover the control API, enrollment and PKI, node agent, usage ledger, entitlements and billing, fair-use policy, fleet updates, installer and images, end-to-end and fault-injection suites, backups and acceptance reporting.

## Needs real-world resources

These items cannot be proven in CI and need real resources:

- Official NetBird apps on Windows, macOS, iOS and Android (G1; AT01, AT09, AT21).
- Ubuntu 24.04 nodes on amd64 and arm64 on real networks, including double NAT (AT05, AT09, AT23).
- A payment provider sandbox account (AT03, AT04).
- A commercially licensed DNS threat feed (AT22).
- A production domain, TLS certificates and identity provider clients.
