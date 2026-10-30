# Operations Runbooks Index (`docs/ops/`)

Operational runbooks, deployment guides, parameter calibration references, and disaster recovery checklists are maintained under [`docs/handover/`](../handover/README.md) alongside domain-specific operational references:

## Hand-Over & Production Runbooks (`docs/handover/`)

- [**Production Deployment Runbook (`../handover/deploy.md`)**](../handover/deploy.md) — PostgreSQL 17 + WAL archiving, TLS & mTLS listeners, OIDC + MFA, NetBird CE `0.82.0`, AdGuard Home `v0.107.59`, release origin, 5 domain config files, PKI & signing key ceremonies, and first node enrollment.
- [**Central PostgreSQL Backups, PITR & Post-Restore Reconciliation (`backups.md`)**](backups.md) — Pinned `pgBackRest 2.60.0` configuration, continuous WAL archiving (`archive_timeout = 300s`), systemd schedules, repository encryption & vault separation (`§16.2`, `§17.2`), monitoring queries/log lines, `deploy/backup/restore-pitr.sh` / `deploy/backup/verify.sh` / `deploy/backup/post-restore-checklist.sql`, ordered post-restore reconciliation protocol (`§17.3`–`§17.4`), and automated CI drill evidence ([`backups/`](backups/)).
- [**Backup and Restore Drill Record (`../handover/restore-drill.md`)**](../handover/restore-drill.md) — `AT30` point-in-time recovery checklist, Wave 4 measured RPO/RTO evidence, `fvpn-usage-admin` ledger freeze/recovery/replay, and Stripe + NetBird state reconciliation.
- [**Parameter Calibration Guide (`../handover/calibration.md`)**](../handover/calibration.md) — Business and engineering parameter matrix (`none`, `test only`, `proposed`), spec constraints, and decision ownership (`Ted` vs. `core team`).
- [**Operator Guide (`../handover/operator-guide.md`)**](../handover/operator-guide.md) — Daily/weekly checklists, reading operations, audit trails, and current OpenAPI contract status, node lifecycle (`D-055`, `D-057`, `D-058`, `D-061`), fleet update batches (`needs_attention` / `unconfirmed`) and `D-053` health attestation with separate candidate/restored health, threat-feed and resolver-document troubleshooting (`D-054`, `D-056`), ledger freeze/adjustments, member suspension, identity continuity and duplicate pre-checks, login limiting, and key rotation.
- [**Member Guide (`../handover/member-guide.md`)**](../handover/member-guide.md) — Bilingual (`zh-TW` / `en`) guide covering candidate client versions (`needs operator run`) and automatic central device authorization (`D-052`) for Windows, macOS, iOS, and Android, manual verification scripts (`AT01`, `AT09`, `AT21`, `AT29`), device limits & revocation (`AT02`, `AT04`), usage (`IPv6Block` mode) & FUP display (`AT17`–`AT19`), DNS behaviour and DoT/DoQ vs. DoH limits, and support flow.

## Domain Operational References

- **Control & Worker**: [`docs/control/README.md`](../control/README.md), [`docs/control/COMPOSITION.md`](../control/COMPOSITION.md), [`docs/control/OPERATIONS.md`](../control/OPERATIONS.md), [`docs/control/WORKER.md`](../control/WORKER.md)
- **Database & Invariants**: [`docs/db/README.md`](../db/README.md) (`F-01`–`F-15`)
- **PKI & Key Rotation**: [`docs/pki/README.md`](../pki/README.md), [`docs/pki/enrollment.md`](../pki/enrollment.md), [`docs/pki/rotation.md`](../pki/rotation.md)
- **Usage Ledger & Recovery**: [`docs/usage/README.md`](../usage/README.md), [`docs/usage/restore-runbook.md`](../usage/restore-runbook.md)
- **Fleet Updates & Node Updater**: [`docs/updates/release-process.md`](../updates/release-process.md), [`docs/updates/central-lifecycle.md`](../updates/central-lifecycle.md), [`docs/updates/node-updater.md`](../updates/node-updater.md), [`docs/updates/node-health-attestation.md`](../updates/node-health-attestation.md)
- **DNS Policy, Threat Feeds & Privacy**: [`docs/dns/PUBLISH.md`](../dns/PUBLISH.md), [`docs/dns/FEEDS.md`](../dns/FEEDS.md), [`docs/dns/BACKEND.md`](../dns/BACKEND.md), [`docs/dns/PRIVACY.md`](../dns/PRIVACY.md)

## Dated Acceptance Evidence

[Security/operations evidence](../e2e/at/secops/README.md) retains the follow-up 2 isolated physical PITR across ledger, grant and audit boundaries. Follow-up 3 rechecks compatibility and proves the default login burst, while its live publication attempt stops at readiness before an operation. It does not establish full-service RPO/RTO, actual node archive recovery or external reconciliation. [Updates follow-up 2](../e2e/at/updates/requests.md) records passing AT23/AT24/AT25/AT27 and the historical AT26 central transition blocker; the [merged fix](../updates/central-lifecycle.md) still needs a fresh live acceptance rerun. Read these records with their source revision; later code merges alone do not change acceptance verdicts.

[Members acceptance follow-up 2](../e2e/at/members/README.md) records payment and controlled DNS subchecks, including node delivery and typed applied evidence, while all assigned full ATs remain blocked. Its [requests](../e2e/at/members/requests.md) identify an unrenewed member lease and a lab node restart waiting for a new enrollment token; the agent-only service hook used for DNS readback does not prove reboot recovery. Native platforms, gateway traffic and native DNS still need acceptance.
