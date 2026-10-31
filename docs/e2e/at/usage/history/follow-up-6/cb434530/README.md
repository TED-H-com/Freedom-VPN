# Follow-up 6 diagnostic cb434530

AT14/16/17/18/19 passed; AT15 was not reached. AT20 accepted and ACKed the
new 1 µs unknown gap at 1 and counted event at 2, with 1,106 upload + 1,515
download = 2,621 bytes matching the actual counter delta exactly. It failed
a new fixture assertion comparing the retired origin's complete usage view
with a baseline taken before the deliberately rejected late-event probe.

[Read-only same-stack diagnosis](checkpoint-diagnostic.json) shows origin
bytes 4,304/6,062 and ACK 2 unchanged. The deliberate rejection updated its
checkpoint from 15:49:33.734927Z to 15:49:37.350835Z; the incarnation API's
`through` is `usage_checkpoints.updated_at`, not an immutable last-event time.
Current traffic did not move origin bytes. The corrected fixture takes its
comparison baseline after the fault probe and still asserts the complete view
stays fixed during current traffic, plus exact bytes across the fault probe.
Existing retired hand-off assertions are unchanged.

The same-stack diagnostic also reads the two stored current batch results
(accepted ACK 1 and ACK 2). It does not replay them or promote this failed run
to pass. The subsequent full live run owns the replay and final pass claims.
