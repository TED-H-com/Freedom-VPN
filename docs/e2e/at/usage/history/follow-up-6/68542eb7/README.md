# Interrupted follow-up 6 attempt 68542eb7

This attempt was deliberately stopped during AT18 after static review found
that the new replay fixture selected pre-adoption upload batches as well as
replacement-incarnation batches. `retirePrevious` preserves `upload_batches`;
the new query must filter the request's current incarnation. This is a harness
selection correction, not a measured product failure. AT20 was not reached.
Only records actually emitted before interruption are retained here; no full-run
or D-130 pass is claimed. The subsequent clean run supplies living pass claims.
