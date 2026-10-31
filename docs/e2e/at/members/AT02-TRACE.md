# AT02 finite grant and device-to-usage trace — Follow-up 8

Run `69f92f49-73dc-4779-8101-dfb6a802a96b`, harness `d79accd9c693aed83713072c925b3b5005b5684c`, product baseline `6848add` (#284), assigned `w4-members / 24000`. All deciding bodies, times, samples and audit rows are in [AT02.json](AT02.json).

| Transition | Authorized / pending before → after | Entitlement revision before → after | Deadline | Grant worker operation | Status |
|---|---|---|---|---|---|
| ReplaceFiniteGrant | 3/0 → 2/1 | 2 → 4 | 2026-10-31T09:53:13.073851374Z | 01a19129-b913-71a0-8715-fa1671c5ee51 | pass |
| ReplaceFiniteGrant | 2/1 → 3/0 | 4 → 6 | 2026-10-31T09:53:36.718773267Z | 01a1912a-1568-77cc-9036-80934af2ed38 | pass |
| ReplaceFiniteGrant | 3/0 → 3/0 | 6 → 8 | 2026-10-31T08:55:49Z | 01a1912a-f5cb-7c8c-835d-b620bbefaa80 | pass |

Production revoke is audited once per grant under same-key replay; expired input is rejected before revocation and leaves the revision/live grants/audits unchanged. The immediate lower-limit transition and the real valid_until expiry are distinct. This member has no fallback plan at expiry, so all three authorized devices become pending. Restoration must reach 3/0, a succeeded parent worker operation, and counted forwarding.

Persistent TCP/9446 and authenticated UDP/9445 senders remain active beyond cutoff; reply increments and both device counters are sampled. The bound is the prepared heartbeat interval. Polling bounds observed reply time and is not a packet capture timestamp.

- pass: event 2026-10-31T08:53:13.073851341Z; last TCP/UDP reply observed 0.944/0.944 s after the event; bound 5 s; sender active True; counters stable or removed True.
- pass: event 2026-10-31T08:55:49Z; last TCP/UDP reply observed -0.842/-0.842 s after the event; bound 5 s; sender active True; counters stable or removed True.

The baseline also rechecks final-slot contention, ownership denial, slot reuse, strict revoked-source fvmb0 ingress versus a counted admitted sibling, restore-fence re-authorization and the real meter duplicate replay. [D077.md](D077.md) indexes HTTP outcomes. The accepted usage replay is read from a real agent payload; it never races the live sequence with a fabricated positive event.

Revoke audit operation IDs: `01a19129-b908-7821-b5e4-2df4cbe96ac9`, `01a1912a-155f-77ad-beca-65650cd4f217`, `01a1912a-f5c1-7ed5-9f5e-fc507ee886ad`. Expiry restoration worker operation: `01a1912c-4906-73e9-98d7-504497144d4b`. Full POST/replay and GET bodies remain in the corresponding AT02 steps.
