# Pinned AdGuard Home response-shape evidence — 2026-10-31

TEST ONLY synthetic DNS data. No member query history, production credentials,
private endpoints or traffic to third-party DNS. This run uses the exact AdGuard
Home and CoreDNS pins in [BACKEND.md §1.1](../../BACKEND.md#11-container-image-pin).
The expected per-type responses and lane answers are in
[BACKEND.md §2.1](../../BACKEND.md#211-expected-responses--measured-2026-10-31).

Base commit: `a9fd962bf204ba83c100010f1cc7708ba6fad392`, plus this change's
`tests/netlab/dns/response_shape_test.go` and `testdata/response-shape` fixtures.
The image identifiers, measured version, platform and capture times are in
`images.json`. Live execution is linux/amd64; an arm64 cross-build is not a
claim of arm64 resolver execution.

The test drives `dnspublish.Compile` and `node/dns.New(...).Apply` for the empty
positive-control document, the blocking document at TTL 60, and the blocking
document at TTL 5. Apply requires configuration/rule read-back and its normal
A probes before returning verified evidence. A transparent local HTTP recorder
captures the actual write bodies, not a hand-maintained substitute. For the
`default` and `null_ip` comparisons, the harness copies the captured production
`dns_config` body, changes only `blocking_mode`, posts it and verifies read-back.
The rules remain exactly those the applier wrote. No product configuration or
contract is changed by these comparison modes.

Files:

- `AdGuardHome.yaml`: exact initial resolver configuration. Test admin auth is
  disabled, query log/statistics are disabled, and `refuse_any` is true. The
  resolver is unprivileged on an **internal Docker bridge with no published
  ports**. The Linux host probes that owned bridge; there are no host network
  commands. Deployment credentials/listeners/client ACLs are intentionally
  outside this DNS-response experiment.
- `Corefile`, `db.example.com`: exact non-recursing authoritative upstream.
  `blocked` and `child.wild` are aliases to a controlled target with real
  A/AAAA/SVCB/HTTPS/TXT/MX/NS records. `chain → hop → blocked → target`
  exercises both completed answer-chain filtering and CNAME-only traversal.
- `documents.json`: JSON contents of all three compiler outputs, pretty-printed
  as an array. Re-canonicalize an individual document before hashing; this
  presentation is not the original wire byte sequence.
- `requests.json`: actual apply and comparison write JSON, pretty-printed.
  Includes the complete DNS config and metadata-bearing rule arrays. The
  direct cache-clear requests before each probe are `{}` and are not repeated
  in this file. No authorization headers are recorded.
- `readback.json`: `/control/dns_info` after nxdomain/60, default/60, null_ip/60,
  nxdomain/5, in that order.
- `probes.jsonl`: **432** raw request/response pairs in hex, each with mode,
  configured TTL, transport, name, type, RCODE, answer types/TTLs/targets/IPs,
  authority types, SOA TTL/MINIMUM and AA/AD. Answer count is the length of
  `answer_types`; `TypeALL` means ANY. TCP length framing is omitted from the
  retained DNS message bytes. Every query sets RD=1, AD=1, without EDNS.
- `run.txt`: foreground conformance output. `SHA256SUMS` covers the captured
  artifacts and this procedure.

Each phase queries four names × nine QTYPEs × UDP/TCP = 72 exchanges. There are
two positive-control phases (authority directly, then unfiltered resolver),
three TTL-60 blocking-mode phases, and one nxdomain/5 phase. Positive controls
require the requested RR type, not merely a CNAME or an empty successful reply.
All synthetic names have upstream data; an absent RR cannot pass as filtering.
The resolver cache is cleared before each query, so the table records fresh
responses, not aged client-cache TTLs. For the first-hop-only CNAME response,
AA and authority details belong to the upstream, and the follow-on H query
must be blocked. Raw bytes retain all sections for independent inspection.

Reproduce on a Linux Docker host from the repository root:

```sh
mkdir -p .agent/tmp
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp"
export CGO_ENABLED=0 GOTOOLCHAIN=local
export FVPN_LAB_PREFIX=fvpn-w3-dns-publish
# Optional: omit this variable to run assertions without exporting evidence.
export FVPN_DNS_SHAPE_EVIDENCE="$PWD/.agent/tmp/response-shape"
make -f mk/dns.mk dns-response-shape
```

The test checks `pgrep -af 'gates.sh|mk/at-'` before starting a resolver and
waits while two or more integration launchers are active. Its only containers
are `fvpn-w3-dns-publish-shape-{resolver1,upstream}` and its network is
`fvpn-w3-dns-publish-shape-net` (`192.0.2.0/24`). An overlapping Docker subnet
is a setup error; do not alter another lane's network. `t.Cleanup` removes
the containers, network and disk scratch even on assertion failure. There are
no volumes and no shell cleanup traps. After an externally killed test, remove
only these exact owned objects, then check owned inventory before rerunning.

`dns-lab` includes this conformance test and the existing two-resolver
feed/allow/withdraw scenario. DNS CI runs `dns-response-shape` on pull requests.
A pin change must preserve the expected shape or deliberately revise the
table and assertion with new measured evidence. A changed RCODE, answer,
SOA TTL/MINIMUM, CNAME target or AA/AD fails the test; it does not regenerate
an accepted golden automatically.

The docs were the wrong side of the disagreement. The observed NODATA is not
caused by the fixture's five-second TTL. That TTL is explicitly configured in
devstack follow-up 16 and members follow-up 8, and the same shape appears at
60 seconds here. The negative SOA TTL follows that setting in both runs.
This experiment does not rerun the native member platform or whole AT20–AT22
campaign, measure real device caches, or establish new publication semantics.
