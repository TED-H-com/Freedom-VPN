# Enrollment acceptance

Every required check must pass. Missing hooks keep the complete AT blocked.

| AT | Status | Decisive observation | Evidence |
|---|---|---|---|
| AT06 | pass | Prior run bebfe230 429 RATE_LIMITED failures on audit_read, issue_reinstall, incarnation_history and probe_process are reviewed as resolved under D-078. Harness Retry-After retries landed in 286eb57b44c2e783257767918680a0e1f097f106 (#210). Run 60a7102e is re-scored in this record: raw http_429 2, both class browser, named checks passed, join phase http_429 0. The saved samples do not include a Retry-After value. Run 35e542f8 probe_process and rate_limit_review stay in prior_checks and stop blocking on that behaviour evidence. Run 1d5c02a4 probe_process was the clone-present parser (wget dropped the 403 body); it stops blocking when this probe exits 0. Limiter class was browser BrowserRate 10/s burst 30. | [AT06.json](AT06.json), [AT06.md](AT06.md) |
| AT07 | blocked | ENR-05 stays blocked, waiting on devstack f15. Expected exports: HoldUsageAcknowledgements(ctx, slot, hold), UsageLedgerSnapshot(ctx, slot), CountedMemberTraffic. fvpn-agent usage snapshot [-json] is merged and is not these hooks. Owner: w4-usage, then devstack. Not emulated. | [AT07.json](AT07.json), [AT07.md](AT07.md) |

See [RUN.md](RUN.md) and [requests.md](requests.md). Immutable attempts are in `history/`.

## Follow-up 4 — run `ec2e6279-0649-4d47-b31a-cf13aeba7303`

UTC `2026-10-31T02:18:00Z`–`2026-10-31T02:23:47Z`. Source `9e493e8` dirty, tree `d57359f925f2860a4efab951ed1c6d10025bfe916d3ff4474b4a57203437d2d5`. Both probes exited 0. Cleanup `{containers:0,networks:0,volumes:0,images:0}`. `make -f mk/at-enrollment.mk at-enrollment` exited 2 because AT07 is blocked.

AT06: 305 passing, 0 blocked, 0 failed. Full-phase `rate_limit_review` passed with `http_429` 2, both class `browser`, `Retry-After` present, and the retry succeeded. Join phase `http_429` 0. Follow-up 3 run `60a7102e` is re-scored in the JSON; its raw count stays in `prior_checks`, and the saved samples have no `Retry-After` value. Disposition `resolved`.

AT07: 16 passing, 6 blocked, 0 failed. `cloned_identity_conflict` passed on `fvpn-w4-enroll-node-clone-net` (`172.29.38.0/24`): one frozen conflict, one audit, stable count, forwarding withdrawn, revoke 202, new incarnation `01a18fc3-386b-716e-960f-fa63faeb507a`, cleared projection, and the clone's old certificate received 403 `IDENTITY_REVOKED`. The six blocked checks are ENR-05 and ENR-06, waiting on devstack f15. The entrypoint re-enroll is not ENR-06. `same_lab_slash24` and `same_origin_slash24` stayed true because the clone netns remains on those bridges; the conflict domain is the presentation `/24`.

Run `1d5c02a4-b254-4a9b-b687-b1535428fb3f` is the earlier attempt in this follow-up. Its process exit is the clone-present parser (wget dropped the 403 body). That history directory is unchanged. The `IDENTITY_REVOKED` pass is this run only.
