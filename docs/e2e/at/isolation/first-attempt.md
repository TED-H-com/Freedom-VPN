# Follow-up 10 first attempt — retained incomplete measurement

Run `17c14696-3eb1-469d-8aab-f947f6f425e4`, source `2da5acfb`, product baseline
`96ec0630`. Assembled driver 770.85 s; outer command 899.883 s; exit 2.
AT10 8/1/0, AT11 2/1/0, AT12 10/0/0, AT13 4/3/0 (pass/blocked/fail).
Those aggregate rows do not erase the failed enforcement operations below.

D-087 passes independently: fixed generation 11, genuine empty/populated signed
leases joined to distinct accepted reports, all invariant snapshots identical,
262 original JSON body digests verified, 29 unchanged desired ETag receipts.
True → accepted serving **2,035 ms** (2,035.055504 ms from original timestamps).
Native admission bytes increase **2,671 → 5,302** afterward; member, node ingress
and uplink captures each contain 16 packets and zero capture drops. First node
uplink packet is 18:20:03.028043 UTC, after the 18:20:00.946155313 serving receipt.
This new sample sits beside historical isolation 4,767 / 5,021 / 5,025 ms and
separate devstack 3,980 / 4,112 ms.

The finite-member setup stopped before its fault window: with the newly fixed
60-second node ceiling, its early lease expires before the three-minute finite
grant. The corrected owned harness waits for natural renewal and accepted
serving, requires a node lease beyond the entire five-probe window, and checks
control stopped before the member deadline. No signed deadline changes.

A separate real enforcement failure occurred during the subsequent renewal
capture. Process `2e7247e0-7303-4080-bd68-ca9ce9abb565`, Apply **712**, has:

| Producer sample | CLOCK_MONOTONIC ns |
|---|---:|
| Apply entry | 54858859674494 |
| Final render | 54858887599306 |
| Install command entry | 54858887720453 |
| Installed-by | 54859103490102 |
| Apply return | 54859414168804 |

Render→installed is **215.890796 ms**, exceeding the unchanged 100 ms bound;
the install command is **215.769649 ms**. Install success and guard release are
false. Baseline producer **713** explicitly names agent `apply_rollback`,
54859107743361–54859384608119 ns. **714/715** name
`verification_HEALTH_CHECK_FAILED`, at 54859414297197–54859726863636 and
54859731657770–54860063408608 ns. This identifies the fresh writer only;
follow-up 9's initial writer remains unattributed.

There are **1,396 renewal probes, 1,117 positives and 279 losses**: 29 captured
ingresses are inside completed guards and 250 outside. Every loss remains a
loss. At the intended whole-node measurement the node held an actual signed
empty lease, so no current D-104 expiry window was executed. Recovery's initial
counted control was unavailable. No inherited packet verdict is used for this
attempt. Host contention is not established as the cause by these brackets.

Nearest rank over all **678 completed guards**, milliseconds:

| Population | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|
| Outer guard | 31.634581 | 41.827776 | 51.362069 | 57.630034 |
| Matching install command | 16.459780 | 22.524217 | 28.101933 | 30.792366 |

All **49 failed Applies** are separate, in process `2e7247e0`:

712, 718, 724, 730, 736, 742, 748, 755, 761, 766, 771, 777, 782, 787, 792, 798, 803, 808, 813, 819, 824, 829, 834, 839, 844, 849, 854, 859, 864, 869, 874, 879, 884, 889, 894, 900, 905, 910, 915, 920, 925, 930, 935, 940, 945, 950, 955, 960, 965.

All reached install; the maximum failed render→installed is **340.349614 ms**.
Their complete brackets, baseline/heartbeat rows and all renewal loss joins are
in [attempt-failure-analysis.json](history/follow-up-10-attempt-1/attempt-failure-analysis.json).
The complete stream retains **5,896 snapshots** across process replacement.
Normal logs were truncated by the shared supervisor; this attempt's final
normal logs do not cover earlier writers. The corrected harness rotates open
logs before replacement and exports all segments after measurement.

The [archive manifest](history/follow-up-10-attempt-1/archive.json) pins all 92
original root files byte-for-byte. Its 23 explicitly marked inherited files
are not measurements of this attempt. The original offline reviewer rejected
mixed clock/namespace inputs; the corrected reviewer refuses an unexecuted
expiry window before reading packet files. All owned Docker inventory was zero
on teardown. This attempt stays retained regardless of later outcomes.

## Follow-up 11 consumer replay — 2026-10-31

The [new replay](history/follow-up-14/first-attempt-host-review.json) reads the immutable first
attempt with agent-core 29's consumer contract. All **49** failures have
`failure_stage=install`, `verification_status=unavailable`, host label
**unavailable** and `cause_status=unproven`. No historical host or command
status is inferred. All **279** renewal losses remain (29 overlapping completed
guards, 250 outside). The original archive and analysis are unchanged.

Agent-core 29's controlled quiet/loaded interventions did not reproduce the
install overrun (maximum render→installed 53.133468 ms, even with 384 busy
workers and 0% aggregate idle); the chain-gate correlation does not establish
cause. See [its assessment](../../../node/followup29-install-contention-2026-10-31.md).
D-135 explains the named rollback as node-wide generation-0 baseline, with
revocation preserved and no previous-generation restoration. Forwarding needs
the next successful Apply; missing verifier records are not verifier failures.
