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
recovery credentials for the checked restored candidate.

The [Chinese](../DELIVERY.md#交接指南修訂) and
[English](../DELIVERY.en.md#handover-guide-revisions) delivery guides retain
follow-up 50's revisions and record this cut's changes. All existing headings
in the three revised documents remain unchanged. The
[pre-cut provenance](../../tests/handover/testdata/followup51-snapshots.json) and
[edit manifest](../../tests/handover/testdata/followup51-edits.json) preserve
13 exact copies from the cutoff and permit exactly 55 replacements: 12 cut-line
edits, 41 runbook/calibration/restore corrections and existing-procedure pointers,
and two delivery-guide revision sections. Every other byte must match. Older
claims retain their assertions; follow-up 50 reads these pre-cut copies.

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

The remaining missing steps are carried forward, one entry per finding:

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

These are documentation and implementation gaps, not new acceptance results.
The [delivery guide's verification limits](../DELIVERY.en.md#verification-scope-and-known-limits)
continue to apply.
