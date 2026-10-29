# Operations Runbooks Index (`docs/ops/`)

Operational runbooks, deployment guides, parameter calibration references, and disaster recovery checklists are maintained under [`docs/handover/`](../handover/README.md) alongside domain-specific operational references:

## Hand-Over & Production Runbooks (`docs/handover/`)

- [**Production Deployment Runbook (`../handover/deploy.md`)**](../handover/deploy.md) — PostgreSQL 17 + WAL archiving, TLS & mTLS listeners, OIDC + MFA, NetBird CE `0.82.0`, AdGuard Home `v0.107.59`, release origin, 5 domain config files, PKI & signing key ceremonies, and first node enrollment.
- [**Backup and Restore Drill Record (`../handover/restore-drill.md`)**](../handover/restore-drill.md) — `AT30` point-in-time recovery checklist, Wave 4 measured RPO/RTO evidence, `fvpn-usage-admin` ledger freeze/recovery/replay, and Stripe + NetBird state reconciliation.
- [**Parameter Calibration Guide (`../handover/calibration.md`)**](../handover/calibration.md) — Business and engineering parameter matrix (`none`, `test only`, `proposed`), spec constraints, and decision ownership (`Ted` vs. `core team`).
- [**Operator Guide (`../handover/operator-guide.md`)**](../handover/operator-guide.md) — Daily/weekly checklists, reading operations and audit trails, node lifecycle, fleet update batches (`needs_attention` / `unconfirmed`), ledger freeze/adjustments, member suspension, and key rotation.
- [**Member Guide (`../handover/member-guide.md`)**](../handover/member-guide.md) — Bilingual (`zh-TW` / `en`) guide for Windows, macOS, iOS, and Android, manual verification scripts (`AT01`, `AT09`, `AT21`), device limits, usage/FUP display, DNS behaviour and DoH limits, and support flow.

## Domain Operational References

- **Control & Worker**: [`docs/control/README.md`](../control/README.md), [`docs/control/COMPOSITION.md`](../control/COMPOSITION.md), [`docs/control/OPERATIONS.md`](../control/OPERATIONS.md), [`docs/control/WORKER.md`](../control/WORKER.md)
- **Database & Invariants**: [`docs/db/README.md`](../db/README.md) (`F-01`–`F-15`)
- **PKI & Key Rotation**: [`docs/pki/README.md`](../pki/README.md), [`docs/pki/enrollment.md`](../pki/enrollment.md), [`docs/pki/rotation.md`](../pki/rotation.md)
- **Usage Ledger & Recovery**: [`docs/usage/README.md`](../usage/README.md), [`docs/usage/restore-runbook.md`](../usage/restore-runbook.md)
- **Fleet Updates & Node Updater**: [`docs/updates/release-process.md`](../updates/release-process.md), [`docs/updates/central-lifecycle.md`](../updates/central-lifecycle.md), [`docs/updates/node-updater.md`](../updates/node-updater.md)
- **DNS Policy & Privacy**: [`docs/dns/PUBLISH.md`](../dns/PUBLISH.md), [`docs/dns/BACKEND.md`](../dns/BACKEND.md), [`docs/dns/PRIVACY.md`](../dns/PRIVACY.md)
