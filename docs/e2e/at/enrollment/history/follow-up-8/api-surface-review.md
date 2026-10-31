# Enrollment API surface review — follow-up 8

2026-10-31. Public APIs provide token issuance, ETag/If-Match revocation,
same-node replacement enrollment, incarnation history, policy/grant/capacity
setup and publication. External revocation is credited only after successful
route/identity operation step read-back. The installer is invoked exclusively
through devstack wrappers; no SQL retirement, hand-off or fabricated observation.

D-116/#302 and active D-118/#312 are exercised. A real installed node serves with
four rendered chains and a separate hookless ownership table; complete nft text
retains the comment hidden by nft 1.0.9 JSON. Held bytes and wrapper rerun pass.

The production uploader uses the current presenter and original stream selector.
Original canonical events/hashes, once-only historical samples, API before+held
equality, stored same-key response replay and independent ACKs pass AT20's named
assertions. Coherent stopped SQLite copies and central read-only queries provide
evidence; no second writer opens a live ledger.

After API republication the new node serves and counts traffic, but its empty
1 ns initial gap fails central timestamp persistence. New ACK remains 0. This
[agent/meter and usage-central defect](current-stream-gap.md) keeps the composed
reinstall row blocked; old-stream success cannot close the fresh stream.

The ordinary-node freeze proof remains separate: populated signed lease/live
grants, real conflict receipt, durable latch and fresh restart observation.
See [results](summary.md), [runbook](RUN.md), [AT20](../usage/AT20.md),
[D-102 answers](../../RESPONSES.md), and [devstack 18](../../runs/2026-10-31/followup18/README.md).
