# Follow-up 45 validation — 2026-11-03

| Gate | Exit | Seconds |
| --- | --- | --- |
| node-lint-build | 0 | 8.592 |
| node-test | 0 | 137.143 |
| node-integration | 0 | 139.254 |
| reviews-r16-r19 | 0 | 20.015 |
| reviews-r20-r26 | 0 | 46.488 |
| handover | 0 | 3.848 |
| go-lint | 0 | 4.754 |
| vet | 0 | 0.395 |
| evidence | 0 | 3.239 |
| fixture-hygiene | 0 | 8.668 |
| final-all-tests | 0 | 152.422 |
| ordinary main lab (13/13) | 0 | 230.853 |
| synthetic tests (41 root; 17 other directory) | 0 | — |

All commands used admission and ran outside reservations. Root go-lint used
the existing pinned staticcheck with `go-with-disk-tmp=set -eu;` to avoid shell
cleanup traps. TMPDIR/GOTMPDIR used repository `.agent/tmp`; GOCACHE came from
`go env GOCACHE` and is recorded home-relative. Python used `-B`.
No production changes or dependency updates. Earlier runner/test/record hashes
remain unchanged. Both reservations are released and owned inventory is empty.

Real output tails and hash-pinned raw receipts are in the [JSON record](followup45-validation-2026-11-03.json).
