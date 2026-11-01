## Summary

Isolation follow-up 12 (session launched 04:33 EST on 2026-11-01 when #369 landed; 4 commits, 207 owned files, merge-tree clean, prepush-scan clean). The regression run on agent-core 31's shutdown change (#369), with the producer consumer joined to the per-Apply `commands` lifecycle records that #369 emits under TestMode + TestTimingPath. No product change. AT10 passed 9 / 0 / 0 in run edd986c4 (#360); this run records one blocked assertion (ISO-F6-01 assertion 6) with the full observation timeline. No forbidden forwarding was observed, and no cause is attributed: the run has no A/B, and the observation path goes to agent-core for assessment.

## Summary

Delivered on `w4-at-isolation`, final commit `0ad967dd8ec9e17ad2808e6390b4a5c95d48969c`.
D-098 first: `b597f995` copies all 103 root files of edd986c4 byte-identically
under `history/follow-up-11/`, with SHA-256 pins. Independently compared all
103 files to product baseline main `5bd751026073444087b32ac2a7115f8566104f8b`
(after netlab 22 #365 and agent-core 31 #369). All 655 archive pins verify;
earlier archives are unchanged. Handover already reads immutable snapshots;
its mutable root strings are rejection inputs, so no D-132 exception edit.

Consumer/measurement commit `99e58bff`: strict command lifecycle validation,
identity joins, phase descriptions and stored-lease/restart observations.
Offline consumer commit `3116ddcd` adds lossless gzip support, indexed summary
joins and corrected D-136 review. Final evidence commit includes compact
successful-Apply correlation references, keeping full failed-Apply records.
All original producer bytes remain in the lossless gzip and private raw export.

Exactly one assembled run: `1c25ee44-1872-4cce-a9f5-d52491d87095`, runtime source
`99e58bff0cf7276e36cb04f35605835208c8638e`, `source_dirty=false`. Runtime binaries
were built before the offline consumer refinements; no product edits in this
lane. Driver 1074.90 s, full command 1207.405 s, exit 2. Results (pass/blocked/fail):

- AT10 **8/1/0 BLOCKED**: real D-087 passes; ISO-F6-01 has 5/6 assertions.
- AT11 **2/1/0 BLOCKED**: native platform matrix.
- AT12 **10/0/0 PASS**.
- AT13 **6/1/0 BLOCKED**: full continuous fault/platform matrix.

AT10's new block is missing dirty-generation central observation evidence,
not observed forbidden forwarding. The final review found it after the live
row's initial pending status; it was never upgraded to a pass. No second run
replaced this sample. Three dated requests: two carried operator gaps and the
new observation convergence hand-off.

## Requirement and AT coverage

Spec §19 AT10–AT13, §8.2–8.3 / §10.3 / §16.3: ISO01–ISO04, ENT02, SEC01,
DNS03. Real D-087 exported false/true seam, natural signed empty/populated
leases and actual HTTP 202 reports, exact generation-13 invariants, counted
native admission bytes 2632→5264. 261 original wire hashes verified; 34 desired
responses. True→accepted serving 243.153733 ms. No generation forcing,
projection stop, fake observations/leases or transport substitute.

D-104 passes: D=110488811839293 ns, 161 positives / 41 drops;
last positive D−99.604363 ms, first drop D−62.610491 ms. One pre-D ingress loss
remains counted; literal zero pre-D loss is false. D-134 [D−101 ms,D] wording
retains the 100 ms installation reserve + 1 ms truncation. No post-D positive
allowance. Historical follow-up 9's early outage remains failed/unattributed.

Renewal: 1396 probes / 1360 positives / 36 losses; two real renewals and 70
completed guards. All 36 lost ingresses overlap completed guards, zero outside;
every loss retained. D-120 is separate: 101 verified Applies / 49 datagrams /
one retained eligible boundary loss / zero TCP timeouts, original predicate.

7915 contiguous producer records; 956 Applies, seven pre-install failures,
zero install overruns. 956 host and lifecycle contexts are available. 53754
commands: all started, 53750 ok, four conntrack revoke exit-1 results inside
successful Applies, zero observed cancellation callbacks. All exit/signal/CPU/
switch samples available; exact child-start and hidden host ancestors unavailable.
No cause inferred from the host or exit status alone. Seven failed Applies:
process 252fd542-4dae-4f26-89fd-e06c870c72c0, sequences
2311/2320/2329/2338/2347/2356/2365, each generation 30, 37 commands, verifier
unavailable, incomplete guard, cause unproven. Agent logs preserve the labelled
TC_FOREIGN_FILTER rejection. D-106 616/616 recompute and match producer reads.

Nearest-rank ms over 949 completed guards:

| Measure | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|
| Outer | 31.733360 | 41.314613 | 49.795099 | 54.955270 |
| Install | 16.616649 | 22.812681 | 28.987567 | 31.429430 |
| Render→installed | 16.747072 | 22.945306 | 29.084397 | 31.716162 |

The summary keeps edd986c4's 945-guard distributions separately. No causal
A/B inference, threshold change or historical ceiling is asserted.

ISO-F6-01 assertions 1–5 pass. Assertion 6 stays blocked: all twelve dirty
kernel samples keep flower/mirred, member link down and serving 0, but central
shows the 09:55:00.452272372 generation-29 report. Desired generation 30 is
received at 09:56:19.012930679 and its seven Applies fail before install.
No generation-30 TC_FOREIGN_FILTER report is in the receiver. The generation-29
rejection observed at 09:56:18.153188288 is accepted at 09:56:41.080290184,
after removal at 09:56:25.090642995. Original 502 retries, report IDs, indices,
local failures and all kernel reads remain in iso-f6-01-observation-gap.json
and its referenced source files. The existing predicate is unchanged.

D-135 / corrected D-136: ordinary stale-lease path is baseline→ACK→renewal→
Apply→verify→heartbeat; only retained-Apply-error recovery uses a verified
deny-all candidate. All 19 policy_missing windows are indexed; 15 populated
old/new-generation windows are consistent with stale leases, baseline→Apply
205.950370–264.804339 ms. Each has later successful live verification and a
serving heartbeat. No diagnostic ingress captures cover these 15 brackets;
loss and independent egress there are unavailable, never zero. Later packet
recovery references cannot cross a subsequent baseline or producer end.
All separately captured 36 renewal / 41 expiry losses stay counted. Desired
accept/branch events are absent, so per-call causes remain unproven. Recovery
is required; install budget is not an outage ceiling.

Stops: normal Docker stop 10.088571 s, stopped SQLite snapshot 52.343 ms,
stop→start completion 10.213515 s; same stored lease ID/original mono anchor
survived and counted recovery passed. Exact signal/exit events unavailable.
Clone fixture whole call 21.475699 s; stopped interval 1.332763 s, lease survived;
recorded SIGTERM→SIGKILL 20.007193 s, exit 137. Not graceful-stop qualification.
Terminal source/clone agent-service stops 456.515 / 456.543 ms, both leases
survived. Later container events are teardown, after the agent stops.

## Files

Owned tests/e2e/at/isolation: producer_timing.py and Python/Go regressions;
readers for raw/gzip historical streams; review_policy_missing.py; driver
observations of stop durations/lease IDs; record_test.go scans decompressed
bytes and validates JSONL. Owned docs/e2e/at/isolation: 103-file archive plus
manifest; fresh raw and derived evidence, exact new observation-gap index,
command and container lifecycle summaries, procedure/summary/requests/README
and validation. Root producer-operations.jsonl becomes lossless .jsonl.gz;
old raw stream and post-freeze review are preserved in history/follow-up-11.
No production, shared harness, dependencies, handover test, earlier archive,
README at repository root, spec or AGENTS edit. 207 changed paths, all owned.

