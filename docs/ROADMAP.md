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
| w2-contracts-r2: review fixes, agent bootstrap endpoint, reinstall target, UpdateTarget flags, ledger-epoch purpose | WP02 | delivered; follow-up 1 adapts merged consumers to presence types (41 endpoints) |
| w2-control-core: HTTP framework, auth, idempotency, ETag, operations, audit, worker | WP04, WP09 | done (#13) |
| w2-pki-enroll: node CA, enrollment, CSR issuance, revocation, signing service | WP04 | done (#11) |
| w2-members-billing: members, entitlement, devices, billing adapter and webhooks | WP06 | done (#9) |
| w2-nodes-desiredstate: node registry, desired state compiler, serving leases, exits | WP04, WP05 | done (#17) |
| w2-usage-central: usage ledger ingest, checkpoints, projection, rollups | WP07 | done (#15) |
| w2-policies-fup: plan, fair-use, DNS and release-support policies, FUP tiers | WP07, WP08 | done (#12) |
| w2-fleet-update: releases, manifests, update batches, serving-version policy | WP09 | done (#10) |
| w2-updater: node updater, update journal, release packaging | WP09 | done (#18) |
| w2-agent-core: node agent, watchdog, control client, time anchor | WP05 | queued (after contracts-r2 consumer adaptation) |
| w2-usage-agent: node meter, usage store, upload with contiguous ACK | WP07 | queued (after contracts-r2 consumer adaptation) |
| w2-db-fixes-2: isolated migration CLI test, contract-conformant retention audit rows, one-time incarnation binding | WP03 | done (#14) |
| w2-portal-wire: portal on generated contract types, auth session, CSRF, contract-example mocks | WP10 | done (#8) |

Later waves cover the control API, enrollment and PKI, node agent, usage ledger, entitlements and billing, fair-use policy, fleet updates, installer and images, end-to-end and fault-injection suites, backups and acceptance reporting.

## Needs real-world resources

These items cannot be proven in CI and need real resources:

- Official NetBird apps on Windows, macOS, iOS and Android (G1; AT01, AT09, AT21).
- Ubuntu 24.04 nodes on amd64 and arm64 on real networks, including double NAT (AT05, AT09, AT23).
- A payment provider sandbox account (AT03, AT04).
- A commercially licensed DNS threat feed (AT22).
- A production domain, TLS certificates and identity provider clients.
