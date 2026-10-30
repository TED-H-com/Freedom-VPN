# Initial follow-up 4 run: rate-limited replay

Run `0e3b86ce-0dcd-4287-8fde-a872a8d1fe6e`, source `a8514ad7366c338f1fb619983d11f7bf886deff8`, at `2026-10-30T06:13:35Z`.
The complete fresh-stack make target exited 2 (the Go test exited 1), wrote all
five records, and removed the stack. AT23/24/25/27 passed; AT26 remained blocked.

The fourth of five rapid conflict requests returned 429 `RATE_LIMITED`; the
other four returned 409 `ACK_CONFLICT`. Both hashes had exactly one conflict
row and one audit. The original accepted receipt was unchanged. All rollback,
revision 7 target-field and observation-authority checks passed.

These unchanged AT26 and summary files preserve the failed run. The HTTP
request/status/response for every replay is inside AT26. This was not a product
receipt-bound failure: the API intentionally rate limits requests, including
traffic from the restarted agent. `internal/config/config.go:81` defaults the
agent bucket to 5/s, burst 10, and heartbeat to 1/s, burst 5;
`internal/httpapi/middleware.go:472` assigns these endpoint classes. The harness
now spaces its TEST ONLY node POSTs by two seconds without altering product
configuration, retrying away a failed assertion, or writing projection state.
The later complete fresh run must independently pass all five conflict calls.
