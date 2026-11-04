# Follow-up 10 second attempt — retained AT10 setup failure

Run `5e9e4041-d942-4562-9e99-489bf735636a`, live source `d9b770e0`, source_dirty=false.
The driver ran 1,214.60 seconds. The outer command also waited for the required
shared-host suite slot. Product baseline remains `96ec0630`.

The D-087 check failed and the AT10 aggregate is blocked: the first counted
control was incorrectly placed after the 65-second invariant wait. The saved
baseline was generation 12; that real flow at 18:45:08 UTC preceded ordinary
member rate-policy projection 1→0 and desired generation 13 at
18:45:15.079131011, before false at 18:45:18.063800980. The actual empty lease
and accepted provisional reports use generation 13 and cannot qualify the
captured generation-12 transition. True was invoked only by cleanup; no new
true→accepted-serving duration is claimed. All original wire and invariant
snapshots remain retained. The owned harness now counts first and then waits
for stable projection; it never stops a projector or forces a generation.

Per-check counts (pass/blocked/fail): AT10 **8/0/1**, AT11 **2/1/0**, AT12
**10/0/0**, AT13 **6/1/0**. AT10's missing qualifying counted control keeps its
aggregate blocked. All six ISO-F6-01 assertions pass without restart or topology
repair; **713/713** D-106 digests also match original producer reads.
The corrected finite-member case passes all five probes while the held node
lease remains valid; 134 read-only snapshots awaited natural renewal.

The new D-104 packet predicate passes: D=`56805760270267` ns, **161 positives /
41 drops**, last positive **D−98.950068 ms**, first drop **D−61.916247 ms**,
no later/post-deadline positives, zero capture drops. **One ingress loss occurs
before D**, so literal zero-pre-D-loss is false. Neither this new pass nor
restart adoption resolves ISO-F6-03 or its historical failed row.

Renewal: **1,396 probes / 1,389 positives / seven losses**; all seven ingresses
are inside completed guards. Two real accepted renewals and 70 completed guards
are retained. D-120 is not applied to these diagnostic losses.

Nearest rank over all **931 completed guards**, milliseconds:

| Population | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|
| Outer guard | 30.749663 | 38.828002 | 46.306456 | 49.844866 |
| Matching install command | 15.779444 | 21.512728 | 26.901864 | 30.439422 |

Failed Applies are separate: process `953516a8-88ac-4ba0-906e-d15750d1effa`, sequences
**2298, 2305, 2313, 2320, 2328, 2335, 2342, 2349**. All eight reject before install during the labelled foreign-tc
scenario; their actual brackets remain in the full analysis. No install-bound
overrun is observed in this sample; the first attempt's 49 failed installs remain
failed evidence.

All **8,230** original producer snapshots are retained: 274 baseline, 3,943
heartbeat, 81 renewal, 939 Apply, 2,814 live and 179 blocked-audit rows. Normal
log segments include the watchdog's actual bootstrap `heartbeat_unreadable`
decision and write. No later watchdog write is inferred from a kernel read.
Recovery containment ≤777.459 ms; removal→link-up ≤1,195.580 ms, serving
≤2,303.006 ms and completed counted control ≤11,546.433 ms. The node-any
capture has 2,493 packets, zero capture drops, and no uplink traffic after
confirmed containment until removal. Four redirected packets and one ordinary
SNAT packet preceded confirmed containment. No unsolicited foreign-tc rejection
is present; the unretained original follow-up 6 creator remains unknown.

This attempt is archived separately under `history/follow-up-10-attempt-2/`;
its failures and passing measurements are not reclassified by later runs.
