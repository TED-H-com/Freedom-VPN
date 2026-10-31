# Pinned handover evidence (D-098)

TEST ONLY — isolated lab evidence; identifiers are test fixtures, not infrastructure credentials.

Run-ID-named files preserve the exact source commits listed in the manifest. Follow-up 14 main snapshot: c6e3eca; follow-up 15: b79ee23 (resumed through #290 from 5ce76e9). `followup14-snapshots.json` records byte-identical source paths, commits and SHA-256 digests. These tests never compare against the lane's mutable current file. Usage follow-up 4 and members follow-up 7 retain their run-ID pins. Follow-up 15 moves isolation follow-up 6 and secops follow-up 10 assertions to the byte-identical final owner archives; old local copies remain immutable for provenance.

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
