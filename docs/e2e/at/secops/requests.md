# Requests to orchestrator — Follow-up 15 (2026-11-03 EST)

AT30 **passed**; SECOPS-21 **closed**. Restore run `5218b7d5-1415-4e1b-8618-48ea2d3cf03e` on application base `7e044c71ecf1d07f63e89721bd213db6611a6a47`. Shared development host measurements; no production SLA.

First delivered offer: HTTP `200`, generation `19` versus retained `15`, delivered `2026-10-31T11:56:09.869156137Z`; complete write `True`. Body SHA-256 `9db8da4cb30ef897e1eaf7f3494eaf8063c7823fcf7d3473d5250052ffc8b0f6`; local accepted/applied `19` / `19`, digest `9db8da4cb30ef897e1eaf7f3494eaf8063c7823fcf7d3473d5250052ffc8b0f6`. Applied ACK HTTP `202`, generation `19`, report `01a191d1-3799-7cee-ba57-53efe5f3dc18`, joined to the [central receipt and ordering evidence](SECOPS-21.json). Capture armed once; no retry or generation repair.

The first separate HTTP **probe** returned `19` at `2026-10-31T11:56:10.163037901Z`. The first new stored **compiler row** was `16`. Neither substitutes for the captured delivery. D-095 exact HTTP 503 / `pending fresh observation after restore` drift and unchanged generation, row count and envelope fingerprint: `True`.

Full AT29 remains **blocked** pending an independently clocked disposable guest. The manual grant-withdrawal export was answered by #297; Follow-up 15 measures its transport behavior. Full AT31 remains **blocked**: candidate hardware and 20-member/60-device mixed relay campaign. Follow-up 12 remains in its archive. Follow-up 15 re-measured all runnable AT29 sub-cases and the bounded AT31 relay row; each passed. [AT29](AT29.json), [AT31](AT31.json), [unchanged resource references](history/follow-up-12/requests.md).

Devstack 18 [proof run](../../runs/2026-10-31/followup18/5f74d7c3-6404-461d-9f47-296bd69958c2/first-desired-offer.md) (2/0/0; first 4 > retained 3; HTTP 202 central receipt) validates the export only. This lane uses the separate restore run above for closure. [Export contract](../../RESPONSES.md).

[Follow-up 12 archive](history/follow-up-12/archive-index.json): transport `cf81f6f8-4ccf-43d2-aae2-10121f6b169d`, restore `dd845a7f-2660-431c-a8e6-c513108cc07d`, byte-identical to PR #301 (`8b5e809b`) with SHA-256 pins. The follow-up 9 and 10 handover archives are unchanged.

| Remaining item | Owner / resource | Required action |
|---|---|---|
| AT29 guest clock rollback | operator + w3-devstack-e2e | Supply an independently clocked disposable guest; never change the shared host clock. |
| AT29 manual grant withdrawal | w3-wire-members / contracts | **ANSWERED by #297**: contract revision 14 (1.10.0) exports revokeMemberGrant. Follow-up 15 uses its public 202 operation and completion read-back; finite expiry stays a separate case. |
| AT31 full mixed relay capacity | operator + acceptance lane | Supply candidate hardware and a 20-member/60-device mixed-load window. |

D-102: only this lane writes this request register. Devstack answers in [RESPONSES.md](../../RESPONSES.md). No export-contract discrepancy was observed in the Follow-up 13 restore or Follow-up 15 transport run.

## 2026-11-03 — Follow-up 15 export disposition

The AT29 grant-withdrawal export request is **answered by #297**. Its absence in
run `cf81f6f8-4ccf-43d2-aae2-10121f6b169d` was historical: that run's application
base predates the operation. The contract change alone does not promote the
transport case. Follow-up 15's measured result is recorded in [AT29](AT29.json).

### Requests to orchestrator

2026-11-03: **w3-devstack-e2e / operator** still owns the disposable guest with
an independently adjustable wall clock for `guest_clock_rollback`. Containers
share the host's clock. No host clock or host network change was made.

2026-11-03: **devstack / handover owners** may refresh their out-of-lane adoption
notes and acceptance references from [the Follow-up 15 hand-off](2026-11-03-follow-up-15.md).
The six node writes are adopted; capacity/publication have live 202 receipts.
Manual withdrawal passed; full AT29 remains blocked. No product change is requested.
