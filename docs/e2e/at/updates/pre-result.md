# Pre-result heartbeat injection — follow-up 5

Run `743c186d-5765-43e9-8b40-7839be178e13`, source `dd72ed4e43e2be21a852e6741d8caa06a8f6ad48`, at `2026-10-30T08:35:19Z`.
Raw requests, HTTP responses, batch GETs, audit pages and post-success evidence:
[pre-result.json](pre-result.json). Every case has an initial heartbeat and
three fresh-ID repetitions, all HTTP 202, with zero accepted results throughout.

| Case | Central current after each heartbeat | Central last-good | Audit count under this job |
|---|---|---|---|
| (a) unrelated registered release in both fields | null (unchanged) | null (unchanged) | 1 for the unrelated current release |
| (b) job target current, another unrelated last-good | job target | null (unchanged) | Previous 1, plus 1 for the unrelated last-good release |
| (c) job target current, boot release last-good | job target | null (unchanged) | Previous 2, plus 1 for boot; expected none for boot requires a recorded-boot prerequisite that is absent |
| Real AT23 success result | job target | job target | Exactly 1 accepted result establishes last-good |

The current/last-good columns started null. The agent's real baseline report
named the configured boot release, but it had no job-bound update evidence and
had never submitted an accepted result. The harness did not manufacture a
recorded boot version. AT24 and AT27 remain blocked for that requested fixture;
see request F5-01. The bounded-audit implementation emits one per pair even
when current and last-good repeat the same unexpected release.
