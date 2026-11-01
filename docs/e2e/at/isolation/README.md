# AT10–AT13 isolation evidence

Follow-up15 run **`0b9e4510-776f-4c51-9e66-500404ce9f7e`**:
AT10 **9/0/0**, AT11 **2/1/0**, AT12 **10/0/0**, AT13 **6/1/0**
(pass/blocked/fail). ISO-F6-01 **6/6**. AT11/AT13 operator matrices remain blocked.

**D-140 schedule qualification remains incomplete:** the unchanged whole-node
packet evaluator passes, but one ingress/egress interleaving is retained.
The post-run spacing correction is not assembled-qualified. Historical
**isolation F14-01** stays failed and is distinct from **members F14-01**.
No retry or loss waiver. Populated hold: **not exercised — devstack 27's fixture pending**.

[Summary](summary.md), [procedure](RUN.md), [boundary qualification](boundary-review.json),
[parent stop receipts](stop-receipt-review.json),
[pre-Apply reservation timelines](pre-apply-reservation-review.json),
[central hold review](generation-holds-review.json), [requests](requests.md),
[validation](validation-results.json), [immutable gzip pins](history/follow-up-15-stream/archive.json).

D-098 first: all **117** run-56d20530 root files and subtrees are byte-identical
under [follow-up14](history/follow-up-14/archive.json). All earlier archives and
historical failures remain unchanged. Owned Docker inventory is zero.
