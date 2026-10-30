# Freedom VPN

Freedom VPN is a member-based, multi-region VPN platform for a small paid community, starting at about 20 members. Every exit node runs the same node package and joins one central control plane, whether it is a pre-installed box, a cloud Ubuntu VM or any other Ubuntu host.

**Status:** early development. Nothing here is production-ready. Progress and remaining blockers are tracked in [docs/ROADMAP.md](docs/ROADMAP.md), the dev-stack run records and shared lab hooks in [docs/e2e/README.md](docs/e2e/README.md) and [docs/e2e/RUN.md](docs/e2e/RUN.md), and the acceptance matrix ([docs/e2e/at/members/README.md](docs/e2e/at/members/README.md), [docs/e2e/at/isolation/README.md](docs/e2e/at/isolation/README.md), [docs/e2e/at/secops/README.md](docs/e2e/at/secops/README.md)).

## What it does

Intended v1 capabilities (subject to the open acceptance items in [docs/e2e/at/members/README.md](docs/e2e/at/members/README.md), [docs/e2e/at/isolation/README.md](docs/e2e/at/isolation/README.md), and [docs/e2e/at/secops/README.md](docs/e2e/at/secops/README.md)):

- **Overlay connectivity (intended, `§7`, `AT01`, `AT09`):** Self-hosted [NetBird](https://netbird.io) (WireGuard) is used for tunnel coordination with candidate versions of the official NetBird apps ([docs/netbird/VERSIONS.md](docs/netbird/VERSIONS.md); native Windows, macOS, iOS, and Android device acceptance remains `needs operator run`).
- **Unified node enrollment and publication (intended, `§5`–`§6`, `AT05`–`AT08`):** Nodes join with a one-time enrollment token, run readiness checks, and stay unpublished until an admin classifies and publishes them. On the assembled dev stack, enrollment and peer correlation (`D-055`) pass ([docs/e2e/runs/2026-10-30/enroll-bootstrap.json](docs/e2e/runs/2026-10-30/enroll-bootstrap.json)), and `D-057` exit-group management (`GET|POST /api/v1/admin/groups`) and `PublicationRequest.group_ids` persistence in `SetNodePublication` are now wired on `main` ([docs/nodes/WIRING.md](docs/nodes/WIRING.md); the dated run in [docs/e2e/runs/2026-10-30/publish.json](docs/e2e/runs/2026-10-30/publish.json) was recorded prior to `w3-wire-nodes` follow-up 11 merging).
- **Server-side Internet-only isolation (intended, `§1`, `§8.1`–`§8.3`, `AT10`–`AT13`):** On the supported server-side exit forwarding path (`inet fvpn`, IPv4 forwarding with member IPv6 blocked in v1), rules are designed to prevent traffic inside the VPN tunnel from reaching exit-node host services, upstream LANs, cloud metadata, or other members. This server-side filter does not control a member endpoint's local Wi-Fi/LAN routing and does not provide a client-side OS kill switch (`§7.3`). Assembled dev-stack forwarding and lease-cutoff acceptance remain `blocked` ([docs/e2e/runs/2026-10-30/forwarding.json](docs/e2e/runs/2026-10-30/forwarding.json), [docs/e2e/runs/2026-10-30/lease-expiry.json](docs/e2e/runs/2026-10-30/lease-expiry.json)).
- **Cross-node usage ledger and Fair-Use shaping (intended, `§10`–`§11`, `AT14`–`AT19`):** The usage pipeline is designed around idempotent, replayable `forward_admission_v1` events and per-member, per-exit HTB shaping tiers within each node's shared capacity ceiling. End-to-end admitted flow byte counting on the assembled dev stack remains `blocked` ([docs/e2e/runs/2026-10-30/usage.json](docs/e2e/runs/2026-10-30/usage.json)), while provider-truth payment hooks, physical PITR backup verification, and loopback control metrics pass ([docs/e2e/runs/2026-10-30/billing.json](docs/e2e/runs/2026-10-30/billing.json), [docs/e2e/runs/2026-10-30/backup.json](docs/e2e/runs/2026-10-30/backup.json), [docs/e2e/runs/2026-10-30/control-metrics.json](docs/e2e/runs/2026-10-30/control-metrics.json)).
- **Domain-level DNS filtering (intended, `§12`, `AT20`–`AT22`):** Controlled DNS resolvers apply malware/phishing lists and custom block/allow rules without TLS interception (`§1.1`, `§12.1`); unmanaged client DoH or Android Private DNS settings can bypass VPN-assigned resolvers unless disabled (`§12.6`). Central threat-feed bootstrap and resolver document compilation (`D-054`, `00182_dns_feed_review.sql`) pass on the dev stack ([docs/e2e/runs/2026-10-30/threat-feed-central.json](docs/e2e/runs/2026-10-30/threat-feed-central.json)), and authenticated `GET /api/v1/agent/resolver-documents/{document_id}` delivery (`D-056`) plus node-side AdGuard Home admin configuration (`D-061`, [docs/node/README.md](docs/node/README.md)) are wired on `main`, while the desired-state compiler `resolver_document` input-hook seam ([docs/nodes/WIRING.md](docs/nodes/WIRING.md)) and end-to-end node delivery acceptance ([docs/e2e/runs/2026-10-30/threat-feed-node.json](docs/e2e/runs/2026-10-30/threat-feed-node.json)) remain open.
- **Signed fleet updates and journaled rollback (intended, `§13`, `AT23`–`AT27`):** One admin batch action targets all non-revoked, non-retired nodes with signed, immutable releases, including offline catch-up and rollback. On the assembled dev stack, candidate update health attestation (`D-053`) and tampered-archive rejection pass ([docs/e2e/runs/2026-10-30/update-good.json](docs/e2e/runs/2026-10-30/update-good.json), [docs/e2e/runs/2026-10-30/update-bad.json](docs/e2e/runs/2026-10-30/update-bad.json)), while live rollback and physical `arm64` execution remain unmeasured ([docs/e2e/README.md](docs/e2e/README.md)).

## Documents

- Product and engineering spec (Traditional Chinese): [docs/spec/PROJECT_SPEC.zh-TW.md](docs/spec/PROJECT_SPEC.zh-TW.md)
- Engineering contract for contributors and AI agents: [AGENTS.md](AGENTS.md)
- Roadmap and task board: [docs/ROADMAP.md](docs/ROADMAP.md)
- API contract conventions and endpoint table: [docs/contracts/README.md](docs/contracts/README.md)
- Database foundation: [docs/db/README.md](docs/db/README.md)
- Deployment, operations, calibration and member guides: [docs/handover/README.md](docs/handover/README.md)

## Repository layout

| Path | Contents |
|---|---|
| `apps/control`, `apps/worker` | Central API and background worker (Go) |
| `apps/portal` | Admin and Member web app (TypeScript, React) |
| `node/agent`, `node/updater` | Node agent and updater (Go, amd64 and arm64) |
| `contracts` | OpenAPI, JSON Schemas, examples, golden vectors |
| `db/migrations` | PostgreSQL schema |
| `deploy`, `packaging` | Dev environments, installer, release tooling |
| `tests` | Network lab, end-to-end and fault-injection suites |

Development is AI-assisted. Every change follows the rules in [AGENTS.md](AGENTS.md) and is reviewed before it reaches `main`.

## License

Apache License 2.0, see [LICENSE](LICENSE). Third-party components such as NetBird keep their own licenses.
