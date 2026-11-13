# Freedom VPN delivery guide

**English** · [繁體中文](DELIVERY.md) · [Project overview](../README.md)

Freedom VPN builds on self-hosted NetBird and its official apps to manage members,
exit nodes and day-to-day operations. This consolidation preserves the existing
membership, metering, shaping and update features while focusing on integration,
necessary corrections and documentation.

**This remains a development and validation version, not a production-ready
release.** Existing implementation, recorded verification and outstanding device
or deployment prerequisites are separated below. A feature's presence does not
mean every platform and failure scenario has passed acceptance.

## Existing features and operating guides

| Use case | Existing implementation scope | Detailed guide |
|---|---|---|
| Use the service as a member | Sign-in, subscription entry points, device management, exit selection, usage and fair-use status | [Member guide](handover/member-guide.md) (Traditional Chinese and English) |
| Manage members and nodes | Membership eligibility, node enrollment and publication, access withdrawal and asynchronous operation results | [Operator guide](handover/operator-guide.md) |
| Manage VPN traffic | NetBird connection coordination, exit isolation, usage accounting, shaping and DNS policy | [Product specification](spec/PROJECT_SPEC.zh-TW.md) (Traditional Chinese) |
| Maintain the service | Node installation, signed updates, failure recovery, backup and restore | [Deployment](handover/deploy.md), [restore drill](handover/restore-drill.md) |
| Manage central signing keys and release trust | Initialize a vault (`fvpn-vault-init`), inspect it and export public trust (`fvpn-vault inspect`, `fvpn-vault export-trust`), rotate and revoke central keys (`fvpn-vault-keys rotate`, `fvpn-vault-keys revoke`), and create and check release trust updates (`fvpn-release trust-update`, `fvpn-release trust-verify`); an update is applied at the centre with `fvpn-vault-keys apply-release-update` as the [rotation runbook](pki/rotation.md#release-verification-trust) describes | [Vault initialization](pki/initialize.md), [vault inspection and export](pki/inspect.md), [central key commands](pki/keys.md), [rotation and restore](pki/rotation.md), [release process](updates/release-process.md) |

## Where to start

1. **Understand the full workflow:** use the isolated lab through the
   [development bring-up and teardown procedure](e2e/README.md#bring-up). Test
   identity, payment and policy settings are for validation and cannot serve
   directly as production configuration.
2. **Prepare a deployment:** follow the [deployment guide](handover/deploy.md) and
   [parameter calibration guide](handover/calibration.md) to confirm identity
   services, domains, payment, plans and node settings.
3. **Run daily operations:** administrators use the
   [operator guide](handover/operator-guide.md); members use the
   [member guide](handover/member-guide.md). Follow the
   [restore drill](handover/restore-drill.md) for backups and failure recovery.
4. **Maintain the code:** read the [engineering contract](../AGENTS.md),
   [API contracts](contracts/README.md) and relevant component documentation first.

## Handover guide revisions

[Follow-up 52](handover/follow-up-52.md) corrects deployment, calibration, restore, operator and backup statements at `6972be03`: enrollment and release-policy fields, the two-resolver resilience recommendation (one is accepted without redundancy), fair-use acknowledgement versus current liveness, the policy event pipeline, HTTP helper names, a repeated recovery-gap notice, node checks and token-file selection, enforcement names and reserved conntrack batches, revoke-step eligibility and public/audit diagnostics, updater space and attempt bounds and rollback health, the member-usage read-back route, configuration defaults and bounded LRU scanning, current incarnation/discovery/probe/schema names, provisional health and lease timing qualifications, fence audit fields and separate peer/identity denial tracking with resource fallbacks, rejected/accepted update-result projections and omitted/null error-code receipt differences, revocation address filters, stale-observation grace, the public-key digest encoding and baseline attempts versus confirmed isolation in deployment and operator guidance, Apply/dependency acknowledgement gates and separate meter-path failures, and the linux/amd64 backup migrator. The dated restore record stays unchanged; the cut records the `UploadOnce` / `Uploader.Once` erratum. This cut adds no procedure or deployment evidence; carried gaps remain as follow-up 51 recorded them.

[Follow-up 51](handover/follow-up-51.md) corrects deployment, calibration, restore-drill and backup statements at `66de6a24`. It records staging authority and grants, installer/DNS behavior, actual configuration and retention bounds, and bounded recovery/reconciliation. The second review adds control-before-worker recovery ordering with its remaining authorization exposure, grant withdrawal, explicit recovery credentials, repository-environment gaps, updater space accounting and credential-only SIGHUP reload. Class review passes looked for credential, reload, process, name, cleanup and gap-notice errors and corrected the cases recorded in the cut. A third review and scripted inventories add repository-environment and retention notices, the node response field, conntrack batch accounting and quoted Go names. These passes are review records, not proof that every statement or dependency has been found. Production procedures and product support absent from the repository remain carried work; no new deployment or acceptance evidence is claimed.

[Follow-up 50](handover/follow-up-50.md) retains the earlier revisions to operator key inspection (§1.2), central rotation/revocation (§8.1), release trust updates (§8.2), and member shadow mode (§5.3, both languages).

Living guides can receive scoped corrections. Each cut freezes its pre-revision inputs and checks every other byte; dated snapshots retain their scope. Current procedures are in [vault initialization](pki/initialize.md), [inspection and export](pki/inspect.md), [central key commands](pki/keys.md), [rotation and restore](pki/rotation.md), [release process](updates/release-process.md), [node installation](node/install.md), the [API contract](../contracts/openapi.yaml) and [portal documentation](portal/README.md).

## Verification scope and known limits

The [merge record](ROADMAP.md) tracks current implementation;
[follow-up 52](handover/follow-up-52.md) preserves the #527
delivery checkpoint at `6972be03`. The earlier [follow-up 51](handover/follow-up-51.md)
retains #524–#526. Later results have their own acceptance records.
Unit and integration tests, Linux lab
tests, actual app devices and production deployments each establish different
evidence. This guide does not announce a production release or replace unfinished
acceptance with documentation.

Retained node verification results (`serving_read`) now flow from node reports
through Central storage and projection to the admin interface and diagnostic
exports ([#492](https://github.com/TED-H-com/Freedom-VPN/pull/492),
[#493](https://github.com/TED-H-com/Freedom-VPN/pull/493),
[#494](https://github.com/TED-H-com/Freedom-VPN/pull/494)). They preserve the original
read time and result. A retained read alone does not authorize current forwarding;
the existing safety checks and lease decisions still apply.

| Area | Current boundary and acceptance reference |
|---|---|
| Official apps | The [NetBird version matrix](netbird/VERSIONS.md) lists pinned components and candidate apps. Windows, macOS, iOS and Android still need real-device acceptance. |
| Exit and member isolation | [D164 acceptance](e2e/at/isolation/d164-acceptance.md) records the complete lab and assembled verification on 2026-11-06: 27 rows passed, 2 remain blocked, and F17-01 is closed. The IPv6 platform and full fault-transition matrices remain unverified; [historical isolation records](e2e/at/isolation/summary.md) retain earlier failure verdicts. |
| Member connectivity | [Member acceptance](e2e/at/members/README.md) records upstream connection and network limitations. Server-side isolation does not provide an OS kill switch on member devices. |
| Updates and hardware | [Update acceptance](e2e/at/updates/summary.md) and [security operations acceptance](e2e/at/secops/summary.md) record gaps including physical arm64, candidate-hardware capacity and a guest with an independent clock. Cross-compilation cannot replace hardware measurements. |
| Production operations | Production IdP/MFA, payment provider, domains/TLS, DNS lists and operating parameters still need configuration and verification. Test defaults do not define production policy. |

Earlier consolidation records remain available:
[isolation acceptance corrections](e2e/at/isolation/followup27-2026-11-05.md)
preserve failed results and evidence; [network lab corrections](poc/G2-followup37-2026-11-05.md)
provide scoped local verification of packet attribution and blocking verdicts.
The latter does not rewrite the original scheduled failure, and neither record
establishes completed final integration verification for the entire system.

## Preserved extension points

Existing modules, APIs, data and historical evidence remain available. Later work
should follow concrete requirements or demonstrated defects; the existence of a
design proposal does not automatically add implementation scope.

NetBird API client reuse, other architecture changes and advanced
diagnostics remain for later evaluation. Necessary completion fixes are
handled according to actual issues.

The [delivery history and known issues](handover/README.md) preserve the full
context, and the [development roadmap and merge record](ROADMAP.md) track individual
deliveries. Results and limits stated at each acceptance entry point remain
authoritative.
