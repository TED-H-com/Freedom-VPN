# Enrollment API surface review — follow-up 7

2026-10-31. Contracts revision 14 and main through devstack #294.
Enrollment tokens, node revocation, replacement enrollment, newest-first
incarnation history and runtime certificate denial use the existing product API.
Serving-source setup uses group/policy/capacity/publication/member-grant APIs and
the shared native member login. A successful mutation is followed by read-back.

The ENR-02/03 freeze proof starts on the ordinary enrolled node, with populated
signed lease and live member rules. Clone conflict uses the existing dedicated
fixture. A replay of a real product observation checks identity_frozen=true;
no fabricated report or SQL mutation supplies an observation. Timing is bounded
conservatively from the central conflict transaction to live-rule read-back.
Agent restart additionally requires the persisted SQLite latch and fresh reports.

D-105 revision 14 describes the retired-stream handoff. Central acceptance
(D-112) and node upload are not merged on the measured source. These interfaces
are not exercised as an implemented product feature. The two corresponding
retention/attribution claims remain blocked.

[Devstack enrollment response](../../RESPONSES.md#enrollment--2026-10-31) and
[installer lifecycle contract](../../installer-layout.md) provide the shared
installer exports. D-116 still prevents an installer serving baseline.
