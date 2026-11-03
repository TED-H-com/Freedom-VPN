# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 32](../handover/follow-up-32.md) (2026-11-03 EST):** Through #412: F27-02 fixture defect, qualified comparison open; first live D-102 survivor continuation; B1 unresolved. F28-01 and F31-01 routed, not yet judged. AT02 / AT09 stay failed; F30-01 has no merged fix. Nine owner requests.
**Previous cut — [follow-up 31](../handover/follow-up-31.md):** #404–#407 at 3573f396; 21 subtests, four rows, nine handover snapshots, nine nightly inputs, 596 checks, three dated documents and 72 findings.

## Follow-up 32 acceptance snapshot (2026-11-03 EST)

Cut **#408–#412**, main **b7449ec1**; no unbriefed PR inside the range or later
merge at task start. [Evidence and nine owner requests](../handover/follow-up-32.md).

| Lane / PR | Result and boundary |
|---|---|
| Updater 20 (#408) | F27-02 fixture defect: unchanged one-second bound now measures the persisted wait deterministically. Four 200-iteration runs unqualified, load never started; qualified comparison stays open. Historical failure stays failed. F30-01 signal proposed only. |
| Handover 31 (#409) | #404–#407 at 3573f396; 21 subtests / four rows / nine handover snapshots / nine nightly inputs / 596 checks / three dated documents / 72 findings / nine owner requests. One codex gpt-6-astra session, 03:07 EST launch, gates after 39-minute admission wait. |
| Members 17 (#410) | 81-file archive plus 18 relative symlinks. 81b638e8 on #406, no retry: 2 pass / 6 blocked / 2 fail, 691/7/5. First live D-102 survivor continuation; 401 did not recur, old cause unknown. AT02 measurement_valid=false, three failed Applies, 1,253 losses counted; AT09 D-067 and combined restart fail, 6.01 s / 12.70 s at generation 63. Separate source-address row not exercised. |
| Netlab 28 (#411) | 100 reloads / 101 Applies / 118 reads, zero TCP failures; 11 retransmissions / seven reads and RTO backoff are a candidate mechanism only. B1 stays FAIL, F27-01 unresolved. AT12 deny-phase +97 bytes fails full-lab gate without retry; F28-01 → netlab 29. Publication mapping retained in the cut. |
| Devstack 31 (#412) | Source-binding hook stopped on feasibility, zero stack attempts; F31-01 → agent-core, design first. One checklist symlink repairs 14 archived OPERATORS references without changing pinned bytes. |

[75-row findings register](../handover/README.md#findings-register-at-the-follow-up-32-cutoff).
F28-01 and F31-01 are routed, not yet judged. F30-01 remains known behaviour,
with no merged fix. Main records no scheduled-run result. The nightly reading
guide below remains applicable; all historical failures stay failed.

