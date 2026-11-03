# Isolation follow-up 16 requests — 2026-11-01

Four rows: **two carried operator assignments**, one continuing D-140 sampling
item, and one new shared-netlab item. Run`83b457ef-ea16-490b-9923-1fea0df86b36`;
[full evidence](summary.md). No retry, waiver or historical reclassification.

1. **Carried — AT11, existing platform/operator assignment.** Native dual-stack
   and member-platform route/capture matrix remains blocked. No renewed ask.
2. **Carried — AT13, existing acceptance/operator assignment.** Full continuous
   crash, NetBird restart, kernel reboot, updater and physical-arm 64 matrix
   remains blocked. Five agent stop receipts do not close it. No renewed ask.
3. **Continuing — orchestrator / next isolation sampling decision, D-140.**
   Correct same-run calibration chose2.2 ms. Packet162 latency3.150529 ms exceeded
   calibration maximum 1.840658 ms; its egress overlapped packet 163 ingress by
   **3.609 µs**. The trace is unqualified even though the unchanged evaluator
   passes and a deciding positive exists. [All times](boundary-review.json) and
   [F16-02](follow-up-16-findings.json) stay failed. The pre-run resolution versus
   noninterleaving trade-off is in [RUN.md](RUN.md). Choose the next independent
   sampling approach; a finite calibration is not an arrival-latency guarantee.
   This task's one-run authorization provides no retry. No evaluator or product
   change is requested. Follow-up15 and isolation F14-01 remain failed.
4. **New — orchestrator / w1-netlab, F16-01.** Outside-owned
   `tests/netlab/scripts/deadline.py:35` requires all adjacent precision-fixture
   expiry differences to equal1 ms. This attempt had a2 ms sixth difference:
   remaining 699,700,701,702,703,704,706,707 ms for requested 1001…1008 ms.
   [Original assertion](mechanism-d104-failure.txt) precedes sockets and deadline
   Apply; no fresh trace was emitted. Investigate the fixture and record a
   justified disposition without weakening D-140 or claiming this packet row
   ran. No source change or retry was made here; cause remains unproven.

The populated-hold dependency is consumed: seven requested categories pass
across two devstack 27 seam incidents. Pure stale_generation classification,
central 30-second fallback and the original opportunistic-cut hold remain
**not exercised**, not newly requested. All per-row joins are in
[populated-holds.md](populated-holds.md). No seam change is requested.

The generation 15 install timeout remains a diagnostic failure: the guard did
not release and rollback/safety baselines and later recovery are retained.
Its delay's cause is unproven. Follow-up15's terminal-source signal 15 remains
observed, with its existing agent-core 36 investigation; no renewed assignment.
Current five parent receipts all record exit 0 and are not used to reinterpret it.

No outside-owned path was edited. Handover claims already use immutable pins;
the D-132 one-file exception was unnecessary. Follow-up15 is byte-identically
archived; every earlier archive is unchanged. Historical requests:
[follow-up 15](history/follow-up-15/requests.md),
[follow-up 14](history/follow-up-14/requests.md).
