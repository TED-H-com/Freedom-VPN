## Counter classification and stopping

`errors.As` now recognizes `*enforce.DuplicateChainError` as `ambiguous_listing` / `duplicate_chain`. Exact and differing duplicate chains have separate real-parser cases, alongside syntax/type failures and a wrapped typed error. The failed `nft_parse` read, absent observation, one-read behavior, private original/error retention and byte/hash receipts stay intact. No parser error text is published and no automatic retry is added. Netlab's 0/3,300 result and F13-01's unproven origin remain unchanged.

The supervisor now waits a monotonic 15 seconds after TERM delivery before attempting KILL. PID1's TERM/INT handler calls that same supervised stop path and keeps the parent waiter and watchdog alive until the receipt is persisted. This is signal forwarding, not scratch cleanup. Compose and secondary/clone nodes allow 25 seconds; shared namespace/fleet/clone wrappers allow 25 seconds (ordinary teardown still allows 20 seconds, above the 15-second agent budget). Container-stop intent permits automatic resume while an explicit service pause persists. Container exit 0 means its handler completed; the parent receipt is required to distinguish agent exit 0 from signal 9.

The dated [D-102 replies](../../../RESPONSES.md) name updates' `tests/e2e/at/updates/stopobserve/main.go`, isolation's whole-node and clone rows in `driver.go.txt`, and its direct-agent `observer_main.go` consumer. Read the retained process identity before stopping; use `docker cp` for the receipt after a whole-container stop. The `.sigkill.json` sidecar records an attempted escalation, while the parent receipt records the actual result. Historical exit-137 rows and missing first-exit observations remain unchanged.

The serving gates change measurement sequencing and timing. These small diagnostic populations do not estimate a transport reliability rate or demonstrate that either historical finding is fixed.

Displayed millisecond intervals are rounded to 0.001 ms; the exact nanosecond bounds remain in the JSON joins.
