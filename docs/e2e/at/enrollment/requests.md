# Enrollment acceptance requests — follow-up 8

2026-10-31 — **1 request to orchestrator.**

## New current-stream gap precision defect — agent/meter and usage-central

[Deciding evidence](current-stream-gap.md): current gap event 1 spans 1 ns;
PostgreSQL stores both endpoints identically and rejects
`usage_events_interval_check`. Repeated retryable uploads cannot advance current
ACK beyond 0; counted event 2 retains 908 bytes. The earlier retired stream is
successfully ACKed through 3 with correct historical attribution.

Route to the owners of `node/internal/meter/delta.go` and
`internal/usage/ingest.go` / central timestamp persistence. Fix generation and
handle already immutable blocked gaps without rewriting this lane's events or
loosening accounting/isolation. Those paths are outside enrollment ownership;
none was edited. Rerun the composed reinstall/current-ACK check after the fix.

| Row | Status | Owner | Evidence |
|---|---|---|---|
| installer_rerun | pass | installer | Same identity and stopped file equality, restored serving and counted traffic; AT07 JSON and text rulesets. Devstack 18 installer-marker-migration / installer-wrappers corroborate. |
| controlled_reinstall | blocked | agent/meter; usage-central | Wrapper, new identity, retained files, strict verifier, serving and fresh traffic pass. New current stream cannot ACK its 1 ns gap; [deciding evidence](current-stream-gap.md). This is not D-116. |
| counted_unacked_usage | pass | agent / meter | Installer-node nonzero held upload/download, TCP/UDP receipts and counter increases; devstack 18 installer-usage corroborates. |
| identity_ledger_preserved | pass | usage / agent | Original canonical payloads/hashes/ACK survive D-118 adoption; original stream delivered first using the current presenter. |
| historical_usage_retained | pass | usage / control | AT20 named equality, historical samples, canonical same-key replay and independent ACKs all pass. |


## D-102 answers consumed

[Enrollment response](../../RESPONSES.md#enrollment--2026-10-31): installer-node
hold/snapshot/transport exports, ConfigureInstallerServing and shared service
restoration are exercised. Its old D-116 blocker wording is superseded by #302,
[devstack 18](../../runs/2026-10-31/followup18/README.md) and this lane's live
marker/serving evidence. Legacy-marker migration is cited from devstack's
installer-marker-migration row; this lane never manually edits markers.

[Usage response](../../RESPONSES.md#usage--2026-10-31): systemd quiescence,
API retirement and same-node token, ControlledReinstall, restore, release,
coherent ledger copies and origin/current ACK separation are exercised. Its
activation-hook gap was closed by agent-core 25/#312. No SQL retirement or
manual hand-off supplies the result.

Before **908** + held **1816** = after **2724 bytes**. Two original events remain attributed once only to the retired incarnation. Origin ACK **3**, independent current ACK **0** through same-key replay; the new stream starts at sequence **1**.

The named [AT20 assertions](../usage/AT20.md) pass here:
BeforePlusHeldByteEquality, OriginalPayloadAndHistoricalAttribution,
AgentSameKeyStoredResultReplay, OriginACKAndIndependentCurrentACK. D-118 can
change SQLite DB/WAL containers during replacement bootstrap; the required
original canonical payloads, hashes and ACK are unchanged. Same-incarnation
rerun still proves quiescent file equality. Empty new-stream event 1 is not
credited as traffic; its subsequent central precision failure is retained above.

Freeze withdrawal was **334 ms**, **+107 ms** against the 227 ms baseline, below the 5000 ms bound. Real receipt replay and durable latch/restart passed.

## Known gaps outside these AT07 rows

The separate post-upgrade seed HASH_MISMATCH remains with installer 6 / updater
17 in [devstack 18](../../runs/2026-10-31/followup18/README.md#candidate-health-and-post-upgrade-reenrollment).
This run does not perform an upgrade. Physical arm64 is unmeasured; compilation
passes. Freeze restart is an agent process restart, not OS reboot. AT20's other
three named assertions retain the usage lane's evidence; no rerun is claimed.
