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
| w2-usage-central follow-up 3: review fixes (freeze fence, gap and replay scoping, overflow, ledger-epoch purpose) | WP07 | done (#23) |
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
| w3-wire-policies-usage: policies and Fair-Use operations, fup worker, usage mount with PKI signer, audit reader | WP07, WP08 | in progress |
| w3-dns-publish: DNS policy compiler, node applier with evidence, lab scenario | WP08 | done (#30) |
| w3-wire-nodes follow-up 1: adopt the composition seam | WP04, WP05, WP09 | done (#32) |
| w3-dns-publish follow-up 1: adopt the worker seam, split document types from the store | WP08 | done (#30) |
| w2-members-billing follow-up 1: review fixes (operations and audit conformance, exit-group eligibility reader, synchronous checkout, patch status) | WP06 | done (#29) |
| w2-control-core follow-up 1: review fixes (conformance helpers, role cookies, limiter split, agent receipts) | WP04 | in progress |
| w2-updater follow-up 1: review fixes (symlink escape, forwarding gate, persistent-data check, nodelock, downloads) | WP09 | done (#33) |
| w1-netlab follow-up 1: review fixes (HTB class cleanup, deadline fail-closed, reset detection, IPv4 validation, evidence dir) | WP02, WP05 | done (#35) |
| w1-dnspolicy follow-up 1: compiled bundle round-trip, source fetch SSRF | WP08 | done (#26) |
| w3-wire-policies-usage follow-up 1: adopt the composition seam over the ledger fixes, single-format audit reader, FUP cap removal | WP07, WP08 | in progress |
| w2-db-fixes-2 follow-up 1: canonical purge_retained audit row (D-035), accepted domain relaxations documented (D-033) | WP03 | done (#28) |
| w2-agent-core follow-up 1: usage ledger integration, required host reserve, fake control usage endpoints | WP05, WP07 | done (#36) |
| w2-control-core follow-up 2: members fixture on role cookies, module-wide integration suite | WP02 | in progress |
| w3-wire-members follow-up 1: retire legacy session job, composition guide refresh | WP02 | done (#31) |
| w3-contracts-r3: additive enums (node.enroll, device_authorize, grant_revoke, admin_identity), node-reported codes, revoked_release_ids, policy_generation, checkout 201-only, role selector, policy schema promotion | WP02 | done (#34) | WP01 | in progress |
| w2-nodes-desiredstate follow-up 3: entitlement serving readers (D-030), dns_generation from publication heads, RenewTx, audit conformance | WP04, WP05 | done (#39) |
| w3-wire-nodes follow-up 2: D-039 fixture isolation (nodestest), dual-stack resolver configuration | WP04, WP05 | done (#32) |
| w2-control-core follow-up 3: terminal dead-lettering with fleet singleton fallback (D-040); follow-up 4: members dead-letter test on D-029 | WP04 | in progress (follow-up 5 adds revision 3 adoption; lands with fleet-update follow-up 2 because the stricter audit writer rejects the legacy fleet actor) |
| w1-netlab follow-up 2: idle counters are not meter-epoch changes (D-041), exported DetectMeterResets | WP02, WP05 | done (#35) |
| w2-fleet-update follow-up 1: review fixes (SubmitResult generation compare, signed intent issued_at, node_name, Register validation order, applied policy_generation) | WP09 | done (#37) |
| w2-pki-enroll follow-up 1: review fixes (bootstrap take, node_id binding, trust bundle filter), migration 00111, node.enroll adoption | WP04 | in progress |
| w3-wire-members follow-up 2: adopt revision 3 kinds (member.device_authorize, member.grant_revoke), D-029 audit conformance | WP06 | done (#38) |
| w2-updater follow-up 2: enforce signed revoked_release_ids, ARTIFACT_INVALID reporting, applied policy_generation (D-043) | WP09 | in progress |
| w3-wire-policies-usage follow-up 2: fail-closed signing and explicit policies/usage settings (D-045), fixture helpers | WP07, WP08 | in progress |
| w2-agent-core follow-up 2: revision 3 node codes, external meter resets via DetectMeterResets (D-041), per-event usage conflicts | WP05, WP07 | in progress |
| w1-deps-bump follow-up 1: golang.org/x/crypto GO-2026-5932 | WP01 | done (no change: GO-2026-5932 is the unused x/crypto/openpgp packages, no fixed release exists, govulncheck symbol scan clean) |
| w3-portal-live: portal on the real control plane, Playwright end-to-end | WP10 | in progress |
| r2-ledger-review confirmation pass on main 3224345: probes re-run read-only, 11 of 12 findings closed; finding 9 (uint64 clamp) still open in internal/fup → policies follow-up | WP05, WP07, WP08 | done (read-only) |
| w3-devstack-e2e: compose dev stack and first end-to-end acceptance run | WP11 | in progress |
| w2-portal-wire: portal on generated contract types, auth session, CSRF, contract-example mocks | WP10 | done (#8) |

Later waves cover the control API, enrollment and PKI, node agent, usage ledger, entitlements and billing, fair-use policy, fleet updates, installer and images, end-to-end and fault-injection suites, backups and acceptance reporting.

## Needs real-world resources

These items cannot be proven in CI and need real resources:

- Official NetBird apps on Windows, macOS, iOS and Android (G1; AT01, AT09, AT21).
- Ubuntu 24.04 nodes on amd64 and arm64 on real networks, including double NAT (AT05, AT09, AT23).
- A payment provider sandbox account (AT03, AT04).
- A commercially licensed DNS threat feed (AT22).
- A production domain, TLS certificates and identity provider clients.
