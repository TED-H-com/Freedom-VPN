# Follow-up 14 requests — 2026-11-01

**2 PASS / 5 BLOCKED / 3 FAIL**, with **628 passed / 6 blocked / 5 failed subchecks**. One assembled ten-case run; live make exit **2**.

D-098 was the first, separate commit `cc04d0c2`: all **61 root files** from run 4182dd3d are byte-identical in [follow-up 13](history/follow-up-13/RUN.md), with [SHA-256 pins](history/follow-up-13/SHA256SUMS). The retained AT09-restart.json still belongs to run 7edb263f. Follow-up-13-captures, follow-up-13-stream and every earlier archive are untouched. Handover claims were checked: tests already read frozen history files; no outside-owned D-132 exception was needed.

**D-136, corrected:** a verified deny-all candidate is used only while an Apply error is retained. The ordinary stale-lease path installs the node-wide baseline, ACKs, renews, then Applies, verifies and heartbeats. No finite outage ceiling follows. F12-01 is labeled **policy_missing baseline during generation 11→13; stale-lease path consistent with code; no failed Apply**. `cause_status=unproven` until the desired input is joined. Netlab 21 request 3 is settled by D-135; this is a cross-reference, not a renewed request.

### Requests to orchestrator

**Nine pending groups, dated 2026-11-01.** Work stops only for the named outside-owned item. No outside-owned file was changed.

1. **F4-03 — platform/operator:** real platform/QUIC, physical second NAT/roaming and PMTU resources; shared native fixtures/platform integrations are outside this lane. Owner-resource status unchanged.
2. **F4-04 — portal/devstack/operator:** rendered expired/suspended screens (`apps/portal`); API/traffic evidence does not certify UI. Owner-resource status unchanged.
3. **F4-10 — packaging/devstack/operator:** boot-flavour provenance and physical arm64 (`packaging`, shared boot fixtures). Cross-build is separate. Owner-resource status unchanged.
4. **F4-05/F4-07 — native integration/operator:** production nameserver distribution, Private DNS, browser DoH and Unicode UI/platform resources. Owner-resource status unchanged; devstack 23’s positive IDNA wire fixture is now consumed and is no longer a missing helper.
5. **F4-06 — usage/database/backup/business:** representative URL/privacy and remaining raw/rollup/file/backup retention surfaces and commercial feed approval (`internal/usage`, retention/backup components and operator resources). Owner-resource status unchanged.
6. **F11-01 and new F14-01 — native NetBird/WireGuard integration:** retain historical F11-01 before-member-IPv6-prerouting investigation, with no repair on main (`deploy/devstack/Dockerfile.node`, shared capture/transport fixtures). Separately diagnose [F14-01’s original failed re-grant](F14-01-regrant-forwarding.json) after valid readiness. Do not assume a common cause; this new attempt lacks detailed native capture. AT01’s successful first selection does not close either historical failure.
7. **F11-03 / D-133 — orchestrator and agent-core:** no failed Apply was observed; retain blocked failed-Apply measurement. All 96 successful Applies now contain commands, so no engine opt-in request remains. Any stronger admitted-sibling availability contract belongs to agent-core design (`node/agent/engine.go` and enforcement). D-136 has no finite outage ceiling; netlab 21 request 3 is already answered by D-135.
8. **F13-01 — netlab/enforcement owner:** inspect the new valid-JSON duplicate-chain sample against `node/enforce/parse_nft.go` (noteChain, around line 400). [Safe public receipt and overlap evidence](F14-F13-01-private-receipts.json); raw/error files remain private. Decide whether identical duplicate definitions are legitimate and, only if so, repair the parser with owner tests. Parser defect and underlying snapshot cause remain unproven. Historical failures stay failed; no retry or predicate relaxation.
9. **Assembled qualification — orchestrator:** the single authorized run used harness e9a215f3. Qualify the final routed-after-relay ordering in the next authorized run; commit 682d4434 fixes the isolated harness ordering. Current relay FAIL remains recorded and confounded. No second run was performed here.

Delivered handoffs: devstack 23 routed hop and IDNA fixture; agent-core 31 command lifecycle and corrected D-136; devstack 24 original-query capture guidance and private parse-failure retention. F12-04 remains historically unattributed, with the original new query fully captured; no missing shared capture helper remains. F11-02 readiness and F12-02 stream consumption stay delivered.

[Current evidence](DIAGNOSTICS.md), [run ledger](RUN.md), [prior requests, unchanged](history/follow-up-13/requests.md).
