# Follow-up 19 — combined-source acceptance (TEST ONLY)

The unchanged real installer-node serving campaign passes **14 / 0 / 0**
(pass / blocked / fail) on source `a9ad1c47`, based on main `33034048` with
#307, #310, #320, #323 and #326. Product source is unchanged in this lane.
The initial real `apply` exits **0**; the candidate pointer and running agent
executable both switch; all five health checks pass. The real central revoke
and candidate re-enrollment pass, with **no HASH_MISMATCH and no tree repair**.

## Serving upgrade and candidate seed

[Run d6aaa798](d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-journaled-upgrade.md)
retains the before/after checkpoint, complete read-only handshake trace,
durable status, coherent journal receipt, actual agent/updater log, staged
pointer, current pointer, running executable and installer check.

- Verified staging leaves the original pointer unchanged.
- At 14:07:30.999575825Z, the member allow-set is populated and control returns
  HTTP 200. The persisted latch is `b455cf8d-3a77-4909-a8a7-c60f9ed89be2`;
  its original window is 14:07:31.266673717Z–14:08:01.266673717Z on boot
  `97c018d7-a925-4723-9127-83c097de1c8a`.
- At 14:07:34.708073795Z, the real five-field checkpoint has that byte-equal
  `barrier_id`, the member allow-set is empty, and authenticated control still
  returns 200. No checkpoint is supplied by the test. The heartbeat sampled
  at that instant is older than the nft withdrawal; both raw observations are
  retained instead of treating the stale heartbeat as the forwarding state.
- The real epoch closes with `close_reason=final_sample` at
  14:07:34.685593870Z. The final sample is quiet: it creates no new usage event.
  The canonical last durable event remains sequence 4, ACK 3, in that epoch.
  `final_sample_event_verified` in the raw record means verified epoch closure
  plus a hash-verified canonical watermark event; it does not assert new bytes
  after withdrawal. The sealed receipt carries those exact sequence/ACK/closed
  epoch values and a distinct new epoch, with its canonical payload hash checked.
- The receipt is `succeeded`, health is 5/5, and the initial process exits 0.
  Publication remains disabled and forwarding blocked as recorded in the receipt;
  successful local application does not assert republication.
- The post-switch installer check reads `release_source=updater_journal` and
  retains the initial install record. The candidate [re-enrollment record](d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-reenroll.md)
  shows successful seed verification of that candidate, retired identity and
  bound state, byte-identical canonical origin events after D-118 adoption,
  restored services, and old-certificate refusal (403 direct and proxy).
  Its deliberate quiescent seed check reports inactive-unit drift; the actual
  seed subprocess exits 0, followed by the recorded successful restoration.

The coherent updater journal is retained privately as
`.agent/tmp/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91-installer-journal.db`,
81,920 bytes, SHA-256
`7ae263bdb423ad4bdd18a34129f49769e8c14dc9c4871f0703bfd081f1c5dce4`.
SQLite integrity is `ok`; the sealed receipt hash is
`10f1cda9ce46c7d3e9a1a99fc7d2f1dcff0b8988ca587daf438a53b27fc9d6e7`.
Binary journals, usage stores and packet payloads are not public artifacts.

## Evidence preservation and limits

Before any new publication, all 138 files in follow-up 18 were copied
byte-identically to `.agent/tmp/followup19/archive-followup18` and pinned by
[followup18-archive.sha256](../../../followup18-archive.sha256).
The follow-up 16, 17 and 18 committed roots remain unchanged. The original raw
pcaps were copied to the follow-up 19 private directory before new capture;
new captures use run-specific names. Original serving and stopped-node journals
remain in place. Historical failures are not relabeled by this success.

Two setup-only attempts were stopped for shared-host admission before apply:
`5e9fb189` (three setup rows) and `4e59a2c4` (six setup rows). Their partial
records remain in this directory; neither is a completed acceptance campaign.
Subsequent heavy commands run behind a visible foreground `gates.sh` marker.

[Retained F8-02 capture decode](capture-decode/README.md) includes every packet
header at the four exact requested size/SHA receipts and distinguishes the
available 22,022-byte failure from the absent original 11,011-byte capture.
No product or NetBird MTU/MSS adjustment is made.

[D-093 reconciliation](../../../portal-d093-reconciliation-followup19-2026-10-31.md)
retains all 15 row agreements, with 13 open and two retired within simulator
scope. Cross-lane answers and the D-087 ownership-boundary request are only in
[RESPONSES.md](../../../RESPONSES.md). D-124's named fixture contains no
`agent.Decide` call on this base; its existing tests pass unchanged.

## Reproduction

Set `FVPN_E2E_PREFIX=e2e`, `FVPN_E2E_PORT_BASE=18000`, `CGO_ENABLED=0`,
`GOTOOLCHAIN=local`, and disk `TMPDIR`/`GOTMPDIR` under `.agent/tmp`.
Check `pgrep -af 'gates.sh|mk/at-'` before heavy runs; wait while two other
suites are active. Each campaign uses a separate fresh stack:

```sh
make -f mk/e2e.mk e2e-up E2E_RESET=1
FVPN_E2E_FOLLOWUP19=1 go test -tags e2e -count=1 -v -timeout 30m ./tests/e2e -run '^TestE2E_Followup19Installer$'
make -f mk/e2e.mk e2e-down
```

Add `FVPN_E2E_CANDIDATE18=1` only for the separate stopped-agent sequence;
it cannot answer a fresh D-125 latch while stopped and must retain that failure.
For native capture use `TestE2E_Followup19Native` on another fresh stack.
For DNS set `FVPN_E2E_TWO_RESOLVERS=true` during up and
`FVPN_E2E_FIXTURES16=1 FVPN_E2E_FOLLOWUP19=1` during its focused runner.
The final gate order and real exits are recorded in the worktree hand-off.
