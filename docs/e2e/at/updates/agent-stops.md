# Agent stop comparison — Follow-up 13

Current run `759d244a-2dc4-460c-9f82-e37ce0ac7560` on product main `5bd751026073444087b32ac2a7115f8566104f8b`; comparison run
`2babf9b6-86a3-489d-a678-0bd447f2f8b0` on `45209609d33d567c5f04d49441799cdd3efac59b`.
[Machine-readable commands, wait statuses, lease rows and journal timestamps](agent-stops.json).

**Historical limitation:** run 2babf9b6 recorded no direct stop-command timings
or process wait statuses. The retained regular journals supply the complete
stage-bracket list below; the installer journal was not retained. These are
larger intervals, not the command durations requested. Exact historical
command-duration/termination comparisons cannot be reconstructed, and no rerun
can recreate that run. Historical success does not exclude a supervisor kill.

| Stop (complete chronological list) | 2babf9b6 command duration | 2babf9b6 stage bracket ms | 2babf9b6 outcome | Current command ms | Current same-stage bracket ms | Current outcome / agent |
| --- | --- | ---: | --- | ---: | ---: | --- |
| installer switch | not recorded | not retained | success inferred from archived installer success | 8.499 | 22.741 | exit 0; exited |
| installer pre-migrate repeat | not recorded | not retained | success inferred from archived installer success | 2.986 | 15.519 | exit 0; already stopped |
| AT23 switch | not recorded | 442.504 | stop returned success | 443.556 | 448.428 | exit 0; unobserved |
| AT23 pre-migrate repeat | not recorded | 227.706 | stop returned success | 229.330 | 233.839 | exit 0; already stopped |
| AT26 failed-candidate switch | not recorded | 442.037 | stop returned success | 441.713 | 447.106 | exit 0; exited |
| AT26 pre-migrate repeat | not recorded | 229.594 | stop returned success | 227.499 | 232.093 | exit 0; already stopped |
| AT26 rollback | not recorded | 949.085 | rollback completed; Stop error is discarded by product | 435.179 | 933.306 | exit 0; exited |
| AT24 catch-up switch | not recorded | 438.58 | stop returned success | 439.400 | 445.742 | exit 0; exited |
| AT24 catch-up pre-migrate repeat | not recorded | 231.621 | stop returned success | 229.107 | 236.324 | exit 0; already stopped |
| AT24 inherited switch | not recorded | 437.058 | stop returned success | 437.477 | 445.587 | exit 0; exited |
| AT24 inherited pre-migrate repeat | not recorded | 232.947 | stop returned success | 225.727 | 231.965 | exit 0; already stopped |

For the old list, switch and repeated pre-migration stops returned success,
inferred from advancement to `stopped` and `migrated`. AT26 rollback completed,
but its Stop return is deliberately discarded by product code; the journal
cannot establish that command's return status. The installer success implies
both required stops returned success. All old kill/timeout observations remain
unknown. AT25 hash refusal, F18-01 checkpoint refusal/replays and HTTP-only
AT27 stimuli perform no updater stop; harness-controlled offline/observation
stops are excluded from this updater-parent inventory.

The direct current measurements use a monotonic clock around the unchanged
original command. The same-stage intervals include observer readback overhead,
so they do not establish a performance regression or a causal shutdown delay.
The regular public stop permits 100 polls at 0.2 s plus process-scan work; its
supervisor escalates after 25 polls at 0.2 s plus scan work. These are nominal
20 s / 5 s polling budgets, not hard wall-clock bounds. The installer systemd
unit reports `TimeoutStopUSec=1min 30s`. `CommandServices` itself adds no deadline;
it inherits its caller's context. Agent-core 31's at-most-10-second local-work
budget is separate from all these command/supervisor limits.

Observed stop findings: **0**. Every current stop returned 0; no command timeout or killed agent was observed. AT23's first agent exit status was not captured; no all-graceful-exit claim is made. Stored lease rows were also checked at every stop and [restart](restart-leases.json). These quiet regular stops do not prove the worst-case bound, preservation under every shutdown boundary, or long-term lease behavior.

See [the run report](follow-up-13.md) for all AT predicates and unchanged measurement gaps.
