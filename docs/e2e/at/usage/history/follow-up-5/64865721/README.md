# Diagnostic run 64865721

AT14/16/17/18/19 passed. AT20 stopped at the harness requirement that the entire revoke operation succeed, before reenrollment. AT15 was not reached.

Read-only operation evidence for `01a191ca-3184-7175-9e77-654d5a964e89` showed `partially_succeeded`: `netbird_route_readback` and `netbird_identity_cleanup` succeeded, while `agent_policy` failed with DEPENDENCY_UNAVAILABLE / “central NetBird route withdrawn; node agent withdrawal unconfirmed”. The old agent had deliberately been stopped in the documented quiescence order; it could not send a later policy ACK. The origin was centrally retired.

The next run checks precisely the two external worker effects required by RESPONSES.md step 3 and preserves the complete operation status. It does not turn a partial operation into a completed one or send a fabricated agent ACK. This interrupted run remains failed.
