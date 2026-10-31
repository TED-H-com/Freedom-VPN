# Enrollment acceptance

Every required check must pass. Missing hooks keep the complete AT blocked.

| AT | Status | Decisive observation | Evidence |
|---|---|---|---|
| AT06 | pass | Prior run bebfe230 429 RATE_LIMITED failures on audit_read, issue_reinstall, incarnation_history and probe_process are reviewed as resolved under D-078. Harness Retry-After retries landed in 286eb57b44c2e783257767918680a0e1f097f106 (#210). Run 60a7102e is re-scored in this record: raw http_429 2, both class browser, named checks passed, join phase http_429 0. The saved samples do not include a Retry-After value. Run 35e542f8 probe_process and rate_limit_review stay in prior_checks and stop blocking on that behaviour evidence. Run 1d5c02a4 probe_process was the clone-present parser (wget dropped the 403 body); it stops blocking when this probe exits 0. Limiter class was browser BrowserRate 10/s burst 30. | [AT06.json](AT06.json), [AT06.md](AT06.md) |
| AT07 | blocked | ENR-05: #266 HoldUsageAcknowledgements, UsageLedgerSnapshot and MemberTransportStack accept node/node2/node-clone, while ControlledReinstall accepts only installer-node. Owner devstack: export the hold, snapshot and counted member transport on that same installer slot, including its proxy/config paths and service restoration. Owner usage/installer: supply an authorized old-incarnation backlog delivery protocol; packaging/installer/reenroll.go explicitly leaves retained old events unuploaded. No central-once or cross-reinstall usage claim. | [AT07.json](AT07.json), [AT07.md](AT07.md) |

See [RUN.md](RUN.md) and [requests.md](requests.md). Immutable attempts are in `history/`.
