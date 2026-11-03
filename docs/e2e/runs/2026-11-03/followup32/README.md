# Follow-up 32 — isolation F19-02 publisher conflicts

2026-11-03. The source change follows spec §14.5 / API01 for node conditional
writes: at most three writes in ten seconds, exact-code conflict re-reads,
intended-value checks, one unchanged command key/body, complete ordered receipts.
Publication requests and downstream completion predicates are preserved.

The [audit](audit.md) covers every owned site, the server transaction/key rules,
and dated requests for read-only AT-lane sites. Isolation 19's original run is
unchanged; its natural race cause is plausible but unproven.

Validation and the two planned fresh-stack attempts are pending host admission
in this implementation commit. The final evidence commit will retain the actual
results of each invocation, including any failures. No stack attempt has been
made at this point.
