# Observation authority and stale ACK — follow-up 7

Run `46c64443-af2d-4295-a3a9-5e6c69bc4828`, source `c8d4a00cf1bf594b20016263c2cfc3c164d92bc9`, evidence `2026-10-30T17:57:42Z`. Prefix `w4-updates`, port base 42000.

[Raw requests, batch GETs, audit pages, job timestamps, results and ACKs](pre-result.json).

| Branch | Result |
|---|---|
| Fresh a: unrelated release in both columns | Columns stay null; exactly one unexpected-release/job audit despite four reports |
| Fresh b: target current / unrelated last-good | Columns stay null; exactly one audit for the distinct unrelated last-good |
| Fresh c: target current / boot last-good | Columns stay null; exactly one boot/job audit |
| Fresh agreeing null last-good | Current seeds to the issued target; last-good stays null; no accepted result or additional audit |
| First accepted 1.0.1 success | Result establishes current and last-good |
| Recorded c1: recorded current | Columns unchanged; zero seeding audits |
| Recorded c2: later 1.0.2 job target | Current/last-good remain 1.0.1; unconfirmed=true; job last_observed_at=null; zero seeding audits |
| Recorded c3: unrelated 1.0.3 | Same retained values and withheld confirmation; zero seeding audits |
| Real 1.0.2 completion | Result establishes current/last-good as 1.0.2 and later observation confirms |
| Replay genuine old success ACK while new job waits | Exact original ACK returned; current job, desired/observed generations and releases unchanged; no new receipt |

> Heartbeats after acceptance can confirm those
> values, but cannot change current/last-good or clear recovery.

Source: `docs/updates/central-lifecycle.md:418`.

> Once a result is accepted, seeding closes even for later jobs in that incarnation.

Source: `docs/updates/central-lifecycle.md:467`.

> A mismatch in
> either release keeps `unconfirmed=true` by withholding that job's observation
> timestamp; a later agreeing report can confirm it.

Source: `docs/updates/central-lifecycle.md:419`.

> Each distinct unexpected
> release/reported-job pair gets one `update_observation.unexpected_release` audit

Source: `docs/updates/central-lifecycle.md:442`.

> A later agreeing null
> last-good permits the issued target to seed current, without establishing
> last-good.

Source: `docs/updates/central-lifecycle.md:463`.

The recorded branches use four fresh report IDs each and retain actual baseline
checks. No projection or journal-health fields are written by the harness.
F6-02 is closed by the orchestrator's corrected expectation, #TBD.
