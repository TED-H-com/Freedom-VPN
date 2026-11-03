# Isolation follow-up 18 requests — 2026-11-03

Three carried items and zero new requests.
Run `9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04`; [evidence](summary.md), [dated nightly diagnosis](nightly-diagnosis-2026-11-03.md).

1. **Carried — AT11 platform/operator assignment.** Full native member-platform
   route/capture matrix stays blocked. No renewed ask.
2. **Carried — AT13 acceptance/operator assignment.** Full continuous crash,
   NetBird restart, kernel reboot, updater and physical-arm64 matrix stays blocked.
   Six exit-0 receipts do not close it. No renewed ask.
3. **Carried — w3-wire-nodes / w2-agent-core, F17-01: pure `stale_generation`.**
   The 13 accepted holds in run `9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04` classify
   **10 check_failed / 3 isolation_unconfirmed**; pure classification remains
   **not exercised** ([raw checks/errors](follow-up-18-findings.json)). No renewed ask.

Both unlisted blocked rows from scheduled run **`41640196805`** (`AT13 short
whole-node lease expiry` and `AT10 foreign ingress redirect containment and D-096
recovery`) were caused by a single owned harness defect in
`tests/e2e/at/isolation/` (`Outcome (a)`), are fixed on this branch, and pass in
the local assembled verification run (`9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04`).
Neither row requires a request to another task (`Outcome (b)`) or addition to
`expected-blocked.json` (`Outcome (c)`).

All owned Docker inventories are zero. No install exceeded the 100 ms bound in
this run. No outside-owned edit or handover pin change was needed.
