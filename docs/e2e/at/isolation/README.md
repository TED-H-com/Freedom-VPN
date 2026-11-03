# AT10–AT13 isolation evidence

Follow-up 18 run **`9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04`**, assembled exit **2**:
AT10 **9/0/0**, AT11 **2/1/0**, AT12 **10/0/0**, AT13 **6/1/0**
(pass/blocked/fail). ISO-F6-01 **6/6**. AT11/AT13 operator matrices remain blocked.

Both unlisted blocked rows from scheduled run **`41640196805`** (`AT13 short whole-node lease expiry` and `AT10 foreign ingress redirect containment and D-096 recovery`) are diagnosed in [dated nightly diagnosis](nightly-diagnosis-2026-11-03.md) as a single post-outage control restoration readiness defect in the owned harness (`tests/e2e/at/isolation/`), fixed and verified in this assembled run (**both rows pass**). Prior [nightly diagnosis](nightly-diagnosis.md) remains untouched as merged.

D-140 **qualifies under D-142 and passes the unchanged evaluator**, with no
overlap anywhere in the sweep (`164 positives / 70 drops`). Shared D-104 passes on netlab 26's fixture.
The original seven hold categories and central's 30-second fallback pass;
pure `stale_generation` is not exercised. Six agent parent receipts record
exit 0; zero installs exceed 100 ms. Historical failures stay failed; no retry.

[Summary](summary.md), [procedure](RUN.md), [hold row joins](populated-holds.md),
[boundary qualification](boundary-review.json), [findings](follow-up-18-findings.json),
[parent receipts](stop-receipt-review.json), [dated nightly diagnosis](nightly-diagnosis-2026-11-03.md),
[requests](requests.md), [validation](validation-results.json),
[immutable stream pins](history/follow-up-18-stream/archive.json).

D-098 first: all **138** run 0ec14b39 root files/subtrees are byte-identical in
[follow-up 17](history/follow-up-17/archive.json). All earlier archives stay
unchanged. Owned containers, images, networks and volumes are zero.
