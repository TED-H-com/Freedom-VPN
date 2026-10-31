# D-116 installer marker scope — 2026-10-31

TEST ONLY lab evidence; ISO01 / §8.2, enrollment isolation / §6.3.
The verifier and renderer are unchanged. The installer defaults to a separate
`inet fvpn_install` table with an unhooked `owner` chain. The installer change
belongs to its own lane.

## Parsed listing controls

`node/enforce/verify_installer_test.go:TestD116_InstallerMarkerScope` parses
complete JSON ruleset fixtures, including the marker objects. For each of
these forms, a correct `inet fvpn` remains verified and a broken forward-chain
policy retains the **identical error text and code**:

- `inet fvpn_install` with only a table comment;
- the same table with an unhooked `owner` carrying
  `managed-by=fvpn-node-install`;
- the same regular chain with an anonymous counter and drop rule, referencing
  no other object and unreachable from any hook;
- a table-level ownership comment on `inet fvpn` itself.

The regular chain becomes an `ISOLATION_POLICY_INVALID` rejection when given
`type filter hook forward priority 0; policy accept`, with the same drop rule.
The verdict identifies `table fvpn_install family inet hook forward priority 0
chain owner statement drop`. An extra `owner` chain inside **inet fvpn** still
fails with `HOOK_PRIORITY_MISMATCH`: five chains instead of the four rendered
chains. There is no installer-name exemption.

The base-chain distinction is deliberate: the existing
`verifyForeign -> hookCrosses -> checkForeignChain -> walkForeign` policy
allows harmless match/counter/log/accept/continue foreign base chains. Merely
adding an empty, accept-policy forward hook is **not** universally rejected.
The hooked drop control proves that the marker table gets the normal foreign
verifier treatment; it does not introduce a blanket ban or loosen that policy.

## Table comment lifecycle

`Render` begins with `add table inet fvpn`, retains named counters, and uses
`replaceChain` (delete/add) for its four chains.
`RenderBaseline` uses the same table-add/chain-replacement pattern. Neither deletes
and recreates the table, nor writes a replacement table comment. D-104 adds
and removes its temporary guard chain in separate atomic batches; it does
not replace the table object.

`ParseRuleset` handles chains/rules/sets/counters/flowtables but does not put a
table metadata object into `Ruleset.Chains`. `noteChain` separates foreign
identities by family/table, and `verifyChainSet` inspects only the four owned
chain identities. A table-level comment on `inet fvpn` is consequently
tolerated. An **owner chain** is a real fifth chain and is retained by the
renderer, explaining today's installer failure rather than hiding it.

On the measured nft 1.0.9 kernel path, the comment survives Boot and serving
Apply. **Use text read-back for this metadata:** nft 1.0.9's JSON table object
omits it, even when text read-back shows it. The initial JSON-only assertion
failed after verified serving and is retained. The corrected fixture creates
a commented TEST ONLY baseline atomically (same rules), proves the comment in
text before enrollment, then checks it after every measured serving snapshot.
The default remains the separate marker table.

## Real-kernel result

Final campaign: **2026-10-31 09:30:16–09:30:46 UTC**, amd64,
Linux `7.0.0-38-generic`, nft `1.0.9`, iproute2 `6.1.0`; source `3101b19`.
`TestD116_AT10_InstallerMarkerEnrollment` passed in **30.56 s**.
[Retained listings and observations](G2-followup17-evidence/agent/TestD116_AT10_InstallerMarkerEnrollment/):

| Read-back | Result |
| --- | --- |
| `before-enrollment.json` / `.txt` | Separate `fvpn_install/owner` already present; no hook; owned table comment proven in text before enrollment |
| `marker-serving.json` / `.txt`, `marker-observation.json` | Generation 1, forwarding enabled, safety passed, exactly four owned chains, one regular marker chain / zero marker base chains |
| `baseline-serving.json` / `.txt`, `baseline-observation.json` | Marker table removed; generation 2, same serving/safety verdict and packet success |
| `final-serving.json` / `.txt`, `final-observation.json` | Marker restored; 20 published-generation reloads through generation 22, forwarding enabled / safety passed, comment retained |

Both marked and unmarked observations have an empty forwarding-block list and
no error. All 30 actual Apply calls (including same-generation renewals) are
verified in `agent.txt`. The fixture's Internet member probe succeeds in both
cases. Ordinary Docker DNS base chains remain in the full listing; “zero
marker base chains” does not falsely assert that all other tables lack hooks.

The initial JSON-only comment check failed **after** generation 1 had already
verified and forwarded; retain it in `agent-initial`. The corrected comment
fixture passed once before the capture-shutdown improvement
(`agent-pre-boundary-fix`, 31.96 s) and again in the final campaign. Neither run
removed or exempted an `owner` chain inside `inet fvpn` to obtain success.

## Reproduce

After checking the shared-host suite limit:

```sh
pgrep -af 'gates.sh|mk/at-'
FVPN_LAB_PREFIX=fvpn-w1-netlab-f17-agent \
FVPN_NETLAB_EVIDENCE_DIR=docs/poc/G2-followup17-evidence/agent \
make -f mk/netlab.mk netlab-agent-measure
```

This target builds the ordinary enrollment binary and fake central fixture,
then uses `fvpn-netlab agent-trace` to run the real agent Engine with a timing
decorator around the production SystemPlane Apply. Every other Plane method
is the production implementation. The fixture installs the marker **before
enrollment**, compares serving with and without it, and performs 20 newer
published-generation reloads. NetBird status and central signatures are test
fixtures; WireGuard, nft, tc and member packets run on the real lab kernel.
It is not a production installer execution or a physical arm64 run.
