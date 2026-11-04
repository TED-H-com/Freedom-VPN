# 2026-11-04 — Follow-up 18 disposition

SECOPS-F17-01 is answered; AT29 changes to blocked. All runnable cases passed,
but D-133 overlap and the missing clock guest prevent full acceptance.

### Requests to orchestrator

- **2026-11-04 — handover/acceptance:** carry current blocked status, finding
  resolution and limits; retain every historical failed row as failed.
- **2026-11-04 — scheduling:** any future unconfounded campaign requires
  separate authorization; no further rerun belongs to this session.
- **2026-11-04 — w3-devstack-e2e/operator:** provide the independent clock guest.
- **2026-11-04 — harness/evidence:** retain the stopped pre-subject recursive
  wrapper incident and tested PATH correction, and the historical renewal-ID
  evidence gap. No corrected old-case product failure needs routing.

See [full diagnosis, receipts and hand-off](2026-11-04-follow-up-18.md).

---

The following requests and dispositions are historical:

# 2026-11-04 — Follow-up 17 disposition

**SECOPS-F16-01 answered:** the new case reached its outage using lease
`bef669a3-1141-45af-86c6-8fc823a4b84a`, issued after the public cap change with
60s lifetime. Its unchanged continuity/cutoff evaluator passed. Historical run
`1bfd5cff` remains failed. Overall AT29 remains failed.

### Requests to orchestrator

- **2026-11-04 — SECOPS-F17-01, secops/admission:** retain the old lease-expiry
  failure and authorize a separate setup/timing follow-up. Two 60s admission
  sleeps left its cached lease deadline stale; both quiet samples preceded the
  heartbeat's newer deadline. No product deadline violation is proved, and no
  old predicate was changed. [Receipt](history/follow-up-17/followup17-lease-expiry-finding.json).
- **2026-11-04 — orchestrator scheduling:** retain D-133 confounding. Foreign
  runs overlapped despite two empty preflight inventories, including thirteen
  samples during the new control-loss window. Future unconfounded acceptance
  needs coordinated scheduling; this lane took no reservation and no retry.
- **2026-11-04 — w3-devstack-e2e/operator:** provide the independently clocked
  disposable guest for guest_clock_rollback.
- **2026-11-04 — handover/acceptance:** carry AT29 failed, SECOPS-F16-01 answered,
  SECOPS-F17-01 open and D-133 limitations into out-of-lane records. No other
  area's files were changed. [Full hand-off](2026-11-04-follow-up-17.md).

---

The following requests and dispositions are historical:

# Requests to orchestrator — Follow-up 16 (2026-11-03 EST)

### Requests to orchestrator

**2026-11-03 — SECOPS-F16-01:** AT29 now fails the new accepted-short-lease
prerequisite. The cap PUT/read-back was 60, but the node retained an earlier
86400-second lease; no control-loss or through-D trace was obtained. Route the
[exact failure](followup16-finding.json) to secops fixture and node/lease owners.
A future setup must establish a post-cap accepted short lease through public
configuration/member generation changes, retaining the predicates and this
failed result. A separate authorized run is required for live agent-core 48
verification; these observations do not establish a probe regression.

**2026-11-03 — w3-devstack-e2e / operator:** provide the independently clocked
guest for `guest_clock_rollback`.

**2026-11-03 — handover owners:** refresh out-of-lane status references from
[the dated note](2026-11-03-follow-up-16.md): AT29 changes blocked → failed;
four earlier cases pass; new continuity remains unmeasured; bounded AT31 relay
passes again. This lane did not edit out-of-lane documents.

---

The following requests and dispositions are historical:

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
