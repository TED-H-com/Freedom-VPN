# Follow-up 8 cursor assertion review

The original records remain unchanged. Installer serving, held usage, wrapper
rerun, authorized reinstall, canonical payload preservation and retired-certificate
denial passed. Production hand-off ACKed the original stream through sequence 3;
the current stream's ACK stayed 0. The stopped current ledger held an empty event
at sequence 1 (no records, no member bytes).

The enrollment harness added `currentCLI.LastEventSeq == "0"`, a condition absent
from usage AT20's `OriginACKAndIndependentCurrentACK` assertion. This rejected a
legitimate empty initial event. Its failure and propagated process failures stay
visible; it is an enrollment assertion error, not evidence of an agent/usage
handoff failure. File hashes changed during D-118 adoption while the canonical
original payloads, hashes and origin ACK remained unchanged.

The corrected probe follows AT20's saved production-batch replay: unchanged
canonical result, ACK selected by the origin, independent current ACK before/after
replay, unchanged totals and historical samples. It separately proves the new
stream starts at 1 and records its actual last/next sequence. Fresh post-republish
traffic must be counted and ACKed; an empty event is never credited as traffic.

These exact rows stop blocking only after all five composed rows, the four named
AT20 assertions, retired certificate denial and both probe processes pass. No
unrelated failure can use this review. The final record retains all original rows.
