# Follow-up 7 attempt 1 — interrupted after a harness failure

Run `63a07f95-6a73-4f6c-95d9-f10a5a22de8c`, source `3685322b0aa247f1a52be682750f6f147cb90dc9`, clean tree at start. Product base `d9e28f8`.

AT02's D-081 fixture could not resolve its admin actor because bootstrap `idplogin.CredFor("e2e-admin")` deliberately returns no ID. No grant was revoked and no plan downgrade occurred. This is a harness setup failure, not a server status-code finding. The preceding D-077 fence/list/cursor/security/re-grant/replay/traffic checks passed; their bodies and counter observations remain in AT02.json.

The foreground test was interrupted with SIGINT (exit 130) after the failure was diagnosed. Any `complete:false` records are partial. Other AT files from follow-up 6 were not copied into this attempt. No pass from this attempt is promoted into the final baseline. The entire assigned stack is torn down and reset before the corrected full run.

Correction: resolve the actor by hashing the current admin session cookie in memory and reading its unexpired `admin_sessions` row. Do not select an arbitrary administrator or equate a bootstrap credential with a subject. The readback harness also preserves primary fail status rather than demoting failures to blocked.

Recorded cases: [('AT01', 'blocked', True), ('AT02', 'fail', True), ('AT03', 'blocked', False), ('AT05', 'blocked', True)]
