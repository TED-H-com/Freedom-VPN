# Follow-up 4 archive — D-098

Final run `351d59aa`, merged as PR #262. All 17 run records are byte-identical to the living files before follow-up 5. `SHA256SUMS` records the snapshot.

No handover pin moved: `tests/handover/handover_test.go` already reads the immutable `351d59aa-AT14.json` through `351d59aa-AT19.json` under its own `testdata/` directory, not the living usage files (D-109).
