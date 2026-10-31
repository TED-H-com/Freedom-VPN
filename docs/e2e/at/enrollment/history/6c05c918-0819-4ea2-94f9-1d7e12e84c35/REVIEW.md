# Follow-up 9 first attempt — early current gap ACK

2026-10-31. Owner: enrollment harness. The raw attempt remains failed.

First failing observation: `installer_prerequisite` at `autonomous origin upload
with new presenter`, inherited `independent` assertion. Retired local ACK and
last sequence are 3 with no final rejection. The fresh CLI already reports
sequence 1, ACK 1, zero unACKed events, and first sequence 1. The old assertion
required current ACK 0; D-130 lets the fresh empty gap succeed before quiescence.
This does not establish the later counted event or close controlled_reinstall.
The full probe exits 1, join exits 0, and all teardown exits are 0 with zero
owned containers, networks, volumes and images.

Correction: allow current ACK 1 only when its own central canonical empty unknown
gap, whole-microsecond interval and CLI ACK match. Retired ACK remains separately
selected. Replaying the original batch must leave that current ACK unchanged.
The later composed proof still requires separate gap ACK 1 and counted event
ACK 2, 550/358 bytes, preserved historical attribution and all four AT20 assertions.
Only this exact prerequisite and its propagated process failures can be reviewed
as resolved after the complete new live proof passes. No product path changed.
