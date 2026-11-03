# Follow-up 44: post-window stream repair — 2026-11-03

Window 2 admitted A3 after three reservation refusals and 90.154318 seconds
waiting. Its non-CPU monitoring qualified, but the lab make failed after
14.106588 seconds, before Go scenarios began. The admission helper printed
`free` to stdout. Nested Docker invocations captured that message ahead of the
image identity and clock probe JSON, causing `JSONDecodeError`. This is a
follow-up 44 harness defect, not an A/B production or D087/D104 failure.

The failed result, suite log, samples and measured runner hash remain retained.
No clock agreement, timing span or packet predicate was obtained in that leg.
Both reservations have been released and are exhausted. No third reservation
or replacement timing leg is authorized or attempted.

The repair moves the admission message to stderr; `free` remains printed and
recorded before the command. A subprocess test executes the actual wrapper
against synthetic admission/Docker stubs in disk scratch. It reproduces the
JSON error before the repair and passes afterward; all 24 inherited and new
synthetic tests now pass. The [repair pins](followup44-runner-repair-2026-11-03.json)
distinguish this delivered runner from the measured version. Original
declarations, pins and window receipts are unchanged. The repaired runner is
not used for any additional timing leg in this follow-up.

The remaining ordinary production lab gate will use main's production code
without the timing overlay. Its separate clock probe can verify the shared
clock on this host, but cannot supply missing timing legs or requalify A3.
