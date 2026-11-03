# AT10–AT13 isolation evidence

Follow-up 17 run **`0ec14b39-467e-4a37-a239-1a33ef12eb71`**, assembled exit **2**:
AT10 **9/0/0**, AT11 **2/1/0**, AT12 **10/0/0**, AT13 **6/1/0**
(pass/blocked/fail). ISO-F6-01 **6/6**. AT11/AT13 operator matrices remain blocked.

D-140 **qualifies under D-142 and passes the unchanged evaluator**, with no
overlap anywhere in the sweep. Shared D-104 passes on netlab 26's fixture.
The original seven hold categories and central's 30-second fallback pass;
pure `stale_generation` is not exercised. Six agent parent receipts record
exit 0; zero installs exceed 100 ms. Historical failures stay failed; no retry.

[Summary](summary.md), [procedure](RUN.md), [hold row joins](populated-holds.md),
[boundary qualification](boundary-review.json), [findings](follow-up-17-findings.json),
[parent receipts](stop-receipt-review.json), [nightly diagnosis](nightly-diagnosis.md),
[requests](requests.md), [validation](validation-results.json),
[immutable stream pins](history/follow-up-17-stream/archive.json).

D-098 first: all **129** run 83b457ef root files/subtrees are byte-identical in
[follow-up 16](history/follow-up-16/archive.json). All earlier archives stay
unchanged. Owned containers, images, networks and volumes are zero.
