# Follow-up 25 — bounded first selections and persistent agent exit status

2026-11-01. TEST ONLY, owned container namespaces. NetBird remains **0.82.0** at the unchanged image digest. No NAT, kernel, member-validation or `node/agent/probe.go` change. All **122** follow-up 24 files remain byte-identical; their pins verified before work and again at completion. New evidence is sealed by [followup25-captures.sha256](../../../followup25-captures.sha256).

## Campaign and counts

The [predeclared plan](plan.md) schedules **eight fresh stacks per family in each arm**, 32 attempts total. Every attempt has one first route selection and one bounded original HTTP request, with no counted-flow prerequisite, reconnect or replacement request. The same two-second passive tail remains. The family constraint is TEST ONLY, not a production preference/fallback policy. Family means the **native peer underlay**; the controlled HTTP origin remains IPv4. These sequential diagnostic counts are not a natural reproduction rate. See [how to run](how-to-run.md) and the [complete ledger](campaign.json).

| Node arm | Native family | Attempts | Pass | Timeout | `::1` response attempts | IPv6 header-error increase attempts |
|---|---|---:|---:|---:|---:|---:|
| kernel | ipv4 | 8 | 7 | 1 | 0 | 0 |
| kernel | ipv6 | 8 | 7 | 1 | 2 | 2 |
| userspace | ipv4 | 8 | 7 | 1 | 0 | 0 |
| userspace | ipv6 | 8 | 5 | 3 | 0 | 0 |

All 32 startups, capture/trace exports, receipt analyses and teardowns completed. Partial observer failures remain explicit: userspace node `wg` commands were unavailable, and the first three userspace attempts precede the additional daemon handshake field. They are retained, not rerun or represented as zero handshakes.

## Every attempt

Each linked row contains endpoint/handshake transitions, native-status versus WireGuard readbacks, first-origin timing, type-4 ordering before/after each response reference, forwarding state and IPv4/IPv6 namespace counters. Type-4 and error deltas below are **member / node**. Origin times are bounded milliseconds from the first member SYN to the first origin packet; HTTP completion/timeout is separate.

| Attempt / full measurements | Result | SYNs | First origin bounds, ms | HTTP ms | Type-4 records M/N | Node `::1` responses | ΔIPv6 header errors M/N | ΔIPv4 header errors M/N |
|---|---|---:|---|---:|---|---:|---|---|
| [kernel-ipv4-01 / 48dbf84e](48dbf84e-00e5-4250-84d2-0a36b074ed1f/measurement.md) | fail | 5 | absent | 5004 | 6/6 | 0 | 0/0 | 0/0 |
| [kernel-ipv4-02 / ce5c270d](ce5c270d-a665-4a9c-bce2-755334327a5b/measurement.md) | pass | 1 | 204.433926–204.438038 | 207 | 19/19 | 0 | 0/0 | 0/0 |
| [kernel-ipv4-03 / e15d15c4](e15d15c4-24a8-4584-bc9d-649bf58a8ea0/measurement.md) | pass | 1 | 205.034107–205.038215 | 209 | 22/22 | 0 | 0/0 | 0/0 |
| [kernel-ipv4-04 / b3641b6b](b3641b6b-d4b8-4590-aef7-b7d9c4aa0964/measurement.md) | pass | 1 | 105.103247–105.106953 | 111 | 20/20 | 0 | 0/0 | 0/0 |
| [kernel-ipv4-05 / d9d7cbcc](d9d7cbcc-8125-4746-8fc4-7e07ee67a7cd/measurement.md) | pass | 1 | 4.487919–4.497104 | 6 | 22/22 | 0 | 0/0 | 0/0 |
| [kernel-ipv4-06 / 4acbbcda](4acbbcda-39a2-439d-8483-7a36e26a1d36/measurement.md) | pass | 1 | 4.851156–4.855078 | 7 | 21/21 | 0 | 0/0 | 0/0 |
| [kernel-ipv4-07 / 9e81482d](9e81482d-eba9-4045-8f78-2cd2d63b107f/measurement.md) | pass | 1 | 105.884161–105.888208 | 108 | 21/21 | 0 | 0/0 | 0/0 |
| [kernel-ipv4-08 / fe2dc734](fe2dc734-f817-4fa3-a0d2-79fbb16cb713/measurement.md) | pass | 1 | 106.015906–106.020234 | 110 | 19/19 | 0 | 0/0 | 0/0 |
| [kernel-ipv6-01 / 5e59201a](5e59201a-53ba-4f10-b1ee-30a78251bea3/measurement.md) | pass | 1 | 6.268232–6.271902 | 8 | 20/20 | 1 | 1/0 | 0/0 |
| [kernel-ipv6-02 / 953565d0](953565d0-2138-4d16-b27a-1b38553ddb36/measurement.md) | fail | 5 | absent | 5004 | 6/6 | 0 | 0/0 | 0/0 |
| [kernel-ipv6-03 / aedd33ae](aedd33ae-ca2f-4c14-a118-9bd818f03fc6/measurement.md) | pass | 1 | 3.431313–3.43514 | 6 | 24/24 | 1 | 1/0 | 0/0 |
| [kernel-ipv6-04 / 74020b4c](74020b4c-45d3-46df-abfd-c054339ae11f/measurement.md) | pass | 1 | 4.364956–4.369397 | 7 | 21/21 | 0 | 0/0 | 0/0 |
| [kernel-ipv6-05 / 5a632fa8](5a632fa8-f968-47cd-b832-5df30fab5d8d/measurement.md) | pass | 1 | 3.429969–3.434137 | 5 | 21/21 | 0 | 0/0 | 0/0 |
| [kernel-ipv6-06 / 5a0cc305](5a0cc305-d25f-4582-b297-f0724cd79079/measurement.md) | pass | 1 | 105.097183–105.100966 | 107 | 22/22 | 0 | 0/0 | 0/0 |
| [kernel-ipv6-07 / 5863db6c](5863db6c-083e-4b68-8c34-f0b22afca0f1/measurement.md) | pass | 1 | 104.578978–104.583209 | 107 | 24/24 | 0 | 0/0 | 0/0 |
| [kernel-ipv6-08 / 1682e3e2](1682e3e2-7518-4d5f-b118-0251a3efa01f/measurement.md) | pass | 1 | 104.398272–104.401954 | 108 | 22/22 | 0 | 0/0 | 0/0 |
| [userspace-ipv4-01 / 5a592c40](5a592c40-e3c9-42eb-b868-9e255e9a9527/measurement.md) | pass | 1 | 4.87803–4.881985 | 7 | 24/0 | 0 | 0/0 | 0/0 |
| [userspace-ipv4-02 / 79feae71](79feae71-c2d7-4d79-831d-9006e7f0c8ba/measurement.md) | pass | 1 | 206.732114–206.736106 | 210 | 18/18 | 0 | 0/0 | 0/0 |
| [userspace-ipv4-03 / b25453a7](b25453a7-c8c2-4a4e-93ec-70477bddd342/measurement.md) | pass | 1 | 205.834115–205.837804 | 207 | 19/19 | 0 | 0/0 | 0/0 |
| [userspace-ipv4-04 / 98e0e91e](98e0e91e-c32e-4f22-a422-16ba6fe014a2/measurement.md) | fail | 5 | absent | 5003 | 6/0 | 0 | 0/0 | 0/0 |
| [userspace-ipv4-05 / e38f2d1a](e38f2d1a-b003-4287-9f35-babdf6718223/measurement.md) | pass | 1 | 105.101194–105.105319 | 108 | 23/0 | 0 | 0/0 | 0/0 |
| [userspace-ipv4-06 / 996a53a4](996a53a4-8efb-49ee-8ba2-581eb11f3daa/measurement.md) | pass | 1 | 204.567901–204.5718 | 207 | 20/20 | 0 | 0/0 | 0/0 |
| [userspace-ipv4-07 / cabe67d1](cabe67d1-708a-4c68-ab75-22b043d7abb3/measurement.md) | pass | 1 | 105.475167–105.478846 | 109 | 20/0 | 0 | 0/0 | 0/0 |
| [userspace-ipv4-08 / 92bb1d45](92bb1d45-183f-4f74-b4f7-c9b61eea42ee/measurement.md) | pass | 1 | 106.118222–106.122038 | 109 | 20/0 | 0 | 0/0 | 0/0 |
| [userspace-ipv6-01 / 2cf62be6](2cf62be6-7684-449d-8f8f-814da0f64cd6/measurement.md) | fail | 5 | absent | 5004 | 6/0 | 0 | 0/0 | 0/0 |
| [userspace-ipv6-02 / 5625cd0e](5625cd0e-b425-4a48-b06a-5cb282ac0b4b/measurement.md) | pass | 1 | 104.581117–104.585114 | 108 | 23/0 | 0 | 0/0 | 0/0 |
| [userspace-ipv6-03 / 1d253f8d](1d253f8d-f771-4edc-af93-f4220df54877/measurement.md) | fail | 5 | absent | 5004 | 6/0 | 0 | 0/0 | 0/0 |
| [userspace-ipv6-04 / 3591e079](3591e079-8c38-4c85-8298-3dec2589e1b2/measurement.md) | pass | 1 | 103.774205–103.777834 | 106 | 20/20 | 0 | 0/0 | 0/0 |
| [userspace-ipv6-05 / 0c27cac0](0c27cac0-ec38-4328-bb11-be2d3545ac04/measurement.md) | fail | 5 | absent | 5004 | 6/6 | 0 | 0/0 | 0/0 |
| [userspace-ipv6-06 / 6e4b6eb3](6e4b6eb3-e7b0-4d94-af8d-b83eb42bbd34/measurement.md) | pass | 1 | 205.595139–205.599054 | 207 | 19/19 | 0 | 0/0 | 0/0 |
| [userspace-ipv6-07 / 9bb914e5](9bb914e5-2871-48a2-b3fc-f6e8a340dd68/measurement.md) | pass | 1 | 106.120139–106.124053 | 110 | 23/0 | 0 | 0/0 | 0/0 |
| [userspace-ipv6-08 / beba4252](beba4252-d205-44a9-9241-acddde335b57/measurement.md) | pass | 1 | 105.15238–105.157546 | 108 | 21/0 | 0 | 0/0 | 0/0 |

## TEST ONLY userspace comparison

The lab daemon alone received `NB_WG_KERNEL_DISABLED=true`; captured status confirms **node userspace / member kernel** in every arm sample. The one-use marker is consumed and cleaned up, leaving no override for an ordinary later boot. The default fixture and Dockerfile pin are unchanged. Each userspace IPv6 attempt has a sampled native IPv6 endpoint; this is not merely absence while no IPv6 path was observed.

In these eight userspace IPv6 captures, the node `::1`-source response and the IPv6 header-error increment **disappear from the observed windows**: zero such responses and zero increase in either namespace. Kernel IPv6 has two matching response/error cases, both on successful requests. This is evidence that the specific response-source boundary is absent in this bounded diagnostic arm, **not a general forwarding fix** or proof of absence outside the captures. Userspace still has retained first-request timeouts. Using it on nodes conflicts with the agreed AGENTS.md §3 kernel-WireGuard baseline; architecture, CPU/throughput, enforcement/metering and both-architecture qualification remain owner decisions. No option is recommended here.

## What the fresh timeouts establish

The retained failing requests have five SYNs at the member tunnel and five at
the node's `fvmb0` tunnel, no origin receipt, zero sampled byte counters before
and after, and **generation 0 / serving=false after the request**. Their prior
state differs: most already sample generation 0 and serving=false;
[1d253f8d](1d253f8d-f771-4edc-af93-f4220df54877/measurement.md) has generation 0
with a true earlier heartbeat; and
[0c27cac0](0c27cac0-ec38-4328-bb11-be2d3545ac04/measurement.md) starts at
**generation 5 / serving=true** and ends at **0 / false**. These independently
timed reads are not an atomic state snapshot or a trace of the exact withdrawal
instant. Counter replacement also means zero endpoint samples cannot prove no
intermediate counter activity.

Their selected member peers remain matched, handshakes complete, and member
type-4 records are present. Neither IPv4 nor IPv6 receive header errors increase
in these failures, and none has a captured `::1` response. Passing requests have
an independent origin receipt and increasing endpoint counter samples. Every
row retains its own state and endpoint transitions; the failures are not
silently collapsed into one pre-request condition.

This places these observed failures after successful native delivery to the
node tunnel and before origin receipt, alongside a closed forwarding state.
It does **not** establish why the node was closed, identify an exact dropping
rule, or retrospectively diagnose follow-up 24's d556673b. That historical row
stays failed and unattributed. The old IPv6 five-SYN/**no-type-4** failure is a
different observation; these fresh failures have type-4 traffic and node-tunnel
SYN receipts. No timeout inherits the `::1` diagnosis merely because its family
is IPv6.

The two kernel IPv6 loopback-source response examples are
[5e59201a](5e59201a-53ba-4f10-b1ee-30a78251bea3/measurement.md) and
[aedd33ae](aedd33ae-ca2f-4c14-a118-9bd818f03fc6/measurement.md). Both requests
succeed with one SYN. In the first, node response **179** joins member **212**
by exact WireGuard message digest; member loopback-source counters are ingress
**1**, prerouting **0**, input **0**, with SNMP6 and nstat header errors **0→1**.
All **20** type-4 records on each peer precede the response. The second has
**24** type-4 records on each peer before its response. These are further
observations of follow-up 24's receive boundary, not evidence that the boundary
caused either successful request to time out. No kernel drop tracepoint or
new distribution-source equivalence is claimed.

## Observation limits and source provenance

The original `wg` command outcomes remain explicit. The userspace node's
captured CLI reads are unavailable; their empty endpoint/handshake fields are
not proof of an empty WireGuard peer or a missing handshake. The member remains
kernel WireGuard and its selected-peer endpoint/handshake snapshots are retained
through the allowed-IP handover. Node ICE endpoints come from native status and
are labelled separately from WireGuard endpoints.

After the first three userspace attempts, the observer additionally records the
native daemon's `lastWireguardHandshake` as `native_latest_handshake_at`. This
cached telemetry is a separate source, never a replacement for an independent
failed `wg` read. Earlier missing timestamps stay missing. Userspace-ipv4-02 also retains three
pre-selection/selection samples without a selected member peer; it is matched
from selection-2 onward, including before and throughout the measured transfer.
Those early gaps are not silently filled. The change has a
focused regression for nonzero and zero timestamps. Attempt userspace-ipv4-04
has runner revision **6b075f99** and a separately built sampler at **d06bdb48**;
its [binary receipt and build-VCS readback](98e0e91e-c32e-4f22-a422-16ba6fe014a2/observer-revision.json)
make this mixed instrumentation revision explicit. Later runs use the same added field on their recorded runner revision.
No original attempt is regenerated to acquire the additional field.

The native type-4 decoder reports structural **packet records**, including
multiple proxy hops, not authenticated or unique WireGuard messages. In the
userspace arm the node's in-process bind/relay path does not expose the same
loopback records: zero node type-4 capture records do not imply zero data, as
node inner SYN receipt and the successful HTTP flows demonstrate. Ordering
against a missing egress response is unknown. Endpoint/handshake snapshots are
bounded samples, not a continuous history. The capture time to first origin
packet is distinct from HTTP completion and the independent origin request
counter. Counter-read success and clock/decoder bounds are preserved in each
original record. No physical arm64 run, throughput result, full platform
acceptance, or production architecture qualification is claimed.

## Persistent stop status for updates 13

The TEST ONLY service parent calls Wait and atomically publishes a mode-0600
receipt keyed by a unique launch ID. The updater observer reads the ID before
stop and the corresponding `agent-exits/<id>.json` after stop; repeated stop and
later restart do not replace it. The receipt records exit status versus signal,
raw wait status, PID/start ticks and UTC wait-observation time. A separate
`<id>.sigkill.json` records a supervisor escalation attempt and whether kill
accepted it. Existing signals, five-second escalation threshold and product
shutdown limits remain unchanged. No production API is added.

[The live graceful stop](d513a413-4597-423c-9dd3-7c0669f03eca/agent-exit-graceful.json)
records exit **0**, signal **null**, at **2026-11-01T10:15:18.957217162Z**;
its service command took **468 ms**. [The explicit SIGKILL](d513a413-4597-423c-9dd3-7c0669f03eca/agent-exit-sigkill.json)
records exit **null**, signal **9**, at **2026-11-01T10:15:20.081162215Z**;
its following stop command took **249 ms**. Both receipts were read after the
agent was reaped and were unchanged by another stop. Unit tests also distinguish
normal exit **137** from SIGKILL and preserve three separate launch receipts.
The live SIGKILL is harness-injected, not an observed supervisor escalation.

These times are parent wait observations, not exact kernel exit instants.
Killing the whole container or the waiter can prevent a receipt; missing data
must stay unavailable. The [dated D-102 answer](../../../RESPONSES.md#updates-13--2026-11-01-follow-up-25-d-102)
gives the exact consumer sequence. Updates' historical first AT23 exit remains
unobserved; no new test repairs that historical record.


## Hand-offs

1. **2026-11-01 — native integration / agent-core owner:** use the complete counts and retained failures alongside [follow-up 24’s owner options](../followup24/analysis.md). Decide whether to investigate an upstream correction/pin, a baseline change for userspace, or a defined IPv4 preference. Investigate the separately observed post-tunnel forwarding gap and generation-0 transitions before assigning a product cause. No production patch is proposed from an unobserved cause; `node/agent/probe.go` stays observation-only.
2. **2026-11-01 — updates:** adopt the D-102 readback in `tests/e2e/at/updates/stopobserve/main.go`. That lane-owned observer remains untouched. Preserve the historical unobserved row; use the new unique-ID receipt for subsequent stops.

[Gate results and real output tails](gates.md). Raw packets, logs and the sampler binary remain private; the ledger records verified receipts. Full AT01/AT09/AT23 acceptance and a repair are not claimed.
