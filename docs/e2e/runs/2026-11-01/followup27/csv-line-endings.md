# Original CSV line endings

2026-11-01. The final staged/base-diff check identified Python's standard CSV
writer CRLF endings as trailing whitespace in `stream-intervals.csv`. The
ordinary working-tree-only diff gate had passed before those new files were
staged. The full comparison produced 58,627 CSV line-ending diagnostics and no
other trailing-whitespace diagnostic.

The original CSV bytes are preserved. This archive's `.gitattributes` permits
CRLF only for its per-run `stream-intervals.csv` files while retaining blank
trailing-space, blank-EOF and space-before-tab checks. No evidence row, timestamp,
interval, original hash or capture was rewritten. The staged and full-base
checks pass with this scoped data-format declaration. This does not change Go
formatting, lint, a product gate, or the historical follow-up 26 archive.
