# Follow-up 24: established download destination denial

TEST ONLY evidence; the synthetic Internet uses documentation addresses inside
an isolated lab. No option (b) intersection-serving implementation is added.
D-141's option (c), tc digest and timeout decisions are unchanged.

## Inputs and lifecycle

`node/enforce/model.go:160` defines `DeniedV4` as an optional replacement for
the spec list, `ExtraDeniedV4` as an unconditional union, and `HostV4` as the
node's own addresses. `Model.denyV4` (`model.go:238`) computes that union.
`Render` also adds the SNAT address to the separate host set.

Production gets `ExtraDeniedV4` and `HostV4` from `agent.Config`
(`node/agent/config.go:53`), loaded by the command's `open`
(`node/agent/cmd/fvpn-agent/main.go:125`). `Engine.input`
(`node/agent/engine.go:759`) parses those stored config slices on each decision;
`decide.go:699–724` passes them to the enforcement model. The override there is
only the explicitly configured TEST ONLY exemption. The desired-state schema
(`contracts/schemas/desired-state-envelope.schema.json`, the signed-object definition and DNS
properties) has resolver addresses and DNS policy references, but no IP-layer
destination-denial field.

No production writer was found that grows this set in a running agent. Config
file edits do not mutate the captured `Config`; SIGHUP calls `reloadResolvers`
for resolver credentials (`engine.go:465`, `dnsconfig.go:199`). There is no
implemented live topology-discovery writer for `ExtraDeniedV4`. A new config
read on restart, or a changed compiled spec list on upgrade, can feed a larger
set to a subsequent full Apply. This is not a claim that restart adoption
immediately installs config differences: adoption verifies retained state and
has its own agent-owned lifecycle (`node/agent/restart.go:15`).

Conntrack can survive a same-kernel process restart or upgrade. Baseline/Apply
replace owned nft rules, not the kernel's conntrack table. `planRevokes`
(`node/agent/revocation.go:47`) queues removed/reassigned member identities;
a remote-set change with stable identities does not queue them. Apply calls
`deleteConntrack` only for `Model.Revoke`; `apply.go:413` performs targeted
source/destination deletion. A reboot or namespace replacement is different;
no persistence across those is claimed. The H-1 lab deliberately retains the
entry across a full Apply and proves that correctness cannot depend on its
deletion. Set sourcing and restart behavior are not changed here.

## Enforcement and counters

The download order is membership/expiry, established-related gate, exact member
and probe resolver replies (TCP/UDP source port 53), current source denials,
then generic probe/member accepts. Probe baseline uses the same source-denial
helper and explicit conntrack checks on its DNS replies. DNS-disabled baseline
probes retain their existing resolver-set scope; they gain deny/host guards.

Host sources and non-53 resolver sources receive the same protection as denied
sources: otherwise a newly protected host address or designated resolver has
the identical established-return loophole. Exact DNS replies keep the upload
exception's precedence even when the resolver is inside a deny/host prefix.
All rules remain bound to the member/uplink pair; control input/output is
unchanged.

New named counter **`ctdenyd`** counts remote-source drops for `denied_v4`,
`host_v4`, and the resolver non-DNS remainder. It contains no remote address,
query, payload, or selector. It belongs to `diagnostic_v1`, never device usage.
Existing upload-only `ctdeny`, `cthost`, and `ctbadns` meanings are preserved.
The new object is reused across renders like existing counters.

Owned enumeration updates: `node/enforce/counter.go` (`diagnosticNamespace`),
`render_nft.go` (`diagnosticCounters` and probe-baseline counter creation), the
serving/probe `.nft` goldens, verifier guard/counter expectations, and positive
JSON fixtures. Outside-owned enumeration/coverage hand-offs are in the report:
usage AT, devstack counterread, and agent counter consumers. Current counterread
selects `u_`/`d_` dynamically and has no closed diagnostic-name whitelist to
silently extend; its owner still needs a new-counter exclusion fixture.

The verifier rejects rulesets without the new counter/guards, guards after
an accept, wrong direction, and extra narrowing predicates. The generic probe
accept cannot take the DNS exception's place. An old serving or probe-enabled baseline ruleset without
the guards therefore fails the new verification path. The ordinary deny-all
baseline has no return accept and needs no additional guard.

## Specification and limits

This changes the previously permissive return behavior to enforce §8.1/§8.2
isolation on existing flows. It supports ISO01/ISO02/ISO04, §16.3 and §19 AT13;
it is not a new DNS blocking rule or IP-rule product API. §12.5's discussion of
DNS changes leaving established HTTPS sessions alive concerns DNS policy and
does not exempt `denied_v4`. Exact designated resolver DNS still works; blocked
returns do not enter §10.1 usage accounting (MTR05/SEC02).

**Spec question, unresolved:** a source guard also rejects a RELATED ICMP error
whose outer source lies in denied/host/resolver material while the quoted
connection's remote endpoint remains allowed. Existing F8-02 PMTU evidence
uses an allowed origin source; it does not establish a private-router exception.
The requested fail-closed source rule is retained. Any narrower allowance for
such errors needs a separately specified rule and evidence; this follow-up
does not invent it or claim private-router PMTU compatibility.

## Evidence

See `G2-followup24-evidence/`. Before/after render and kernel proofs retain the
review names and setup. The two D-141 design-only tests and their unused
`allowedElementTimeout` helper were removed; `forwardRules` remains used.
`proof-source-audit.json` proves the counter-preservation review test and the
entire established-member-revocation file are byte-identical to their sources.

The first probe extension lacked the stream helper's TCP/9001 seed tuple;
`kernel-after.txt` retains that fixture failure. Both H-1 and the unchanged
member-revocation proof passed in the same attempt. The complete lab initially
exposed an existing reload-fixture assumption: a removed device's reading is
in `Finals`, not `Samples`; the added sibling control made its prior count
nonzero. The test now checks the final reading with the same monotonicity
assertion. Neither failed attempt is reclassified as passing.

## Same-host comparison

D-133/D-138 same-host comparison: A is immutable main `335047a2b650db1d64bb4e0c59fc042fad74da51`; B is `a71313990140409dee70934fc5cfeb96f3166fab` (production source unchanged since `d11f809e`). All timing quadruples below are nearest-rank p50 / p90 / p99 / max in milliseconds. Ordinary Apply has lifecycle sampling disabled; reload tracing is a separate population. The one extra warm-shape trace per attempt is excluded from timings.

| Leg / qualified attempt | Fresh rules / bytes | Guarded rules / bytes | Ordinary applies / failed | Guard ms | Install ms | Verify ms | Reload result |
|---|---:|---:|---:|---|---|---|---|
| A1 / A1-attempt5 | 34 / 9053 | 36 / 6603 | 60 / 0 | 30.403 / 40.191 / 46.649 / 46.649 | 14.944 / 20.979 / 23.971 / 23.971 | 23.189 / 25.061 / 27.886 / 27.886 | PASS |
| B1 / B1-attempt2 | 37 / 9404 | 39 / 6924 | 60 / 0 | 31.337 / 40.028 / 43.954 / 43.954 | 15.553 / 21.595 / 25.939 / 25.939 | 23.051 / 24.452 / 26.141 / 26.141 | PASS |
| A2 / A2-attempt2 | 34 / 9053 | 36 / 6603 | 60 / 0 | 28.868 / 38.538 / 47.178 / 47.178 | 15.111 / 20.837 / 27.264 / 27.264 | 21.988 / 23.999 / 25.716 / 25.716 | FAIL (D-120 missing successor) |
| B2 / B2-attempt4 | 37 / 9404 | 39 / 6924 | 60 / 0 | 31.580 / 43.669 / 51.586 / 51.586 | 16.419 / 23.817 / 31.490 / 31.490 | 25.563 / 27.847 / 28.977 / 28.977 | PASS |

| Population / arm | Applies / completed guards / failed Applies | Guard ms | Install ms | Verify ms |
|---|---:|---|---|---|
| Ordinary / A | 120 / 120 / 0 | 29.902 / 38.935 / 46.649 / 47.178 | 15.097 / 20.979 / 24.705 / 27.264 | 22.476 / 24.551 / 26.565 / 27.886 |
| Ordinary / B | 120 / 120 / 0 | 31.508 / 42.461 / 47.867 / 51.586 | 16.294 / 23.302 / 30.527 / 31.490 | 24.205 / 27.126 / 28.630 / 28.977 |
| Reload trace / A | 202 / 202 / 0 | 29.107 / 36.201 / 42.793 / 45.881 | 14.977 / 19.911 / 27.832 / 29.092 | 29.104 / 31.554 / 32.964 / 33.844 |
| Reload trace / B | 202 / 202 / 0 | 30.583 / 40.468 / 44.141 / 46.995 | 15.253 / 20.499 / 24.859 / 27.251 | 28.016 / 30.715 / 33.545 / 34.043 |

Qualified host receipts (each max/min covers all six measured 5-second intervals; the since-boot vmstat aggregate is retained separately):

| Leg / checkpoint | max r / min idle % | load before → after | D-state threads before → after | matching processes before / after |
|---|---:|---|---:|---:|
| A1 / prestart-2 | 2 / 86 | 10.71/10.01/10.42 → 11.28/10.22/10.48 | 12 → 11 | 0 / 0 |
| A1 / ready-1 | 3 / 86 | 11.26/10.23/10.48 → 10.49/10.15/10.44 | 7 → 9 | 0 / 0 |
| A1 / postflight | 3 / 81 | 10.36/10.15/10.43 → 10.10/10.12/10.41 | 5 → 9 | 0 / 0 |
| B1 / prestart-1 | 3 / 84 | 10.04/9.75/10.08 → 9.03/9.53/10.00 | 8 → 7 | 0 / 0 |
| B1 / ready-1 | 1 / 82 | 9.26/9.57/10.01 → 9.70/9.65/10.02 | 9 → 10 | 0 / 0 |
| B1 / postflight | 2 / 82 | 10.88/9.95/10.11 → 11.69/10.24/10.20 | 5 → 12 | 0 / 0 |
| A2 / prestart-2 | 3 / 84 | 9.67/10.16/10.24 → 9.80/10.15/10.23 | 11 → 11 | 0 / 0 |
| A2 / ready-2 | 1 / 85 | 10.31/10.33/10.30 → 10.25/10.32/10.29 | 10 → 10 | 0 / 0 |
| A2 / postflight | 0 / 86 | 9.96/10.24/10.27 → 9.63/10.14/10.23 | 10 → 9 | 0 / 0 |
| B2 / prestart-1 | 3 / 87 | 5.96/8.76/9.96 → 5.58/8.40/9.80 | 4 → 4 | 0 / 0 |
| B2 / ready-1 | 3 / 86 | 5.53/8.34/9.77 → 5.23/7.99/9.61 | 4 → 1 | 0 / 0 |
| B2 / postflight | 1 / 93 | 6.04/7.91/9.53 → 5.88/7.70/9.41 | 4 → 9 | 0 / 0 |

Comparison result: **complete; qualified functional failure**. Qualified functional failures: A2-attempt2. 9 completed attempts remain excluded for failed postflight quiet checks. Every rejection, including prestart/ready polls, is retained with load, vmstat and D-state data. No two-leg envelope is used. All qualified observations are pooled per arm; one window cannot establish a repeatable causal timing shift. The 100 ms bound/reserve and tc rate/ceil settings are unchanged.

The A2 reload failure is the same evidence condition seen in `gate2-replay`: the final lost datagram has no recorded successor. A2 generation 1087 / sequence 57 failed D-120 clause (iii); every Apply succeeded. The failed run remains failed, and is not recast as an H-1 regression or an AT acceptance pass.

Harness corrections are explicit in `comparison/qualification-correction.json` and `analysis-provenance.json`: the initial admission version conservatively tested the since-boot row and admitted no leg; the resumed version tests current intervals, matching the existing D-133 harness. The original 90-minute clock was preserved. Postprocessing corrected trace field names and changed tc text filename suffixes only; exact executing harnesses and measured bytes are retained.

## Final validation

The required lane lint/unit/integration/amd64+arm64 build gate passed. The final
privileged full suite passed (`gate2-completion`: 236.514 seconds including
setup/cleanup), with H-1, both probe modes, D-120, and the unchanged member
revocation test green. Both unchanged consumer commands, evidence hygiene,
handover, root go-lint, vet, tracked Go formatting and diff check passed.
`gate*-completion.{json,txt}` retain exact commands, exits, timings and output.

The comparison completed in 4,432.181 seconds (73.87 minutes), with 85 host
checks. All 240 ordinary and 404 traced Apply calls in the qualified timing
populations succeeded. Both B reload scenarios passed; unchanged main A2's
D-120 missing-successor failure remains failed. This is one comparison window,
so no causal timing shift is attributed under D-138. Earlier failed full-suite
attempts also remain retained and are not reclassified by the final pass.

D-098 final check: all 109 follow-up 23 files still match their retained lengths
and SHA-256 pins, with complete manifest coverage and no diff from task-start
main. No prior evidence path was written. `cleanup.json` records zero owned
Docker containers, images, networks or volumes. Consumer enumeration coverage
and the RELATED ICMP question remain four dated owner hand-offs in the report.

Raw text presentation: the staged diff check exposed trailing whitespace in command output. Exact original public bytes for 83 affected files are retained as `.raw.gz`; readable copies only trim trailing whitespace and terminal blank lines. `text-normalization.json` maps raw/readable hashes. Earlier log digests refer to decompressed originals. Measurements, test verdicts and failed attempts are unchanged. The draft pre-normalization retention manifest is retained as a compressed artifact; the final manifest covers the delivered tree.

Shared main advanced to `aa7f306c008d8021b3921d083b70e19db7c8f865` while this lane ran. The merge-base and frozen comparison A remain `335047a2b650db1d64bb4e0c59fc042fad74da51`; validation is for this branch. No rebase or merge was performed. The final three-dot scope check contains only owned paths.
