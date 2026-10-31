# D-132 evidence-read audit — 2026-10-31

Audited main `96ec0630e4834e13a3a14d65553ba8108bb5da55` and the follow-up 11 harness.
`git grep -n 'docs/e2e/at/' -- internal node packaging tests` returned 1,028
matches after the harness changes. This is a reference inventory, not 1,028
runtime reads. No product test consumes living AT-root evidence.

| References inspected | Actual use |
| --- | --- |
| `internal/fleetupdate/d062_integration_test.go:24`, `:73` | Two provenance comments name immutable follow-up 9 archives and their SHA-256 digests. Executable reads name `testdata/d062-harness-result-delivery.json` and `testdata/d062-harness-rollback-delivery-finding.json`. Both copies match the archive bytes. |
| `internal/nodes/publication_evidence_test.go:21` | Comment explicitly forbids the living secops input. The test reads `testdata/secops11_recorded_baseline.json`. |
| `internal/nodes/testdata/d126-mixed-observation.txt:6` | Provenance text only. `observations_followup32_test.go` and `httpwire/followup32_integration_test.go` read the frozen `d126-mixed-observation.json` fixture. |
| `node/`, `packaging/` | No literal `docs/e2e/at/` reference. |
| `tests/e2e/at/updates/` | Archive tests read only `history/follow-up-9`, `history/follow-up-10`, and the existing follow-up 2 fixture. The other references write or identify this lane's new evidence. |
| Other `tests/e2e/at/` matches | Lane evidence writers, readback/analysis of their own run, hygiene checks, immutable archive checks, and documentation citations. These do not feed product behavior tests. |
| `tests/handover/` | Archived claim inputs and provenance, documentation consistency checks (including living `secops/scope-review.json`), and negative path-policy examples. These are documentation/evidence tests, not product fixture reads. Living mentions are not all asserted to be comments. |
| `tests/netlab/evidence_test.go` | Positive fixture uses `members/history/follow-up-8/AT09.json`; living/lookalike paths are rejection stimuli for the immutable-pin validator. |

The four `internal/` references claimed by the hand-off are confirmed by source
inspection; their actual file-read sites were checked. No `internal/`, `node/`,
`packaging/`, shared harness, or other lane file was edited.

All 39 former living-root files for run `695f977c-b587-4768-88f1-0514f3a4f1ac`
are preserved byte-identically in [follow-up 10](history/follow-up-10/ARCHIVE.md),
with [SHA-256 pins](history/follow-up-10/SHA256SUMS). Product tests may pin only
history paths, never the new living root. Existing F10 diagnostic archives
remain unchanged. The audit's full grep output is retained in the worktree
report logs; no private runtime state is part of that inventory.

## Follow-up 12 archive and consumer closures — 2026-10-31

Before any follow-up 12 run, all **43** root files on main `45209609d33d567c5f04d49441799cdd3efac59b`
were copied byte-identically to [follow-up 11](history/follow-up-11/ARCHIVE.md).
Every copied byte was compared with its original main blob; the manifest keeps
the follow-up 10 `SHA256SUMS` shape. Earlier archives are untouched.

Updater 19 / #351 accepted run `daddad24-b6f7-4c32-ba8e-1c7d24f76ea2`
and deliberately cited living paths without digests. Its installer hand-off,
transport capture and uncached-origin observation now have immutable paths under
`history/follow-up-11/`. Installer 8 / #353's producer test and updater 19's
consumer test construct fixtures themselves; neither reads AT evidence.
Devstack 21 / #352 answers F11-01 in `docs/e2e/RESPONSES.md` under D-102.
The four `internal/` reference sites above remain unchanged; `node/` and
`packaging/` still contain no literal `docs/e2e/at/` references. Documentation
citations are not runtime reads. Any new digest-dependent consumer must use
this archive, never a new living-root digest. No outside-owned file was edited.
