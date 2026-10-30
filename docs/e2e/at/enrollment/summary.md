# Enrollment acceptance — follow-up 1

Run `75dbb974-04d2-4e45-a6bd-cfe0d923bd2c`, 2026-10-30 18:07:10Z–18:09:56Z,
using the corrected `w4-enroll / 12000` binding. Fresh startup and live probes
exit 0; the acceptance target exits nonzero because both complete ATs are blocked.

| AT | Complete status | Decisive observation | Passing live evidence | Record |
|---|---|---|---|---|
| AT06 | blocked | Enrollment audit changes expose only `reason`, no stored token hash (ENR-02) | Dropped-response retry returns the same certificate/incarnation; two-key race gives 201 + 409 ENROLLMENT_KEY_MISMATCH; expiry/revocation/proof negatives and secret scans pass | [AT06.json](AT06.json), [AT06.md](AT06.md) |
| AT07 | blocked | `MachineIdentityFingerprints` is not exported (ENR-07) | Both documented node slots join from the same immutable image with distinct API Node/incarnation IDs and bound peers | [AT07.json](AT07.json), [AT07.md](AT07.md) |

AT06 has 85 passing and 4 blocked recorded subchecks; AT07 has 10 passing and
9 blocked. No subcheck failed. The 54/26 prior checks from both initial attempts
remain retained, with their original run IDs and blocked results.

Complete AT06 also awaits complete incarnation evidence (ENR-03) and the
own-key join/peer inventory exports (ENR-04). AT07 awaits the fingerprint/clone,
held-ACK usage and real installer/reinstall exports (ENR-05/06/07). The real
first-boot peers do not stand in for AT06's own-key race-winner peer.

[API review](api-surface-review.md) distinguishes current identity, audit
history, member aggregate usage and the agent checkpoint. [Requests](requests.md)
retain the exact product questions and devstack follow-up 12/13 owners.
[RUN.md](RUN.md) has provenance, image IDs, real command output and cleanup.
Both earlier attempts and this run remain under `history/`.

Control, worker, both nodes' streams, current probe evidence and 30 existing
public evidence files were scanned against 64 in-memory token/key patterns:
zero matches. No secret values or raw logs are published. Teardown exits 0;
owned containers, networks, volumes and images are all zero.
