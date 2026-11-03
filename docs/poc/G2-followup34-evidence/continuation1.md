# Follow-up 34 continuation 1 — consumer fixture

Date: 2026-11-03. Initial delivery: `873b83d9`; fixture extension: `b059756d`.

The orchestrator clarified that the two owned agent test adapters may be
extended faithfully. The original [gate 3a failure](commands/gate3a.txt) and its
[metadata](commands/gate3a.json) remain unchanged. This continuation adds the
rerun beside them; it changes no production file or original evidence.

## Translation and regression coverage

Only `node/agent/followup12_test.go` changes. Its parser now translates the
rendered TCP SYN bit test, MSS comparison and maxseg mangle, ICMP type/code,
and singleton RELATED state (nft JSON `in`). It preserves the actual numeric
MSS values. It also translates rendered hexadecimal priorities to nft JSON
integers, allowing the exact PMTU verifier to check the device class.

The first [focused attempt](commands/focused-continuation1-attempt1.txt)
retains the older priority-string mismatch. The [second attempt](commands/focused-continuation1-attempt2.txt)
passes after correcting the translator. Independent expected JSON tests cover
both new expression sequences and distinct literal MSS values. Negative
batches include `meta nftrace set 1`, an unknown TCP option/field, malformed
or out-of-range MSS, an unsupported flag operator and an unknown ICMP field.
They fail atomically: no rule, expiry anchor or successful batch history is
committed from a rejected batch.

Every existing assertion is preserved. Existing add-if-missing and atomic
warm/cold nft tests pass, as do follow-up 32's tc prepare/commit/retire/release,
crash-cut and partial-failure tests. `revocation_executor_test.go` needs no
change; its tc implementation remains byte-identical.

## Gates

The unchanged consumer command `go test -count=1 ./node/... ./packaging/installer/...`
now passes. No test in another agent-core file required an edit or hand-off.

| Gate | Result | Full output |
| --- | --- | --- |
| 3a — node / installer consumers | PASS; node/agent 145.435 s, installer 8.736 s | [rerun](commands/gate3a-continuation1.txt) |
| 4 — committed-evidence hygiene | PASS; 2.826 s | [output](commands/gate4-continuation1.txt) |
| 5 — handover | PASS; 3.564 s | [output](commands/gate5-continuation1.txt) |
| 6 — make go-lint | PASS; exit 0, empty output | [metadata](commands/gate6-continuation1.json) |
| 7 — go vet ./... | PASS; exit 0, empty output | [metadata](commands/gate7-continuation1.json) |
| 8 — tracked Go formatting | PASS; exit 0, empty output | [metadata](commands/gate8-continuation1.json) |
| 9 — git diff --check | PASS; exit 0, empty output | [metadata](commands/gate9-continuation1.json) |

Gates ran in the requested order. Their JSON siblings contain the exact command,
exit code and output SHA-256. The final staged evidence, including the new
manifest, is checked again before the records commit.

All test commands use `python3 -B tests/netlab/scripts/disktmp.py`, with disk
TMPDIR/GOTMPDIR under `.agent/tmp`. Admission via `$FVPN_HOLD_SH` must print
`free` before each test or make command. The active orchestrator reservation
was waited out; its denial was not counted as a test result. No reservation
was requested and no Docker command or lab was run in this continuation.
The prior lab evidence, all 13 agent scenarios and D-087/D-104 results stand.

## Retention and hand-offs

The [audit](continuation1-audit.json) compares all 7,262 pre-existing G2 files
against delivery `873b83d9` by Git blob identity: none changed. It also checks
that all original test code outside `parseRule` remains byte-identical before
the newly appended tests. The original retention manifest is unchanged.
New records are pinned separately by [continuation pins](continuation1-pins.json).

The original [design and hand-offs](../G2-followup34-2026-11-03.md) stand:
members' native/IPv6-underlay AT09 rerun, isolation's AT13 rerun, and the
handover updates are routed by the orchestrator. No production change or
additional outside-path edit is requested by this continuation.

D-098: earlier evidence and design trees remain byte-identical.
