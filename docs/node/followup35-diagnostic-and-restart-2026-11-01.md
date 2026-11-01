# Follow-up 35: diagnostic exclusion and restart deny sets

Baseline: `48665f70899fb1914dbb8f016c23f26d6cac73cc` (`main` after #389).
Production code is unchanged. These are deterministic tests of that runtime,
not a new kernel traffic measurement or full AT acceptance result.

## `ctdenyd` never becomes member usage

Netlab 24's [return-denial guard and counter](../poc/G2-followup24-2026-11-01.md)
count rejected return traffic in `diagnostic_v1`. The agent already excludes
that name from usage:

| Consumer | Existing boundary | Proof |
| --- | --- | --- |
| Meter selection | `node/internal/meter/delta.go`: `parseUsageCounter` accepts only `u_`/`d_` plus a canonical 32-hex device ID; `usageCounters` filters before attribution, reset comparison and persistence. | `TestAT14_F35_CtdenydUsageSelection` includes positive upload/download controls. |
| Sample-path reset classification | `node/agent/meterreset.go`: `usageName`, `usageOnly`, `liveUsage`, `nudgeOne`, `classifyExternal`. | `TestAT16_F35_CtdenydResetAndReportConsumers`: increasing, decreasing and disappearing diagnostic readings, alone and with live member counters. |
| Apply reset/error consumers | `applyNeedsForce`, `kernelSource.noteApply`, `publishMeterEpoch`, `noteResetsWithoutKernel`. | The same test supplies real raw reset evidence, checks no nudge or `METER_EPOCH_CHANGE`, and retains a positive member-reset control. |
| Raw Apply persistence | `Engine.enforce` saves all `ApplyResult.Counters` to agentstore; these are enforcement snapshots, not the usage ledger. | `TestAT16_F35_CtdenydApplyPersistenceDoesNotBecomeUsage` runs Step/Apply, reads the raw diagnostic through an independent DB connection, and samples through the real kernel source/meter. |
| Usage persistence and reporting | The meter stores only selected readings; `usage snapshot` reads the resulting ledger via `usagestore.ReadSnapshot`. | The agent proof and `TestAT16_F35_CtdenydDoesNotChangePersistedUsageOrEpoch` check no events, extra epochs or diagnostic readings. The latter closes/reopens SQLite and the meter on the same boot, and proves exact member upload/download deltas. |
| Counter sample reports | `enforce.Samples`, also used by `tests/netlab/cmd/fvpn-netlab`'s `countersCmd`, excludes diagnostics before joining device/member bindings. | `TestAT14_F35_CtdenydCounterSamples` and the probe-exception fixture. |

The memPlane diagnostic fixture and both probe-exception excluded-name
assertions now include nonzero `ctdenyd`. Meter tests include values above
2^53, disappearance/reappearance and restart. Opening a new ledger still
creates its ordinary baseline epoch. The assertion is **no diagnostic-induced
epoch**, not that a running meter has no epoch at all. Likewise the generic
`enforce.DetectResets` can report a diagnostic decrement in raw evidence; the
agent must not convert it into a member reset or gap.

No path from `ctdenyd` to member usage was found. No selector repair, schema,
dependency, updater, installer or enforcement change was needed. This covers
MTR01/MTR03/MTR05 and SEC02, §10.1/§10.5/§10.7, with AT14/AT16 mechanism proofs.

## When configured denial grows

The actual configuration fields are `ExtraDeniedV4` (`extra_denied_v4`) and
`HostV4` (`host_v4`); there is no `ExtraHostV4` field. The command's `open`
calls `LoadConfig` and then `Start`, which captures that Config. `Run` calls
`Boot` before the first `Step`. SIGHUP calls `reloadResolvers`; it does not
reload these fields.

Every new decision goes through `Engine.input` and `Decide`/`buildModel`:
`input` parses the captured slices, and the model contains `ExtraDeniedV4`
and `HostV4`. `Model.denyV4` unions extra denials with the spec list;
`enforce.Render` flushes/fills `denied_v4` and `host_v4` in the full Apply
transaction, including the SNAT address in the host set. Therefore the
**first full Apply after restart renders the new sets**. With working control
and an available lock, this occurs in the first Step. An ordinary Boot
baseline is deny-all; it is not a full Apply that fills these sets.

There is, however, **no guarantee that serving waits for that Apply**:

1. `Boot` loads the retained model, desired document, same-boot lease and
   heartbeat, then calls `adoptStoredServing` (`node/agent/restart.go`).
2. Adoption rebuilds a candidate model from the **new config** and invokes
   `verifyServingModel`, retaining only the installed time sample. It does
   not install that candidate.
3. `enforce.VerifyServing` calls `verifyServingRuleset`
   (`node/enforce/verify_tc.go`, `node/enforce/verify.go`). The verifier checks
   guards/order and member lifetimes, and compares resolver contents where
   applicable, but does **not compare denied/host set contents with the
   candidate model**. Its `verifyModelRules` checks rule shape, not set values.
4. Thus old rules containing #389's guards but smaller deny/host sets can pass.
   Adoption sets `appliedModel` to the rebuilt candidate, resumes sampling,
   sets `serving=true` and publishes a serving heartbeat **before any Apply**.
   The retained kernel still has the old sets.
5. The next successful full Apply installs the new sets. Stable member
   identities yield no `Model.Revoke` and no targeted conntrack delete.
   Netlab 24's guards enforce the newly installed sets even for surviving
   conntrack entries; restart alone does not clear those entries.

`TestAT13_F35_RestartFirstApplyRendersConfiguredDenyGrowth` pins both same-boot
adoption and missing-heartbeat baseline paths. It reads the new config through
`LoadConfig`, calls real Start/Boot/Step, and executes real Apply/VerifyServing
against one retained emulated nft/tc executor. It checks the **first** Apply
model and rendered transaction, successful verification, retained old and new
prefixes, and zero conntrack deletes. The adoption branch explicitly records
the current unsafe pre-Apply serving behavior; it is not a safety guarantee.
Documentation prefixes are exempted using the existing TEST ONLY setting so
the test actually enlarges the effective deny set. No host networking runs.

The first-Step statement is conditional: `Engine.enforce` can defer Apply
on a busy node lock, and Step's control-timeout `retain` branch can skip Apply
when the documents/deadline are unchanged. Both can retain adopted service.
Original lease/kernel deadlines still apply. No new maximum adoption-to-Apply
delay is claimed. This is ISO01/ISO02 and §8.1/§8.2/§16.3, AT13 scope; live
restart traffic and physical arm64 behavior remain unmeasured here.

## Dated hand-off

**2026-11-01 — Orchestrator / netlab enforcement owner, with agent-core:**
resolve the pre-Apply adoption gap. `node/enforce/verify.go` is outside this
lane's ownership. Reject retained rules whose effective `denied_v4` or
`host_v4` contents fail the new model's protection, or coordinate an agent
adoption rule that baselines when those protections grow. Add the owner-side
verifier proof and a same-kernel restart traffic test with a surviving
established connection, including delayed/blocked first Apply. Do not infer
installation from adoption's rebuilt `appliedModel`. No production fix or
hot-reload feature is proposed by this proof-only delivery.
