## updater — 2026-10-31 (follow-up 21, F18-01, D-102)

[The negative scenario contract](scenarios/updater-negative.md) adopts updater
18's stopped-producer interpretation: apply/replay/recover **1/1/2**, sealed
`CHECKPOINT_FAILED`, blocked forwarding, unchanged baseline/pointer and receipt,
no candidate success. The old 13/0/1 campaign stays failed and archived. Future
negative D-125 coverage must require those facts separately from the unchanged
positive upgrade assertions. No updater behavior or assertion is relaxed.
