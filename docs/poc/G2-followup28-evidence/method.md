# TCP observation method

TEST ONLY; enabled only by `FVPN_NETLAB_TCP_OBSERVE=1` in the existing AT10
reload harness. The diagnostic's source is the claim in `window/claim.json`.
The declaration was committed before the harness and before any live window.

The TCP capture is independent of the existing UDP tcpdump processes. Three
additional `any` captures in the origin namespace, node namespace and synthetic
responder namespace retain all TCP/9001 segments in both directions, starting
before the connection opens and stopping after the sidecar's final read/end.
Raw nano-PCAP files exist only in disposable private container scratch. The
strict parser exports header fields and deletes the pcaps; only derived records
are published. It exports no MAC address or payload. Capture completion requires
zero kernel drops and exact parsed-versus-captured counts.

Node ingress is `fv-fh`, node Internet egress is `fv-ih`; return traffic reverses
those interfaces. Origin and responder captures independently establish actual
emission and endpoint arrival. Packet identity uses sequence, ACK, flags,
length, IP ID and TCP timestamp options, which survive this lab's NAT. All
stages remain separate records. Ambiguous duplicate identities or stage copies
fail correlation; a missing endpoint observation remains explicitly unmatched.
Retransmissions mean overlap in consumed TCP sequence space at the emitting
endpoint, including SYN/FIN and sequence-number wrap. Repeated pure ACKs are
reported separately. Copies of one segment at multiple hops are not retries.

Kernel capture timestamps use CLOCK_REALTIME, the same clock as the independent
Apply/guard `wall_ns` samples. Producer verifier brackets use CLOCK_MONOTONIC;
the analyzer maps them with the paired anchor for that Apply and reports the
anchor drift. It retains a 2 microsecond classification margin. All intersecting
install/release/verify brackets are listed; the analyzer never selects a single
convenient generation. A command bracket encloses the atomic transaction,
not an observed kernel commit instant. Capture-time monotonic timestamps are
not fabricated. Start/end paired clock samples are also retained.

The opt-in lab executor reads filtered `conntrack -L` and member `ss -tin` state
before and after each Apply. Read start/end, stdout, and success/error remain
in the Apply record. These reads cannot run between install and release;
`tcp_analysis.py` rejects such a record. Snapshots can show persistence or change
between reads, not exclude every transient transition between them. They add
measurement overhead outside Apply and may change the phase of probe/reload
collisions. They do not add production D-133 cost.

The flow remains the original persistent socket, including its one `recv(128)`
per cycle and reuse after a timeout. Packet TCP sequence numbers distinguish
wire bytes from the sidecar's application cycle numbers. The TCP sidecar does
not validate payload equality or delimit response frames; a nonempty read alone
cannot establish that it belongs to the current token. Those limitations remain
visible and unchanged. UDP behavior, its trace, and the entire D-120 analyzer
and existing tests are byte-identical to task-start main.

To reproduce the offline analysis, pass a retained reload directory and a fresh
output path to `python3 -B tests/netlab/scripts/tcp_analysis.py`. It reads both
plain and gzip artifacts. This analysis is additional evidence, never a D-120
verdict input. Replay the unchanged `reload_analysis.py` separately after
materializing any compressed inputs in unique disk scratch under `.agent/tmp`.
Do not reuse delivered evidence paths for a new live run.
