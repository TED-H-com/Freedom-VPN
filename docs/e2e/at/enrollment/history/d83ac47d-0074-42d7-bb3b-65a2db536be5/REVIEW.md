# Follow-up 7 diagnostic attempt review

TEST ONLY, 2026-10-31. Source `123a5d098b4ebb35a1819f1f7a41445e2ae16a87`.
The original JSON and Markdown records are retained unchanged.

AT06: **245 pass / 1 blocked / 1 fail**. AT07: **34 pass / 5 blocked / 2 fail**.
The full probe completed; the join phase did not start. While admission was
occupied, the idle first stack was torn down through the normal harness. The
runner was then interrupted during admission waiting so the corrected attempt
could run both fresh phases. Its finally block persisted these records and
performed another successful teardown. Inventory is zero in every category.

The only failed assertion is `serving_freeze_source`: the harness passed the
container name as the primary member's NetBird peer name. The fixture actually
uses `fvpn-member-e2e` (`deploy/devstack/entrypoint-member.sh`), so native lookup
timed out despite real API authorization and a populated lease. The resulting
exit 1 accounts for both ATs' `probe_process` failures. The unrun join is the
blocked AT06 `peer_uniqueness` row; it is not a successful fresh join.

An independent positive control still read one live lease member, a fresh
serving heartbeat and six member accept rules before the clone. Withdrawal
within 350 ms of the central conflict and restart persistence passed. All seven
new installer checks passed. These measured rows stay as originally recorded;
the failed setup and shared process outcomes are not relabeled.

The correction selects the primary fixture's actual peer name. Resolving these
exact historical failures requires a new passing source check, freeze withdrawal
and restart, and passing AT06/AT07 probe processes. Unknown failures remain
blocking. The subsequent attempt's JSON carries the explicit review disposition.
