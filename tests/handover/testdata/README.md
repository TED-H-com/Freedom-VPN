# Pinned handover evidence (D-098)

TEST ONLY — isolated lab evidence; identifiers are test fixtures, not infrastructure credentials.

Run-ID-named files preserve the exact source commits listed in the manifest. Follow-up 14 main snapshot: c6e3eca; follow-up 15: b79ee23 (resumed through #290 from 5ce76e9). `followup14-snapshots.json` records byte-identical source paths, commits and SHA-256 digests. These tests never compare against the lane's mutable current file. Usage follow-up 4 retains its run-ID pins; members follow-up 7 now uses the final owner archive and its 12 local copies have been removed. Follow-up 15 moves isolation follow-up 6 and secops follow-up 10 assertions to the byte-identical final owner archives; old local copies remain immutable for provenance.

The two devstack `*-verdicts.json` files are explicit extracts of scenario id, status, run id and product revision from the dated follow-up 15 acceptance/export directories; each entry also pins the original source SHA-256. They omit traffic dumps irrelevant to the count assertions. The D-085 cost excerpt comes from the immutable #260 commit body (`git log -1 --format=%B 2df26aca`); it is not a new measurement.

`followup15-snapshots.json` records all 44 final isolation follow-up 6 files and
104 secops follow-up 10 files, compared byte-for-byte with their original main
commits. It also pins the new isolation follow-up 7 and updates follow-up 9
run-ID copies, the immutable enrollment follow-up 5 and devstack follow-up 16
runs, the older devstack follow-up 14 run note, and the raw r16 report from
commit e672d5d. No test compares these dated claims with mutable lane records.
The D-085 commit excerpt is retained for provenance; the assertion now names
its existing Go proof and owned operator wording, under D-109.

The raw r16 report is deliberately excluded from current-path/schema lint:
its proof files and line numbers belong to the review branch at base 5575470.
Its complete bytes, original metadata and current owner dispositions are
checked by `TestHandover_Followup15Claims`. Only owners adopt its proofs.

The resumed follow-up 15 pins the eight original enrollment follow-up 5 files
under `history/follow-up-5/` and all seven files from enrollment follow-up 6
run `1af27741-cbd9-4fdb-aa1c-7d02790e7a46`. The roadmap snapshot includes
#288–#290. Contract revision 14 and deadline claims pin existing Go test names
and owned operator wording; no token pin reads a living netlab note.

Follow-up 16 pins main 982c209 through #303 (13 exact roadmap rows), retains
byte-identical follow-up 15 register/review snapshots under history/follow-up-15,
and adopts members follow-up 7, enrollment follow-up 6 and secops follow-up 11
archives. Enrollment's four AT files match its old run directory exactly; the
older runbook/requests/summary pins remain where originally recorded because
the final root archive contains later finalized prose. No prose was substituted.
New members and secops evidence uses run-ID copies; enrollment 7 and devstack
17 already have immutable run directories. followup16-snapshots.json records
the exact source and SHA-256 of each pin. D-114 and D-119 are explicitly pending
or interpretive orchestrator decisions, not claimed merged implementations.

## Follow-up 17 provenance

`followup17-main.json` contains the ten #304–#313 roadmap rows at main
`7e044c71ecf1d07f63e89721bd213db6611a6a47`. `followup17-snapshots.json`
records source paths, that commit and SHA-256 for 59 immutable evidence inputs.
Owner prose is copied into `history/follow-up-17/`; devstack measurements use
their existing run-ID paths. No claims test reads a lane's current evidence.
Source notes may describe an older product base: their original failures and
pending work are historical, not silently updated to the handover cutoff.

Follow-up 16 claim tests now read its frozen register/operator/review text.
The register snapshot is a byte-identical contiguous excerpt beginning at its
dated decisions heading, excluding unrelated catalogue text; the manifest
records source byte offset, length, full-source digest and excerpt digest.
All other snapshots are complete source files, byte-identical at the cutoff.
Live document digests are separate in the owned secops scope-review manifest.


## Follow-up 18 provenance

`followup18-main.json` freezes 12 roadmap rows from main
`6d0022d83345ae78d26aaaf31c64f026d38dea3b`: #312, #314/#315 and #316–#324.
`followup18-snapshots.json` pins 26 byte-identical source files by path, source
commit and SHA-256. Lane results use run-ID filenames (including usage's short
run ID 170ea3e3); dated source prose and secops' carried-forward AT29/AT31 rows
are under `history/follow-up-18/`. Those secops rows retain their original
transport run identity, not the newer restore run ID. The register17 snapshot
keeps the old cutoff explicit while the current register advances through #324.
The r17 review is copied directly from `.agent/REVIEW.md` on review commit
`dcabc41e7d0c12725b1a7090425e2831faa3a12e`; its bytes are not reformatted.

Snapshot Markdown retains original links and source wording; resolve those
links at the manifest's original source location and commit. These fixtures
are test inputs, not additional operator runbooks. No claims test reads a
lane's current acceptance file. Living document-integrity digests remain in
`docs/e2e/at/secops/scope-review.json` and are checked separately.

## Follow-up 19 — #337 cutoff

`followup19-main.json` pins main 57a8cc9f and thirteen roadmap rows (#325–#337).
`followup19-snapshots.json` records 37 immutable inputs: thirty byte-identical
file copies under `history/follow-up-19/`, five existing enrollment owner
archives, and two exact merged-PR message paragraphs (D-129 from #328 and
D-132 from #337). Message excerpts include the source commit, complete-message
SHA-256, byte offset/length and excerpt SHA-256; no author metadata is copied.
File snapshots are exact `git show <source_commit>:<source>` bytes. Their
original relative links remain historical; operator links resolve from the
current handover page. `register18.md` freezes the prior register before its
current decision/findings update. Tests never read a living lane AT record.

`TestHandover_Followup19Claims` keeps ENR-07 / F9-01 closure, ISO-F6-03 OPEN,
regular AT23–AT27 blocked and unmeasured platforms distinct from local/harness
proofs. Live document hashes in scope-review.json are checked separately from
these historical evidence hashes.
## Follow-up 20 — #344 cutoff

`followup20-main.json` pins main 96ec0630 and seven roadmap rows (#338–#344).
`followup20-snapshots.json` records 38 immutable inputs: 27 byte-identical source
files, nine exact file excerpts, one merged-PR D-133 excerpt with byte offsets
and full-message hash, and one dated, selected-field Dependabot #22 API capture.
The alert is external status; the main-cut lockfile excerpt is separate.
Original source links and wording in copied Markdown resolve at the source
location/commit. The snapshots are evidence, not current operator instructions.

`register19.md` and seven original entrypoint excerpts freeze follow-up 19's
16 subtests / 13 roadmap rows / 37 original inputs. Follow-up 18 is unchanged.
`TestHandover_Followup20Claims` separately checks the #344 dispositions, exact
fresh gap and byte counters, stored ACK replay, immutable run verdicts, archive
hashes, D-133 attribution and still-open measurements. No test reads a living
AT-root file or binds another lane's mutable prose. Current owned-document
integrity hashes stay separate in `scope-review.json`.

Follow-up 21 fixes main `6d1e320d`, through #351, with seven exact roadmap rows
in `followup21-main.json`. `followup21-snapshots.json` records 13 immutable
inputs: 12 exact excerpts with byte offsets and full-source digests, plus the
byte-identical dated updater 19 hand-off. Eight excerpts freeze the follow-up
20 register and entrypoints; its 18 assertions, 7 rows and 38 inputs retain
all original expectations. Current acceptance results use roadmap rows only:
no members 11, isolation 10 or updates 11 living root is copied or read.
Historical owner archives are verified directly: members 10 (35 files),
updates 10 (39), isolation 9 (86), and both isolation 10 attempts (92 / 98).
The live scope-review manifest remains separate from dated claims.

Follow-up 22 fixes main `45209609`, through #356, with five exact roadmap rows
in `followup22-main.json` and 22 exact prose/manifest snapshots in
`followup22-snapshots.json`. Eight excerpts freeze follow-up 21's remaining
register/entrypoint reads; its 21 assertions, seven rows, 13 original snapshots
and 350 owner-archive files remain intact. Dated installer 8 / netlab 21 /
agent-core 29 hand-offs and devstack 21 prose/answers are frozen with full-source
hashes and byte offsets. No living raw AT or current netlab/agent JSON is read.
The 114 devstack follow-up 20 run files and 82 netlab follow-up 20 archived
files are checked by their manifests (four unscoped devstack prose files use
exact owned snapshots); the manifest's provenance paths are never read.
D-134 and provisional D-135 are registered separately from historical outcomes.
Current owned-document digests in scope-review remain live integrity checks.

## Follow-up 23 provenance

`followup23-main.json` pins the six #357–#362 roadmap rows at `bc011b6f`.
No listed follow-on delivery or security-ops 14 was on main at start. The
18-entry `followup23-snapshots.json` holds exact dated hand-offs and PR body
excerpts, prior owned register/entrypoint excerpts, and devstack manifests.
Each records its source commit, full-source SHA-256, byte offset, length and
excerpt SHA-256. The PR message excerpts are from #357 and #361's immutable
merge commits; no living AT root is copied or read by the claims test.
Members/updates/isolation current measurements use roadmap rows only.

Archive integrity covers updates 43 + its first attempt 17, isolation 100,
members 43, devstack 63 prior files and 155 UUID-scoped capture files.
Follow-up 22's 19 subtests, original assertions, source manifests and prior
redirects are unchanged except exact frozen-input redirects; a whole-function
hash enforces this. D-135 stays provisional, D-136/D-137 are the orchestrator's
provisional interpretations. Live owned-document integrity is separate.

## Follow-up 24 provenance

`followup24-main.json` pins seven #363–#369 roadmap rows at `5bd75102`.
Nothing merged after cutoff at task start; r18 and devstack 24 have no results
in this cut. The 21-entry `followup24-snapshots.json` retains exact dated notes,
immutable PR message excerpts and devstack run analyses/manifest, plus eight
owned excerpts freezing follow-up 23's register and seven entrypoints.
Full-source hash, source commit, byte offset, length and excerpt hash permit
an independent Git-object audit. No living members F13 or usage 7 root is
copied or read. Results use roadmap/PR pins, not those provenance-only paths.

The claim test checks 357 owner archive/run files: netlab 132, usage 19,
members 52, devstack 154. A normalized whole-function hash proves follow-up
23's original assertions are unchanged apart from frozen document inputs.
D-135 through D-138 stay provisional; D-136 correction and D-138 acceptance
never rewrite historical failures. Live owned-document hashes in secops
scope-review remain integrity checks only.

Follow-up 25 freezes #370–#373 at 9cd7a053: four exact roadmap rows and 15
byte-exact snapshots/excerpts (source commit, full-source digest, offset and
length). Updates 13 and isolation 12 use merged-PR hand-offs, never living root
records. Owner history inventories and the dated devstack 24 captures are
checked directly. r18 is an exact archive from 1a65598f, with no adopted proof
files or acceptance promotion. Follow-up 24's live register/entrypoints are
frozen; its normalized whole-function hash preserves every original assertion.

## Follow-up 26

`followup26-main.json` pins the nine #374–#382 roadmap rows at `9dde5eeb`,
including the #381 correction `5f4b455c`; #369 is supporting historical context.
`followup26-snapshots.json` records 17 immutable source excerpts with source
commit / full hash / offset / length / excerpt hash. Members 14 and isolation
13 living roots are neither read nor copied by the claims test. Their new-run
claims use only the exact roadmap rows. D-098 owner archives and the dated
906-file devstack capture inventory have independent hash checks. The previous
cut's register and entrypoint inputs are frozen; its body-content check permits
only removal of the single pinned installer note added by this cut.

## Follow-up 28 provenance

Main aae6763f through #390: five exact final roadmap rows, 16 immutable snapshots
with source digest / offset / length, and 1,644 owner archive/run file checks
(987 devstack 26, 609 netlab 24, 48 updates 13). Follow-up 27's register and
entrypoints are frozen; its old guide-body check excludes only the exact new
node-diagnostics note, whose current text is independently pinned. The #381
row still equals the original follow-up 27 snapshot (400d5cf9…).

Updates 14 run 2704600f is cited only by roadmap row / PR: living roots are
never copied or read by claims. D-098 includes dated devstack runs and the
per-follow-up netlab tree. D-109 implementation claims name Go tests; dated
hand-offs and merged-PR excerpts supply immutable context. Earlier claims,
decisions and historical failures remain intact.

## Follow-up 30 provenance

Main b9439411 through #403: seven exact final roadmap rows
(`followup30-main.json`), nine frozen follow-up 29 inputs with source digest /
offset / length (`followup30-snapshots.json`), and 449 archive/run file checks
(`followup30-evidence.json`: 123 isolation follow-up 15, 64 members follow-up
15, 165 netlab 26 and 97 devstack 29 files). Devstack 29's dated run has no
owner manifest, so `followup30-devstack29.sha256` lists its 97 files from main.
Follow-up 29's register, entrypoints, operator-guide §3.2 and ops snapshot are
frozen under `history/follow-up-30/`; its whole-function hash is checked after
reversing only those redirects. The revised node-configuration note replaces
follow-up 29's note in place, so older guide-body checks remove either note.

Living roots (isolation 83b457ef, members 0ffb355a, usage b8a6e9e1) are never
read by claims. Agent-core 36's dated hand-off is pinned by tokens from its own
sections only. The r20 review is an exact archive, SHA-256 af15fb7c…, with no
adopted proof files. The five scheduled workflows at b9439411 are frozen
(`followup30-workflows.json`) and checked structurally; no run result is read
or claimed.

## Follow-up 31 (#404–#407)

`followup31-main.json` pins four final roadmap rows at
`3573f396d60836f3f203ab5f883575c96a0ec45c` (#407), with no extra in-range PR
or later merge at task start. Agent-core 38 (carrying held 37), updater 20,
netlab 28 and members 17 are outside the cut.

`followup31-snapshots.json` pins nine byte-exact owned-document slices from
that commit: follow-up 30's register, seven entrypoint slices (the ops slice
also holds its old reading guide), and the operator configuration section.
Only follow-up 30's input paths are redirected; its normalized function hash
remains `3c4fc87eb6255f28ef46a156d03a7b99db392012828034953cc4871c0288a02d`.
Earlier snapshots and follow-up 30's cut document are unchanged.

`followup31-nightly.json` pins nine whole-file snapshots at the same commit:
the two workflows, two expected-blocked lists, isolation's exit mapper,
summary writer and runner, and e2e's gate library and command. These frozen
structural inputs permit future owner edits without changing this cut's
claims. No live workflow, list or implementation is a claims-test input.

`followup31-evidence.json` pins three dated evidence/review documents and
596 archive/run file checks: isolation follow-up 16's 129 entries, netlab 27's
446-entry final retention manifest, and 21 dated devstack 30 run files in
`followup31-devstack30.sha256`. Both compressed and uncompressed hashes are
checked where the owner manifest provides them. r21 is exactly 29,439 bytes,
SHA-256 `0fcf888bbb59e8e3032abf041f4d600fcb661525a3c390e15850c61e4df2edc0`.

Living isolation 0ec14b39, members 0ffb355a and usage b8a6e9e1 evidence is
never an input (D-132). Their statements use final roadmap rows/PR numbers.
Go proof names are checked as declarations on main; r21's unmerged proof
names are checked only inside its raw archived review. Follow-up 31 has
21 claims subtests, 72 findings and the same nine owner requests. Owned
current-document digests remain a separate live-integrity check.

## Follow-up 32 — cut #408–#412

`followup32-main.json` pins the five final roadmap rows at
`b7449ec1931a5cc954862908690969cb8bb2305d`: updater 20, handover 31, members 17,
netlab 28 and devstack 31. There was no unbriefed in-range PR or later merge at
start. Agent-core 38 (carrying 37), updater 21 and netlab 29 are outside the cut.

`followup32-snapshots.json` pins nine byte-exact source slices: follow-up 31's
register, seven entrypoints (ops includes its acceptance and nightly reading
sections), and the operator guide. Source commit, whole-source SHA-256, offset,
length and snapshot digest are retained. The old claims test only redirects
these inputs; its normalized function hash is unchanged.

`followup32-evidence.json` checks 81 members archive payloads and 220 netlab 28
payloads (301 total, including netlab gzip originals), seven dated/additive
documents (updater hand-off and timing JSON, members publication mapping, four
devstack run documents), 18 members relative archive symlinks and the devstack
checklist symlink, plus all 14 archived OPERATORS source files. Every symlink
must resolve within the repository and match its target-byte digest. The
81 archived payloads were independently compared byte-for-byte with the old
root files on #409; those living roots are never claims-test inputs.

The netlab manifest's main digest is
`dda5a42a27bd1e215aeb7384d71cdb5ef69286b9bfa873172d318138749f8e5d`.
The roadmap's additive publication mapping is authoritative for the two
retained old harness IDs; no old evidence is rewritten.

`history/follow-up-32/pr409-session.txt` is the exact public summary excerpt
from squash commit `17801d4d` (`git show -s --format=%B`), starting at
"Handover follow-up 31: session launched" and ending before "The orchestrator's
reading." Its SHA-256 is pinned separately. It establishes the one-session
03:07 EST launch and 39-minute admission wait without copying private receipts.

Members 81b638e8, isolation 0ec14b39 and usage b8a6e9e1 remain living. Only
final roadmap rows / PR numbers support their current acceptance claims.

## Follow-up 33 — cut #413–#415

`followup33-main.json` pins three final roadmap rows at
`339db17215db583330a2395f0484f2bdad9681e7`: updater 21, handover 32 and netlab 29.
No unbriefed PR or post-cutoff merge was present at task start. Agent-core 39
(carrying 37 and 38), updater 22 and netlab 30 are outside the cut. Three exact
public PR excerpts are retained from their squash commits on main: #413's
answered requests, #414's session and seven-minute wait, and #415's routing.

`followup33-snapshots.json` pins ten byte-exact #415 source slices: the previous
register, seven entrypoints, full operator guide and full reviews index. Each
has source commit, source hash, byte offset, length and retained hash. Earlier
claims only redirect their inputs; the source-reader normalization reverses
those redirects for historical function hashes. Follow-up 32's normalized
SHA-256 is `ad4e9f28727b56e2373f0a64eabc22f559811c80a187c2cc84683f75432ef4ee`.
Its actual counts remain 20 subtests, five final rows, nine frozen inputs,
301 payload checks (81 members, 220 netlab), seven documents, 19 relative
symlinks, 14 archived navigation sources, 75 findings and nine requests.

`followup33-evidence.json` pins both dated updater 21 documents and r22's raw
37,600-byte review, SHA-256
`05e7806a1f5dfec8388e4fe2cbe0e5e1ed11ec1d4c291f3bb5236d7c0fe43c02`.
Netlab 29's manifest is
`d9f9e7d539a884c113a461646448539343c1da803cf9dc378c4e0ab59dbddefa`:
512 payload hashes and lengths, including 42 gzip originals. The updater
JSON independently pins all seven legs and both exact deterministic-budget
checks. Review proof names are checked only in the archive, never as adopted
main tests. The stop-budget erratum is in the reviews index, not the archive.

The cut has 16 claims subtests, 81 findings and the same nine owner requests.
Only F27-02, F28-01 and F27-01/B1 prior rows change; all others and all 47
decision rows remain byte-identical. Living members 81b638e8, isolation
0ec14b39 and usage b8a6e9e1 roots remain excluded. Live owned-document integrity
uses a separate scope-review inventory.

## Follow-up 34 — cut #416–#418

`followup34-main.json` pins the three final roadmap rows at
`cc0ed4ae713f6c8242171f923bdc799a0c07a74c`: handover 33, updater 22, netlab 30.
Its three public PR excerpts are byte-exact slices of the squash-commit bodies
on main, with source hashes and offsets. No unbriefed or post-cutoff merge
was present at task start. Agent-core 39 / r23, updater 23 and netlab 31
remain outside the cut.

`followup34-snapshots.json` pins ten #418 source slices: register, seven
entrypoints, full operator guide and reviews index. Earlier claims redirect
only those reads; follow-up 33's normalized function SHA-256 remains
`9c03014cb655ab6e68d92eb1322f9415569bbb59aa956f9b64f7ab2c26bc05ac`.
Its counts are verified from code and manifests: 16 subtests, three rows,
ten frozen inputs, 512 payloads / 42 gzip originals, three documents,
three PR excerpts, 81 findings and nine owner requests.

`followup34-evidence.json` pins updater 22's dated hand-off and timing JSON,
and netlab 30's final manifest, SHA-256
`cd23561b8e978f126a213b6fa5ed3eb192ccc9de37058ad430c68e4788cd1254`.
The suite checks 312 payload hashes and lengths, including 19 gzip originals;
the 283 updater iterations' exact budgets; the finite collector controls;
all seven unique drop matches and install brackets; cost and host
qualification; and the byte-identical five prior manifests / 1,506 payloads
recorded at both endpoints. The actual prior payloads remain protected by
the earlier claims suites. No living AT root is a claims input.

There are 14 claims subtests, 81 findings, 47 unchanged decisions and nine
unchanged owner requests. Only F21-01, F27-02 and F27-01/B1 change. #417's
reported 16 failing control cases are explicitly separated from the JSON's
15 failure labels; 26 passing test/subtest results remain the recorded fixed
summary. The counting unit is an orchestrator clarification, not an edit to
producer evidence. Live document integrity uses a separate scope inventory.

Follow-up 35 pins the #419–#421 cut at `dff6bdb0`: `followup35-main.json`
contains three exact final roadmap rows and three public PR excerpts with
squash-commit, offset and hash provenance. `followup35-snapshots.json` seals
ten #421 input snapshots used to preserve earlier claims. The separate
`history/follow-up-35/source-audit.json` records the checked renderer/marker,
counter, stop-budget and proof-name errata, r23's L-1 code-trace boundary,
and r24's after-cutoff review commit without its results.

`followup35-evidence.json` seals updater 23's two dated documents, the exact
41,791-byte r23 archive, and all six netlab 31 design files. No living lane
root is a claims input. `TestHandover_Followup35Claims` has 16 subtests,
checks the declared load-leg qualifications and all 352 logged Recover
budgets, verifies F23-01 controls across both sessions and the 15-case
erratum, preserves follow-up 34's assertions, and verifies 87 findings,
47 unchanged decisions and ten owner requests. Live document digests remain
separate. Historical failures retain their recorded verdicts.

## Follow-up 36 (#422–#423)

`followup36-main.json` seals two final roadmap rows and two exact public PR
message excerpts. #422 uses a reversible JSON string to preserve its original
whitespace-only line without introducing trailing whitespace in the diff; its
decoded bytes, digest and source length are retained. `followup36-snapshots.json` records ten exact #423 byte slices
for earlier claims; source commit, source digest, offset and length are retained.
`followup36-evidence.json` seals r24 (46,001 bytes), all three netlab 32 design
files and all 171 evidence files, including 12 gzip originals. No living lane
root is read. The separate frozen source audit records code verification at
#423, the review-branch proof boundary, 1,830 unchanged historical artifacts,
jitter arithmetic and the lock-path discrepancy returned to netlab. The raw
archive and producer evidence are unchanged. Follow-up 35's assertions change
only their rewritten-document inputs; their restored function hash stays exact.
Live document integrity remains a separate check of the secops inventory.

Follow-up 37 seals the #424–#426 cut at 792e3e41. Its main manifest freezes
three final roadmap rows and three lossless JSON-encoded public PR excerpts.
Ten #426 owned-document slices preserve follow-up 36's original assertions.
The evidence manifest pins the dated isolation diagnosis, netlab 33's additive
correction and the 138-file follow-up 17 archive; follow-up 36's complete
netlab 32 inventory remains sealed. The separate source audit records code
readings and commit-scoped searches. Living isolation 9f1bc0ff, members
81b638e8 and usage b8a6e9e1 roots are never claims inputs.

Follow-up 38 seals #427 only at 77626af4. Ten #427 slices preserve every
follow-up 37 assertion; its normalized claims function remains exact. The
main manifest has one final row and one lossless public PR excerpt. r25 is
sealed byte-for-byte, with proof names pinned only inside the archive. The
separate frozen audit records source readings at 8cc27609 and proof metadata
at 1afecb29, neither on main, and verifies #427's 258 digest-only changes.
The proof file is never copied. No living lane evidence is a claims input.

Follow-up 39 seals #428–#429 at 1b87ab32. Its main manifest pins two final
roadmap rows and two lossless public PR excerpts. Ten frozen #428 input slices
preserve follow-up 38's assertions and follow-up 28's live diagnostic-note read;
only those document reads move. The source audit seals 98 cited/corrected source
excerpts, 24 declaration references (seven unique stale-line citations), exact
r25/r26 import metadata, no-sensitive-pattern inspection and #428's 258
strictly digest-only changes. The r26 archive remains 19,545 exact bytes.

The evidence manifest seals 15 dated agent-core documents and 75 Go test
names now on main. The judgment manifest binds every requested row to its
record, verdict, sub-trace limits, fix and proof names; no rejected claim is
promoted to a fix. Thirty finding rows change and 69 remain byte-identical.
Eight confirmations are status/date substitutions only; six new decisions
bring the count to 53, with all ten historical owner requests answered.
The 11 follow-up 39 subtests keep the two unqualified timing windows out of
both zero-sized arm pools. Living AT roots remain excluded; live document
digests in scope-review.json stay separate from these frozen claim inputs.

## Follow-up 40 provenance

`followup40-main.json` pins #430–#434 at main `3f7dbc25`: five final roadmap
rows, ten first-parent commits and five lossless public PR body excerpts.
`followup40-snapshots.json` freezes ten #430 handover inputs; earlier assertions
are unchanged after reversing only six input redirects. The separate source
audit checks baselineDecision at 8dec4100: lines 170–209, mono starts at 211.
Neither the r26 archive nor #430's erratum changes.

`followup40-evidence.json` pins 76 immutable dated/per-follow-up records,
11 exact cutoff snapshots and 84 Go test declarations on main. The owned
members-run2 and isolation-assembled directories are frozen copies of selected
#434-cutoff records; tests never read living roots. `case-checks.json` extracts
the ten case verdicts/counts/blockers and all 77 AT09 names/statuses from the
original JSON (including AT02 gzip), with full source hashes and cutoff commit.
No assertion, failure or payload is invented. Historical relative links in
these snapshots resolve at their source paths/commits, not their copied paths.
The D-146 header archives and isolation's D-147 lab archives remain in place.

Decision and finding manifests pin 55 decisions and 102 findings: only three
old decision statuses change, two provisional decisions are added; four finding
rows change, three are added and 95 remain byte-identical. Live scope-review
digests are separate from immutable claims. Agent-core 45, devstack 32 and
members 19 are recorded by briefing status only, with no post-cutoff results.

Follow-up 41 seals #435–#438 at c359d4a9: four final roadmap rows and public
PR excerpts, ten exact #435 handover inputs and three cutoff-frozen members
19 notes, 101 immutable agent-core 45/devstack 32 records and 36 Go declarations.
The source audit records #437's complete changed-file list and nine excerpts
supporting the product/contract reading. Four follow-up 40 reads now use #435
bytes; all 34 earlier claims functions remain exact after reversing those
redirects. Prior 14-subtest/76-record/11-run/84-declaration counts and the
members ten-case/77-subcheck inventory are independently checked. Five live
publisher invocations retain their outcomes; neither a later pass nor the
zero-header proof upgrades a historical failure or establishes the observation
race. Only D-150's status changes: 55 decisions, 105 findings (three updated,
three added, 99 unchanged). No living members/isolation/usage evidence root
is a claims input. TEST ONLY — all run IDs and keys here identify lab fixtures.
