# Updates requests — follow-up 10, 2026-10-31

Measured on product main `fb5e002549f79e8ef61bebcf42cce1678ab65cb1`, harness `0df8599dc84d4b85e48182f20aa4b0fc741bc72a`, run `695f977c-b587-4768-88f1-0514f3a4f1ac`. Installer extension passes; the regular live gate fails. Original F9 observations remain [byte-identically archived](history/follow-up-9/requests.md).

### Requests to orchestrator

| ID | Owner | Deciding observation | Requested change outside this lane |
|---|---|---|---|
| F10-01 | Devstack; agent-core consult | [AT23](AT23.json): candidate logs `agent: active release identity is inconsistent`; `executable_layout` shows `<current>/fvpn-agent` absent and `<current>/bin/fvpn-agent` present. Journal ends `recovery_required`, `ROLLBACK_FAILED`, no last-good and no candidate attestation. Thus AT24–AT27 cannot run their later scenarios. | Align the shared regular release layout and supervisor with the active-release contract. `tests/e2e/serving.go` builds `bin/` unless InstallerLayout; `deploy/devstack/agent-service.sh` and shared assertions expect `bin/`; `node/agent/release.go` verifies root `fvpn-agent`. Keep executable/journal identity validation. Also update shared recovery FileMeter configuration in `tests/e2e/cmd/fvpn-e2e-nodeconfig/main.go` and `tests/e2e/scenarios_test.go`: they still name the legacy four-member `e2e/meter.json`; this lane's actual apply config uses genuine `state/meter.checkpoint` and keeps the agent active through final sampling. Publish the answer in `docs/e2e/RESPONSES.md` (D-102), then request the complete AT23–AT27 rerun. |
| F10-02 | Installer | [installer hand-off](installer-handoff.json), `generated_config`: `result_url` is empty. Real product delivery and central acceptance pass only after this harness explicitly configures the owned endpoint/CA/origins. | In `packaging/installer/config.go` (`renderUpdater` and its caller), derive/require the authenticated central result endpoint and its trust configuration, with a real generated-config delivery test. F9-04's central 422 is resolved, but the reporting configuration part remains open. No production endpoint is invented here. |

### F9 closure by measured observation

| Prior request | Follow-up 10 deciding evidence |
|---|---|
| F9-01 | Install exit 0; real NetBird and agent active; new daemon socket exists. |
| F9-02 | Default-umask and 077 seed both exit 0 with the same baseline hash/timestamp. |
| F9-03 / D-125 | Genuine checkpoint UUID equals persisted latch; finalized epoch; apply 0, current switch, unit restarts and five health checks. Withdrawn-evidence failure and sealed replay exit 1; blocked recovery exits 2. |
| F9-04 / D-110 / D-111 / D-127 | Real success accepted from null central baseline; installer summary/batch null; exact ACK replay, conflicts, wrong-job refusal, bounded correction, preserved history and ETags pass. Generated reporting remains F10-02. |
| F9-05 | Downgrade exit 1 DOWNGRADE; pending absent. |
| F9-06 / D-080 | Agent, seed, job, result and lock holder agree with the kernel boot identity. |

The extension's quiet local apply and HTTP acceptance are separate from devstack 19's [serving measurement](../../runs/2026-10-31/followup19/README.md). D-127's rolled_back input is an explicit TEST ONLY HTTP contract stimulus; no physical rollback pass is inferred. Scope limits are listed once in [follow-up-10.md](follow-up-10.md#evidence-and-limits).
