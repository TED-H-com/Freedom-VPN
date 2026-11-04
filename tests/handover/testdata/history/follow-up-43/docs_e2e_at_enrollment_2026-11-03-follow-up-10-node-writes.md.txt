# Follow-up 10 — node conflict handling adoption

2026-11-03. Offline adoption of [devstack 33's hand-off](../../RESPONSES.md#2026-11-03--at-owners-exported-node-conflict-handling-follow-up-33), implementing the command-concurrency contract in §14.5 for AT06/AT07 (§19; ENR01/ENR02/ENR03, PKG01, DATA01). Live proof waits for the next authorized assembled enrollment run. No stack, reservation or Docker object was created. Existing committed run records are unchanged.

## Sites and predicates

All five commands at the three assigned sites now call `Stack.WriteNode` through the receipt-embedding `probe.writeNode` adapter in `tests/e2e/at/enrollment/node_write_test.go`.

| Site | Command and exact predicate | Receipt in the consuming check |
|---|---|---|
| `freeze_test.go`: `prepareFreezeSource` → `prepareFreezeNode` | Capacity PATCH: `NodeFieldsMatch(body)` includes all four capacity fields: `down_bps=200000000`, `up_bps=200000000`, `member_down_bps_cap=100000000`, `member_up_bps_cap=100000000`. | `serving_freeze_source.detail.PATCH.node_write` |
| Same freeze setup | Publication PUT: `NodePublished(group.GroupId)` requires published intent and exactly the new group's ID. | `serving_freeze_source.detail["PUT/publication"].node_write` |
| `handoff_test.go`: `installerPublish` | Capacity PATCH: the same four-field `NodeFieldsMatch(body)`. | Installer check detail: `installer_publications[].PATCH.node_write` |
| Same installer setup, initial and after reinstall | Publication PUT: `NodePublished(p.servingGroup)` requires published intent and exactly the serving group's ID. | Installer check detail: `installer_publications[]["PUT/publication"].node_write`; append preserves both invocations. |
| `live_test.go`: `revokeNodeReason` | Revocation POST: `NodeRevoked`. The command-only reason is excluded from the read-back predicate. Used by incarnation history, clone recovery and controlled installer reinstall. | `incarnation_history.detail.node_writes["POST/revoke"].node_write`; clone recovery's returned detail and installer check detail retain `["POST/revoke"].node_write`. |

A search of the owned sources found no other node mutation. Enrollment-code revocation is a different resource and retains its existing behavior. There are no node-write exceptions and no expected-error stale-node-ETag test in this lane to change.

Receipts retain all attempts, re-reads, timestamps, exact body hashes, keys, ETags, real response statuses and outcomes, including failures. The helper alone owns conflict handling: only exact `412 RESOURCE_VERSION_CONFLICT`, up to three writes and ten seconds, with the same body and key. The old revoke loop's retries on transport errors and unrelated 412s are removed. A 429 stops and remains visible in the lane's limiter review, without an outer retry. An initial GET failure records its safe response and an empty receipt because no write was attempted.

## Observed state still gates acceptance

Command acceptance, including already-intended read-back, does not establish applied-state completion. Freeze retains readiness, signed populated serving lease, heartbeat, rules and the positive forwarding control. Installer publication retains readiness, native exit selection, actual serving read-back and counted traffic. Revocation retains incarnation retirement, preserved historical usage, clone recovery and retired-identity checks.

For controlled reinstall, an ordinary 202 supplies the real operation ID. An already-intended 412 supplies no operation; the consumer resolves the existing successful `node.revoke` audit event for this node since the observed incarnation retirement. Missing, ambiguous, old, wrong-resource or incomplete audit evidence fails closed. The operation GET must match the ID, node and kind, and both `netbird_route_readback` and `netbird_identity_cleanup` must succeed as before. Full-operation completion is still not claimed for the stopped old agent.

## Stack-free audit and its exceptions

`TestF10_NodeWriteAudit` parses every Go source under the enrollment package, regardless of build tag. It rejects node writes through general API or HTTP calls, unknown write destinations, method aliases and direct `Stack.WriteNode` calls outside the receipt adapter. `TestF10_NodeWriteAuditRejectsBypasses` injects literal, concatenated, aliased, constant and dynamic bypasses. Unused exception entries also fail the audit.

These are the only transport-boundary exceptions; callers remain audited:

| Exception | Reason |
|---|---|
| `live_test.go:call:callHeaders` | API dispatch boundary; every `call` site is audited. |
| `live_test.go:callHeaders:Call` | API dispatch boundary; every `callHeaders` site is audited. |
| `live_test.go:post:NewRequestWithContext` | Anonymous agent POST constructor; every `post` URL is audited. |
| `live_test.go:post:Do` | Transport for the same audited anonymous agent POSTs. |
| `proof.go:RoundTrip:RoundTrip` | Existing lost-enrollment-response fault transport; callers and request constructors remain audited. |
| `proof_test.go:TestAT06_ResponseActuallyLostBeforeCaller:Post` | Loopback fault test makes two deliberate single-shot calls to measure response loss and subsequent recovery. Adding conflict retries would change that measurement. |

## Offline evidence

`TestF10_NodeWriteSites` calls the freeze setup, installer setup and revoke helper against local TLS cookie/CSRF stubs. Each command covers acceptance, conflict/retry, exact already-intended read-back, unrelated 412, three-write exhaustion, failed re-read, transport failure and stopped write/re-read rate limits. Capacity tests independently change and omit every field. Publication tests reject wrong, extra, duplicate, missing and null groups and disabled intent. Revocation rejects active, retired and missing lifecycle values. Checks are serialized through the real probe spool to verify receipt embedding, key/body/ETag evidence and credential exclusion; both installer publication receipts must survive.

`TestF10_RevokeOperationReadback` covers ordinary and already-intended operation resolution and rejects missing, ambiguous, old, wrong-node, wrong-action, failed, paginated and unreadable evidence. The offline command proof does not replace the next live serving/retirement proof.

Run admission separately with `bash "$FVPN_HOLD_SH"` before every gate; proceed only after `free`. Gates use disk scratch through `python3 -B tests/e2e/at/enrollment/disktmp.py`. The lane lint target includes offline tests and amd64/arm64 test-binary compilation. Standalone commands:

```sh
make -f mk/at-enrollment.mk at-enrollment-lint
python3 -B tests/e2e/at/enrollment/disktmp.py go test -count=1 ./tests/e2e/at/enrollment/...
python3 -B tests/e2e/at/enrollment/disktmp.py go test -tags e2e -count=1 -run '^TestF10_' ./tests/e2e/at/enrollment/...
```

### Requests to orchestrator

2026-11-03 — **0 new requests.** Devstack 33's adoption hand-off is answered. Live proof remains scheduled for the lane's next authorized assembled run; this offline session makes no new live acceptance claim.
