# Follow-up 9 interrupted attempt 1 — 2026-10-31

Run `f678f06d-83c2-4591-b77a-c6ef2d4395e9`, source `f17c0df85ebcbb6927903efb531d8e791c9a112f`.
This incomplete run was deliberately stopped during AT02 after three stale
harness audit queries failed. AT01 and AT05 completed blocked with 18 and 30
passing checks. AT02's recorded `complete` flag remains false; its partial
checks and failures are retained byte-identically. No AT09 result came from
this attempt.

The harness still selected `action='grant.revoke'`. Current
`internal/members/service.go` emits `member.grant_revoke`, as required by its
integration tests. [Read-only diagnostic](audit-diagnostic.json) found exactly
one current-action audit per each of the three revoked grants, with operation
IDs. The correction changes the audit selector in owned acceptance code;
assertions still require exactly one event per revoked grant, and expired
input must leave the audit count unchanged. No product state was repaired.
The next run starts with a fresh stack and repeats the full mutation sequence.
