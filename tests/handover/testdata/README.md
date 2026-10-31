# Pinned handover evidence (D-098)

TEST ONLY — isolated lab evidence; identifiers are test fixtures, not infrastructure credentials.

Run-ID-named files preserve the exact source commits listed in the manifest. Final handover main snapshot: c6e3eca. `followup14-snapshots.json` records byte-identical source paths, commits and SHA-256 digests. These tests never compare against the lane's mutable current file. Missing lane archives (security-ops follow-up 10, usage follow-up 4 and members follow-up 7 final run) are requested in the hand-off report.

The two devstack `*-verdicts.json` files are explicit extracts of scenario id, status, run id and product revision from the dated follow-up 15 acceptance/export directories; each entry also pins the original source SHA-256. They omit traffic dumps irrelevant to the count assertions. The D-085 cost excerpt comes from the immutable #260 commit body (`git log -1 --format=%B 2df26aca`); it is not a new measurement.
