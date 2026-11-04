# Follow-up 16 — AT29 live-agent continuity prerequisite failed

AT29 is **failed**, previously blocked. Run `1bfd5cff-e593-481b-a4d2-94ca08cf7c3d`
passed the four existing runnable cases. The new `control_lost_agent_alive`
case failed to obtain a locally accepted 60-second lease before its outage.
Control loss was not injected, so live evidence for agent-core 48 remains missing.
Guest clock rollback remains blocked. One attempt, no rerun; zero owned Docker
inventory after cleanup. The bounded AT31 relay row passed again.

See [the dated hand-off](2026-11-03-follow-up-16.md), [AT29](AT29.json),
[SECOPS-F16-01](followup16-finding.json), and [the pre-run archive](history/before-follow-up-16/archive-index.json).

---

The following text is historical; its results belong to the archived earlier runs:

# Follow-up 15 — AT29 grant withdrawal

Run `35116d25-122c-4724-adfe-5a06d623c04f` passed all four runnable AT29
sub-cases, including public-API manual grant withdrawal. AT29 overall remains
**blocked** on the independently clocked guest. The bounded AT31 relay row also
passed again; full AT31 is still blocked on candidate hardware and mixed load.
One live run, no rerun; zero owned Docker inventory after cleanup.

See [the dated hand-off](2026-11-03-follow-up-15.md), [AT29](AT29.json),
[AT31](AT31.json), and [the byte-identical prior records](history/before-follow-up-15/archive-index.json).
AT30 was not rerun; its latest result remains the Follow-up 13 pass below.

---

The following is the retained Follow-up 13 report:

# Follow-up 13 — first post-restore delivery

AT30 **passed**; SECOPS-21 **closed**. Restore run `5218b7d5-1415-4e1b-8618-48ea2d3cf03e` on application base `7e044c71ecf1d07f63e89721bd213db6611a6a47`. Shared development host measurements; no production SLA.

First delivered offer: HTTP `200`, generation `19` versus retained `15`, delivered `2026-10-31T11:56:09.869156137Z`; complete write `True`. Body SHA-256 `9db8da4cb30ef897e1eaf7f3494eaf8063c7823fcf7d3473d5250052ffc8b0f6`; local accepted/applied `19` / `19`, digest `9db8da4cb30ef897e1eaf7f3494eaf8063c7823fcf7d3473d5250052ffc8b0f6`. Applied ACK HTTP `202`, generation `19`, report `01a191d1-3799-7cee-ba57-53efe5f3dc18`, joined to the [central receipt and ordering evidence](SECOPS-21.json). Capture armed once; no retry or generation repair.

The first separate HTTP **probe** returned `19` at `2026-10-31T11:56:10.163037901Z`. The first new stored **compiler row** was `16`. Neither substitutes for the captured delivery. D-095 exact HTTP 503 / `pending fresh observation after restore` drift and unchanged generation, row count and envelope fingerprint: `True`.

Pre-replay database RPO (s): `92.163101` s versus `106.45` s; delta `-14.286899` s.
Total outage RTO (s): `81.016113709` s versus `74.53` s; delta `6.486113709` s.

Full AT29 remains **blocked**: independently clocked disposable guest and manual grant-withdrawal export. Full AT31 remains **blocked**: candidate hardware and 20-member/60-device mixed relay campaign. Follow-up 12 transport sub-cases and bounded relay passes are retained without promotion or rerun. [AT29](AT29.json), [AT31](AT31.json), [unchanged resource references](history/follow-up-12/requests.md).

[Follow-up 12 archive](history/follow-up-12/archive-index.json): transport `cf81f6f8-4ccf-43d2-aae2-10121f6b169d`, restore `dd845a7f-2660-431c-a8e6-c513108cc07d`, byte-identical to PR #301 (`8b5e809b`) with SHA-256 pins. The follow-up 9 and 10 handover archives are unchanged.

[Preparation failure](followup13-restore-attempt1-failure.json): the first attempt stopped before ArmFirstDesiredOffer because its negative-drill central connection had been closed. No delivery was captured or discarded. The local-only retained read fixes that harness error; [failed driver and zero cleanup](followup13-restore-attempt1-run.json) remain preserved.

Cleanup **passed**: [container/network/volume/image inventory and disk state removal](cleanup.json). [Run commands](RUN.md).
