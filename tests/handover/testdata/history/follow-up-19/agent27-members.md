# D-126 producer confirmation for members 10 — 2026-10-31

The F9-01 report is a deliberate composition in a single synchronous decision,
not a race or a merge of two reporters. D-126 documents the existing behavior;
no HTTP contract or precedence change is needed. In members run `11e305fe`,
`2026-10-31T12:06:32.527615Z` is the receiver time; the mixed body's
`observed_at` is `2026-10-31T12:06:32.521093013Z`. Its retained tc read failed
on the member interface's qdisc listing (`read_failed`). The earlier report's
probe also failed; “provisional” alone does not mean safety passed.

`Engine.Run` calls `Engine.Step` serially (`engine.go`; concurrent Step is
explicitly unsupported). `syncLease` / `acceptLease` (`sync.go`) accept and
persist the fresh, matching generation-30 empty lease and arm the grant-less
renewal schedule. They do not write a separate observation. `evaluate` and
`Decide` compose the decision from that lease and the retained enforcement
failure. On the initial provisional pass, `enforce` installs the baseline.
The deferred `finishServingStep` (`serving.go`) calls `auditBlocked` on the
retained applied model; a failed tc read calls `noteApply` (`engine.go`),
quarantines, recomputes `Decide`, and emits a new report if needed. This explains
the two close reports on this restart without concurrent reporters. The
`ForwardingBlocked` / `meterBarrierIdentity` helpers in `gate.go` read the
updater latch only; `latch` feeds that gate to `Decide`. They have no reporting
goroutine and do not overwrite lease/check errors.

The exact composition rules in `decide.go` are:

- `Decide` accumulates distinct reasons. A retained Apply/verification error
  adds `safety_check_failed`. `grantsProvisional` requires desired bindings and
  a fresh, matching, still-in-force empty lease; it adds
  `awaiting_member_grants`. It does not add `lease_expired`.
- `checks` first evaluates serving authorization, then overrides the `lease`,
  `qos` and `source_binding` codes with a retained enforcement failure (or
  missing dependency). Thus `TC_FOREIGN_FILTER` outranks
  `AWAITING_MEMBER_GRANTS`, which outranks ok. These checks all depend on a
  verified enforcement path. Other checks keep their own causes; the captured
  `tunnel_probe` is `HEALTH_CHECK_FAILED`.
- Stage errors retain the stage's cause: the **sole stage=lease error** remains
  `AWAITING_MEMBER_GRANTS`; the probe error remains `HEALTH_CHECK_FAILED` and
  stage=apply remains `TC_FOREIGN_FILTER`. “Sole stage=lease” does not mean
  there is only one error in the entire report. Containment appends its
  explicitly historical tc/quarantine rows separately.
- `safetyOf` selects failed while the retained enforcement/probe failure is
  active. `report` (`observe.go`) copies the decision checks and attaches
  diagnostic evidence without changing their status/code. `observe` serializes
  one body into the durable outbox before another Step; retries retain that
  report ID and body. A later clean read cannot rewrite the queued report.

For members 10's AT09 revision-11 predicate, require the awaiting reason and
lease-stage error while the lease is provisional, but permit a more severe
active safety code on the three dependent checks. Preserve the simultaneous
safety reason and its stage evidence. Require all three awaiting check codes
for a **clean** provisional report. Do not turn a safety failure into a pass,
call the in-force provisional lease expired, or reject this captured mixed
report. After clean recovery the awaiting checks can return, and only actual
signed grants plus verified enforcement can restore serving (§16.3, §19 AT09).

`TestD126_AT09_CapturedProvisionalSafetyComposition` replays the captured
inputs (generation, lease times, desired bindings, failed probe and retained tc
failure). It asserts the exact two reasons, every check's status/code, all
three decision-stage errors with their own times, no serving, and the adjacent
clean provisional composition without mutation of the earlier report. Test
identities are synthetic; the captured report remains unchanged in members'
owned evidence. No producer race fix is claimed or necessary.
