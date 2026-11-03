# Follow-up 44: pre-measurement clock-check amendment — 2026-11-03

Window 1's only entry was refused at preflight: another lane's container was
running. No measured leg or scenario started. Its committed receipts remain
unchanged, and it consumes the first reservation.

A subsequent read-only probe found `/proc/1/ns/time` unreadable to the host
user (`PermissionError`). The original clock check would therefore fail before
executing its container probe. This amendment removes that unnecessary init
lookup. It compares the runner's `/proc/self/ns/time` inode with the probe's
container namespace inode, requires zero monotonic and boottime offsets in both
`/proc/self/timens_offsets` files, and retains all three host/kernel/timeanchor
CLOCK_MONOTONIC brackets. Equal namespace identities prove the container did
not introduce a separate time namespace relative to the runner. Any mismatch
or unreadable required state still stops the timing item.

The local read measured runner namespace `time:[4026531834]` and zero offsets.
Actual container agreement remains to be measured inside each admitted leg.
There is no change to CPU classes, qualification, pooling, span boundaries,
failure rules, arms, overlays, scenarios, limits or the remaining A3,B3,A4,B4
order. The original declaration and pins are preserved. The runner checks
these [amended pins](followup44-timing-amendment-pins-2026-11-03.json) before the
remaining reservation. All 23 synthetic tests pass again. This amendment is
committed before the first measured leg; there is still no third reservation.
