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

## Verification scope and known limits

The [current engineering record](handover/follow-up-49.md) collects merged fixes,
verification sources and carried issues. Unit and integration tests, Linux lab
tests, actual app devices and production deployments each establish different
evidence. This guide does not announce a production release or replace unfinished
acceptance with documentation.

| Area | Current boundary and acceptance reference |
|---|---|
| Official apps | The [NetBird version matrix](netbird/VERSIONS.md) lists pinned components and candidate apps. Windows, macOS, iOS and Android still need real-device acceptance. |
| Exit and member isolation | [Isolation acceptance](e2e/at/isolation/summary.md) preserves incomplete scenarios and the latest attempt results. Historical failures retain their original verdicts. |
| Member connectivity | [Member acceptance](e2e/at/members/README.md) records upstream connection and network limitations. Server-side isolation does not provide an OS kill switch on member devices. |
| Updates and hardware | [Update acceptance](e2e/at/updates/summary.md) and [security operations acceptance](e2e/at/secops/summary.md) record gaps including physical arm64, candidate-hardware capacity and a guest with an independent clock. Cross-compilation cannot replace hardware measurements. |
| Production operations | Production IdP/MFA, payment provider, domains/TLS, DNS lists and operating parameters still need configuration and verification. Test defaults do not define production policy. |

This consolidation also has two closing records:
[isolation acceptance corrections](e2e/at/isolation/followup27-2026-11-05.md)
preserve failed results and evidence; [network lab corrections](poc/G2-followup37-2026-11-05.md)
provide scoped local verification of packet attribution and blocking verdicts.
The latter does not rewrite the original scheduled failure, and neither record
establishes completed final integration verification for the entire system.

## Preserved extension points

Existing modules, APIs, data and historical evidence remain available. Later work
should follow concrete requirements or demonstrated defects; the existence of a
design proposal does not automatically add implementation scope.

The optional `serving_read` contract has been added. Central storage and display,
diagnostic reads and node data production are not all enabled. These interfaces
remain available for later extension; a field's presence does not mean the feature
is live. NetBird API client reuse, other architecture changes and advanced
diagnostics also remain for later evaluation. Necessary completion fixes are
handled according to actual issues.

The [delivery history and known issues](handover/README.md) preserve the full
context, and the [development roadmap and merge record](ROADMAP.md) track individual
deliveries. Results and limits stated at each acceptance entry point remain
authoritative.
