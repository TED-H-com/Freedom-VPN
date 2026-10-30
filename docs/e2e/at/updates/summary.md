# Updates follow-up 7

Run `71078ab8-edf8-4d23-af72-ea247f704011`, source `c8d4a00cf1bf594b20016263c2cfc3c164d92bc9`, evidence `2026-10-30T18:09:36Z`. Prefix `w4-updates`, port base 42000.

Fresh AT24, AT26 and AT27 **pass**. Make exits **2** (Go test 1) only because
AT23/AT25 retain their follow-up 6 blocked records. All records are written
before the nonzero exit and teardown.

| AT | Status | Decisive observation |
|---|---|---|
| [AT23](AT23.md) | blocked, retained | F5-02: w3-devstack-e2e follow-up 12 (probe-peer export); not rerun |
| [AT24](AT24.md) | pass | Offline target progresses waiting_online → queued → succeeded through a real result; the post-batch node's separate first-sync job also succeeds; original denominator stays 1 |
| [AT25](AT25.md) | blocked, retained | Same F5-02 owner; not rerun |
| [AT26](AT26.md) | pass | Genuine candidate failure restores healthy 1.0.1; one current rollback ACK; confirmed target with revision 7 evidence |
| [AT27](AT27.md) | pass | Accepted values remain unchanged, mismatches withhold the job observation timestamp, zero post-acceptance seeding audits; fresh unexpected claims audit once; stale ACK cannot rewind the newer job |

AT24's restated authority criterion:

> Heartbeats after acceptance can confirm those
> values, but cannot change current/last-good or clear recovery.

Source: `docs/updates/central-lifecycle.md:418`.

> Once a result is accepted, seeding closes even for later jobs in that incarnation.

Source: `docs/updates/central-lifecycle.md:467`.

AT27's restated confirmation/audit criteria:

> A mismatch in
> either release keeps `unconfirmed=true` by withholding that job's observation
> timestamp; a later agreeing report can confirm it.

Source: `docs/updates/central-lifecycle.md:419`.

> Each distinct unexpected
> release/reported-job pair gets one `update_observation.unexpected_release` audit

Source: `docs/updates/central-lifecycle.md:442`.

The fresh-null case also proves that a later agreeing null last-good permits
current to seed to the issued target while last-good remains null. Only the
real accepted result establishes last-good. F6-02 is resolved by the corrected
expectation in this follow-up (**#TBD**, to be filled by the orchestrator).

AT23/AT25 keep run `a83d5d64-505c-488c-b560-5a777566702e` and its evidence timestamps;
their evidence is not relabeled as this run. Probe evidence remains from
follow-up 6 and is not rerun or worked around here.

[Raw authority/stale-ACK evidence](pre-result.json), [offline and inheritance
results](inheritance.json), [run and per-record provenance](run.json),
[requests](requests.md), [how to run](RUN.md), and [follow-up 6 history](history/follow-up-6/README.md).
The [first diagnostic run](history/follow-up-7/first-run/README.md) is retained: its AT27 gate incorrectly expected HTTP 200 for a replay that correctly returned the contract's 202. The assertion was corrected and all fresh cases rerun.
The [subsequent functional pass](history/follow-up-7/provenance-run/README.md) has accurate harness file hashes but an incorrect source-difference flag. Git now runs from the repository root, and this final run verifies both hashes and the flag.
Full original §19 coverage, including the additional pre-existing enrolling
snapshot member and physical arm64, remains outside this focused follow-up.
