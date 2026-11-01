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
