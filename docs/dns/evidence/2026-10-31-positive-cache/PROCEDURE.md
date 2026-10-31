# F10-03 positive-cache measurement — 2026-10-31

TEST ONLY synthetic authority and traffic; no real DNS data, credentials or member
history. See [BACKEND §2.1.5](../../BACKEND.md#215-f10-03--positive-cache-and-ttl-measured-2026-10-31)
for the decision, inherited-setting caveat and acceptance attribution rule.

The pinned AdGuard Home v0.107.59 image is the exact §1.1 digest. Its image ID,
platform, measurement source commit and SHA-256 of the actual test/adapter source
are in `provenance.json`. Both resolver processes run that image unprivileged on
an internal Docker bridge, without published ports or public recursion. The
owned authority is a static Go binary built from
`tests/netlab/dns/cmd/cache-authority` and mounted into the same pinned image;
it is not a different recursive resolver or an external DNS service.

The real `node/dns.New(...).Apply` publishes one compiled document to both targets.
`requests.json` contains the complete captured mutation bodies, without headers,
credentials or host paths. `document.json` is the compiled document. `readback.json`
contains GET `/control/dns_info` and the persisted `dns` YAML map for each resolver.
The starting `AdGuardHome.yaml` deliberately omits cache, upstream mode and timeout
settings. Thus the running 4 MiB cache, TTL overrides 0/0, optimistic=false,
load_balance, fastest_timeout=1s and upstream_timeout=10s are measured defaults.
The auth-free admin listener and documentation-range bridge are TEST ONLY fixture
choices; production management-plane authentication/access controls remain required.
The configured blocked TTL 60 is the existing starting value, not a new production
decision. No blocks are needed for the positive-cache experiment.

Apply's normal initial cache clear is captured; no cache clear, policy publication,
restart, setting change or resolver selection by a native stub occurs afterward.
Each case uses a distinct closed synthetic label. Both resolvers share policy and
authority, but have independent caches. Direct connected UDP sockets identify the
response source and validate ID/question/QR/no-TC. `resolver_events` here are those
client-side exchange receipts, not an assertion that the native metadata relays ran.
The controlled fallback case explicitly tries primary for 2500 ms then secondary.
It demonstrates independent cache semantics, not platform qualification.

The authority accepts only the closed cases plus the adapter's smoke name. A answers
have TTL 10, class `other_ipv4` (198.51.100.10) before a phase change and
`changed_ipv4` (203.0.113.20) afterward. Both are documentation addresses. The latter
corresponds to the members fixture's changed `private_ipv4` outcome; no connection
to a private destination or rebinding enforcement test is made here.

`traces.json` preserves UTC query bounds, selected responder, RCODE, answer class,
TTL remaining, query duration, correlated authority events, and a separately queried
SOA serial/time after each response. The SOA query bypasses the resolver, is excluded
from upstream counts, and does not warm or invalidate the positive cache. It answers
1001/1002 for the selected case's authority phase. Positive A records contain no SOA.
`authority-events.json` retains the complete cross-query history, including starts,
completions, source role, outcome and the authority phase sampled when a request
arrived (`snapshot_serial`, **not a serial in an A response**). Only answered A events
have answer classes/TTLs; pending/dropped requests do not claim a response. No packet,
full query name, answer address or credential is logged in either metadata artifact.

A trace's authority events cover requests **started within** its query bounds, so
pending old work may complete later. The whole-run history must be joined for the
in-flight finding. Absent events support only an inferred hit. Privacy read-back
checks leave AdGuard query logs/statistics disabled. All Docker resources and disk
test directories are removed by `t.Cleanup`; the export survives separately.

## Procedure and regression assertions

```sh
mkdir -p .agent/tmp
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp"
export CGO_ENABLED=0 GOTOOLCHAIN=local
export FVPN_LAB_PREFIX=fvpn-w3-dns-publish
export FVPN_DNS_CACHE_EVIDENCE="$PWD/.agent/tmp/positive-cache"
make -f mk/dns.mk dns-positive-cache
```

Omit `FVPN_DNS_CACHE_EVIDENCE` for assertions only. Ordinary gates do not overwrite
this archive. Linux Docker bridge access is required; the fixed documentation subnet
must be free. Before **each** resolver launch, the harness checks
`pgrep -af 'gates.sh|mk/at-'` and waits while at least two integration launchers run.
The source uses no shell cleanup traps. DNS CI runs this target on pull requests;
`dns-lab` also includes it. The second resolver is hosted by this fixture, so no
separate manual two-resolver procedure is necessary.

1. Warm primary with TTL 10, flip only the authority, query primary every second
   through second 9, and require old class with decreasing TTL and no upstream
   request. At second 1, query secondary and require changed class/TTL 10 plus a
   fresh authority exchange. At 10.1 seconds require primary's miss/new answer.
2. Prime a separate name late in a wall-clock second; after its whole-second expiry
   require a fresh fetch inside the stopwatch TTL, between 9 and 10 seconds.
3. Delay a first primary reply by 2500 ms. Require one exchange, cached old hits
   through seconds 1–9 after receipt, then a fresh answer after TTL.
4. Drop the first primary packet. Require two attempts and about 10 seconds before
   its positive reply; cache it. Drop both attempts on a separate name and require
   SERVFAIL at about 20 seconds, with no positive response.
5. Delay the first reply by 11 seconds and flip phase while it is pending. The
   retry fetches new at 10 seconds; after the late old reply, require new on a hit.
6. Model a client's 2500-ms primary deadline with a dropped first upstream packet;
   secondary answers old. Flip authority and require primary fresh/new while
   secondary still returns cached old. Drain the outstanding primary retry.
7. Delay the first primary reply by 3500 ms, exceeding the client deadline but not
   AdGuard's. Secondary answers old after the client abandons primary. Flip phase,
   query primary again and require fresh/new; after the old outstanding reply
   completes require cached old. The full authority history must prove the old
   request began before and completed after the fresh primary reply. This guards
   a known inherent limitation and forces review if a future pin changes it.

The assertions are explicit; evidence does not auto-update expected answers. This
run is Linux/amd64. The test and authority cross-compile for arm64; no native member
path, cache-pressure campaign, real arm64 runtime, or historical-cause recovery is
claimed. The 4 KiB acceptance fixture and native NetBird fallback timing remain
separate variables. Source/build changes require a new archive, not editing this one.

## Every measured query

All timestamps below are UTC on 2026-10-31. JSON retains nanoseconds and completion
bounds. `Δs` is from the first completed response for that case; initial rows use 0.
`requests/replies` counts upstream A requests begun in the query window and those
already answered by return (not the separately sampled SOA). `0/0` is an inferred
hit; it does not exclude completion of an older request. Milliseconds are truncated
elapsed durations; 0 means less than one millisecond. Every SOA serial was sampled
after its corresponding response.

| Case / step | Query start UTC | Δs | Responder | Class | TTL | ms | Upstream requests/replies | Authority serial after |
|---|---|---:|---|---|---:|---:|---:|---:|
| steady / initial | 18:15:48.700484649 | 0.0000 | primary | other_ipv4 | 10 | 0 | 1/1 | 1001 |
| boundary / initial | 18:15:48.702301695 | 0.0000 | primary | other_ipv4 | 10 | 0 | 1/1 | 1001 |
| steady / inside_1s | 18:15:49.701714759 | 1.0007 | primary | other_ipv4 | 9 | 0 | 0/0 | 1002 |
| steady / other_resolver_fresh_fetch | 18:15:49.70259322 | 1.0016 | secondary | changed_ipv4 | 10 | 0 | 1/1 | 1002 |
| steady / inside_2s | 18:15:50.702301557 | 2.0013 | primary | other_ipv4 | 8 | 0 | 0/0 | 1002 |
| steady / inside_3s | 18:15:51.701482863 | 3.0006 | primary | other_ipv4 | 7 | 0 | 0/0 | 1002 |
| steady / inside_4s | 18:15:52.701882023 | 4.0009 | primary | other_ipv4 | 6 | 0 | 0/0 | 1002 |
| steady / inside_5s | 18:15:53.701695714 | 5.0007 | primary | other_ipv4 | 5 | 0 | 0/0 | 1002 |
| steady / inside_6s | 18:15:54.701733514 | 6.0008 | primary | other_ipv4 | 4 | 0 | 0/0 | 1002 |
| steady / inside_7s | 18:15:55.701660044 | 7.0008 | primary | other_ipv4 | 3 | 0 | 0/0 | 1002 |
| steady / inside_8s | 18:15:56.701918779 | 8.0010 | primary | other_ipv4 | 2 | 0 | 0/0 | 1002 |
| steady / inside_9s | 18:15:57.702008177 | 9.0011 | primary | other_ipv4 | 1 | 0 | 0/0 | 1002 |
| boundary / whole_second_expiry | 18:15:58.050508536 | 9.3484 | primary | changed_ipv4 | 10 | 0 | 1/1 | 1002 |
| steady / after_ttl | 18:15:58.801514772 | 10.1008 | primary | changed_ipv4 | 10 | 0 | 1/1 | 1002 |
| slow / first_fetch_delayed_2500ms | 18:15:58.802799235 | 0.0000 | primary | other_ipv4 | 10 | 2502 | 1/1 | 1001 |
| slow / inside_1s | 18:16:02.305653309 | 1.0011 | primary | other_ipv4 | 9 | 0 | 0/0 | 1002 |
| slow / inside_2s | 18:16:03.30591329 | 2.0011 | primary | other_ipv4 | 8 | 0 | 0/0 | 1002 |
| slow / inside_3s | 18:16:04.305719278 | 3.0009 | primary | other_ipv4 | 7 | 0 | 0/0 | 1002 |
| slow / inside_4s | 18:16:05.305715372 | 4.0009 | primary | other_ipv4 | 6 | 0 | 0/0 | 1002 |
| slow / inside_5s | 18:16:06.30571323 | 5.0009 | primary | other_ipv4 | 5 | 0 | 0/0 | 1002 |
| slow / inside_6s | 18:16:07.30573072 | 6.0009 | primary | other_ipv4 | 4 | 0 | 0/0 | 1002 |
| slow / inside_7s | 18:16:08.305757483 | 7.0010 | primary | other_ipv4 | 3 | 0 | 0/0 | 1002 |
| slow / inside_8s | 18:16:09.30565689 | 8.0009 | primary | other_ipv4 | 2 | 0 | 0/0 | 1002 |
| slow / inside_9s | 18:16:10.305908194 | 9.0013 | primary | other_ipv4 | 1 | 0 | 0/0 | 1002 |
| slow / after_ttl | 18:16:11.405689342 | 10.1014 | primary | changed_ipv4 | 10 | 0 | 1/1 | 1002 |
| retry / first_packet_dropped | 18:16:11.406928686 | 0.0000 | primary | other_ipv4 | 10 | 10001 | 2/1 | 1001 |
| retry / after_authority_change | 18:16:21.409335699 | 0.0011 | primary | other_ipv4 | 10 | 0 | 0/0 | 1002 |
| late / timeout_then_retry | 18:16:21.409837614 | 0.0000 | primary | changed_ipv4 | 10 | 10001 | 2/1 | 1002 |
| late / after_discarded_late_old_reply | 18:16:32.710779343 | 1.3000 | primary | changed_ipv4 | 9 | 0 | 0/0 | 1002 |
| failed / both_upstream_attempts_dropped | 18:16:32.711853907 | 0.0000 | primary | SERVFAIL | — | 20001 | 2/0 | 1001 |
| fallback / client_2500ms_then_secondary | 18:16:52.714026745 | 0.0000 | secondary | other_ipv4 | 10 | 2501 | 2/1 | 1001 |
| fallback / primary_fresh_fetch_inside_secondary_ttl | 18:16:55.21630723 | 0.0013 | primary | changed_ipv4 | 10 | 0 | 1/1 | 1002 |
| fallback / secondary_keeps_old_entry | 18:16:55.217511212 | 0.0023 | secondary | other_ipv4 | 10 | 0 | 0/0 | 1002 |
| inflight / client_timeout_then_secondary | 18:17:03.015719191 | 0.0000 | secondary | other_ipv4 | 10 | 2501 | 2/1 | 1001 |
| inflight / primary_while_first_fetch_outstanding | 18:17:05.518300702 | 0.0010 | primary | changed_ipv4 | 10 | 0 | 1/1 | 1002 |
| inflight / primary_after_first_fetch_completed | 18:17:06.816044583 | 1.2990 | primary | other_ipv4 | 10 | 0 | 0/0 | 1002 |

The decisive F9-INFLIGHT ordering is:

| Event | UTC timestamp |
|---|---|
| Original primary upstream request snapshots old (1001) | 2026-10-31T18:17:03.016270874Z |
| New primary reply completes, TTL 10 | 2026-10-31T18:17:05.518637949Z |
| Original old upstream reply completes | 2026-10-31T18:17:06.517069987Z |
| Primary query receives old, no new upstream request | 2026-10-31T18:17:06.816611758Z |

The last answer changes class only 1.297974 seconds after the
new primary response. This is not optimistic/stale serving after TTL expiration;
an earlier request populates the cache later. By contrast, the 11-second reply
arrives after its upstream attempt has timed out and does not replace the retry.
F10-03's old → new native observation has no such retained history, so its cause
and failed verdict remain unchanged.

`SHA256SUMS` covers every artifact in this directory except itself. `run.txt` is
the real foreground conformance output (36 traces, no skipped test).
