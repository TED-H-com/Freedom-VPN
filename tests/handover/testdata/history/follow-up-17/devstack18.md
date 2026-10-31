# Follow-up 18 — installer serving, genuine checkpoints and native evidence

TEST ONLY, prefix `e2e`, port base `18000`, product base `f88bf1c` (D-116 and
agent-core follow-up 24). All changes are harness/evidence changes. No product
verifier, marker implementation, ledger event, observation or generation was
modified to manufacture a passing result.

**ENR-05 passes on the real installer node.** The complete serving campaign
`09cb1198` has **13 pass / 0 blocked / 1 fail**, exit **1**. Its remaining failure
is the serving-node updater barrier, not installer ownership. The native campaign
`daf790fc` has **9 pass / 0 blocked / 2 fail**, exit **1**: MTU transfer failure and
an intentionally retained first-304 capture that cannot prove new acceptance.
Separate supplements below isolate candidate health and exercise a new envelope.
The isolated candidate campaign `8a1124a1` is **13/0/1**, exit **1**: candidate
health passes, but subsequent reenrollment fails baseline seeding. The final
first-offer supplement `5f74d7c3` is **2/0/0**, exit **0**. These results do not
close all AT07, AT09 or SECOPS-21 cases.

## Installer outcomes

D-116's fresh installation serves counted member traffic. The fault fixture then
places the exact legacy follow-up 3 marker in a stopped node; **the installer
itself** migrates it on rerun. Complete nft listings show only the four rendered
chains in `inet fvpn` and the marker in separate hookless `inet fvpn_install`.
The installer check, fresh heartbeat, active service and verifier read-back pass.

The held CLI snapshot has sequence 2, ACK 1, one unacknowledged event with
1,376 upload / 1,001 download bytes. Before/after wrapper ledger hashes and raw
CLI bytes match. Services restore, the fresh heartbeat serves and persistent
TCP/UDP traffic resumes. Releasing the 503 hold reaches sequence/ACK 3 with no
unacknowledged events. This establishes same-incarnation preservation and ACK
catch-up; it does not establish D-105 retired-stream delivery.

The serving-node upgrade stages, then `apply` exits **0** while sealing
`state=failed`, `stage=final_sample`, `journal_stage=rolled_back`, health null.
The genuine checkpoint is absent before and immediately after apply; the current
release does not switch. The coherent SQLite journal copy includes WAL and has
an 81,920-byte SHA-256 receipt; its private copy passes `PRAGMA integrity_check`.
Agent-core documents why a serving node has no stale checkpoint and why the
updater needs a bounded real barrier handshake. Owner: **updater**,
`node/updater/host.go` (outside this lane); process exit handling is also that
owner's follow-up 15. No checkpoint or success exit is substituted here.

## Live scenario table

Each link contains the immutable JSON/Markdown observations and source commit.
Passes are scoped to their row, not full acceptance cases.

| Campaign / scenario | Result | Owner / evidence scope |
|---|---|---|
| Installer [admin-login](09cb1198-435f-4609-9d0b-d53e6c1ed441/admin-login.md) | pass | e2e — §14.2 |
| Installer [capture-and-boot](09cb1198-435f-4609-9d0b-d53e6c1ed441/capture-and-boot.md) | pass | e2e — AT23, ISO01, F9-06 |
| Installer [enroll-bootstrap](09cb1198-435f-4609-9d0b-d53e6c1ed441/enroll-bootstrap.md) | pass | e2e — §6, §19 AT06 |
| Installer [fixture-member-admission](09cb1198-435f-4609-9d0b-d53e6c1ed441/fixture-member-admission.md) | pass | e2e — §8, AT02, AT12 |
| Installer [installer-journaled-upgrade](09cb1198-435f-4609-9d0b-d53e6c1ed441/installer-journaled-upgrade.md) | fail | updater — missing serving-node barrier handshake; apply exits 0 on sealed failure |
| Installer [installer-marker-migration](09cb1198-435f-4609-9d0b-d53e6c1ed441/installer-marker-migration.md) | pass | e2e — AT07, ENR-05, D-116 |
| Installer [installer-reenroll](09cb1198-435f-4609-9d0b-d53e6c1ed441/installer-reenroll.md) | pass | e2e — ENR-06, AT07, §6.3 |
| Installer [installer-serving-readback](09cb1198-435f-4609-9d0b-d53e6c1ed441/installer-serving-readback.md) | pass | e2e — ENR-05, D-116, AT07 |
| Installer [installer-usage](09cb1198-435f-4609-9d0b-d53e6c1ed441/installer-usage.md) | pass | e2e — §10.5, ENR-05 |
| Installer [installer-wrappers](09cb1198-435f-4609-9d0b-d53e6c1ed441/installer-wrappers.md) | pass | e2e — §6.3, ENR-06, AT07 |
| Installer [node-restart](09cb1198-435f-4609-9d0b-d53e6c1ed441/node-restart.md) | pass | e2e — §5, §6, F2-05 |
| Installer [publish](09cb1198-435f-4609-9d0b-d53e6c1ed441/publish.md) | pass | e2e — §5.2, §14.2 |
| Installer [regular-kernel-boot](09cb1198-435f-4609-9d0b-d53e6c1ed441/regular-kernel-boot.md) | pass | e2e — AT23, F9-06 |
| Installer [serving-catalogue](09cb1198-435f-4609-9d0b-d53e6c1ed441/serving-catalogue.md) | pass | e2e — §9, §14.2 |
| Native [admin-login](daf790fc-81f7-49a2-8a65-f3ec8faec656/admin-login.md) | pass | e2e — §14.2 |
| Native [desired-state](daf790fc-81f7-49a2-8a65-f3ec8faec656/desired-state.md) | pass | e2e — §5, §8 |
| Native [enroll-bootstrap](daf790fc-81f7-49a2-8a65-f3ec8faec656/enroll-bootstrap.md) | pass | e2e — §6, §19 AT06 |
| Native [first-desired-offer](daf790fc-81f7-49a2-8a65-f3ec8faec656/first-desired-offer.md) | fail | e2e assertion — actual first response was 304; see separate new-envelope supplement |
| Native [fixture-member-admission](daf790fc-81f7-49a2-8a65-f3ec8faec656/fixture-member-admission.md) | pass | e2e — §8, AT02, AT12 |
| Native [namespace-restart-90](daf790fc-81f7-49a2-8a65-f3ec8faec656/namespace-restart-90.md) | pass | agent-core / members — current fixture recovers; original high-generation failure not closed |
| Native [native-mtu-comparison](daf790fc-81f7-49a2-8a65-f3ec8faec656/native-mtu-comparison.md) | fail | netlab / wire-nodes — 1280 native transport path |
| Native [node2/publish](daf790fc-81f7-49a2-8a65-f3ec8faec656/node2/publish.md) | pass | e2e — §5.2, §14.2 |
| Native [publish](daf790fc-81f7-49a2-8a65-f3ec8faec656/publish.md) | pass | e2e — §5.2, §14.2 |
| Native [second-exit-counted](daf790fc-81f7-49a2-8a65-f3ec8faec656/second-exit-counted.md) | pass | e2e — AT09, F8-01 |
| Native [serving-catalogue](daf790fc-81f7-49a2-8a65-f3ec8faec656/serving-catalogue.md) | pass | e2e — §9, §14.2 |
| Candidate [installer-quiescent-candidate](8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179/installer-quiescent-candidate.md) | pass | e2e / updater — genuine shutdown checkpoint; all five candidate checks and sealed success |
| Candidate [installer-reenroll](8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179/installer-reenroll.md) | fail | installer / updater — upgraded-tree baseline seed HASH_MISMATCH |
| Supplement [first-desired-offer](5f74d7c3-6404-461d-9f47-296bd69958c2/first-desired-offer.md) | pass | security-ops hook — actual first response, exact local digest and durable applied receipt |

## Candidate health and post-upgrade reenrollment

`8a1124a1` explicitly stops the real watchdog/agent after its serving/usage
proof. The agent closes its meter epoch and writes the genuine checkpoint
(sequence 4 / ACK 3). Staged installer apply then switches to the candidate,
passes **processes, identity/state, internet-only, DNS policy and tunnel probe**,
and seals `succeeded / locally_succeeded`. The subsequent genuine checkpoint is
sequence/ACK 4. The coherent journal is 81,920 bytes with SHA-256
`d78f02ceea0bf781c0bb4e7ab9b83e3af40c006cb25df0b0ff1635ea66efff08`, and its
private snapshot passes SQLite integrity checking. No checkpoint is synthesized.
This isolates candidate identity/health; it does **not** repair or supersede the
serving-node barrier failure above.

The later real revoke/re-enroll command preserves the ledger and retires the
old identity, but exits 1 (`baseline_failed`). A separate read-only check returns
2 and retains the seed error: **`HASH_MISMATCH: seed tree differs from the signed
artifact`**. The installer resolves the current upgraded tree before calling
seed; the wrapper does not supply an older artifact to reenroll. Owner:
installer/updater, `packaging/installer/baseline.go` and updater seed.
[Exact read-only diagnostic](8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179/post-upgrade-reenroll-diagnostic.md).
Do not count this as a successful post-upgrade reinstall. The earlier
pre-upgrade controlled reinstall passed separately in `09cb1198`.

## SECOPS-21 first delivered offer

The final focused supplement `5f74d7c3`, source `4e0ff498`, changes capacity via
real PATCH/ETag, reads the requested capacity back, and resumes the retained
agent through the hold. Compilation and generation advancement happen on its
actual GET. Retained generation **3** receives **4** first at
`2026-10-31T10:54:54.90936473Z`; HTTP 200, 1,479 bytes, digest
`05bc28b50a0c50e1a8622edc1779bc540df5b926fe0084bca170404596fcae58`.
SQLite accepted/applied generation and envelope digest match. The subsequent
applied observation returns 202 and its exact report ID exists centrally with
applied generation 4. This is first-delivery plus acceptance/ACK evidence, not
inference from a compiler row. The node proxy binary is unchanged from the fresh
`a0089b8` image; only the invoking test routine changed between supplements.

The earlier first 304 remains archived and cannot be overwritten by later 200s.
Security-ops must use this hook in its own restore workflow; this test does not
perform PostgreSQL restore or close the full SECOPS-21 row.

## F8-01 namespace restart

The second exit has counted traffic before destruction of its node, resolver and
namespace processes. Generation remains **3 → 3** throughout. `daf790fc` records
88 heartbeat samples over 90.37 seconds, complete boot/end nft/tc reads, all
central observations and the agent log. The first fresh serving observation is
**5.96 seconds** after restart. The earlier `2454c833` attempt recovered after
15.79 seconds (its pre-fault second-exit route assertion was a harness failure).
Both capture an initial `TC_FOREIGN_FILTER` / missing-link read, followed by
recovery. No generation bump, tc repair or state clearing was used.

This does **not reproduce members' persistent failure at generation 30 after
its full mutation history**. Members and agent-core should use the exported
`RestartNodeNamespace90` in that exact sequence. No product fix is claimed from
this smaller fixture; kernel boot identity is unchanged.

## F8-02 MTU comparison and limits

All rows request 64,064 echoed payload bytes. The selected exit remains directly
connected before and after each transfer. Availability filters select a family;
the table reports the observed family rather than assuming it.

| Input / reconnect | `2454c833`: chosen family; bytes / milliseconds | `daf790fc`: chosen family; bytes / milliseconds |
|---|---|---|
| dual 1500 / no | ipv6; 64,064 / 1 | ipv4; 64,064 / 2 |
| dual 1280 / no | ipv6; 10,010 / 12,011 | ipv4; 64,064 / 1 |
| IPv4 1500 / yes | ipv4; 64,064 / 1 | ipv4; 64,064 / 2 |
| IPv6 1500 / yes | ipv6; 64,064 / 2,061 | ipv6; 64,064 / 205 |
| dual 1500 / yes | ipv4; 64,064 / 1 | ipv6; 64,064 / 1 |
| dual 1280 / yes | ipv4; 64,064 / 1 | ipv6; 2,002 / 12,010 |
| IPv6 1280 / yes | ipv6; 2,002 / 12,010 | ipv6; 10,010 / 12,011 |
| IPv4 1280 / yes | ipv4; 22,022 / 12,010 | ipv4; 64,064 / 2 |

MSS is **1240 from the member and 1460 from the origin** in both families,
including all rows in the second run. The member underlay interfaces change to
1280; the native overlay remains 1280. Captures from both member and exit show
**no IPv4 fragmentation-needed or IPv6 packet-too-big message** in these bounded
windows. IPv6 neighbor/router discovery is present and is not a too-big signal.
The second run uses immediate capture mode and retains tcpdump statistics,
with zero reported kernel drops. The earlier fast-success captures could be
empty because capture shutdown preceded buffer delivery; retain that limitation.

The failure occurs without reconnecting as well as with reconnecting, so it is
**not explained by the matrix's reconnect step alone**. It remains a measured
native transport/PMTU-path failure for netlab and wire-nodes. Both families have
failed in at least one attempt; neither family is universally broken in this
small sample. Exact packet-loss/MSS/fragmentation root cause is not proved.
The unchanged bulk client sends 64 lines and validates each echo; no retry or
counter reset converts incomplete transfers into passes. Device counters grow
on failed transfers. Private bounded pcaps are exported with size/SHA-256
receipts under `.agent/tmp`; packet contents are not committed.

## Preserved attempts

[Follow-up 17](../followup17/README.md) and all its failures remain untouched.

| Attempt / source | Actual outcome and limitation |
|---|---|
| `abd7ce3a` / `0b9fb5e` | Interrupted when shared-host admission changed. 9 recorded pass / 1 fail, not a complete run. Rapid deliberate service restarts hit packaged start limits; held usage existed but restored traffic failed. Its serving-readback pass used a stale heartbeat and is **invalidated**, not accepted. Candidate health later sealed succeeded with all five checks and a genuine checkpoint on this already-stopped source; that does not prove a serving-node handshake. Raw journal export was not yet implemented. |
| `09cb1198` / `84b6a34` | Complete serving installer campaign, exit 1, 13/0/1. Restart campaigns are paced beyond the unchanged packaged start-limit interval; service/heartbeat freshness is checked. Genuine final-sample failure remains. |
| `2454c833` / `114d31d` | Native exit 1, 8/0/3. Missing explicit release argument in agent restart and single-route selection with two published exits were harness failures. MTU failures and the full namespace window are retained. |
| `daf790fc` / `fc57099` | Native exit 1, 9/0/2. Correct selected exit has counted traffic. The first response after an unchanged-generation hold is really 304; the observer correctly refuses to credit a later response. Runtime images and the compiled runner stayed at this source; subsequent edits changed only not-yet-run test routines. |
| `7c8338ec` / `fbe4a7b` | First-offer supplement exit 1, 1/0/1. The test waited for a compiled envelope before issuing the agent GET that triggers compilation. The archived timeout remains; later observed delivery is not retroactively credited. The final test reads the requested capacity back before releasing the agent; it does not wait for compilation or generation advancement while GETs are held. |
| `8a1124a1` / `a0089b8` | Isolated candidate campaign exit 1, 13/0/1. All candidate health and journal checks pass; the subsequent post-upgrade reenrollment fails genuine seed tree validation. |
| `c1a63711` / `a0089b8` | First-offer supplement exit 1, 1/0/1: waiting for desired-generation advancement while the agent GET is held is also invalid. |
| `14ea102e` / `7cd1b64` | Same timeout, exit 1, 1/0/1, after making decimal parameter encoding explicit. There is no query error; capacity read-back replaces the invalid generation prerequisite. |
| `5f74d7c3` / `4e0ff49` | Final first-offer supplement exit 0, 2/0/0; actual delivered 4 > retained 3, local digest/applied match and central receipt. |

One setup-only down/up overlap exited 2/2 because teardown removed freshly built
image tags. No acceptance test ran in that failed setup. It was followed by a
serial successful teardown, zero owned inventory and fresh up. Subsequent stack
transitions wait for completion. No other lane's objects were touched.

## Reproduce

Check shared-host admission first. Each installer/native campaign requires a
fresh stack; do not run them together against one mutable fixture. Use disk
scratch under `.agent/tmp` for `TMPDIR` and `GOTMPDIR`.

```sh
export FVPN_E2E_PREFIX=e2e FVPN_E2E_PORT_BASE=18000
mkdir -p .agent/tmp
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp"
make -f mk/e2e.mk e2e-up E2E_RESET=1
FVPN_E2E_FOLLOWUP18=1 go test -tags e2e -count=1 -v -timeout 30m \
  ./tests/e2e -run '^TestE2E_Followup18Installer$'
make -f mk/e2e.mk e2e-down
```

For native MTU and namespace evidence, use a separate fresh up and
`FVPN_E2E_NATIVE18=1` with `-run '^TestE2E_Followup18Native$'`. The first-offer
supplement uses the retained enrolled primary from either campaign:
`FVPN_E2E_FIRST_OFFER18=1` with `-run '^TestE2E_Followup18FirstOffer$'`, before
down. For the isolated candidate-health run, add `FVPN_E2E_CANDIDATE18=1` to
the installer command on another fresh stack. This explicitly stops the agent
and is separate from the required serving-node upgrade verdict.

Always run `e2e-down` even on an expected failed assertion. Service state,
observed traffic, durable sealed outcomes and captured first delivery decide
success, not shell exit alone. [D-102 responses and hooks](../../../RESPONSES.md),
[portal D093 reconciliation](../../../portal-d093-reconciliation-2026-10-31.md).
