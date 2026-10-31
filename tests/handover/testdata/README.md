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
