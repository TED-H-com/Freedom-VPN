# SECOPS-11: live gate and zero-binding publication verified

The real ready baseline, with capacity unset, accepted PUT as 202 and recorded
`data_path_probe=failed`, `NODE_NOT_SERVICEABLE`, `member_forwarding_unconfirmed`.
The operation was still running/retrying at that snapshot. This proves the step
refusal, not a terminal failed operation. After the finite capacity input, its
retry could recover; both snapshots remain in the record.

The next publication succeeded with an issued **published** generation containing
**zero device bindings**. All required operation steps succeeded after #148.
Publication with one genuinely admitted device also succeeded. No observation
report or projected row was written by the harness.

The real watchdog reported serving=1, reason=serving, with DNS/lease/QoS/source
binding checks passed. The observation-named lease generation matched applied
generation 9. The controlled native request failed and both per-device admission
counters stayed at zero. This keeps AT29 and full AT31 blocked on the devstack
follow-up 9 counted-flow gate; no forwarding diagnosis beyond these counters.

SECOPS-13's exact owner test passes against its frozen testdata fixture (#130),
including after this live record changed. [JSON record](publication.json),
[live evidence](live.json), [component results](components.json) and
[requests](requests.md) preserve command, UTC, incarnation and digest evidence.
