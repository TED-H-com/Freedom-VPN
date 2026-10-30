# Follow-up 6 verification — 2026-10-30

The changed independent scenarios passed on three fresh stacks: physical PITR,
built Portal billing return, control metrics, API serving baseline and both
real-binary update cases. The complete acceptance command remains non-zero
because eight scenarios are blocked. This is **not** three green full-suite
runs. Provider snapshot/event and browser-ticket unit tests also passed with
`-count=3`; live paid-period reconciliation could not pass the D-057 prerequisite.

| Run | Run ID | Source | Pass / blocked / fail | Test seconds | PITR total ms |
|---|---|---|---|---:|---:|
| 1 | `5a29bd5f-dbdf-454b-b7f2-8656a8be0e04` | `177a293` | 12 / 8 / 0 | 315.070 | 7474 |
| 2 | `a4d88567-7a40-40bb-b650-2b82f707a97a` | `177a293` | 12 / 8 / 0 | 332.066 | 15386 |
| 3 | `4797ad88-13f2-47e2-a003-a17a33b99c88` | `177a293` | 12 / 8 / 0 | 330.655 | 9951 |
| Final | `5de03570-0436-477d-9847-6f6984432bd8` | `a7a712a` | 12 / 8 / 0 | 328.797 | 8828 |

The final run changes only blocker-owner wording and documentation relative to
`177a293`; the tested fixture and probe behavior is unchanged. Its complete
20-record set replaces `runs/2026-10-30/`. Earlier complete sets and logs remain
in the ignored task state. No record in the published set comes from another
run. All four `make e2e-test` invocations exited 2 (underlying Go test: 1).

`make go-lint`, `make -f mk/e2e.mk e2e-lint`,
`go vet -tags e2e ./tests/...`, and
`go test -count=3 ./tests/e2e/... ./tests/e2e/cmd/... ./tests/e2e/internal/...`
exited 0. Compose validation, shell syntax and the Portal probe's JavaScript
syntax checks also exited 0. Both node architectures were built and packaged;
only amd64 ran here.

The final Part B gate fetched `origin` and observed `origin/main=5a22aa9` with
wire-nodes follow-up 10 as the latest implementation; follow-up 11 was absent.
The live catalogue record has `GET /api/v1/admin/groups` **501 NOT_IMPLEMENTED**.
As instructed, Part B was stopped at this gate and no upstream merge was made.
The task branch is based on main at `dab7274` (#89); later producer/feed merges
are not claimed as executed by this run.
