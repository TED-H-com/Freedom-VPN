# Handover follow-up 51 — deployment guide corrections

This cut records **three merged pull requests: #524–#526**, through
`66de6a24376237a2c674450bb9a24d41313ff1b5`. The cutoff is the roadmap commit after
#526, whose squash is `995a56792340d151475c382e5320935b0c5a5ba5`.
[Follow-up 50](follow-up-50.md) covered #486–#518 and #520–#523 and was itself
merged as #526. It belongs in this range by the same rule that included #487,
the preceding cut, in follow-up 50.

The [source manifest](../../tests/handover/testdata/followup51-main.json) records
titles, full merge and head commits, merge times and the following roadmap
commits. The summaries below describe the cited merged work. This cut does not
rerun those deliveries or claim a new deployment, measurement or CI result.

## Merged deliveries

| PR | Merged title | Merge commit | Head commit | Effect and boundary |
|---|---|---|---|---|
| [#524](https://github.com/TED-H-com/Freedom-VPN/pull/524) | tests: move the secops guard checks into CI and check every scope pin on each pull request | `3ab457f80da1a25228226437236a6ce3b6e1a86f` | `031da40f0cc143dc616e3e8e0393d44df5bacf37` | Moves applicable historical security-operations guard checks into CI-discovered tests and checks every live scope pin in the unfiltered Go workflow. Developers get failures on the change that invalidates a pin; dated acceptance records retain their original results. |
| [#525](https://github.com/TED-H-com/Freedom-VPN/pull/525) | tests: date signing-key fixtures from a test clock or the run's clock so they do not expire | `dfa266244110f1379996382678b17a57cdc24de7` | `c23e831f948ee6e8a1b7d3f7358bc2502b3ef419` | Derives signing-key fixture windows from an injected test clock or the run's clock so calendar expiry does not invalidate the intended checks. This changes test fixtures, not production key policy or runtime behavior. |
| [#526](https://github.com/TED-H-com/Freedom-VPN/pull/526) | handover: follow-up 50 — cut #486–#523, and revise the out-of-date key, trust and fair-use guidance in place | `995a56792340d151475c382e5320935b0c5a5ba5` | `84b40882b66a3b23ff0e50a87c7feb521f113a82` | Publishes follow-up 50 and updates operator key/trust instructions and member shadow-mode guidance in place. Earlier cuts and exact pre-revision inputs remain frozen. |

## Guide revisions at this cut

The [deployment runbook](deploy.md) now uses `fvpn-vault-init`, followed by
`fvpn-vault inspect -check-keys` and `fvpn-vault export-trust`, with explicit
mapping from generated outputs to runtime configuration. The
[initializer](../pki/initialize.md), [inspection and export guide](../pki/inspect.md)
and [rotation/restore limits](../pki/rotation.md) define the staging FileProvider
boundary: production custody and anti-rollback are not implemented, and local
artifact creation does not prove deployment or node adoption.

The runbook corrects enrollment and lease routes, plan price naming, database
guard wording, the MFA callback outcome, domain configuration loading, the
portal build's working directory, release tree contents and verification trust
path, and the network executor/address-read names. Node provisioning follows
[`fvpn-node-install`](../node/install.md#what-install-changes-63), including
[token intake](../node/install.md#token-intake-182-enr03) and
[Node trust files](../node/install.md#node-trust-files); the incomplete manual
enrollment and unit-install sequence is replaced by those existing procedures.
Bootstrap is described as subsequent agent runtime work.

The [calibration guide](calibration.md) corrects plan and DNS fields, device
limits, capacity types/nullability, login rates, concurrency and request-body
bounds against the [plan schema](../../contracts/schemas/policies/plan-payload.schema.json),
[DNS schema](../../contracts/schemas/policies/dns-profile-payload.schema.json),
[API contract](../../contracts/openapi.yaml) and
[Central defaults and validation](../../internal/config/config.go).
The [restore drill](restore-drill.md) uses local vault key-use inspection with
the whole-directory rollback limitation and selects separate migrator and
recovery credentials for the checked restored candidate. Its logical restore
paragraph now distinguishes the append-only guards and authorized retention
exception from the specific historical replay checks controlled by
`fvpn.restore`; the example remains production-only and unverified, with
its validation carried below. The [database restore contract](../db/README.md#restore-of-historical-snapshots)
describes the existing prerequisites.

Eight stale link fragments are corrected: the two historical restore
measurements, identity continuity, control rate limiters, the fleet offline
lease ceiling, the operator changes section, and the findings register
links from operations and reviews. All relative links in the 14 live cut
sources are checked against their paths and heading anchors; frozen copies
and dated documents retain their bytes.

The deployment review adds the following corrections, including the current
[backup guide](../ops/backups.md) as the fourteenth source. These describe the
implementation at the cutoff; they do not add a deployment or restore result.

| Finding | Correction and boundary |
|---|---|
| R1 | Live authority paths follow the staging ceremony. Control can write the vault directory for revocation-state replacement; worker can open its existing lock file read/write. Initialization precedes service startup. |
| R2 | The executable grants are labeled broad staging grants: runtime roles receive recovery-only writes beyond the intended privilege contract. A complete tested production grant set remains absent. |
| R3 | Node reconciliation handles generation holds, fleet job state and bootstrap expiry; member sync walks existing devices, while discovery may admit eligible unknown member peers. There is no automatic global orphan-peer deletion guarantee. The D-076 fence explanation follows its actual conditions and audit reason. |
| R4 | Installer output has no DNS configuration. Add the documented DNS object and credential file after install, restart the agent and inspect node-reported resolver evidence. A successful installer re-run rewrites the file and drops the added DNS object. |
| R5 | The service-log retention column is stored without a log-retention implementation; `service_logs` is not an accepted purge target. |
| R6 | Grace and grant durations are bounded by 366 days; positive grace needs a reason, and trial duration cannot exceed the configured grant maximum. |
| R7 | Changing the FUP mode override requires restarting control and worker. A read-back procedure confirming both loaded effective modes is still missing. |
| R8 | Database retention columns check positivity; the stricter JSON bounds do not populate those columns. A 30-day payload cutoff conflicts with the 35-day database default. Rollup and audit purge scheduling remains absent. |
| R9 | Shadow mode removes only the fair-use tier cap; plan and node per-member caps remain. |
| R10 | Token neutralization acts on the supplied file or prompted temporary file, with no fixed path. |
| R11 | The read-only logical backup role is distinct from the local `postgres` account used by physical pgBackRest backups. |
| X1 | ACKed payloads stay at least 72 hours after local ACK, not a 30-day default. The unacknowledged limits are 72 hours or 256 decimal MB; the archive has its own 1,000,000,000-byte engineering budget. Zero effective usage RPO depends on the retained replay window. |
| X2 | Updater check/rollback constants bound transaction-step entries, not periodic retry intervals. Failed health begins rollback immediately; attestation timing constants cannot be changed by configuration. |
| X3 | One credential pair serves every resolver target of an exit. Per-target credentials remain unimplemented. |

The second review and six class sweeps also correct the following current
instructions. These are source checks, not a live recovery drill.

| Finding | Correction and boundary |
|---|---|
| S1 | Start control alone, commit recorded post-snapshot revocations, suspensions and grant withdrawals, then start the worker and verify observed enforcement before public traffic. Control can still issue leases from restored authorization during reapplication; restricted recovery startup remains absent. |
| S2 | The backup guide now describes the same bounded node/member/discovery reconciliation as the restore drill and carries orphan inventory/revocation. |
| S3 | Use the existing grant-withdrawal endpoint and operator procedure. Its accepted operation records durable withdrawal intent, not completed enforcement. |
| S4 | Backup recovery commands select the checked candidate's separate migrator and recovery credentials explicitly, rather than an unprovisioned generic DSN file or the control-role default. |
| S5 | PostgreSQL archiving and native/manual restore or verification need repository credentials in their process environment. The backup services' environment file does not configure those other invocations; production delivery of that environment remains a gap. |
| S6 | The review added local `Carried gap:` notices for the dependencies it identified, including build/install/start prerequisites, evidence creation, networking, routing, secrets, retention and recovery limits. Later review found further missing or misplaced notices. |
| S7 | Updater space accounting sums artifact size, installed size and the signed reserve; staging also needs artifact space. There is no enforced 512 MiB floor. |
| S8 | SIGHUP reloads only resolver credential contents at the loaded path. Resolver targets or path changes require an agent restart. |
| S9 | The worker publishes resolver documents; the agent applies them to its local configured resolver targets. Check node DNS evidence separately from publication. |
| S10 | A non-empty invalid drill prefix fails. Only an unset/empty prefix gets a random suffix. |

The six review passes looked for errors in credentials and roles, reload/restart
behavior, process responsibilities, named symbols, automatic cleanup/reconciliation,
and gap dependencies in the four guides. They corrected the recorded cases;
they do not prove that every statement is correct or every dependency has a notice.
The operator and member guides were not reviewed in this cut; the operator
guide's node-detail field is corrected from `observation.checks` to `checks`
because it repeats T3.
Additional corrections identify the actual
enforcement executor methods and binding identities, installer configuration
replacement, optional bootstrap DNS, the validator's nonnegative FUP minimum,
and the restart needed after changing the host reserve. Billing reconciliation
visits stored subscriptions; recovery of subscriptions absent from the snapshot
requires a separate inventory. Database TLS inputs and the OIDC client secret
also require explicit provisioning that this repository does not document.

The third review and the closing name/dependency inventories add these corrections:

| Finding | Correction and boundary |
|---|---|
| T1 | An isolated restore from an encrypted repository needs the caller's exported repository environment; the script loads no secret file. The default unencrypted lab profile does not need a decryption passphrase. |
| T2 | The usage configuration step now warns about independent JSON/database retention and absent audit/rollup purge scheduling, with the calibration explanation linked after the example. |
| T3 | Node detail exposes `checks` at the top level. |
| T4 | The conntrack limit counts durably reserved serving-apply delete batches, including slots consumed by a crash or an earlier failure. Restart does not reset it; successful cleanup can clear it. Baseline-only retries do not authorize forwarding. |
| T5 | The system-plane call uses `enforce.RequireConntrack(p.executor())`. |
| T6 | Review summaries state what was inspected and corrected, without claiming completeness. |

The inventories also correct the member-usage read-back route, the `meter_path`
check name, the Stripe event-parser name and loopback-evidence helper references.
They move or add notices inside identified dependent steps before the commands
or configuration they qualify. Configuration examples were decoded with their
loaders and command examples were inspected without execution; these are source
checks, not deployment or acceptance evidence. No new carried category is added.

The [Chinese](../DELIVERY.md#交接指南修訂) and
[English](../DELIVERY.en.md#handover-guide-revisions) delivery guides retain
follow-up 50's revisions and record this cut's changes. All existing headings
in the three revised documents remain unchanged. The
[pre-cut provenance](../../tests/handover/testdata/followup51-snapshots.json) and
[edit manifest](../../tests/handover/testdata/followup51-edits.json) preserve
14 exact copies from the cutoff and permit exactly 220 replacements: 12 cut-line
edits, 192 guide corrections, gap notices and existing-procedure pointers,
eight link-fragment fixes, and eight delivery-guide revision edits. Every other
byte must match. Older claims retain their assertions; follow-up 50 reads these
pre-cut copies. Follow-up 8 reads the pre-cut deployment guide because T4
removed the phrase it pins. The earlier-claims checks reverse that one read
expression and continue to read the live test source.

## Dated text retained

These five historical passages stay byte-identical. They retain their own
cutoffs and do not describe current acceptance status:

- Deployment §3.1: the 2026-10-30 restore-generation hold and “SECOPS-21 remains open”; later assembled-stack closure is recorded in [follow-up 18](follow-up-18.md).
- Deployment §9.4: the follow-up 15 cutoff's red journaled upgrade; later installer evidence is recorded in [node installation](../node/install.md#upgrade).
- Deployment's “Follow-up 16 restore evidence (2026-10-31 EDT)”: its SECOPS-21 pending/current-snapshot wording belongs to that date; [follow-up 18](follow-up-18.md) records the later closure.
- Restore drill §2.5: the 2026-10-30 measured recovery command sequence, including its old flags, remains the original record.
- Restore drill §2.6: the follow-up 9 snapshot shorthand `fvpn-agent usage snapshot -json` remains historical text, not a current invocation template.

## Existing procedures linked and carried gaps

The deployment runbook adds pointers for backup configuration/scripts/timers
([backup guide](../ops/backups.md)), manifest input fields
([release commands](../updates/release-process.md#commands)), pinned bootstrap
publication ([verified download](../node/install.md#verified-download-182-pkg01)),
desired-state trust and updater configuration, and NetBird/unit/timer startup
([node installation](../node/install.md)). These pointers do not supply a new
procedure or certify a complete production deployment.

The remaining missing steps and logical restore validation are carried forward:

- Deployment §§1/7.1: production Portal static hosting and SPA routing.
- Deployment §§1/9.3/9.4: complete endpoint routing compatible with the installer's one control origin and the public/agent listener split.
- Deployment §3.1: complete first-host pgBackRest provisioning, including initial stanza/base-backup setup; the linked backup guide covers repository settings, credentials, scripts and scheduled units.
- Deployment §4.3: production AdGuard bootstrap, services and host-local/overlay listener wiring.
- Deployment §5.3: production custody and independent anti-rollback authority implementation and deployment.
- Deployment §6.1: server TLS issuance, installation, renewal and trust-chain distribution.
- Deployment §6.2: production payment provider, webhook and billing-secret provisioning.
- Deployment §7.1: installation of built central binaries into the service executable paths.
- Deployment §7.2: creation of the central service account/group and configuration/secret directories.
- Deployment §7.2: central daemon reload, service enablement and startup.
- Deployment §8.1: creation of the SBOM, notices and source evidence files before publication.
- Deployment §9.1: real interface/address/route provisioning; the installer creates dummy links without addresses or routes.
- Restore drill §2.1: validated logical-dump, target-schema and dependency-order selection for the retained example; the database restore contract gives prerequisites, but this logical path remains unverified.
- Deployment §§3.2–3.3: a complete tested grant set conforming to the database privilege contract; the broad staging block is not production least privilege.
- Restore drill §5.2: inventory and revocation of peers authorized after the restore point, including peers absent from restored records.
- Deployment §9.1: installer DNS inputs and preservation of DNS configuration across re-runs; the current installer takes no such input and rewrites the file.
- Calibration §1.6: journald or logrotate retention setup; the stored service-log period has no implementation.
- Calibration §2.2: read-back confirming the effective FUP mode in both restarted services.
- Deployment §6.6 / calibration §1.6: coordination of `usage.json` retention values with the independent database policy; no automatic synchronization exists.
- Deployment §6.6 / calibration §1.6: scheduling authorized audit-event retention purges.
- Deployment §6.6 / calibration §1.6: scheduling authorized usage-rollup retention purges.
- Deployment §§2/9.1: per-target resolver credentials; one pair currently serves all resolver targets of an exit.
- Restore drill step 3.2 / backup Steps 6–8: restricted recovery startup completing usage replay, entitlement computation and policy synchronization before the automatic worker starts; current control-only startup can still serve restored authorization.
- Deployment §3.1 / restore drill Phase 2 / backup §§2.1, 4, 5.2, 6.1–6.3 and Step 2: production delivery of repository passphrase and object-store credentials to PostgreSQL archiving and native/manual and isolated restore or verification processes (including an encrypted lab repository).
- Restore drill step 4.1 / backup Step 7.2: inventory and reconstruction of provider subscriptions absent from restored database rows.
- Deployment §3.1: database TLS certificate/key provisioning and client CA distribution before migration/runtime connections.
- Deployment §4.1: production provisioning and delivery of the confidential OIDC client secret used by control startup.

These are documentation and implementation gaps, not new acceptance results.
The [delivery guide's verification limits](../DELIVERY.en.md#verification-scope-and-known-limits)
continue to apply.
