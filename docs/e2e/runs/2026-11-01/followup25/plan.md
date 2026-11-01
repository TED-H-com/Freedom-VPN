# Follow-up 25 — predeclared bounded campaign

2026-11-01, TEST ONLY. Declared before the first stack starts: **eight fresh
attempts per family per arm**, 32 attempts total. Order: kernel IPv4 ×8, kernel
IPv6 ×8, node userspace IPv4 ×8, node userspace IPv6 ×8. Each attempt has a reset
stack, one initial route selection and one bounded first HTTP request through
TestE2E_Followup25Native (the follow-up 24 measurement). Keep every attempt,
including startup/instrumentation failures, without replacement. A passive
two-second tail is unchanged. Counts are observations, not reproduction rates.

The TEST ONLY userspace marker is armed before node enrollment. Only the node's
lab daemon receives NB_WG_KERNEL_DISABLED=true. Member and ordinary fixtures
remain kernel WireGuard; NetBird 0.82.0 and its digest are unchanged. This arm
conflicts with the agreed node kernel-WireGuard baseline and is evidence for
an owner decision, not a recommendation or product repair.

The 122 follow-up 24 pins verified before changes. Its entire directory is
snapshotted privately for a final byte-for-byte comparison. No historical
artifact will be regenerated. New pins refer only to this immutable directory.
