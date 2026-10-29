# Freedom VPN

Freedom VPN is a member-based, multi-region VPN platform for a small paid community, starting at about 20 members. Every exit node runs the same node package and joins one central control plane, whether it is a pre-installed box, a cloud Ubuntu VM or any other Ubuntu host.

**Status:** early development. Nothing here is production-ready. Progress is tracked in [docs/ROADMAP.md](docs/ROADMAP.md).

## What it does

- Self-hosted [NetBird](https://netbird.io) (WireGuard) provides connectivity. Members use the official NetBird apps.
- Nodes join with a one-time enrollment code, check themselves, and wait until an admin classifies and publishes them.
- Every node enforces fixed Internet-only isolation. Members can never reach node hosts, upstream LANs, cloud metadata or each other.
- A cross-node usage ledger uses idempotent, replayable events. Fair-use shaping slows heavy users gradually, per member.
- DNS-level malware and phishing protection supports custom block and allow rules.
- One admin action updates every node with signed, immutable releases, including offline catch-up and rollback.

## Documents

- Product and engineering spec (Traditional Chinese): [docs/spec/PROJECT_SPEC.zh-TW.md](docs/spec/PROJECT_SPEC.zh-TW.md)
- Engineering contract for contributors and AI agents: [AGENTS.md](AGENTS.md)
- Roadmap and task board: [docs/ROADMAP.md](docs/ROADMAP.md)

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
