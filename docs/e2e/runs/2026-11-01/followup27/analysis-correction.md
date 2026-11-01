# Initiation-origin correction

2026-11-01. Reviewing IPv6 attempt 03 exposed a derived-label issue in the
original `measurement.json` convenience list: `member_initiations` selected
every member loopback type-1 copy, including the **received** copy of a
node-originated initiation. Packet bytes, hashes, timestamps, full handshake
timelines and send/ingress intervals were correct and remain unchanged.

`initiation-origins.json` is the authoritative supplemental origin join for
every run. It identifies SLL2 outgoing underlay packets or the pinned harness's
local WireGuard loopback source on port 51820, then computes intervals only
between initiations from the same endpoint. The regression test rejects an
incoming relay copy as a local initiation. Older convenience lists are retained
byte-identically and explicitly superseded; no original outcome is reclassified.

For `ae1a2829`, member initiation `e85074a2` is followed by node initiation
`25905953`; the received member copy of the latter is not another member
initiation. The next real member initiation, `3597bf0a`, is **15.600070 seconds**
after `e85074a2`. The paired node-originated handshake had already carried data
before the `::1`-source response to `e85074a2` arrived.

After the ordinary timeline is available, create this supplement once with
`python3 tests/e2e/summarize_initiations27.py RUN_DIRECTORY`. Create the separate
source-anomaly view with
`python3 tests/e2e/summarize_loopback27.py RUN_DIRECTORY`. Both refuse an existing
output. The final campaign joins use the corrected origins.
