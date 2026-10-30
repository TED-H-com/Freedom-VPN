# Pre-final diagnostic

This run used source `e0a84fb288db42a7ea0c45ff9a8de097f94734f7` with uncommitted
harness changes. Its JSON records retain `source_dirty: true`. It established
counted origin controls and exercised isolation, publication and revocation.
It is superseded by the root records.

Two instrumentation limits were corrected before the final run:

- Restart readiness initially accepted the old forward chain, so its next
  capture overlapped interface replacement. The final test waits for a new
  serving heartbeat and records bounded recovery attempts.
- The expiry assertion required the native route to remain present, but the
  still-running worker withdrew it. All five connections timed out and bytes
  stayed frozen; its recorded `fail` was not evidence of a forwarding leak.
  The final test stops control and worker together and measures node ingress.

Final protected-target probes add directional captures, and revocation adds
conntrack before/after evidence. This diagnostic must not be used to claim
those later checks ran here.
