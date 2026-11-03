# AT10–AT13 isolation evidence

Follow-up 16 run **`83b457ef-ea16-490b-9923-1fea0df86b36`**, assembled exit**2**:
AT10 **9/0/0**, AT11 **2/1/0**, AT12 **10/0/0**, AT13 **6/1/0**
(pass/blocked/fail). ISO-F6-01 **6/6**. AT11/AT13 operator matrices remain blocked.

D-140's evaluator passes; qualification **fails** on one **3.609 µs** overlap.
The separate shared D-104 fixture fails before its packet trace. Both attempts
remain failed, with no retry. Historical failures stay unchanged. Populated
stale-lease seam rows pass **seven coverage categories across two incidents**;
all five agent parent stop receipts record exit 0.

[Summary](summary.md), [procedure](RUN.md), [hold row joins](populated-holds.md),
[boundary qualification](boundary-review.json), [findings](follow-up-16-findings.json),
[parent receipts](stop-receipt-review.json), [requests](requests.md),
[validation](validation-results.json), [immutable stream pins](history/follow-up-16-stream/archive.json).

D-098 first: **123** run 0b9e4510 root files and subtrees are byte-identical under
[follow-up 15](history/follow-up-15/archive.json). All earlier archives remain
unchanged. Owned Docker containers, images, networks and volumes are zero.
