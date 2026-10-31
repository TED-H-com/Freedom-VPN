# Enrollment acceptance requests — follow-up 7

TEST ONLY. Dated 2026-10-31. D-102 enrollment answer is present in
[RESPONSES.md](../../RESPONSES.md#enrollment--2026-10-31).
Shared installer-node usage exports and RestoreInstallerServices are delivered
by devstack follow-up 17 (#294); this run checks their real behavior.

## ENR-06 — serving installer prerequisite (D-116)

Installer/enforcement: move the installer marker out of inet fvpn (installer
follow-up 4), then rerun serving installer retention and restoration checks.
The existing marker fails verifyChainSet with HOOK_PRIORITY_MISMATCH.
[Authoritative product proof](../../runs/2026-10-31/followup17/a7c41436-82b3-4665-b75c-e2887f2e1009/installer-owner-chain.md).
Stopped file hashes, identity and restored systemd state cannot close serving
rerun/reinstall or nonzero retained usage.

## ENR-05 — retired-stream delivery (D-105 / D-112)

Usage-central follow-up 4 and usage-agent follow-up 3: complete the revision-14
retired-stream handoff, including authorized once-only delivery after revocation.
Before-plus-held-byte equality and old-incarnation attribution remain blocked
until both implementations merge. No retired event is submitted under the new
identity by this harness.

## D-098 archive

Follow-up 6 root records were moved byte-identically before the new run to
[history/follow-up-6](history/follow-up-6/README.md). The current run retains all
prior check outcomes. No pin outside this lane is changed.
