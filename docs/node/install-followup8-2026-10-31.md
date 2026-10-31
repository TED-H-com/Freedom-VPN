# Installer follow-up 8 — 2026-10-31

The generated updater configuration matches [updater 18's reporting hand-off](../updates/node-followup18-2026-10-31.md#reporting-settings-installer-7-must-render)
(#342) and [updater 19's reconciliation](../updates/node-followup19-2026-10-31.md#installer-7-field-reconciliation-and-consumer-test)
(#351). **No installer behavior changed.** This audit starts from main
`6d1e320d`, which includes installer 7 (#341) and updater 19 (#351).

F10-01 is **CLOSED** by devstack 20 (#343) and updates 11 (#346): the
[D-102 release-root answer](../e2e/RESPONSES.md#updates--2026-10-31-follow-up-20-f10-01-d-102)
and [updates 11](../e2e/at/updates/follow-up-11.md) record the aligned layout,
installer extension **27/27 without a reporting adapter**, and scoped regular
AT23–AT27 **5/5**, including physical rollback. No further F10-01 work is
requested. These are consuming-lane measurements, not a new central campaign
or complete §19 acceptance in this lane.

## Producer / consumer comparison

Producer: [`renderUpdater` and `repairUpdaterReporting`](../../packaging/installer/config.go).
Consumer: [`TestUPD05_InstallerReportingConfigShape`](../../node/updater/followup19_test.go),
subtests `control_cgnat` and `test_only_loopback`. Paths below are production
defaults; both suites relocate files to temporary directories. The consumer's
loopback TLS listener uses an allocated port; `9444` is the shared lab proxy's
port. Neither changes the exact-origin rule. All rows agree.

| JSON field | Updater 18 / 19 expectation | Initial rendering | Same-release reporting repair |
|---|---|---|---|
| `result_url` | Full HTTPS result endpoint, one contract path | Validated control origin + `/api/v1/agent/update-results`; trailing input slash normalized | Restores that exact endpoint |
| `allowed_artifact_origins` | Exact origins granting private/CGNAT exceptions to reporting and downloads; not an exclusive host list or CA trust | One entry: validated control origin, including port, without path/trailing slash | Restores that one entry; separately provisioned private mirrors need reconfiguration |
| `allow_loopback` | False for `control_cgnat`; explicit TEST ONLY true for local TLS fixture; an origin entry alone is insufficient | Request flag, default false | Restores the request flag |
| `mtls_ca` | Endpoint-server PEM trust, distinct from client issuer; empty means system roots with verification | `/etc/vpn-node/ca.pem`, copied from required enrollment `--server-ca`; never empty | Restores path and server CA bytes from the validated input |
| `mtls_cert` | Enrolled client certificate path | `/var/lib/vpn-node/identity/cert.pem` | Preserves the existing enrolled path and bytes |
| `mtls_key` | Node-held private-key path, never key bytes | `/var/lib/vpn-node/identity/key.pem` | Preserves the existing enrolled path and bytes |
| `layout.releases` | Release root directory | `/opt/vpn-node/releases` | Preserved |
| `layout.current` | Current release pointer | `/opt/vpn-node/current` | Preserved |
| `layout.identity` | Enrolled identity directory | `/var/lib/vpn-node/identity` | Preserved |
| `layout.state` | Durable agent state | `/var/lib/vpn-node/state` | Preserved |
| `layout.journal` | Updater journal directory | `/var/lib/vpn-node/update-journal` | Preserved |
| `layout.lock` | Shared node lock | `/var/lib/vpn-node/update.lock` | Preserved |
| `layout.staging` | Retained/staged artifacts | `/var/lib/vpn-node/update-staging` | Preserved |
| `layout.trust`, `release_trust` | Release public trust store | `/etc/vpn-node/release-trust.json` | Preserved |
| `node_id`, `incarnation_id` | Valid enrolled UUIDs | Parsed from enrolled `node.json` | Preserved |
| `revoked`, `revoked_release_ids` | `false`, JSON `null` in consumer fixture | `false`, JSON `null` | Preserved |
| `platform.os_id`, `os_version`, `architecture` | `ubuntu`, `24.04`, `amd64` in fixture | Detected platform; supported `amd64` or `arm64` | Preserved |
| `platform.updater_version`, `control_api_major` | Fixture `1.0.0`, `1` | Signed manifest component version and API major | Preserved |
| `platform.policy_schema`, `usage_schema` | Fixture `1`, `1` | First compatible manifest versions, fallback `1` | Preserved |
| `platform.state_schema` | Fixture `1` | Manifest `state_schema_to` | Preserved |
| `desired_trust` | Desired-state public trust | `/etc/vpn-node/desired-trust.json` | Preserved |
| `desired_state` | Agent desired-state file | `/var/lib/vpn-node/state/desired-state.json` | Preserved |
| `meter_checkpoint` | Genuine agent FileMeter checkpoint | `/var/lib/vpn-node/state/meter.checkpoint` | Preserved |
| `stop_command` | Stop agent, then watchdog | `["systemctl", "stop", "fvpn-agent.service", "fvpn-watchdog.service"]` | Preserved |
| `start_command` | Start watchdog/agent; appended release root is a shell positional argument | `["/bin/sh", "-c", "exec systemctl start fvpn-watchdog.service fvpn-agent.service", "fvpn-updater-start"]` | Preserved |
| `boot_id` | Absent configured fallback | Explicitly omitted | Absence preserved |
| `meter_checkpoint_wait_ms`, `probe_peer`, `agent_config` | Absent in consumer fixture | Omitted; consumer defaults apply | Existing unrelated settings preserved |

Same-release reporting repair assumes the existing enrolled certificate/key
paths are intact; it does not repair arbitrary identity/configuration damage.
A same-release repair that rewrites units/config uses `renderUpdater` again.
No discrepancy with the updater's acceptance was found in either path.

[`TestUPD05_RenderedReportingConfigShape`](../../packaging/installer/followup8_test.go)
pins the six reporting fields using literal JSON keys and expected paths,
independently of updater struct tags or installer path helpers. Both CGNAT
control and TEST ONLY loopback exercise initial install, stale reporting repair,
and a byte-identical repeat rerun. It checks the copied enrollment server CA,
unchanged client identity, absence of a boot fallback, normalization of a
trailing slash, and repair of a missing URL, wrong CA path, foreign origin and
opposite loopback flag. Host/enrollment commands are fakes and neither origin
is dialed. Existing input tests prove loopback without opt-in and malformed
origin/CA inputs fail before host commands; updater 19 supplies the real TLS
consumer test. No test reads living acceptance evidence.

The shared `installerSpec` in `tests/e2e/installer_exports.go` on this starting
main has no loopback field. Devstack 21's exact-local-proxy TEST ONLY opt-in
(F11-01) is not merged on this base. Updates 11 records ordinary-wrapper usage
exit 64 and success with an explicit test flag. That assigned shared-wrapper
delivery remains with devstack; no broader default or wrapper edit is made here.

## Retained archives

The installer guide and follow-up 3 now explicitly cover initial **and
same-release** retention and price storage per distinct retained archive,
including staging/repair and updater verification scratch. Their authoritative
reference is the updater's [Offline installation paragraph](../updates/node-updater.md#installer-hand-off-d-080-d-089-d-110),
corrected by updater 19. `retainArchive` already implements this behavior;
neither the archive nor seed implementation changes. Other installer follow-up
notes contain no contradictory archive-retention claim, so their historical
observations remain intact.

## Scope and remaining measurement

PKG01 / ENR03 / ENR04 (§6.1–§6.3, AT05/AT07) cover installation, identity and
idempotent repair. UPD02 / UPD03 / UPD05 (§13.3–§13.10.3, AT23/AT25/AT26/AT27)
cover authenticated staging, seed and reporting. SEC01 / SEC02 (§16.2) keep
node keys local and distinguish server trust from client identity. The new
test proves configuration production, not enrollment or central authorization.

Distinct-endpoint CA and private-mirror deployments remain **outside the
measured claim**, as updater 19 requires. `allowed_artifact_origins` grants
private/CGNAT exceptions only; updates 11's non-private release-origin probe
needed none. No new artifact-origin input is justified by that measurement.
Real central consumption of the generated config is closed **in the lab** by
updates 11 and remains consuming-lane evidence; it is not remeasured here or
promoted to a production deployment claim.

Physical arm64, cold boot/power loss, production signing/distribution/rollout,
real-device kernel/throughput, complete fault matrices and worst-case 30 s wait /
100 ms poll bounds remain 待實測. Cross-building arm64 is not hardware execution.

### Requests to orchestrator

2026-10-31: **No new requests.** F10-01 is closed by #343/#346, generated
reporting consumption by #346, and updater's stale offline paragraph by #351.
The existing F11-01 shared-wrapper item remains assigned to devstack 21 in
`tests/e2e/installer_exports.go`; it does not require an installer change.
