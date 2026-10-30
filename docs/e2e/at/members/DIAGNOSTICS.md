# Historical first-pass diagnostics

Current diagnoses and closed findings are in [requests.md](requests.md).
The older observation below is retained, not asserted for Follow-up 3.

# Component diagnostics, not live acceptance

Executed 2026-10-29 UTC on the task worktree. All 13 selected existing tests
passed. These use isolated PostgreSQL databases through `internal/db/dbtest`,
real domain/HTTP code and test adapters. They do not exercise the devstack, native clients or real gateway traffic;
the later live run is recorded separately in the per-AT JSON files. **AT01–AT04 remain blocked.**

Command (TMPDIR/GOTMPDIR pointed at private disk scratch under `.agent/tmp`,
removed afterwards):

```sh
go test -tags integration -count=1 -p 1 -parallel 2 -v \
  ./internal/devices ./internal/billing ./internal/members/httpadapter \
  -run '^TestAT0[1-4]_'
```

Selected actual output, omitting random isolated database identifiers:

```text
--- PASS: TestAT04_FiniteBindingRenewalAndExpiredObservation (0.37s)
--- PASS: TestAT02_BlankPeerIdentifiersDoNotWrite (0.34s)
--- PASS: TestAT02_LastSlotRaceIsolationAndRevoke (0.46s)
--- PASS: TestAT01_VerifiedCrossClientMappingAndReadBack (0.59s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/internal/devices	1.771s
--- PASS: TestAT03_DuplicateTenAndEveryEventOrder (0.83s)
--- PASS: TestAT03_ForgedReturnInvalidSignatureAndCardMinimization (0.62s)
--- PASS: TestAT03_LateFailureCannotOverwriteRenewal (0.38s)
--- PASS: TestAT04_CancelRefundRenewSuspendAndExactlyOneOutbox (0.48s)
--- PASS: TestAT04_KeepFullRefundPolicyAndConflictingInvoiceAnomaly (0.45s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/internal/billing	2.772s
--- PASS: TestAT01_AT03_LoginCheckoutWebhookDeviceAuthorizationHTTP (0.84s)
--- PASS: TestAT03_WebhookRawBytesReplaySignatureFailureAndPortal (0.57s)
--- PASS: TestAT02_DeviceLimitRevokeOwnershipReadbackHTTP (0.81s)
--- PASS: TestAT04_SuspensionGrantGraceAndHTTPPreconditions (1.29s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/internal/members/httpadapter	3.516s
```

This provides additional support for the intended fail-closed mapping and quota
behaviour in [AT02-TRACE.md](AT02-TRACE.md). No source in `internal/` was changed.
The isolated databases were dropped by the test cleanup; the admin database was
not migrated, reset or reconfigured.
