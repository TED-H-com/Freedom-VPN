## Summary

Agent-core follow-up 30 (session launched 23:12 EDT on the #356 tip 45209609; 1 commit, 1 owned file, merge-tree clean, prepush-scan clean; no live suite, lab or Docker object). Urgent fix: on unchanged `main`, `TestUsageCheckpointRoundTrip` fails from 2026-11-01 00:00 UTC (`period_id 2026-10 does not overlap the interval`) — the handover 22 chain's gates failed on it at 23:09 EDT and four other chains were stopped before any push.

## Summary

Fixed the month-dependent `TestUsageCheckpointRoundTrip` without changing product validation. Checkpoint issuance, checkpoint verification and store timestamps now share a fixed `2026-11-01T00:00:00Z` clock. The usage interval is `[2026-10-31T23:59:00Z, 2026-10-31T23:59:10Z)` and its period comes from its own start. A real test run crossing UTC midnight cannot change these values.

The owned-test sweep found one required fix across 87 test files. Twenty-three other calendar-containing files were reviewed and left unchanged for the reasons below. No product, API contract, migration, dependency or other-lane file changed. No live suite, lab, Docker object or host-network operation was run.
