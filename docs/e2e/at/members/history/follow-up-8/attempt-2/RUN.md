# Follow-up 8 interrupted attempt 2

Run `63a4fb5f-5492-474b-b299-d799ce179386`, harness `4a2715a8c4ddf683e37a314c9a11f38ab0cf7128`, baseline `6848add` (#284), 2026-10-31. AT01 and AT05 completed blocked with passing subchecks. AT02 is partial and retains its failed re-grant transfer. Only records from this run are archived here; unchanged root files from attempt 1 are not represented as new results.

Stopped after discovering that moving installProbe to `.agent/tmp` left copy/hash reads at the former artifact location. The stale and current binary SHA-256 values differed (7178620ad24937213260a4f5b419ab1cc68233f02f9b3c4bcf45b408a2901845 versus 52e9c1028f3b089bb2fa24256409eedd3d493a4659a127e447ecfeb112677975). The old artifact was reachable only because earlier runs left it behind. The next attempt resolves all references through the same disk-scratch path.

D-077 member-list authorization and generation-9 forwarding checks passed, but the 08:41:06Z HTTPS probe got no origin receipt. Counters moved only 340 upload / 60 download bytes, with six outbound and one inbound member packets. This does not prove HTTP success. The inherited probe reused TCP source port 38443 across fresh connections; the next attempt assigns a distinct bounded port per transfer and passes it to all three aggregate packet observers. Port reuse is a possible contributor, not an established product root cause. No retry changes this failed check into a pass.

The interrupted stack is torn down before a clean full rerun. Partial evidence is historical only. SHA256SUMS protects the archived case bytes.

The test process was explicitly terminated; Go reports `signal: terminated`, 410.534 seconds, and make exit 2. This is an interrupted diagnostic run, not a completed gate.
