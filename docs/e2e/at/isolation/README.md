# AT10–AT13 isolation evidence

Follow-up 13 run **`38074149-1137-4782-80fd-8669c36c6fcb`** on the #376 baseline:
AT10 **pass 9/0/0**, AT11 **blocked 2/1/0**, AT12 **pass 10/0/0**,
AT13 **blocked 6/1/0** (pass/blocked/fail).

ISO-F6-01 passes all six assertions, including the accepted dirty-candidate
report joined to its original pre-install command trace. D-087, D-104 and
D-120 pass. Every renewal/expiry loss remains counted. Direct source/clone
agent stops are within the 13 s cooperative bound and preserve stored leases.
Hold reports carry the required signal; no alarm was observed. M-1 cancellation
is not observable here. Two carried operator rows remain.

[Summary](summary.md), [procedure](RUN.md),
[requests](requests.md#follow-up-13-requests--2026-11-01),
[producer analysis](producer-timing-analysis.json),
[immutable producer pins](history/follow-up-13-stream/archive.json),
[validation](validation-results.json).

Run `1c25ee44` remains blocked on its missing report in the byte-identical
[follow-up 12 archive](history/follow-up-12/archive.json). Earlier archives and
failures are unchanged. No timeout inflation, loss waiver or historical promotion.
