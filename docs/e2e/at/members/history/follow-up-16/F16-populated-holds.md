# Follow-up 16 populated stale-lease holds

**D-133 contention:** the orchestrator reported CPU-heavy handover gates at **2026-11-01 17:32:43–17:37:12 UTC**. Every timing, failure and anomaly overlapping that window is **contention-confounded**, with `cause_status=unproven`; measured values and verdicts stand. [Complete time and step index](F16-contention.json) includes 46 AT02 steps completed in the window and cumulative references in later records. Finite-expiry withdrawal/restoration, the second re-grant, SIGTERM/usage stop/resume, node2 preparation, alarm recovery and sibling removal overlap. The 4,724-send stream, initial AT01 selection, sole-member/retained-safety holds and fresh AT09 measurements are outside this reported window.

Run `0ffb355a-945b-4198-a551-3794120a686c`, measured harness `3a8b61594b447d164bb7c9481545e207f8688ec2`. Four fresh TEST ONLY arms use devstack 27 unchanged. All four incidents have a validated reported alarm and exactly one escalation before the 30-second central fallback. The 13 accepted hold samples are joined by node, incarnation, report and incident ID; each escalation uses its exact `after_diff.incident_id`.

All four rows use node `01a1982c-6a22-7bc6-bed1-039a0980255b`, incarnation `01a1982c-6c15-706d-95ad-1b97f1fab978`. The original lease population and live bindings were checked before arming. The original AT02 mutations and full re-grant captures precede these scenarios.

[Complete joins](F16-populated-hold-joins.json); [supplemental original member-patch audits](F16-AT02-member-patch-audits.json). The live query omitted the `member.patch` action; the supplement reads the original audit rows through the AT02 completion timestamp from the same owned stack. It exports only identifiers and suspension booleans. No report or audit was synthesized.

## alarm-recovery

Exercised: alarm, single escalation, generation-matched closure, exact forwarded/local lease digest and a separate accepted serving report. Closure and the serving report are separate observations; no full-h or partial-service classification is claimed.

- Arm: `c47acb0c-bb22-43fb-af5b-30f34ad34cfd`; target generation `6`.
- Incident: `01a1982d-ce09-7996-b9da-f8e61397d591`.
- First seen: `2026-11-01T17:35:01.128498+00:00`; last seen: `2026-11-01T17:35:11.004116+00:00`.
- End: `2026-11-01T17:35:16.115458+00:00`; reason `generation_matched`.
- Escalation audits: `01a1982d-f49d-751e-876d-bd3dc3d3a8f3`.
- Reports: `01a1982d-ce01-71f6-bdcb-2627335c634e`, `01a1982d-f496-7063-95e1-01227ad8d946`.
- Original leases: `be17e437-90e7-471c-8e23-79e1775b2de4` (2 members).

[Original row](F16-hold-alarm-recovery.json).

## sibling-removal

Survivor continuation is **not exercised**. A real member suspension committed, but the final accepted hold report precedes it. The proxy released on generation_superseded before any accepted post-command survivor hold report. Do not infer survivor coverage from the earlier two-member samples or the later generation-matched ending.

- Arm: `f5321384-0e5f-4260-aab8-3afab12a6734`; target generation `10`.
- Incident: `01a1982f-415a-7505-8b09-830051bab7d1`.
- First seen: `2026-11-01T17:36:36.185473+00:00`; last seen: `2026-11-01T17:36:46.06157+00:00`.
- End: `2026-11-01T17:36:51.186077+00:00`; reason `generation_matched`.
- Escalation audits: `01a1982f-67ef-7056-832e-0bb6fc770e0f`.
- Reports: `01a1982f-4154-79d7-8888-71948aa21fb8`, `01a1982f-67e6-7c3c-832c-8a7147febc4d`.
- Original leases: `7e1ca8b4-fbe9-45f4-b936-30828dc87ee9` (2 members).
- Member revocation: `01a19824-21a0-758e-b551-5b99a7c32483`; audit `01a1982f-70fb-7b16-8945-404aee049082`; operation `01a1982f-70fb-761e-99b7-a4d89d0a74b2`; committed `2026-11-01T17:36:48.376109+00:00`; suspension `False → True`.

[Original row](F16-hold-sibling-removal.json).

## sole-member

Exercised: the sole member in the original one-member populated lease was really suspended, and this incident closed members_revoked. The member suspension audit is false → true. No service-recovery claim and no exact forwarded/local matching-lease claim are made for this closure.

- Arm: `bbe760b0-8eb6-485a-a3e6-11fac8798402`; target generation `16`.
- Incident: `01a19830-c813-758e-99d5-2be0b969aae6`.
- First seen: `2026-11-01T17:38:16.210767+00:00`; last seen: `2026-11-01T17:38:26.118915+00:00`.
- End: `2026-11-01T17:38:28.164465+00:00`; reason `members_revoked`.
- Escalation audits: `01a19830-eec8-71b0-b1ea-4d77ea209e15`.
- Reports: `01a19830-c80d-76f4-bf10-f4c7d9c8a884`, `01a19830-eec1-7184-8e8b-b63337c2350c`.
- Original leases: `289766e9-2870-4e28-965a-1b725da97421` (1 members).
- Member revocation: `01a19824-21a0-758e-b551-5b99a7c32483`; audit `01a19830-f680-7844-81d9-ff303d3e72da`; operation `01a19830-f680-704a-bb69-94d73feed983`; committed `2026-11-01T17:38:28.092799+00:00`; suspension `False → True`.

[Original row](F16-hold-sole-member.json).

## retained-safety

Exercised: two failed Applies during the same hold, followed by five accepted reports retaining TC_FOREIGN_FILTER in lease, qos and source_binding checks while the lease-stage STALE_GENERATION remains recorded. The publication classification replay retains safety priority. This is a TC fault affecting both safety checks, not a separately injected source-address fault. No node nft observation rule was installed.

- Arm: `a946b9ce-c7aa-46d6-916d-a77bd4f0c1bc`; target generation `22`.
- Incident: `01a19832-3b79-7079-91f7-7c9e0d96ca6e`.
- First seen: `2026-11-01T17:39:51.287566+00:00`; last seen: `2026-11-01T17:40:16.284276+00:00`.
- End: `2026-11-01T17:40:21.30219+00:00`; reason `generation_matched`.
- Escalation audits: `01a19832-6213-7ade-a6ac-9aea333004e5`.
- Reports: `01a19832-3b70-7f24-9454-3bda77f6d2ab`, `01a19832-620b-72d3-93f8-e16817ecd1c9`, `01a19832-75ca-7565-979c-f11bf7f753df`, `01a19832-8957-7d1a-930f-1e3165573aef`, `01a19832-89c4-7895-bfb2-d9eb6d504de6`, `01a19832-9ce3-7ec8-a73a-5d5a9a4afc48`, `01a19832-9d16-7795-9b85-4b96ac4dcd4e`.
- Original leases: `9da17b2d-43c6-476e-ab0a-f5c41f27fd37` (2 members).
- Failed Apply producer identities: `a294e6ac-33a0-454e-b400-bf100e461080/614` at `2026-11-01T17:40:11.179506985Z`, `a294e6ac-33a0-454e-b400-bf100e461080/622` at `2026-11-01T17:40:16.180506799Z`.

[Original row](F16-hold-retained-safety.json).

No incident ending alone proves service recovery. The observation shape has no per-member hold field. Historical hold gaps and all historical failed rows stay unchanged.
