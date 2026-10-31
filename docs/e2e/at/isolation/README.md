# AT10–AT13 isolation evidence

Follow-up 9 run **`2be2797b-7a28-4e9f-a1fe-5fffa4c1b047`**:
**AT10 blocked, AT11 blocked, AT12 pass, AT13 fail.**

Exact producer timing correlates all three new renewal losses with install
brackets. Whole-node expiry has 96 early drops followed by later forwarding;
95 of those drops are outside completed guards. The strict predicate stays
failed. Historical 17 losses are preserved without retrospective attribution.
Recovery passes all six assertions; 557/557 D-106 digests match. The separate
AT10 reload stream passes under unchanged D-120 with one eligible loss.

[Summary](summary.md), [procedure](RUN.md), [requests](requests.md),
[producer analysis](producer-timing-analysis.json),
[expiry failure](expiry-failure-correlation.json), [validation](validation-results.json).
D-087 remains blocked on wire-nodes 33 → devstack 20 → isolation 10.

Corrected immutable archives:
[follow-up 6](history/follow-up-6/archive.json),
[follow-up 7](history/follow-up-7/archive.json),
[follow-up 8](history/follow-up-8/archive.json).
No handover claim pin moved. Mechanism evidence under `lab-*` / `reload-stream/`
does not replace assembled acceptance or the real-device matrix.
