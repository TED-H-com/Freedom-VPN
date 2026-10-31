# Updater follow-up 19 — 2026-10-31

Updates 11 (#346) closes the generated-reporting and regular-release-layout
hand-offs from [follow-up 18](node-followup18-2026-10-31.md), consuming installer
7 (#341), updater 18 (#342) and devstack 20 (#343). Its installer extension is
**27/27 without a reporting adapter**; its scoped regular AT23–AT27 rows are
**5/5**, including a physical rollback. This audit starts from main `03f808db`.
It consumes the recorded run; it does not rerun the external lab or claim full
§19 acceptance from those historical lane scenario labels.

## Evidence and D-132 citation policy

All U11 references below are living paths under `docs/e2e/at/updates/`, read on
2026-10-31. **No evidence-file SHA-256 digest is pinned and no test reads these
files.** A run UUID recorded inside a living root file does not make that path
immutable. Digest pins require an owner-provided `history/` snapshot or a UUID
run directory. Updates 10 now has its own
[history archive](../e2e/at/updates/history/follow-up-10/ARCHIVE.md); follow-up
18's older measurements remain historical rather than being rewritten.

- **U11:** [run.json](../e2e/at/updates/run.json),
  [source-main.json](../e2e/at/updates/source-main.json),
  [follow-up-11.md](../e2e/at/updates/follow-up-11.md) and
  [RUN.md](../e2e/at/updates/RUN.md). Run
  `daddad24-b6f7-4c32-ba8e-1c7d24f76ea2`, measured harness `3ac13f7e`, product
  main `96ec0630`; live make/test exit 0, test duration 511.968 s.
- **I11:** [installer-handoff.json](../e2e/at/updates/installer-handoff.json)
  and [readable hand-off](../e2e/at/updates/installer-handoff.md), 27 required checks.
- **T11:** [installer-delivery-transport.json](../e2e/at/updates/installer-delivery-transport.json),
  route-only capture through installer extension completion, before regular
  enrollment, with no other update producer in the interval.
- **O11:** [artifact-origin.json](../e2e/at/updates/artifact-origin.json),
  isolated post-extension uncached seed of the retired installer fixture.
- **A11:** [AT23.json](../e2e/at/updates/AT23.json),
  [AT24.json](../e2e/at/updates/AT24.json),
  [AT25.json](../e2e/at/updates/AT25.json),
  [AT26.json](../e2e/at/updates/AT26.json) and
  [AT27.json](../e2e/at/updates/AT27.json).

## Follow-up 18 claim reconciliation

Requirement scope: UPD02/UPD03/UPD05, MTR03 and API01;
[spec §13.3–§13.10.3](../spec/PROJECT_SPEC.zh-TW.md#133-不可變-release-與簽章-manifest),
§19 AT16/AT23/AT25/AT26/AT27. AT24 supplies offline catch-up evidence.

| Follow-up 18 claim | U11 measurement by path/observation | Disposition and limit |
|---|---|---|
| Explicit nil-sink `Engine.Flush` returns `VALIDATION_FAILED`, exit 1 | I11 `check_generated_reporting`, `check_generated_delivery` and `check_no_missing_result_url` show a configured sink; U11 does not inject a nil sink. | The unchanged `TestUPD05_MissingResultURLKeepsReceiptPending` remains the direct failure proof. Do not label the nil-sink injection externally measured. |
| `result_url_missing` retains the sealed receipt pending, never delivered or ACKed | I11 `generated_config`, unchanged whole-config hash before/after, and command/service logs show the generated endpoint is present and this diagnostic is never observed. | Now externally measured as absent on the installer-provisioned path. The new config-shape test proves `LoadConfig` → `Config.Engine` creates an `HTTPSink`, making this nil-sink branch unreachable while that valid generated config is intact. Missing/manual/older configs retain the existing pending-receipt negative. |
| Repair of missing reporting sends the identical receipt exactly once | I11 `generated_delivery_counts` is history 1/outbox 0 before manual replays; `generated_repeat_flush` exits 0 with unchanged ACK/admin view. T11 `no_repeat_flush_post=true`. | U11 measures automatic delivery/deduplication with correct config from installation. It does not deliberately remove then repair the URL; the exact repair sequence remains the unchanged follow-up 18 component test. Installer's same-release reporting repair is code/installer-test coverage. |
| Root `fvpn-agent` fixture and service contract | I11 `running_executable`, `check_restarts`, `check_apply`; A11 AT23 runs the release-root executable with five passing checks, and AT26 restores the prior root executable. | F10-01 closes in the scoped regular campaign after devstack 20. Updater's root-fixture/hash-drift test remains unchanged. |
| Installer must render result endpoint, origin policy, server CA and client identity | I11 `generated_config`, `server_trust_matches_enrollment`, `server_trust_differs_from_client_issuer`, `check_generated_reporting`, `check_reporting_unchanged`; T11 records real central acceptance. | F10-02 closes without the U10 reporting adapter. Field-by-field reconciliation is below; no rendered-field divergence found. |
| D-125 correlated apply, genuine finalization and nondestructive stopped-producer failure | I11 `correlated_checkpoint`, `correlated_wait`, `finalized_epoch`, `check_apply`, `check_checkpoint_failure`, `check_failure_replay`. | Sequence 10→11, real restarts, health 5/5, success seal; quiet final-sample durable/ACK 0/0. Stopped producer apply/replay/recover remains **1/1/2**, failed seal retained, current 11 retained and forwarding blocked. F18-01 stays the intended negative. Serving withdrawal/nonzero accounting is still attributed to devstack 19. |
| Real central result acceptance, replay and D-127 correction | I11 `check_job_id`, `check_central`, `check_replay`, `check_conflict`, `check_wrong_job`, `check_correction`, `check_etag`; T11 automatic requests. | D-110 job recomputed; source installer/batch null; local/central ACK and immutable payload agree. Exact replays return 202, modified/wrong-job/third-outcome stimuli 409. Installer correction is still a TEST ONLY HTTP stimulus, with local current 11; physical rollback comes separately from A11 AT26/AT27. |
| Seed has no extractor-induced drift after upgrade or across umasks | I11 `seed_default_umask`, `seed_umask077`, `check_seed` preserve the same verified baseline/hash/timestamp. O11 verifies current sequence 11 against the signed artifact. | Real revoke/re-enrollment remains follow-up 18's devstack R19 evidence. U11 adds a post-upgrade seed without tree repair; it does not repeat re-enrollment. |
| Tamper and downgrade refuse candidate promotion | I11 `apply_tampered`/`check_tamper` and `downgrade`/`check_downgrade`; A11 AT25 independently measures bad archive refusal. | Installer pending tamper exits 2 with `HASH_MISMATCH` before switch; downgrade exits 1 with `DOWNGRADE`. Neither is the sealed final-sample failure's apply-1 path. |
| FileMeter load check 40/40; no mutable AT evidence dependency | U11 [D-132 audit](../e2e/at/updates/d132-audit.md) also finds no product-test read of living AT evidence. | The 40/40 race/load result remains follow-up 18 evidence, not a fresh U11 timing measurement. This test constructs its entire config and TLS fixtures locally. |

T11 counts **13 POSTs** to `/api/v1/agent/update-results`: **11 explicit HTTP
contract stimuli + one automatic success + one automatic stopped-producer
failure**. The first automatic success precedes the stimuli; its stored ACK
`received_at` is `2026-10-31T18:17:13.343497Z`. A repeat product flush adds no
POST. This is one automatic transmission per receipt in the measured interval,
not a claim that an unreliable transport can never retransmit a pending receipt.
The existing authenticated split-listener enrollment proxy verifies the client
and forwards to the real central used by the lab; it is not a new reporting
adapter or evidence of a production central deployment.

## Installer 7 field reconciliation and consumer test

Read-only producer: [`renderUpdater` and `repairUpdaterReporting`](../../packaging/installer/config.go).
Consumer: [`LoadConfig`, `Config.Engine`, `NewMTLSClient`, `HTTPSink`](../../node/updater/host.go)
and [`checkURLPolicy`](../../node/updater/verify.go).

| Follow-up 18 field | Installer 7 / I11 actual value | Reconciliation |
|---|---|---|
| `result_url` | Validated control origin plus `/api/v1/agent/update-results`; I11 `https://127.0.0.1:9444/api/v1/agent/update-results`. | Full endpoint, no doubled path. Automatic success and failure reach the authenticated proxy and real central. |
| `allowed_artifact_origins` | Exactly the validated control origin; I11 `["https://127.0.0.1:9444"]`. | Correct private/CGNAT exception policy for the shared result/download client; not an exclusive host list or TLS trust source. O11 closes the separate-artifact-origin question for the measured non-private origin. |
| `allow_loopback` | Request's explicit TEST ONLY flag; I11 true. | Production default remains false. I11's ordinary wrapper exits usage/64 without the opt-in before host changes; explicit test input succeeds. F11-01 remains devstack-owned. An origin entry alone cannot allow loopback. |
| `mtls_cert`, `mtls_key` | `/var/lib/vpn-node/identity/cert.pem`, `/var/lib/vpn-node/identity/key.pem`. | Enrolled client identity; authenticated lab proxy accepts it. Same-release reporting repair preserves these existing paths; it is not a repair of arbitrary damaged identity material. |
| `mtls_ca` | `/etc/vpn-node/ca.pem`, copied from enrollment server trust. | I11 comparison matches enrollment trust and differs from `/var/lib/vpn-node/identity/ca.pem`, the client issuer. This satisfies the endpoint-server trust requirement; no assumption that the client issuer signs the server. |

No field divergence requiring a product fix was found. The updater's
[`TestUPD05_InstallerReportingConfigShape`](../../node/updater/followup19_test.go)
uses literal JSON keys for the full rendered shape, null revoked-release IDs and
no configured boot fallback, with temporary paths and generated client identity.
It passes through `LoadConfig`, constructs a non-nil `HTTPSink`, checks the exact
control origin, loaded server CA and client certificate, and submits through a
local TLS fixture requiring client authentication. It also checks a CGNAT control
origin without dialing it. Removing the CGNAT exception or the TEST ONLY
loopback permission fails construction. The server trust is distinct from the
client trust; normal server verification stays enabled. This proves configuration
consumption, not enrollment or central authorization, which come from I11/T11.

## F11-02: retained archive and missing-cache fallback

The [Offline installation paragraph](node-updater.md#installer-hand-off-d-080-d-089-d-110)
now matches [`install.go`](../../packaging/installer/install.go) and
[`retainArchive`](../../packaging/installer/archive.go): initial and same-release
installs retain a verified archive as well as the envelope before seed. The
prior statement that only the envelope was retained was stale documentation.
There is no runtime change or installer-owned code defect to fix for this item.

[`verifySeedTree`](../../node/updater/seed.go) uses and verifies a private copy of
an existing cache. Bad hash/size/unreadable cache fails with the documented
`HASH_MISMATCH` / `SIZE_MISMATCH` / `ARTIFACT_INVALID` diagnostic; it never silently
downloads a replacement. Only absence chooses the guarded, size/hash-verified
download path. Exhausted network retries give `STORE_UNAVAILABLE: download`, or
`DOWNLOAD_TIMEOUT: attempt` for an exhausted attempt timeout; seed exits 1.
The guide gives the same-release installer repair and absent-cache retry actions.

O11 deliberately withholds the archive, uses unchanged generated config and the
unchanged signed release URL, and exits 0. Its only changed status field is
`seeded_baseline`; current/last-good and failed transaction outcome remain intact.
The original cache is restored byte-identically. The release hostname resolves
to a controlled non-private/non-CGNAT Docker address, so no additional origin
exception is needed. This neither changes hosting policy nor establishes private
mirror reachability. An unverified artifact or live-tree-only hash is never an
acceptable fallback.

## Remaining measurement from follow-up 18

| Previously unmeasured item | U11 disposition |
|---|---|
| Generated reporting without an adapter | Closed for I11/T11's installer-provisioned lab node, including automatic failure delivery and no missing-URL diagnostic. |
| Regular candidate startup and rollback | A11 AT23–AT27 pass 5/5. AT26 physically switches/restarts/restores health; AT27 `check_physical_rollback=true` names the same-run transaction `01a19330-0749-7ce3-aad0-ddf926cada16`. This closes the missing physical witness in this lab scope, without turning the installer HTTP correction into a physical rollback. |
| 30 s wait / 100 ms poll under worst-case CPU/storage load; controlled near-deadline publication | Still 待實測. I11's correlated success and stopped-producer timeout do not bound worst-case scheduling/filesystem latency; follow-up 18's 40/40 component load result is unchanged. |
| Live malformed/oversized/special-file checkpoint; foreign/alternate-spelling UUID; other-boot evidence; withdrawal after accepted sampling | Still component coverage only. I11 measures matching kernel/agent/job boot identities, a correlated positive and an absent-checkpoint negative, not these fault injections. |
| Mid-handshake crash at `blocked`/`final_sampled`, original-deadline preservation, identical producer republication and valid publication after the deadline | Still component coverage only. I11's sealed terminal replay is not a mid-handshake restart or a late-publication injection. |
| Newly counted final-sample bytes and wider accounting continuity | I11 is quiet at 0/0. Follow-up 18's devstack 19 serving/nonzero watermark evidence remains its own scope; no new byte claim is made here. |
| Physical arm64 forwarding/metering/updating | Still 待實測; [arm64 operator row](../e2e/at/updates/at23-arm64.json) is unchanged. Cross-build is not hardware execution. |
| Filesystem/power-loss durability, complete disk/migration fault matrix | Still 待實測; successful lab switching and component crash tests do not close actual power-loss acceptance. |
| Real fleet rollout, three join sources, mixed populations, production central | Still unmeasured here. A11 is a scoped amd64 lab campaign, and I11's real central is a test deployment behind the authenticated proxy. |

## Verification

All required gates passed: updater lint/test/integration/build (including
linux/amd64 and linux/arm64), fresh updater/nodelock/installer/installerjob tests,
the unchanged `TestMTR03_UPD05_DurableFileMeterBarrier`, whole-module
`make go-lint`, direct `go vet ./...`, formatting and diff checks. The root lint
helper was overridden with `go-with-disk-tmp=set -eu;` to retain exported worktree
disk scratch without executing the cleanup trap in unowned `mk/go.mk`; its
gofmt/vet/pinned-staticcheck checks were unchanged. Integration used `dbtest`
per-test databases on the supplied DSN, with sweeping unset. No container was
created or shared PostgreSQL setting changed.

The new test and unchanged reporting regressions pass together. A recursive
audit of 37 updater and 8 updatejournal test files found no living AT evidence
reads or constructed `docs`/`e2e` path fragments. No dependency, contract,
migration or runtime behavior changed. The worktree hand-off report retains
real output tails and `.agent/tmp/f19/` logs.

### Requests to orchestrator

- **2026-10-31 — installer 8:** consume the new config-shape test and I11/T11
  closure: all rendered reporting fields are compatible; preserve exact control
  origin, enrollment server CA, enrolled cert/key and TEST ONLY loopback opt-in
  on initial install and same-release reporting repair. F11-02 is corrected in
  this lane; no installer code change or new artifact-origin input is requested
  by this measurement. Keep distinct-endpoint CA/private-mirror deployments
  outside the measured claim. This is a nonblocking hand-off, not a field defect.
- **2026-10-31 — acceptance owners (updates/devstack):** retain the remaining
  measurements above and the F18-01 stopped-producer negative. Carry forward
  devstack's existing F11-01 explicit test-loopback wrapper request; this lane
  did not edit `tests/e2e/installer_exports.go`. Archive U11 before any future
  digest-dependent evidence consumer; this test needs no evidence-file input.
