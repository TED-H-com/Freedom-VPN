# Scheduled isolation job — Follow-up 17, 2026-11-03

Read-only investigation used `gh run view <id> -R TED-H-com/Freedom-VPN --log`
for [41304317016](https://github.com/TED-H-com/Freedom-VPN/actions/runs/41304317016)
and [41456092596](https://github.com/TED-H-com/Freedom-VPN/actions/runs/41456092596).
Neither workflow was triggered. The runs' artifact API lists both have total_count=0.
Raw workflow logs remain private because they contain runner paths and build state.

Both logs contain these four blocked row lines (the AT10 line is provisional pending the final six-assertion review):

| Row | Nov 1 | Nov 2 | Expected on this job? |
|---|---|---|---|
| AT11 IPv6 platform matrix | positive=true | positive=true | Yes: full native platform route/capture matrix is operator work. Container IPv6 capability does not prove it. |
| AT13 continuous crash/NetBird/reboot/updater transitions | positive=true | positive=true | Yes: full fault/physical-device matrix is operator work; container restart proves its own row only. |
| AT13 short whole-node lease expiry | positive=false | positive=false | Not established. Leave off expected-blocked list. |
| AT10 foreign ingress redirect containment and D-096 recovery | provisional positive=true | provisional positive=true | This line precedes final review; neither final aggregate reports AT10 blocked. Do not treat the intermediate line as an expected block. |

The final `TestIsolationAssembled` aggregate logs only AT11 and AT13 as blocked.
The foreign-redirect row starts blocked while awaiting receiver export;
`reviewFollowup8` can then set it to pass from the completed six-assertion review.
The provisional log line alone cannot prove a final AT10 block.

The expiry line follows the finite-member-expiry pass by about 0.3 seconds in
both logs. That timing is not a diagnosis. The old `add` logger prints only row,
status and positive validity. It does not print `Check.Actual`, its owner or the
positive-control observations. There are no uploaded records to recover those
fields. `wholeNodeExpiry` has several distinct blocking returns (login, route,
lease acquisition, measurement window and counted control); the logs cannot
select one safely. A positive=false field does not establish a GitHub capability
limit. The exact root cause remains **unproven**, and the job must remain red if
the unexplained expiry block recurs in final records. Follow-up 17 adds the actual reason/owner to
row logs and a records-only artifact for the next naturally scheduled run.

The two review failures differ:

- Nov 1: after `TestIsolationAssembled` fails, `review_followup10.py` validates
  producer data despite expiry not having run, and raises
  `ValueError: invalid lifecycle resource block`. No boundary review is reached
  in that log. This invalid producer input is not diagnosed from absent artifacts.
- Nov 2: the producer review exits with its refusal; `review_boundary.py` then
  raises `ValueError: Current attempt did not execute expiry; refusing inherited boundary evidence`.

Both scripts now first write a current-run JSON record with status `blocked`,
reason `expiry not executed` and the original check. They return nonzero with
one line, before reading any inherited packet, schedule or producer file.
Tests provide deliberately invalid inherited data and require that exact result.
No boundary evidence is inherited or silently passed.

[D-144 expected-blocked entries](../../../../tests/e2e/at/isolation/expected-blocked.json)
name each established row once with reason, owner and how the expectation was
established. Only scheduled jobs enable the exit mapping; PRs and manual/local
runs retain strict behavior. Every fail, error, traceback or unlisted block
fails the scheduled job. A passing listed row is reported as unblocked for list
removal. Acceptance records and the operator rows remain blocked as measured.

The always-run upload step targets only `.agent/nightly-records/`, a fresh output
directory containing the harness's redacted records. It does not target stack
state, cookies, credentials, raw workflow logs or the rest of `.agent`. Retention
is seven days. `scheduled-job.json` and `scheduled-job.md` retain each row's
status, reason, owner and job mapping plus nonzero phase exits; the latter is
also appended to the GitHub job summary.
