# Enrollment acceptance — follow-up 6

Run `1af27741-cbd9-4fdb-aa1c-7d02790e7a46`, **2026-10-31T07:28:01Z–2026-10-31T07:34:14Z**.
Source `3f604e2c2d0fd89ee7e2d91014ab31ff46aed7a8`, clean at invocation. Both historical failure reviews
are resolved. All fixtures and identifiers are TEST ONLY.

| AT | Status | Pass / blocked / fail | Evidence |
|---|---|---|---|
| AT06 | pass | **376 / 0 / 0** | [JSON](AT06.json), [Markdown](AT06.md) |
| AT07 | blocked | **26 / 5 / 0** | [JSON](AT07.json), [Markdown](AT07.md) |

AT06 repeats the previous 344 passing checks and adds 32 evidence-file scans:
16 new files (nine follow-up-5 archive files and seven current-attempt files),
scanned in both phases. Both phases recorded **0 HTTP 429**; D-078 passes.

Installer PR #279 is verified through the devstack wrappers: initial install
`0 / installed`, same-release rerun `0 / already_current` with unchanged
hashes, controlled reinstall `0 / reenrolled`. The new incarnation belongs to
the same Node; the mode-0700 retired identity hashes match, all four retired
bound-state file observations match, and usage.db is unchanged. Old certificate
requests to desired-state and observations both returned **403 IDENTITY_REVOKED**.
The fixed socket exists under a mode-0700 directory; the old socket and both
prior read-only diagnostics are absent. No journaled upgrade is claimed.

AT07 still has five blocked rows: `counted_unacked_usage`, `installer_rerun`,
`identity_ledger_preserved`, `controlled_reinstall`, `historical_usage_retained`.
`docs/e2e/RESPONSES.md` is absent. Hold/snapshot/member transport exports still
reject `installer-node`; no service-restoration export is documented. Each
wrapper leaves agent/watchdog/updater inactive. API forwarding is null and
safety unknown, so subprocess success cannot establish a serving baseline.
Usage history is zero bytes, with no counted backlog or new member traffic.
Authorized delivery of retained old-incarnation events also remains undefined.

Live make exited **2** for these documented blockers; area lint, root Go lint,
vet, untagged Go tests and whitespace checks passed. Cleanup inventory:
**0 containers / 0 networks / 0 volumes / 0 images**.

See [RUN.md](RUN.md), [requests.md](requests.md), and
[follow-up 5 archive](history/follow-up-5/README.md). Historical files remain
byte-identical; new records use explicit SHA-256 digest labels.

Final moving-main recheck: RESPONSES.md has since appeared on main, with only
members/updates answers and no ENR-05 response. The measured worktree and
remaining blockers are unchanged; see the dated requests entry.
