# Live collector validation: PASS

The second reserved attempt, on declaration `b82ffd09`, validates the corrected
collector without changing it. [Attempt 1](collector1-outcome.md) stopped at host
preflight and remains retained; no third attempt is needed.

[The result](collector2/collector-result.json) matches all eight injected packets
exactly once: five upload and three reverse. The extra unlogged negative control
has no record. Rule counters independently show 5 / 3 / 1 packets.
[The receiver](collector2/collector.meta.json) reports eight captured, eight
received, zero kernel drops. All eight records decode, each with
`pcap_precision_ns=1000` and its own kernel timestamp. The collector is unchanged
from the corrected source on main; [source pins](collector-source-pins.json)
bind the collector and guard executor used for the forthcoming window.

The quiet check passed throughout: five checks, maximum interval 4.002595 seconds,
maximum foreign CPU 0.144907 cores, no other lane suite or container. The shared
PostgreSQL service remained visible and its CPU counted as foreign. The initial
60-second settle and subsequent waits are retained in the reservation/preflight
receipts. The reservation was released at 11:24:58 UTC.

The foreground command's exit is **1 from cleanup**, not a collector failure:
after removing all lab objects, cleanup still rejected the deliberately retained
admission-wrapper directory. [The complete output](collector2.txt) preserves the
error. Cleanup now also removes those wrappers; the separate
[completion inventory](collector2-cleanup-inventory.json) proves zero owned
containers, images, networks, volumes or task scratch. Cleanup completion is not
a new collector attempt. The executed cleanup and runner sources are preserved
under `collector2-source/`.

This check proves the finite injected set, parser, timestamp fields and receiver
accounting. It is not an AT10 measurement or proof of lossless NFLOG under every
load. The one guard window remains separately declared. No capture options,
backend, group, production rules, D-120 inputs or historical statuses changed.
