# Handover follow-up 54 — guide and engineering document corrections

This cut records **four merged pull requests, in merge order: #533, #534, #535 and #536**, through `2f5159f377cb807cd4ed203ef6e4d1785f020ba9`. The cutoff is the roadmap commit after #536, whose squash is `e965ea2b2f757afd700dac21c96b9fc8b2c56fcd`. [Follow-up 53](follow-up-53.md) was itself merged as #534 and is therefore included here.

The linked pull requests are the sources for these summaries. This cut does not re-verify them or claim a new run, measurement, deployment, restore or CI result.

## Merged deliveries

| PR | Merged title | Merge commit | Head commit | Effect and boundary |
|---|---|---|---|---|
| [#533](https://github.com/TED-H-com/Freedom-VPN/pull/533) | db: every pool connection scans timestamps in UTC, so API times end in Z on a host in any time zone | `6321c3ec13dd0ae3c88efde2985a651d75d57140` | `b1862238836bae9987d66380a4dc0de0b818c923` | PostgreSQL pool connections scan timestamps in UTC, allowing API timestamps to use UTC independently of the host time zone. |
| [#534](https://github.com/TED-H-com/Freedom-VPN/pull/534) | handover: follow-up 53 — cut #528–#532, and correct the deployment, calibration, operator, member, database, restore-drill and backup guides against the code | `abd0d850b43e623f4f68098127761bb449ed0a8f` | `1d45500bb7866d8b76df7ca664834b7e5acea84b` | Publishes follow-up 53 and corrects guide statements against the code. Its frozen inputs and bounded edits preserve earlier claims. |
| [#535](https://github.com/TED-H-com/Freedom-VPN/pull/535) | node: run the same-boot adoption test and 13 other positive adoption tests on virtual time | `8b7b15a380077da72e868a7e2f0de30587e341bc` | `741ffd40ea26df019a89beb0c7ede9e97bf50172` | Node adoption tests use virtual time for positive adoption scenarios, reducing their dependence on wall-clock waits. |
| [#536](https://github.com/TED-H-com/Freedom-VPN/pull/536) | worker: size the database pool from worker_concurrency and refuse a setting above 11 | `e965ea2b2f757afd700dac21c96b9fc8b2c56fcd` | `7ede0c43c217d00672e21f10e366f40786e7d8e5` | The worker sizes its PostgreSQL pool from its concurrency and registered connection peaks, and rejects a budget above the pool limit at startup. |

## Guide revisions at this cut

The deployment, operator, member, database, operations and contracts guides and the node and network-lab engineering documents are corrected against the code at this cutoff. The deployment guide also describes the worker pool budget and PostgreSQL connection planning. The [delivery guide](../DELIVERY.en.md#handover-guide-revisions) retains the earlier revisions.

## Dated-record errata

These passages remain unchanged:

- `docs/handover/follow-up-51.md`, [“Existing procedures linked and carried gaps”](follow-up-51.md#existing-procedures-linked-and-carried-gaps), deployment §9.1: agent datapath interface names must start with `fv`; the installer creates dummy links without addresses or routes, and real interface, address and route provisioning remains a documented gap. See [current node prerequisites](deploy.md#91-node-host-prerequisites-directory-layout-and-configuration-agentsmd-4-nodeenforceapplygo-nodeenforceexecutorgo).
- `docs/handover/follow-up-51.md`, [“Existing procedures linked and carried gaps”](follow-up-51.md#existing-procedures-linked-and-carried-gaps), restore startup: restricted recovery startup remains absent; a host forwarding fence used for staging is not a product recovery mode. Control can issue leases from restored authorization before newer intent is reapplied; see [current restore ordering and exposure](restore-drill.md#phase-3-rotate-ledger_epoch-and-start-central-services-173-steps-26).
- `docs/handover/member-guide.md`, “3.2 AT09、AT29”, historical follow-up 7 qualification in zh-TW: AT30 first passed in [follow-up 9](../e2e/at/secops/history/follow-up-9/AT30.md), then passed again in [follow-up 10](../e2e/at/secops/history/follow-up-10/AT30.md). The [current AT30 record](../e2e/at/secops/AT30.md#at30-restore--follow-up-13) reports a later pass.
- `docs/handover/member-guide.md`, “3.2 AT09, AT29”, historical follow-up 7 qualification in English: AT30 first passed in [follow-up 9](../e2e/at/secops/history/follow-up-9/AT30.md), then passed again in [follow-up 10](../e2e/at/secops/history/follow-up-10/AT30.md). The [current AT30 record](../e2e/at/secops/AT30.md#at30-restore--follow-up-13) reports a later pass.

## Carried gaps and verification boundary

Carried gaps remain as [follow-up 53 recorded them](follow-up-53.md#carried-gaps-and-verification-boundary). This cut supplies no missing procedure and closes no product or acceptance gap.
