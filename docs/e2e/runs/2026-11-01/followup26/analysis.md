# Follow-up 26 — central closure records, native re-grant and stop budgets

2026-11-01. TEST ONLY, kernel fixture only. All attempts and failed prerequisites are retained. [Predeclared plan](plan.md), [complete campaign ledger](campaign.json), [gate results](gates.md), and per-run measurements below. Follow-up 25 remains byte-identical under its original 906 pins. New pins are only under this immutable archive.

The historical node tracer reproducibly closes an admitted node with `ISOLATION_POLICY_INVALID`: the local baseline becomes generation 0 while central applied generation stays 5. Normal observation now omits node marking and proves serving before measurement. Separately, a passive re-grant timeout retains serving generation 7 while native handshake/data delivery misses the request deadline. Historical F11-01/F14-01 causes and the transport choice are not retroactively settled.

## Campaign

16 historical-tracer reproductions (eight measured attempts per family), four corrected controls (two per family), and two separately declared 12-interval tracer-hold fault probes: **22 measured fresh stacks**, plus one compile-only fresh stack. The first stack also had a compile-only command before any runner executed. Both build failures are retained; no failed request was replaced. The extra ipv4-09 supplies the eighth measured IPv4 reproduction.

| Attempt | Run / measurements | Initial request | SYNs member/node; origin receipts | Closed report in initial capture | Re-grant request |
|---|---|---|---|---|---|
| ipv4-01 | [68591f0c](68591f0c-0fa0-414f-8187-590d8ce11bc5/measurement.md) | pass | 1/1; 1 | 1 | pass (traced) |
| ipv4-02 | no runner / no packet | compile failure | — | — | not attempted |
| ipv4-03 | [82b02856](82b02856-f33c-4219-8d5a-acfc1287903b/measurement.md) | pass | 1/1; 1 | 1 | pass |
| ipv4-04 | [822ad3c7](822ad3c7-b203-4793-82bb-2d3649d5fee4/measurement.md) | pass | 1/1; 1 | 1 | pass |
| ipv4-05 | [2d9dc8ee](2d9dc8ee-50ca-4ad1-9ed9-f077041f2f64/measurement.md) | pass | 1/1; 1 | 1 | pass |
| ipv4-06 | [c1f20cc7](c1f20cc7-a51c-48dc-9c55-e538603169b5/measurement.md) | pass | 1/1; 1 | 1 | pass |
| ipv4-07 | [85c2169c](85c2169c-5aac-4f45-b0f6-4ebc9659f208/measurement.md) | pass | 1/1; 1 | 1 | pass |
| ipv4-08 | [5f47e04b](5f47e04b-d955-44e4-bd29-9dd80d949a06/measurement.md) | pass | 1/1; 1 | 1 | pass |
| ipv6-01 | [7ab2130a](7ab2130a-bf7c-48a0-a498-f95967a5ed96/measurement.md) | pass | 1/1; 1 | 1 | pass |
| ipv6-02 | [11a4b4b6](11a4b4b6-b3c3-4082-97d0-1f91a63124f7/measurement.md) | pass | 1/1; 1 | 1 | pass |
| ipv6-03 | [fa2cf325](fa2cf325-1959-4ace-b456-de5c669aabbe/measurement.md) | pass | 1/1; 1 | 1 | pass |
| ipv6-04 | [528e1feb](528e1feb-9898-4405-9396-dd8bc69d87de/measurement.md) | pass | 1/1; 1 | 1 | pass |
| ipv6-05 | [76018d07](76018d07-bf89-4bbb-9d8a-ccbf40292446/measurement.md) | pass | 1/1; 1 | 1 | pass |
| ipv6-06 | [ce8ccbe1](ce8ccbe1-ed7d-4fc0-b1ed-786ef0e7578c/measurement.md) | fail | 5/5; 0 | 2 | prerequisite failed; no request |
| ipv6-07 | [af718646](af718646-166a-4a3f-af55-3f7a3ee8e17b/measurement.md) | pass | 1/1; 1 | 1 | fail |
| ipv6-08 | [65260e25](65260e25-8aa5-4e7c-994f-51525514e801/measurement.md) | pass | 1/1; 1 | 1 | pass |
| corrected-ipv4-01 | [0343c6b9](0343c6b9-3796-40b0-bc5e-8d5d86e6dfd9/measurement.md) | pass | 1/1; 1 | 0 | pass |
| corrected-ipv4-02 | [0d25f1ba](0d25f1ba-af4c-49a6-bc4f-0fb0f42f1008/measurement.md) | pass | 1/1; 1 | 0 | pass |
| corrected-ipv6-01 | [e79579f7](e79579f7-51af-4fc2-ad1a-7e4ba64fe181/measurement.md) | pass | 1/1; 1 | 0 | pass |
| corrected-ipv6-02 | [efd253e7](efd253e7-c9a0-4fdc-86e3-f35f2ef2fe64/measurement.md) | pass | 1/1; 1 | 0 | pass |
| ipv4-09 | [3be19193](3be19193-d023-44c7-9388-a6a1f697ba8f/measurement.md) | pass | 1/1; 1 | 1 | pass |
| trace-hold-ipv4 | [86153543](86153543-1251-48f4-b368-cf11e5b0939d/measurement.md) | fail | 5/5; 0 | 4 | prerequisite failed; no request |
| trace-hold-ipv6 | [a3154028](a3154028-4296-4a61-afba-fee20fc0aa5f/measurement.md) | fail | 5/5; 0 | 4 | prerequisite failed; no request |

| Population | Pass | Fail |
|---|---:|---:|
| historical-tracer-hold/ipv4 | 0 | 1 |
| historical-tracer-hold/ipv6 | 0 | 1 |
| historical-tracer/ipv4 | 8 | 0 |
| historical-tracer/ipv6 | 7 | 1 |
| member-only-tracer/ipv4 | 2 | 0 |
| member-only-tracer/ipv6 | 2 | 0 |

Re-grant population: not_attempted=3, passive_fail=1, passive_pass=17, traced_pass=1. A prerequisite failure is not a forwarding attempt. The first traced re-grant is excluded from the passive population.

## Failure-specific joins

The natural first-selection timeout [ce8ccbe1](ce8ccbe1-ed7d-4fc0-b1ed-786ef0e7578c/timeout-analysis.md) has proven generation-5 admission, a baseline closure at +1.650…1.704 seconds, a failed later generation-5 Apply while the observer remains, and recovery at the same generation after removal. Its first SYN reaches the node only after 10.444 seconds, beyond the 5.003-second request deadline. Closure is explained; closure alone is not a complete timeout attribution.

The two separately declared held-tracer fault probes retain their failures and are detailed in [held-tracer-analysis.md](held-tracer-analysis.md). They are not first-selection acceptance rows.

The separate passive re-grant timeout [af718646](af718646-166a-4a3f-af55-3f7a3ee8e17b/regrant-analysis.md) keeps generation 7 and serving intact. A loopback-source type-2 response, one member IPv6 header error, handshake 0 during the request and a later native handshake accompany first node delivery at 5.671 seconds. This localizes that fresh delay to native establishment/data delivery; it does not retroactively assign 258ba504 a cause.

## What closed the reproduced nodes

The TEST ONLY node nft observer can invalidate the safety property it is observing. Its broad IPv6 UDP `meta nftrace set 1` statements are unscoped foreign `mangle` operations under the unchanged verifier. The exact captured listing identifies the tracer table and `ISOLATION_POLICY_INVALID`; later projections identify netdev ingress as the first rejection. The isolated inet prerouting rule also fails the real verifier regression. This affects verification even when the measured request uses the IPv4 underlay.

The accepted central reports retain `applied_config_generation=5` while forwarding becomes false with `safety_check_failed` and `ISOLATION_POLICY_INVALID` at stage `apply`. The producer stream distinguishes the failed live verification from subsequent enforcement attempts: stage `apply` alone does **not** prove a failed Apply transaction. The lease, QoS and source-binding checks can all carry that same rejection code; the exported lease/member windows remain live. Those failed checks are not evidence of lease expiry. The local baseline writer records `verification_ISOLATION_POLICY_INVALID`. That deny baseline explains a kernel counter generation of 0 without any central desired generation moving backwards to 0. The observer remains installed through the request and passive tail; later attempts cannot establish verified serving while the same foreign rule remains. The post-removal recovery gate is independent of the original request result.

The owned fix is to remove node marking from ordinary observation, retain passive native captures, and require serving admission before the first request. The historical observer is available only through the explicit diagnostic flag. No product verifier, NAT, kernel, member validation, NetBird pin or `node/agent/probe.go` changed. This evidence identifies harness interference, not a demonstrated product defect requiring a product patch.

## Admission and history

The old `fixture-member-admission` row includes native identity, finite grant and selection setup, but its final success depended on the same first request and counter increase. Its six failures therefore restated the failed request; they did not establish that admission had never completed. The corrected 90-second gate independently joins a fresh accepted serving report, a populated live same-generation lease and intended authorized binding, local desired/kernel generations, and a fresh serving heartbeat. A failed counter read ends the gate without replacement. A gate that never becomes serving remains a failed admission row and launches no first-selection request. Corrected controls recheck after observer setup. Historical-tracer and held-tracer diagnostics are labeled separately from unconfounded acceptance.

[Historical read-only extraction](history-review.json): d556673b already sampled generation 0 / serving=false before and after its request. The earlier IPv6 five-SYN rows 9b3a89e7 and a26d9b0f sampled generation 5 / serving=true before and after; they do not show the same closed-node signature. Follow-up 25's four initially false samples, 1d253f8d's older true heartbeat alongside generation 0, and 0c27cac0's 5/true → 0/false transition remain exactly as recorded. They have no retained accepted central cause history or baseline writer stream. The new mechanism is relevant to the observer used there, but it cannot retroactively name each historical closure's exact reason or instant. “Four never served” is not established by those samples. The four pre-request closed samples lack a demonstrated serving prerequisite and cannot serve as qualified first-selection acceptance; the retained data cannot decide never-admitted versus previously-admitted then withdrawn. No historical failure is reclassified, and F11-01's remaining transport options still belong to Ted.

## F14-01 is a separate population

The original 258ba504 failure remains `cause_status: unproven`: readiness at generation 10 passed, then five member packets reached neither node nor origin, with unchanged counters and no failed Apply. Here, native session capture starts before the D-077 authorization fence, samples worker withdrawal and explicit same-key idempotent re-authorization, and continues through populated serving readiness, route selection and the single re-grant request. No request retry replaces a failure. Raw capture/log bytes remain private; public headers, peer/handshake/route samples and their hashes are retained.

These are reduced fresh-stack D-077 populations. They do not replay every preceding mutation in the members assembled sequence. Serving readiness deliberately does not demand an already established native handshake: in retained passing examples the old peers disappear on withdrawal, new peers return with handshake 0 / disconnected status, and the first request establishes the new handshake before reaching node ingress. Packet-level transit bounds use matching TCP SYN headers, measured monotonic/wall-clock envelopes and pcap quantization. Encrypted outer datagrams are not assigned to an inner request by timing alone. A passing first selection or re-grant closes neither F11-01 nor F14-01.

## Counter classification and stopping

`errors.As` now recognizes `*enforce.DuplicateChainError` as `ambiguous_listing` / `duplicate_chain`. Exact and differing duplicate chains have separate real-parser cases, alongside syntax/type failures and a wrapped typed error. The failed `nft_parse` read, absent observation, one-read behavior, private original/error retention and byte/hash receipts stay intact. No parser error text is published and no automatic retry is added. Netlab's 0/3,300 result and F13-01's unproven origin remain unchanged.

The supervisor now waits a monotonic 15 seconds after TERM delivery before attempting KILL. PID1's TERM/INT handler calls that same supervised stop path and keeps the parent waiter and watchdog alive until the receipt is persisted. This is signal forwarding, not scratch cleanup. Compose and secondary/clone nodes allow 25 seconds; shared namespace/fleet/clone wrappers allow 25 seconds (ordinary teardown still allows 20 seconds, above the 15-second agent budget). Container-stop intent permits automatic resume while an explicit service pause persists. Container exit 0 means its handler completed; the parent receipt is required to distinguish agent exit 0 from signal 9.

The dated [D-102 replies](../../../RESPONSES.md) name updates' `tests/e2e/at/updates/stopobserve/main.go`, isolation's whole-node and clone rows in `driver.go.txt`, and its direct-agent `observer_main.go` consumer. Read the retained process identity before stopping; use `docker cp` for the receipt after a whole-container stop. The `.sigkill.json` sidecar records an attempted escalation, while the parent receipt records the actual result. Historical exit-137 rows and missing first-exit observations remain unchanged.

The serving gates change measurement sequencing and timing. These small diagnostic populations do not estimate a transport reliability rate or demonstrate that either historical finding is fixed.

Displayed millisecond intervals are rounded to 0.001 ms; the exact nanosecond bounds remain in the JSON joins.


## Native re-grant transit bounds

Intervals match the first inner SYN TCP header at member egress and node ingress; they are not encrypted-payload inference. Late receipts remain late and never replace the failed request.

| Attempt | Outcome | First SYN transit interval (ms) |
|---|---|---|
| [ipv4-01](68591f0c-0fa0-414f-8187-590d8ce11bc5/session-intervals.json) | pass | 5.211 … 5.215 |
| [ipv4-03](82b02856-f33c-4219-8d5a-acfc1287903b/session-intervals.json) | pass | 4.901 … 4.907 |
| [ipv4-04](822ad3c7-b203-4793-82bb-2d3649d5fee4/session-intervals.json) | pass | 105.073 … 105.084 |
| [ipv4-05](2d9dc8ee-50ca-4ad1-9ed9-f077041f2f64/session-intervals.json) | pass | 104.792 … 104.800 |
| [ipv4-06](c1f20cc7-a51c-48dc-9c55-e538603169b5/session-intervals.json) | pass | 203.902 … 203.907 |
| [ipv4-07](85c2169c-5aac-4f45-b0f6-4ebc9659f208/session-intervals.json) | pass | 5.032 … 5.037 |
| [ipv4-08](5f47e04b-d955-44e4-bd29-9dd80d949a06/session-intervals.json) | pass | 3.444 … 3.458 |
| [ipv6-01](7ab2130a-bf7c-48a0-a498-f95967a5ed96/session-intervals.json) | pass | 103.260 … 103.264 |
| [ipv6-02](11a4b4b6-b3c3-4082-97d0-1f91a63124f7/session-intervals.json) | pass | 105.675 … 105.679 |
| [ipv6-03](fa2cf325-1959-4ace-b456-de5c669aabbe/session-intervals.json) | pass | 4.657 … 4.664 |
| [ipv6-04](528e1feb-9898-4405-9396-dd8bc69d87de/session-intervals.json) | pass | 3.515 … 3.521 |
| [ipv6-05](76018d07-bf89-4bbb-9d8a-ccbf40292446/session-intervals.json) | pass | 3.301 … 3.308 |
| [ipv6-07](af718646-166a-4a3f-af55-3f7a3ee8e17b/session-intervals.json) | fail | 5671.079 … 5671.087 |
| [ipv6-08](65260e25-8aa5-4e7c-994f-51525514e801/session-intervals.json) | pass | 3.511 … 3.517 |
| [corrected-ipv4-01](0343c6b9-3796-40b0-bc5e-8d5d86e6dfd9/session-intervals.json) | pass | 3.230 … 3.236 |
| [corrected-ipv4-02](0d25f1ba-af4c-49a6-bc4f-0fb0f42f1008/session-intervals.json) | pass | 203.521 … 203.527 |
| [corrected-ipv6-01](e79579f7-51af-4fc2-ad1a-7e4ba64fe181/session-intervals.json) | pass | 103.923 … 103.927 |
| [corrected-ipv6-02](efd253e7-c9a0-4fdc-86e3-f35f2ef2fe64/session-intervals.json) | pass | 104.526 … 104.534 |
| [ipv4-09](3be19193-d023-44c7-9388-a6a1f697ba8f/session-intervals.json) | pass | 105.883 … 105.894 |

## Stop receipt proofs

| Run | Mode | Container exit | Agent exit / signal | Stop elapsed (ms) | Post-TERM budget before KILL (ms) |
|---|---|---:|---|---:|---:|
| [2cd90524](2cd90524-b768-4abc-8314-b9cded71a9ad/container-stop-forced.json) | forced | 0 | None / 9 | 15790 | 15200 |
| [2cd90524](2cd90524-b768-4abc-8314-b9cded71a9ad/container-stop-graceful.json) | graceful | 0 | 0 / None | 509 | None |
| [38fa3ba8](38fa3ba8-6830-420d-bd6c-dca1125b67f7/container-stop-forced.json) | forced | 0 | None / 9 | 15723 | 15150 |
| [38fa3ba8](38fa3ba8-6830-420d-bd6c-dca1125b67f7/container-stop-graceful.json) | graceful | 0 | 0 / None | 568 | None |
| [7853c630](7853c630-c61c-48fd-bf37-475d29cb6608/container-stop-forced.json) | forced | 0 | None / 9 | 15640 | 15080 |
| [7853c630](7853c630-c61c-48fd-bf37-475d29cb6608/container-stop-graceful.json) | graceful | 0 | 0 / None | 514 | None |

The first stop run is retained preliminary evidence. Later runs exercise automatic resume and the final post-delivery deadline implementation. SIGSTOP is deliberately used to prevent cooperative completion; the supervisor sidecar and actual parent signal-9 receipt prove escalation after the full budget. Ordinary exit-0 is a separate parent receipt, not inferred from Docker.

## Evidence and limits

Verified 289 private raw-artifact receipts across the captured runs, including packet captures, debug logs, exact tracer listings and producer streams when present. The first run predates producer export and traced its re-grant; its first trace was retained separately before the second trace export. Its central report still names the closure, but no writer instant is fabricated. Earlier derived joins used null `peer_status`/`connected` aliases; immutable `session-intervals.json` supplements read the correct original `peer_connected` field. Original capture records were not changed.

The public central snapshot is one read-only repeatable-read transaction: every accepted observation with lease/QoS/source-binding checks and code/stage/time errors, all immutable desired generation/signing times, and all issued lease/member windows. Reports are discrete observations, not continuous state. Producer sequence continuity is verified per stream; packet clocks retain their measured bounds. Runtime raw bytes stay private.

Live evidence is Linux amd64 in the owned lab. Arm64 is cross-compiled, not claimed live-qualified. Full members history, all real devices/networks, historical causes and the transport decision assigned to Ted remain outside these bounded measurements. No production parameter or pin changed.
