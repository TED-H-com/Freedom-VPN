# Freedom VPN

A member-based VPN platform built on self-hosted NetBird and its official apps.
One control plane manages members and Ubuntu exit nodes across locations.

**Status: development and validation. This is not a production-ready release.**
The implementation, verification boundaries and remaining deployment prerequisites
are summarized in the [交付導覽 / Delivery guide](docs/DELIVERY.md).

## What is included

- Member and administrator portals, subscription integration, device management,
  node enrollment and publication, and access withdrawal.
- NetBird connection coordination and manual exit selection through official apps.
- Server-side Internet-only isolation, usage accounting, fair-use shaping and
  DNS/domain policies.
- Signed node updates, recovery and rollback, with deployment and restore runbooks.

These describe the existing implementation scope. Recorded acceptance results and
remaining gaps are linked from the delivery guide; Linux lab results do not
establish Windows, macOS, Android or iOS device compatibility.

## Start here

| Task | Guide |
|---|---|
| Understand this delivery and its limits | [交付導覽](docs/DELIVERY.md) |
| Run the isolated development stack | [Dev-stack bring-up and teardown](docs/e2e/README.md#bring-up) |
| Prepare a deployment | [Deployment](docs/handover/deploy.md) and [configuration calibration](docs/handover/calibration.md) |
| Operate members and nodes | [Operator guide](docs/handover/operator-guide.md) |
| Connect a member device | [Member guide](docs/handover/member-guide.md) |
| Back up and recover | [Restore drill](docs/handover/restore-drill.md) |

The development stack uses test identities, payment fixtures and policy settings.
Production identity, payment and network configuration require separate setup.

## Architecture and maintenance

NetBird supplies the overlay and existing client apps. Freedom VPN adds membership,
accounting, policy enforcement and node lifecycle management through upstream APIs
and the CLI. The [version matrix](docs/netbird/VERSIONS.md) records pinned components
and the client platforms still awaiting real-device validation.

| Location | Responsibility |
|---|---|
| `apps/control`, `apps/worker` | Central API and background work |
| `apps/portal` | Administrator and member interface |
| `node/agent`, `node/updater` | Node enforcement, accounting and updates |
| `contracts`, `db/migrations` | API definitions and persistent data |
| `deploy`, `packaging`, `tests` | Deployment, release tooling and verification |

Read the [product specification](docs/spec/PROJECT_SPEC.zh-TW.md),
[engineering contract](AGENTS.md) and [API documentation](docs/contracts/README.md)
before changing behavior. This consolidation preserves existing functionality and
extension points; optional diagnostic and architecture work remains later work.

## Evidence and history

**Current cut:** [follow-up 49](docs/handover/follow-up-49.md), #482–#485 at aaffdc7f.
**Previous cut:** [follow-up 48](docs/handover/follow-up-48.md), #478–#481 at b921c75f.
The [findings and decisions](docs/handover/README.md), [acceptance records](docs/e2e/README.md)
and [roadmap](docs/ROADMAP.md) retain the detailed engineering history.

## License

Apache License 2.0; see [LICENSE](LICENSE). Third-party components, including
NetBird, retain their own licenses, recorded in the [version matrix](docs/netbird/VERSIONS.md).
