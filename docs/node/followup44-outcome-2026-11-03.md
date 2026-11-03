# Follow-up 44 outcome — 2026-11-03

| Leg and window | Pooled (reason if not) | Scenarios passed | Marked intervals | Retained / excluded whole Step | Outer guard p99 / max (ms) | Whole serving Step p99 / max (ms) | Failed operations |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A1 / 1 | No: foreign-container preflight | 0/13 | 0 | 0 / 0 | — / — | — / — | 0 (no scenario started) |
| B1 / 1 | No: not started after window stop | 0/13 | 0 | 0 / 0 | — / — | — / — | — |
| A2 / 1 | No: not started after window stop | 0/13 | 0 | 0 / 0 | — / — | — / — | — |
| B2 / 1 | No: not started after window stop | 0/13 | 0 | 0 / 0 | — / — | — / — | — |
| A3 / 2 | No: clock-reader setup failed; no Step spans | 0/13 | 0 | 0 / 0 | — / — | — / — | 0 producer; 1 harness failure |
| B3 / 2 | No: not started after window stop | 0/13 | 0 | 0 / 0 | — / — | — / — | — |
| A4 / 2 | No: not started after window stop | 0/13 | 0 | 0 / 0 | — / — | — / — | — |
| B4 / 2 | No: not started after window stop | 0/13 | 0 | 0 / 0 | — / — | — / — | — |

No timing comparison is possible: both pooled arms contain zero legs and all
five retained distributions are empty. No attributable B shift or B-only
production failure is established. D-133's post-merge measurement remains open.

Window 1 reserved immediately (0.037303 s), drained 60 seconds, then refused
A1 because `fvpn-w1-netlab-f34-before` was running. Admission printed free,
but the unchanged foreign-container rule prevented any scenario. It released
at 18:24:38.969843Z. No other container was changed.

Window 2 first requested admission 600.040058 s after that release. It waited
90.154318 s through three refusals, reserved at 18:36:09.124642Z, drained
60 seconds and admitted A3. Four in-leg samples plus preflight satisfied the
non-CPU rules, but the clock reader failed on an admission `free` message
prepended to Docker stdout. Make exited 2 after 14.106588 seconds; no scenario
or producer span ran. Its raw image field is likewise contaminated (`free`
plus the SHA-256), and is preserved as received, not claimed as a clean
parsed identity. The image build log separately records manifest list
`sha256:7c313978f98f3cb75a64e77c3520d0b4e4a00cbf38ed8058415d547970f00263`.
Window 2 released at 18:37:28.006636Z. No third reservation or timing retry.

There were six qualification samples, zero CPU marks, zero excluded spans and
zero producer failed operations. These zero counts reflect no scenario data;
they are not success or latency evidence. Source, available A binary/fixture
identities, raw hashes, refusals, samples and all five empty per-leg/pooled
statistics are in the [results](followup44-timing-results-2026-11-03.json).

The actual wrapper regression reproduced the JSON failure. Moving admission
diagnostics to stderr fixes it; all 24 synthetic tests pass. The measured and
delivered runner hashes are distinct in the [repair record](followup44-runner-repair-2026-11-03.json).
The original declarations and both window records are preserved.

A separate post-reservation probe then measured runner time namespace
`time:[4026531834]` and lab namespace `time:[4026534102]`. All three monotonic
brackets aligned and both offsets were zero, but the declared no-separate-time-
namespace prerequisite failed. The [clock receipt](followup44-clock-prerequisite-2026-11-03.json)
retains the measurements. That preliminary ordinary setup aborted before
scenarios; it is neither a timing leg nor a gate pass. No clock or host setting
was changed. Resolve this prerequisite before any newly declared comparison.

The subsequent **ordinary main lab passed all 13 scenarios**, including
D087/D104, without overlays or a timing-only namespace gate. Source main is
`1b87ab32344fa3be3216df0d5f7b489eb853e011`. Its clock probe was not linked into the
agent. This is functional evidence only. Expiry predicate: True;
97 positives, 15 drops, last positive offset
-100.328716 ms and first drop offset
-63.815657 ms, with
0 positive egresses at/after expiry.
Renewal: 2 actual renewals,
1518/1518 delivered,
0 losses (0 ingress inside
guards, 0 outside); zero-loss predicate
True. No D-120 tolerance is added. Every diagnostic
failed operation and packet loss remains in the [functional receipt](followup44-production-lab-2026-11-03.json).

| Gate | Exit | Seconds |
| --- | --- | --- |
| node-lint-build | 0 | 3.221 |
| node-test | 0 | 153.182 |
| node-integration | 0 | 152.398 |
| reviews-r16-r19 | 0 | 19.898 |
| reviews-r20-r26 | 0 | 44.038 |
| handover | 0 | 3.743 |
| go-lint | 0 | 2.188 |
| vet | 0 | 0.421 |
| all-tests | 0 | 145.467 |
| evidence | 0 | 2.791 |
| fixture-hygiene | 0 | 8.833 |
| final-all-tests | 0 | 149.742 |
| ordinary main production lab (13/13) | 0 | 227.301 |
| synthetic monitor/stream tests (24) | 0 | — |

Node builds include linux/amd64 and linux/arm64. R16–R19, r20–r26, handover,
root pinned staticcheck/vet, plain whole-module tests and evidence hygiene pass.
The [validation record](followup44-validation-2026-11-03.json) retains real tails,
hashes, source identities, environment and zero final owned inventory. No
production file, node/enforce file, dependency, earlier record or other lane
was changed. Physical arm64 and field-network measurement remain outside this run.

## Requests to orchestrator — 2026-11-03

- Arrange a later declared D-133 comparison only after resolving the measured
  namespace prerequisite. Both reservations here are exhausted; no stopped
  or ordinary functional attempt may be pooled. No production fix is inferred.
- Review admission/teardown coordination: window 1 received free admission
  while a foreign lane container remained running. This script and the foreign
  container are outside this lane's paths and were not changed.

An unlocated D087/D104 failing operation conservatively stops the delivered
runner; only a located operation's interval may excuse it. The frozen install-
bound assertion has exact endpoint attribution. No arbitrary whole-test window
is used to classify a failure as marked.
