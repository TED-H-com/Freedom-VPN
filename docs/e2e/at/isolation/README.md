# AT10–AT13 isolation evidence

Follow-up 10 run **`bfcc3880-dc02-485d-990c-13811d6e8835`**:
**AT10 pass, AT11 blocked, AT12 pass, AT13 blocked.**

The real D-087 transition passes at **962 ms** true→accepted serving, followed
by counted native forwarding at unchanged generation 13. All 297 original wire
hashes verify. D-104 passes with 161 positives / 41 drops, including one pre-D
ingress loss. Renewal retains 30 losses among 1,396 probes; none is forgiven.
ISO-F6-01 passes six assertions; all 654 D-106 digests match producer reads.

[Summary](summary.md), [procedure](RUN.md), [requests](requests.md#follow-up-10-requests--2026-10-31),
[grant audit](member-grants-audit.json), [producer analysis](producer-timing-analysis.json),
[validation](validation-results.json).

Historical follow-up 9 expiry remains failed and unattributed in its
[86-file immutable archive](history/follow-up-9/archive.json). The
[first attempt](first-attempt.md) retains its install-bound failure;
the [second attempt](second-attempt.md) retains its AT10 setup failure.
Earlier archives and netlab's follow-up 8 pins are untouched. Handover needed
no pin move. Operator platform coverage remains open; no physical arm64 result
is inferred from helper cross-compilation.
