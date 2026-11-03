# Follow-up 27: populated hold and per-family native measurements

2026-11-01. The populated stale-lease fixture is ready for isolation and members.
The declared native campaign completed **four fresh IPv4 stacks and four fresh
IPv6 stacks**, each with first selection after proven serving and a 150-second
serving-node stream. **Neither historical five-SYN failure nor the five-second
missing-ingress gap reproduced.** IPv6 did reproduce loopback-source underlay
anomalies, including responses that arrived after another handshake had already
carried the first request. These are separate observations, not a common-cause
assignment to the three historical samples.

## Preservation and attempts

All **987 follow-up 26 pins verified** before work and again after the campaign;
that archive and manifest are byte-identical to main. Historical members
F15-F11-01, F15-AT02 and archived F14-01 remain byte-identical. F14-01 retains
`cause_status: unproven`.

The [predeclared plan](plan.md) and [150-second amendment](stream-duration-amendment.md)
precede all transport attempts. The original one hold attempt was followed by
[two declared harness-only reruns](hold-fixture.md); both earlier failures and
separate read-only local-byte supplements remain. Total: **11 fresh stacks**,
with no discarded or substituted attempt. The final hold runner is 13 / 0 / 0;
each native runner is 16 / 0 / 0 for its fixture/instrumentation conditions.
These counts do not grant AT01/AT02 acceptance: every stream loss remains counted.

The [campaign joins](campaign.json) retain every native attempt, accepted serving
report/lease, source commit, capture checks and corrected initiation origins.
Large originals are lossless gzip with compressed and uncompressed SHA-256 and
byte counts in each run's `archive.json`. Plain Markdown navigation paths remain
present. Pins in `docs/e2e/followup27-captures.sha256` name only this immutable
archive. The [origin-label correction](analysis-correction.md) adds supplements;
it does not rewrite any original capture, result or earlier convenience list.

## Product-owned populated hold

The lease path on main is **POST `/api/v1/agent/lease-renewals` only**. GET desired
state returns `DesiredStateEnvelope`; it does not embed a serving lease. The
TEST ONLY proxy forwards the signed desired g+1 and observation reports, while
returning prompt 503s for successful renewal responses matching that generation.
This permits real Steps and reports to continue. No signed state, lease or
report is forged, replayed or synthesized. Release forwards the next fresh
central-issued lease, and the local stored body matches its digest exactly.

[The final proof](hold-fixture.md) has two members on the retained generation-7
lease, applied desired generation 8, ten withheld renewal responses, and a
**50.000298-second** central incident. The configured withholding bound is 45 s;
the next normal poll supplies the matching lease after the bound. The agent's
Step-3 report carries `alarm=true`, elapsed 9891 ms. Central emits exactly one
`node.generation_hold.escalated` audit at incident age **9.887144 s**: the
**reported alarm**, not the 30-second central-age fallback, triggered it. Further
alarm samples add no second event. The matching generation-8 report ends the
incident `generation_matched`, followed by independent serving proof.

[All joins](hold-joins.json) include node/incarnation/report IDs, old and released
lease IDs and generations/member windows, incident first/last/end timestamps,
every accepted hold sample and the audit. No product defect was demonstrated.
The exact arm/release sequence and consumer ownership are in the dated
[response](../../../RESPONSES.md) and
[export contract](../../../scenarios/harness-exports.md). Unrelated-failure
precedence, agent-stop retention/local reset, same-node later incidents and
sole-member `members_revoked` acceptance remain the consumers' rows.

## Native results, every attempt

Each first request has one SYN at member, node and origin. All three capture
points have zero drops, header-read errors and truncations; all 64 private
receipts verify. No accepted closed report appears after serving admission.
The node ruleset receives no tracing object. The historical tracer flag is
rejected by this runner. The first-selection boundary is member tunnel egress;
stream intervals use each socket write's before/after monotonic bounds.

| Attempt / timeline | First SYN transit, ms | Stream sends | Missing node ingress | Missing origin | Largest consecutive ingress gap, ms |
|---|---:|---:|---:|---:|---:|
| [ipv4-01 · e4c58b1d](e4c58b1d-505d-40b1-b899-e929f3d20846/transport-timeline.md) | 105.912113–105.972855 | 7304 | 0 | 26 | 26.112 |
| [ipv4-02 · 0b6e58f1](0b6e58f1-a47f-45d4-ae63-60adc117a790/transport-timeline.md) | 106.703229–106.727972 | 7351 | 0 | 21 | 24.729 |
| [ipv4-03 · 0a8a05f7](0a8a05f7-7c54-4863-a98e-a0e92e73e7c4/transport-timeline.md) | 3.350965–3.369257 | 7337 | 0 | 24 | 26.385 |
| [ipv4-04 · d992b04f](d992b04f-f83f-4393-b375-891cfb90cf7d/transport-timeline.md) | 104.996068–105.044803 | 7308 | 0 | 22 | 25.071 |
| [ipv6-01 · e530b86a](e530b86a-ab8a-411a-bfbf-6130355c6721/transport-timeline.md) | 104.935676–104.965715 | 7342 | 0 | 23 | 27.761 |
| [ipv6-02 · 0876ef42](0876ef42-5948-4a1f-b15f-ee83785e3771/transport-timeline.md) | 3.534703–3.559238 | 7297 | 0 | 33 | 26.463 |
| [ipv6-03 · ae1a2829](ae1a2829-7834-485d-a4df-557cf6d090d3/transport-timeline.md) | 3.585356–3.630897 | 7322 | 0 | 18 | 25.641 |
| [ipv6-04 · 1ecee8fe](1ecee8fe-69a4-4a38-8f72-aa392ee88eb1/transport-timeline.md) | 4.766312–4.812435 | 7358 | 0 | 16 | 27.642 |

IPv4: **4/4 first selections**, 29,300 stream sends, zero missing node ingress,
93 missing at origin. IPv6: **4/4 first selections**, 29,319 stream sends, zero
missing node ingress, 90 missing at origin. There are no ≥1 s ingress gaps or
stream deliveries with upper bounds above 1 s. The **183 ingress-without-origin
losses are retained**, not waived or assigned the handshake-delay hypothesis.
This campaign does not perform the consumers' full Apply/egress loss attribution.

### What the handshake joins support

- **IPv4:** initial loopback handshake responses remained on loopback. No
  loopback-source response crossed the underlay. Both ends later show ordinary
  IPv4 rekey initiation/response pairs near 120 s, with continuous node ingress.
  The historical first-selection and stream-gap shapes did not reproduce.
- **IPv6 attempt 01:** loopback establishment, IPv6 endpoint selection and later
  IPv6 rekey completed; no loopback-source underlay message or observed header
  error increment occurred.
- **IPv6 attempt 02, `0876ef42`:** [eight type-4 data messages](0876ef42-5948-4a1f-b15f-ee83785e3771/loopback-underlay.md)
  from node `eth2` to member `eth1` use source `::1`; identical opaque hashes join
  both ends and the member header-error counter increases by eight. They occur
  during asymmetric endpoint handover. Valid-source data resumes on the same
  receiver index/counter sequence without an intervening handshake. The first
  request and all member→node stream sends arrive. This is a data-source event,
  not evidence of a handshake-retry ingress stall.
- **IPv6 attempt 03, `ae1a2829`:** [one type-2 response](ae1a2829-7834-485d-a4df-557cf6d090d3/loopback-underlay.md)
  uses source `::1` on the underlay with one observed member header error. The
  concurrent node-originated handshake had already carried data and the first
  SYN before that response arrived. The next actual member initiation is
  **15.600070 s** after its previous initiation. The sustained stream starts
  after this later handshake; no uninterrupted-forwarding claim is made for the
  preceding idle interval.
- **IPv6 attempt 04, `1ecee8fe`:** [another type-2 response](1ecee8fe-69a4-4a38-8f72-aa392ee88eb1/loopback-underlay.json)
  from `::1` reaches member packet 666 at Unix 1793547758.827656, after the first
  SYN had already arrived using the node-originated session. One header error
  is observed. No further member initiation appears in the capture; the next
  initiation is the node's normal rekey 120.406534 s after its prior initiation.
  The first request and all stream sends reach node ingress.

The nominal WireGuard five-second retry remains a plausible timing hypothesis
for the historical 5.671 s and 5.0975 s observations, not an attribution from
these new rows. The upstream [timer constants](https://git.zx2c4.com/wireguard-linux/tree/drivers/net/wireguard/messages.h)
set REKEY_TIMEOUT to 5 s; that constant alone does not identify which timer,
handshake direction or available session governed a captured packet. The fresh
rows show why a loopback-source response by itself is insufficient to infer a
five-second delivery stall.

Every run retains both-end type-1/type-2 message hashes and sender/receiver
indexes, per-hop source flags/addresses, NetBird peer status/endpoint history,
independent WireGuard latest-handshake reads, complete member route tables,
per-send CSV intervals, IP error-counter windows and producer/central histories.
Known loopback addresses are named; other addresses remain stable tokens.
Structural headers and bounded counters do not authenticate a handshake or
identify the exact kernel discard/source-selection callsite.

## Limits and hand-off

Fresh reduced-population stacks do not reconstruct members 15's full preceding
history. Stream readiness is checked independently and can leave an idle window
between the first request and continuous traffic; the exact timestamps are
retained. First selection keeps the original members two-second wait after
route selection. There is no successful traffic warmup, reconnect or repair.
A small declared sample does not estimate reliability.

Historical F15-F11-01 (five SYNs, zero node/origin receipts), F15-AT02 (249 missing
node ingresses, 5.0975 s neighbouring-ingress gap without original send times),
and F14-01 retain their outcomes and unproven causes. Members 16's original
send-time work remains necessary. F11-01 repair direction stays with Ted; no
transport, NAT, kernel, member validation, `node/agent/probe.go` or NetBird pin
changed. No arm64 live run is claimed.

See [reproduction commands](how-to-run.md) and the final [gate record](gates.md).
Suite admission was checked before and after all eleven starts. Waiting for two
other suites consumed 20.042 s and 40.067 s on the first two hold starts; total
admission-check wall time was **60.531 s**. Stacks ran serially and teardown
removed the owned inventory. All network observation stayed in owned namespaces.
