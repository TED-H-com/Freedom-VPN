# AT30 physical PITR subset: passed

UTC target: `2026-10-30T04:00:20.471778Z`. Base backup: `20261030-040016F`.
Source and immutable image/binary digests: [pitr.json](pitr.json),
[pitr-source.json](pitr-source.json). Complete AT30 remains **blocked** on
actual node archive and external provider/peer/revocation reconciliation.

The source uses real browser sessions, finite API grants, API policy/group
inputs, native member admission and authenticated synthetic node uploads.
No projector-owned or configuration SQL fixture writes occur. It ACKs 1–5,
takes the owning pgBackRest full backup, ACKs 6–7 and extends the grant, records
PostgreSQL's UTC clock and WAL position, then ACKs 8–10 and extends the grant
again. The target is at least one whole second after backup completion because
pgBackRest backup selection uses whole-second stop times.

The existing `deploy/backup/restore-pitr.sh --target-time` restores into a new
`fvpn-w4-secops-restore-pitr` container and volume. Its network is `none`, it has
no published ports, the repository is read-only and archiving is off. The image
comes from the repository's backup Dockerfile under a task-owned temporary tag.
A primary-overwrite attempt is refused before mutation. All six sections of the
shipped post-restore checklist run successfully; 21 migrations are applied.
The current signing vault is copied independently of the database backup.

| Check | Expected | Actual |
|---|---|---|
| Restored ledger/checkpoint | 7 events, ACK/max seen 7 | Passed; 7 samples / 2,100 bytes |
| Pre-target grant/operation/audit | Present; original entitlement end | Passed; two canonical audit rows unchanged |
| Post-target grant/operation/audit/events | Absent | Passed |
| Freeze, epoch rotation, retained replay | 3 accepted, 7 duplicates | Passed; ACK10 |
| Repeat replay | 10 duplicates | Passed; exactly 3,000 member bytes, no gap or pending reconciliation |
| Entitlement after replay | Unchanged | Passed |
| Restore diagnostics scan | No known password or generic secret markers | Passed |
| Cleanup and primary | Copy objects removed, source healthy | Passed; final e2e-down leaves zero task objects |

Grant → operation → request → audit correlation is checked. The schema has
append-only audit records; no cryptographic audit hash chain is claimed.

Lab diagnostics: DB recovery window **1.040352 seconds**;
restore/promotion **5.316 seconds**; isolated copy through
checklist and service replay **5.793 seconds**.
The last duration excludes image/binary build and all full-service recovery.
Service RPO/RTO remain null. Retained events came from the test sender after real
mTLS ACKs, not the node SQLite archive. No packet-forwarding or capacity claim.

Reproduce on a fresh own-prefix stack with `make -f mk/at-secops.mk at-secops-pitr`
([setup/cleanup](README.md)). The [attempt history](pitr-attempts.json) retains
failed harness/image/bind/timing attempts and their resolutions. Raw backup,
checklist, canaries and logs stay private; public evidence contains digests,
counts and exact command arguments with worktree placeholders.
