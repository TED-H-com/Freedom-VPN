# Forwarding observation — Follow-up 3

Run `f81a8365-7839-4756-bee2-9bb2a5506cdd`, source `ac93203dbabc05aa3c1c70f644370a205a704f16`. Full binary/image and probe digests are in [AT01.json](AT01.json) and [AT02.json](AT02.json). Times below are UTC.

Initial counted forwarding and current-generation leases work on this main; the first real device revocation later exposes a missing lab dependency. No application, policy projection, route on the host, or node firewall was changed by the harness. The missing step in the earlier shared forwarding driver is native exit selection under `skip_auto_apply=true`.

| Observation | Before | After |
|---|---|---|
| AT01 controlled-origin route | `eth0` | `fvnb0`, table `netbird` |
| Device upload bytes | 0 | 2214 |
| Device download bytes | 0 | 2411 |
| Forward-chain generation / member accept rules | 5 / 6 | unchanged during the bounded request |
| Origin receipt | no request yet | CA-verified HTTPS 200, exact controlled `ok` body |

The first positive trace began at `2026-10-30T07:07:58Z`. The member `fvnb0` and node `fvmb0`/`fvup0` packet observations each saw 9 packets toward the origin and 7 back. The origin observer in this trace ended early and saw no packets; the exact TLS response proves application receipt independently. AT02 also records a positive origin-interface observation. This is a bounded functional transfer, not a throughput or metering-accuracy benchmark.

The first full-suite attempt initially attempted selection before the route was available to the native client. Its route remained `eth0`, no origin response arrived and device counters stayed zero. The corrected harness waits for the advertised route before selecting through `netbird routes select`; it then polls the product lease separately. The earlier unsuccessful trace is retained in [attempt 1](history/follow-up-3/attempt-1/AT01.json).

**Owner: devstack**, `tests/e2e/traffic_scenario_test.go:countedFlow` (and the calling member-traffic scenario). That driver performs `wget` without selecting an advertised exit. Add bounded native route-availability/selection before the controlled transfer, retaining `skip_auto_apply=true`. Evidence points to the member route-selection hop; it does not justify changing the agent or bypassing the node firewall. Once selected, no stop is observed: member → node ingress → node uplink → origin and return are counted.

The `forwarding` arrays retain interface routes, counter snapshots, aggregate packet observations and origin response per transfer. `elapsed_ms` is the actual capture window; no claim is made that an early-ended observer proves packet absence. Source identities, keys, credentials and raw packets are omitted.

AT01 also repeats the counted transfer after pausing/unpausing the member container. A real network move and Windows/macOS/iOS/Android remain **needs operator run**. AT02 distinguishes the admitted client from the quota loser: no selected exit or successful controlled transfer for the loser, with a contemporaneous counted positive control.

## Later stop after the first revocation

At `07:09:36Z`, AT02 revoked A2 and admitted the former loser through the
production service. The first `HEALTH_CHECK_FAILED / apply failed` observation
arrived at `07:09:39.760469Z`. The node image has no `conntrack` executable;
`node/enforce/apply.go:deleteConntrack` invokes it for dropped addresses.
No package was installed and no enforcement rule was changed by the harness.

AT03's paid-member transfer at `07:12:14Z` had a selected `fvnb0` route, but
node ingress `fvmb0` saw five SYN packets while `fvup0` saw none. The node's
forward chain was generation 0, with zero member accept rules. Its `ctbase`
increased **0 → 300 bytes**, and the device counters remained **0 / 0**.
No origin receipt occurred. This first stop is the **node's fail-closed forward
chain**, following revocation failure, not the original member route selection
or lease-generation gap. Owner: **devstack**, add the required conntrack utility
to `deploy/devstack/Dockerfile.node`; details and reproduction are F3-02 in
[requests.md](requests.md). The existing agent error is retained separately from
its misleading all-passed check list and empty forwarding-block reason list.

The earlier AT02 pass covered the quota/winner/loser checks before the apply
failure was observed. Its supplemental dependency observation changes the
complete AT02 result to blocked, preserving every original passing check.
AT03/04 and subsequent A/B publication traffic remain blocked by this condition.
