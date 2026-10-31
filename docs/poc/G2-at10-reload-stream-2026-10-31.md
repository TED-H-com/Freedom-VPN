# AT10 reload-stream timing — 2026-10-31

TEST ONLY measurements for ISO01 / ISO02 / §8.2 and AT10 (§19).
This campaign does not change the UDP or TCP **1.5 s timeout**, the one-send/
one-read echo pattern, the 200 ms inter-iteration sleep, the 100 reloads, or
AT10's rejection of any failed echo. It makes no acceptance decision for
isolation follow-up 8.

## Method

`stream.py` retains its original text verdict log and adds a JSONL sidecar:
sequence, TCP send/read, UDP send, source port, send wall/monotonic nanoseconds,
read completion, timeout stage, and whether the returned echo matches. No
payload is logged. A TCP timeout is kept separate from a UDP datagram that
was actually sent. An unmatched response is reported, without silently
changing the pre-existing acceptance rule.

The independent **node `any` trace** records request ingress on `fv-fh`,
request egress on `fv-ih`, and the reverse response path. An `any` capture in
a network namespace cannot see interfaces in a different namespace. Therefore
additional `any` traces in `fv-a1` and `fv-inet` record origin egress/return
and responder ingress/egress. Those are packet observations, not claimed
application-receive timestamps. All three use the same host realtime clock,
header summaries only, immediate delivery, and a checked zero kernel-drop
count. The parser requires the number of parsed packets to equal tcpdump's
final capture count.

The original enforcement CLI and the real agent run through the same
**lab-only timing decorator**. It invokes the unmodified `enforce.Apply` with
a System executor, sampling Apply begin/end and nft guard-install/release
command begin/end. It buffers those samples and writes them only after Apply
returns. No log write occurs while the guard is installed. Real-agent records
are in `agent.txt` alongside normal agent events; the CLI log is explicitly
labeled as a driver log. No production agent or enforcement file is edited.

Each actual kernel transaction happens somewhere within its command bracket:

- **inside**: node ingress is after install-command completion and before
  release-command start, with a 2 µs margin for microsecond packet timestamps;
- **boundary**: ingress overlaps the wider install-start → release-end bracket,
  including the timestamp margin, but is not definitely inside;
- **outside**: ingress is outside every such bracket;
- **unobserved**: no node ingress was captured; never turn missing ingress into
  an outside-guard result.

Send classification, the nearest guard generation and delta, every guard
intersecting the socket read interval, and all eight packet observation stages
are retained per datagram. The single outstanding UDP request and the header
trace ordering provide correlation without reading payloads. Null means an
unobserved stage. Ambiguous duplicate matches fail the analyzer.

Guard duration has a lower bound (install-end → release-start) and an upper
bound (install-start → release-end); these are not kernel commit timestamps.
The D-104 **100 ms bound actually applies to final-render-start → installed-by**,
not to the duration of the guard. A delayed guard release can reduce
availability while remaining fail closed. Both durations are reported so an
upper bound below 100 ms is not mislabeled as a new product guarantee.

The final capture shutdown is a handshake: the sidecar closes only after its
last recorded read completes; captures stop afterward. The acceptance stream
continues unchanged for revoke checks. That keeps every measured datagram
within capture coverage without changing its timeout or verdict.

## Reproduce

Check the shared-host slot before each privileged run. Run the original full
lane gate on a fresh topology and retain its real pass/fail result:

```sh
pgrep -af 'gates.sh|mk/at-'
FVPN_LAB_PREFIX=fvpn-w1-netlab-f17 \
FVPN_NETLAB_EVIDENCE_DIR=docs/poc/G2-followup17-evidence/lane \
make -f mk/netlab.mk netlab-test
```

The separate real-agent campaign uses normal enrollment, fake central signing
keys and NetBird-status fixture, real WireGuard/nft/tc, and 20 newer published
generations. See [the D-116 run instructions](G2-installer-marker-2026-10-31.md).
`netlab-agent-measure` is a measurement test, not a replacement AT10 acceptance
gate. Its log must still be evaluated for every timeout.

For either retained directory, run `tests/netlab/scripts/reload_analysis.py`
with `--stream <sidecar> --apply <Apply log> --trace-base <capture prefix>
--output <analysis.json>`. The `netlab-unit` target runs correlation/parser
regressions through the disk-temp wrapper. Each live harness tears down its
container and bridge; remove the task-tagged image after the campaign.

## Measured results

Platform: amd64, Linux `7.0.0-38-generic`, nft `1.0.9`, iproute2 `6.1.0`.
Final instrumentation source: `3101b19`. The first full gate used the earlier
capture shutdown; the actual Apply/guard decorator and stream timeouts are
identical. The real-agent campaign below uses the final instrumentation.

| Campaign | Reloads / all Apply calls | Recorded UDP timeouts | At node ingress: definitely inside / boundary overlap / outside / unobserved | Guard upper bound min / median / p95 / max (ms) | Max render → installed (ms) |
| --- | --- | --- | --- | --- | --- |
| Full lane, 09:26:13–09:26:26 UTC | 100 / 101 | 2 | 0 / 2 / 0 / 0 | 20.858 / 33.942 / 48.224 / 54.861 | 29.520 |
| Focused final fixture, 09:33:04–09:33:17 UTC | 100 / 101 | 1 | 0 / 1 / 0 / 0 | 18.174 / 32.905 / 44.661 / 51.318 | 27.863 |
| Real agent, 09:30:16–09:30:46 UTC | 20 published generations / 30 calls | 0 | 0 / 0 / 0 / 0 | 18.040 / 32.315 / 43.621 / 44.707 | 28.107 |

Every measured Apply verified. No installation or guard upper bound exceeded
100 ms. The full lane's guard lower bounds span **0.002307–0.019787 ms**;
the final agent's span **0.002785–0.013020 ms**. These wide brackets explain
why the lost packets cannot honestly be called definitely inside a guard.
Maximum wall-versus-monotonic elapsed disagreement was 479 ns and 596 ns,
respectively, below the classification's 2 µs timestamp margin. The final
focused fixture has guard lower bounds **0.002273–0.016988 ms** and maximum
wall/monotonic elapsed disagreement **410 ns**.

The final agent has **117 datagrams**, no TCP/UDP timeouts, no mismatched echo,
no missing read completion and no missing node ingress. Its captures contain
118 request/response exchanges: one extra exchange from the still-running
acceptance stream after the sidecar closes. All 117 recorded exchanges have
all eight packet observations. Extra trace headers remain retained and counted.

The first full gate **failed AT10 in 15.64 s**, and finished in **212.715 s**
with no other failed scenario. D-104 passed in 3.79 s and the ISO-F6-01 library
recovery case passed in 1.14 s. It is not a green acceptance gate. The final
agent's successful 30.56 s measurement does not erase that failure or promote
assembled AT10.

The final focused AT10 rerun also **failed**, in **14.047 s**, after all 100
reloads. Its **46 datagrams** have complete read records and node ingress;
45 have the full successful eight-stage path, and one is the timeout below.
There are no TCP timeouts or mismatched echoes. This verifies the capture
shutdown fix without converting a real acceptance failure into a pass.

### All three observed timeouts

All times below are UTC on 2026-10-31. Null egress/responder means no packet
observed in the checked zero-drop captures, not a fabricated timestamp.

| Datagram / generation | Send | Origin egress | Node ingress | Read completion | Node egress / responder |
| --- | --- | --- | --- | --- | --- |
| 24 / 1050 | 09:26:19.315661529 | 09:26:19.315673000 | 09:26:19.316201000 | 09:26:20.817370782 | null / null |
| 29 / 1068 (full gate) | 09:26:22.853706846 | 09:26:22.853716000 | 09:26:22.853778000 | 09:26:24.354847947 | null / null |
| 39 / 1068 (final focused) | 09:33:13.741777369 | 09:33:13.741791000 | 09:33:13.743195000 | 09:33:15.242671773 | null / null |

Generation 1050's outer guard bracket is
**09:26:19.313273348–09:26:19.354888201**; install returns at
09:26:19.340198822 and release starts at 09:26:19.340204264.
Generation 1068's bracket is
**09:26:22.850922778–09:26:22.888966648**; install returns at
09:26:22.876203369 and release starts at 09:26:22.876209422.
The final focused generation 1068 bracket is
**09:33:13.740746639–09:33:13.773264421**; install returns at
09:33:13.753206860 and release starts at 09:33:13.753213248.
All three sends and node ingresses overlap guard installation. None reaches
node egress or the responder, and each missing response consumes about
1.501 s at the reader. **Three bracket overlaps, zero outside-bracket timeouts;
zero packets proven to arrive in the narrow definitely-guarded interval.**
This is packet-level correlation with the guard's installation transaction;
the command brackets do not establish the precise drop instant or prove
which nft rule discarded the packet.

The first full-gate sidecar includes a few trailing rows after capture shutdown,
including one unfinished read; neither of the two timeout rows is affected.
That retention defect prompted the final handshake: close the sidecar only
after the pending read completes, then stop captures while the original
acceptance stream continues for revoke checks. The earlier datasets are
retained explicitly as pre-fix evidence, not silently replaced.

### Retained evidence

- [Full-gate packet/Apply correlation](G2-followup17-evidence/lane/TestAT10_AT13_ReloadRevokeBaseline/analysis.json),
  with raw `at10-node.txt`, origin/responder traces, capture statistics,
  `at10-apply.jsonl`, `stream-a1.log.jsonl`, and the unchanged text verdict.
- [Final focused AT10 correlation](G2-followup17-evidence/lane-final/TestAT10_AT13_ReloadRevokeBaseline/analysis.json),
  with complete read/capture coverage and its retained failing verdict.
- [Final real-agent correlation](G2-followup17-evidence/agent/TestD116_AT10_InstallerMarkerEnrollment/analysis.json),
  with raw `agent.txt`, all three captures and `agent-stream.txt.jsonl`.
- `agent-initial` preserves the initial JSON-only metadata assertion failure;
  `agent-pre-boundary-fix` preserves the successful earlier agent run and its
  one trailing positive row without capture coverage.

The orchestrator and isolation follow-up 8 must decide how these measured
brief forwarding denials relate to normal-Internet continuity. No threshold,
pass criterion, retry behavior, production guard or verifier is changed here.
