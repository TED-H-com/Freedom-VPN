# Agent stop comparison — Follow-up 14

Run `2704600f-e076-4a59-9338-09befa2af70e`, product main `f680d6ddb8fd19979c0471a2ce06ffc6e16c7cd2`; immutable baseline
[run 759d244a](history/follow-up-13/agent-stops.json) on `5bd751026073444087b32ac2a7115f8566104f8b`.
[Complete command, identity, parent receipt, lease and journal data](agent-stops.json).

| Stop (complete chronological list) | 759d244a command ms / exit | 759d244a agent | Current command ms / exit | Current agent |
| --- | ---: | --- | ---: | --- |
| installer switch | 8.499 / 0 | exited 0 | 52.688 / 0 | exit 0 / signal null |
| installer pre-migrate repeat | 2.986 / 0 | already stopped | 3.028 / 0 | exit 0 / signal null (same receipt; no new exit) |
| AT23 switch | 443.556 / 0 | unobserved | 448.124 / 0 | exit 0 / signal null |
| AT23 pre-migrate repeat | 229.330 / 0 | already stopped | 231.254 / 0 | exit 0 / signal null (same receipt; no new exit) |
| AT26 failed-candidate switch | 441.713 / 0 | exited 0 | 449.261 / 0 | exit 0 / signal null |
| AT26 pre-migrate repeat | 227.499 / 0 | already stopped | 227.978 / 0 | exit 0 / signal null (same receipt; no new exit) |
| AT26 rollback | 435.179 / 0 | exited 0 | 433.314 / 0 | exit 0 / signal null |
| AT24 catch-up switch | 439.400 / 0 | exited 0 | 439.985 / 0 | exit 0 / signal null |
| AT24 catch-up pre-migrate repeat | 229.107 / 0 | already stopped | 226.091 / 0 | exit 0 / signal null (same receipt; no new exit) |
| AT24 inherited switch | 437.477 / 0 | exited 0 | 434.169 / 0 | exit 0 / signal null |
| AT24 inherited pre-migrate repeat | 225.727 / 0 | already stopped | 222.130 / 0 | exit 0 / signal null (same receipt; no new exit) |

| Stop | Launch ID | Pre-stop child PID | Start ticks | Raw parent wait status | Agent exit / signal | UTC wait observation |
| --- | --- | ---: | --- | --- | --- | --- |
| installer switch | 29922bbaf65d48ecb9fd669c2d4b53b1 | 2266 | 12800871 | waitid 1/0; encoded 0; kernel sample 0 | 0 / null | 2026-11-01T14:47:56Z |
| installer pre-migrate repeat | 29922bbaf65d48ecb9fd669c2d4b53b1 | 2266 | 12800871 | waitid 1/0; encoded 0; kernel sample -1 | 0 / null | 2026-11-01T14:47:56Z |
| AT23 switch | 8f159491-bf4b-4b31-bbf5-1b5d5db15973 | 18338 | 12810102 | 0 | 0 / null | 2026-11-01T14:49:33.759333325Z |
| AT23 pre-migrate repeat | 8f159491-bf4b-4b31-bbf5-1b5d5db15973 | 18338 | 12810102 | 0 | 0 / null | 2026-11-01T14:49:33.759333325Z |
| AT26 failed-candidate switch | 09e966f4-4d2b-4369-a266-042c7578a629 | 23276 | 12811201 | 0 | 0 / null | 2026-11-01T14:50:14.850504038Z |
| AT26 pre-migrate repeat | 09e966f4-4d2b-4369-a266-042c7578a629 | 23276 | 12811201 | 0 | 0 / null | 2026-11-01T14:50:14.850504038Z |
| AT26 rollback | e70d9d16-69b7-45c4-bdbd-c174ad4ec22b | 35598 | 12815308 | 0 | 0 / null | 2026-11-01T14:50:46.379240341Z |
| AT24 catch-up switch | ec212d6c-01ba-4163-ba33-8fe24ad82820 | 141112 | 12842598 | 0 | 0 / null | 2026-11-01T14:54:58.892957702Z |
| AT24 catch-up pre-migrate repeat | ec212d6c-01ba-4163-ba33-8fe24ad82820 | 141112 | 12842598 | 0 | 0 / null | 2026-11-01T14:54:58.892957702Z |
| AT24 inherited switch | 8051b988-bd76-4e34-b3c3-b0d5d6c6dec9 | 1152 | 12845056 | 0 | 0 / null | 2026-11-01T14:55:28.415011886Z |
| AT24 inherited pre-migrate repeat | 8051b988-bd76-4e34-b3c3-b0d5d6c6dec9 | 1152 | 12845056 | 0 | 0 / null | 2026-11-01T14:55:28.415011886Z |

`null` means null, never inferred zero. Five repeated pre-migration stops reuse
an existing exit and are not five new terminations. The regular fixture uses
devstack's exact retained-ID sequence; raw parent wait receipts are read after
stop and remain identical after subsequent starts. The installer agent's parent
is systemd: its native waitid code/status and UTC parent exit observation are
retained. The encoded wait status is explicitly derived, and the independent
raw kernel sample is separate (-1 means unavailable). Systemctl's displayed
exit timestamp has one-second precision; devstack uses RFC3339Nano. Neither is
the exact kernel exit instant. Container exit 0 is not used to classify agents.

Every command is timed monotonically around the original service call. The
observer changes no command argument, agent binary, generated config or limit.
Devstack 26's supervisor allows 15 s after TERM before SIGKILL; PID1 forwards
TERM/INT and compose/clone grace is 25 s. SIGKILL sidecars record attempts only;
actual termination is the separate parent receipt. The collector uses docker cp
and can read receipts after a whole-container stop, without docker exec.

AT25 hash refusal, F18-01 checkpoint refusal and HTTP-only AT27 stimuli perform
no updater stop. Harness-controlled offline and observation stops are outside
this updater-parent inventory. A successful slower stop within budget is a
measurement; reaching 15 s, signal termination, command failure or lost lease
is retained as a failed row, with agent-core or updater ownership. Journal-stage
intervals include additional work and are not direct stop durations.

Updates F13-01 remains unobserved for run 759d244a; this AT23 receipt is a new
sample. Its historical row is not filled retrospectively. The older 2babf9b6
missing direct timings also remain missing. Stored-lease [restart readbacks](restart-leases.json)
cover only the immediate observations, not worst-case shutdown or lease expiry.

See [Follow-up 14](follow-up-14.md) for acceptance results and unchanged gaps.
