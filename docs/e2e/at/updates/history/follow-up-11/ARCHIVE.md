# Follow-up 11 immutable archive — D-098 / D-132

Archived before follow-up 12 execution from main `45209609d33d567c5f04d49441799cdd3efac59b` on 2026-10-31.
All 43 files directly under the living updates root were copied byte-for-byte,
including run `daddad24-b6f7-4c32-ba8e-1c7d24f76ea2`, its narrative, source
pins, and carried-forward operator/historical records. Original contents and
links are preserved, not rewritten. See `SHA256SUMS` for every pin. Every
previous archive remains unchanged.

Updater 19 (#351) cites the former living `installer-handoff.json`,
`installer-delivery-transport.json` and `artifact-origin.json` by path without
digests. This archive now supplies stable pins for that accepted measurement.
Product tests may consume only immutable history paths, never living AT roots.
Current acceptance claims must come from the new run, not this archive.

Verify with `sha256sum -c SHA256SUMS` from this directory.
