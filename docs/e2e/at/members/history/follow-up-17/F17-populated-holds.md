# Follow-up 17 populated holds

Run `81b638e8-be59-4996-9594-14dd402ebb86`, measured harness `3ba0856487288b9be3cf81f3e3afa37b0fa43a47`. Four fresh TEST ONLY arms, **14 accepted hold samples**, four reported alarms and exactly one escalation per incident, each before the 30-second central fallback. [Full joins](F17-populated-hold-joins.json), [qualification](F17-qualification.json). These are this run’s observations, not devstack’s proof.

## alarm-recovery

- Node/incarnation: `01a1a07c-e057-7586-9060-b6eb57c33446` / `01a1a07c-e0a6-7422-8725-1d608ef6c68e`.
- Arm: `0f9f7111-881a-4263-ac96-1037369609c3`; base generation `5`; initial target `6`; hold 90 seconds.
- Incident: `01a1a07e-41fd-7a7f-92c3-e10158fefd8c`; first seen `2026-11-03T08:19:51.420813+00:00`; last seen `2026-11-03T08:20:01.312999+00:00`; ended `2026-11-03T08:20:06.411569+00:00` as `generation_matched`.
- Entry report: `01a1a07e-41f5-7664-babe-80dabc66a38d`; alarm report: `01a1a07e-689b-763d-becf-c440f53d90c8`; escalation audit: `01a1a07e-68a2-7413-b6c8-24ab629ce0b4`.
- Accepted report IDs: `01a1a07e-41f5-7664-babe-80dabc66a38d`, `01a1a07e-689b-763d-becf-c440f53d90c8`.

Exact forwarded/local lease digest agreement and a separate accepted serving report were retained. Incident closure and service observation remain distinct.

[Original row](F17-hold-alarm-recovery.json).

## sibling-removal

- Node/incarnation: `01a1a07c-e057-7586-9060-b6eb57c33446` / `01a1a07c-e0a6-7422-8725-1d608ef6c68e`.
- Arm: `427f7620-a7a0-42df-844e-3198981f4a66`; base generation `8`; initial target `9`; hold 90 seconds.
- Incident: `01a1a07f-a1c7-76b6-8630-828c3ae49e96`; first seen `2026-11-03T08:21:21.478533+00:00`; last seen `2026-11-03T08:21:36.363628+00:00`; ended `2026-11-03T08:21:41.466961+00:00` as `generation_matched`.
- Entry report: `01a1a07f-a1c0-7d5c-bcf9-aec8d0755dff`; alarm report: `01a1a07f-c85b-750c-9288-02f1ae3530f6`; escalation audit: `01a1a07f-c862-75a3-9385-4ba9f7acddd2`.
- Accepted report IDs: `01a1a07f-a1c0-7d5c-bcf9-aec8d0755dff`, `01a1a07f-c85b-750c-9288-02f1ae3530f6`, `01a1a07f-dbe5-7dea-a894-f38323a91f94`.
- Real sibling/member suspension: `01a1a073-cb57-7b8b-b48e-708bf74404a5`, HTTP 202, response complete `2026-11-03T08:21:33.848203874Z`; audit IDs `01a1a07f-d216-7bbf-bab2-d171e2ea2246`.

**Survivor continuation exercised.** Survivor `01a1a073-c87d-7195-bf9f-49a396f4955e` remains eligible. The original two-member lease `0efb3544-2d77-4c71-ad61-88f069e8e851` (SHA-256 `3e719b6cf3022c740290fb372b1fb0fb962c2d74f8a7f3901c033376de1c498d`) still contains the survivor and suspended sibling. The accepted post-command report `01a1a07f-dbe5-7dea-a894-f38323a91f94` arrived `2026-11-03T08:21:36.363628+00:00` while the same incident was open, at generation `11`. The proxy records `continuation_started` and `withheld` at that same generation, then `explicit_release` and `matching_lease_forwarded`. Generation 9 → 11 is one observed continuation; generation numbers need not be consecutive.

Recovery lease `a4d6649f-b721-4b1d-82de-c1bd66ab3e05` matches forwarded SHA-256 `28f618bf69e3222f12b06af5b6edc2863d790dfeb02dce29b9d312bd41081137` byte-for-byte. Independent accepted serving report `01a1a07f-efd5-707b-b212-366410520e5d` arrived `2026-11-03T08:21:41.466961+00:00`, generation `11`. This is accepted serving evidence, not an end-to-end traffic or full-h claim.

[Original row](F17-hold-sibling-removal.json).

## sole-member

- Node/incarnation: `01a1a07c-e057-7586-9060-b6eb57c33446` / `01a1a07c-e0a6-7422-8725-1d608ef6c68e`.
- Arm: `b41cfba0-b671-47e0-9872-7cf47e8e4551`; base generation `14`; initial target `15`; hold 90 seconds.
- Incident: `01a1a081-2885-7d88-aa6d-f741c3dde6cf`; first seen `2026-11-03T08:23:01.509212+00:00`; last seen `2026-11-03T08:23:11.419458+00:00`; ended `2026-11-03T08:23:14.485269+00:00` as `members_revoked`.
- Entry report: `01a1a081-2881-73fa-b508-0181d262647e`; alarm report: `01a1a081-4f35-7d00-b6f8-03c45d788394`; escalation audit: `01a1a081-4f3d-7f34-ad4c-240b69a82aca`.
- Accepted report IDs: `01a1a081-2881-73fa-b508-0181d262647e`, `01a1a081-4f35-7d00-b6f8-03c45d788394`.
- Real sibling/member suspension: `01a1a073-cb57-7b8b-b48e-708bf74404a5`, HTTP 202, response complete `2026-11-03T08:23:13.758267295Z`; audit IDs `01a1a081-585c-7bc7-b94e-b35161dac867`.

The original one-member lease closes `members_revoked`; no service-recovery claim.

[Original row](F17-hold-sole-member.json).

## retained-safety

- Node/incarnation: `01a1a07c-e057-7586-9060-b6eb57c33446` / `01a1a07c-e0a6-7422-8725-1d608ef6c68e`.
- Arm: `01f1b5e7-e4c0-4379-b8aa-adef9b9846bf`; base generation `20`; initial target `21`; hold 90 seconds.
- Incident: `01a1a082-9bd0-7515-9aee-ac791cafdd20`; first seen `2026-11-03T08:24:36.559665+00:00`; last seen `2026-11-03T08:25:01.56848+00:00`; ended `2026-11-03T08:25:06.595458+00:00` as `generation_matched`.
- Entry report: `01a1a082-9bcb-7285-8eb7-eb6834cf6a31`; alarm report: `01a1a082-c28d-7b2a-b840-c841b45151cc`; escalation audit: `01a1a082-c295-70aa-a328-5aa5ffef7b28`.
- Accepted report IDs: `01a1a082-9bcb-7285-8eb7-eb6834cf6a31`, `01a1a082-c28d-7b2a-b840-c841b45151cc`, `01a1a082-d641-7974-a1d1-d2278d237c99`, `01a1a082-e9bf-7c53-854c-625721f950b5`, `01a1a082-e9ec-71a1-8a7e-8bfe9ba63304`, `01a1a082-fd45-7c58-82c5-b1c75e0bed1d`, `01a1a082-fd7a-77a7-90e7-30080daa2c75`.

2 failed Applies and 5 accepted safety-priority reports were observed during the bounded TC fault. The original fault was removed. TC errors in source_binding do not constitute a separate source-address fault.

[Original row](F17-hold-retained-safety.json).

The **primary continuous sibling failed-Apply row was exercised and failed**: three failed Applies occur inside its original capture; the existing provisional measurement remains invalid, with zero failed Applies bracketed by that evaluator. Every loss counts. A separate source-address-fault hold is **not exercised**; [owner request](requests.md). No node nft observer or signed-document edit was added.
