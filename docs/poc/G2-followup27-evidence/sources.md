# Follow-up 27 source inventory (before implementation)

Task-start main: `b943941157ab6aeba6b04c79377de686c22bfbb4`.
D-098 first: 163 follow-up 25 and 165 follow-up 26 payloads match their final
manifests. All new artifacts stay in this directory.

`node/enforce/model.go:Model.denyV4` uses explicit non-nil `DeniedV4`, or the
16 compiled `SpecDeniedV4()` prefixes for nil, then unions `ExtraDeniedV4`.
The helper masks, deduplicates and sorts; the kernel additionally merges
coverage. Empty non-nil DeniedV4 is an error. `Render` validates the IPv4
prefixes, installs this union in denied_v4, and installs HostV4 plus the
Datapath.SNAT /32 in host_v4. SNAT is included even when HostV4 omits it.
Both sets use interval/auto-merge. Existing sets are flushed and filled;
`State` suppresses recreation, but supplies no desired contents.

`RenderBaseline` with a probe installs the same two sets from the same
sources. Probe DNS/destinations do not add denial exemptions. Probe lifetime
retention affects only probe_v4/probe_dst_v4. Without a probe, a fresh baseline
creates neither denial set. A previously installed set currently survives
that transaction: this is an additional historical-state source on the bare
baseline path. The requested absence contract requires deleting both after
replacing the chains, in that same atomic baseline transaction. Guard and
quarantine transactions do not populate these sets; rollback uses baseline.

`node/agent/engine.go:input` parses ExtraDeniedV4, HostV4 and TestExemptV4 from
the captured startup Config. `decide.go:buildModel` uses SpecDeniedV4 minus
exact masked TestExemptV4 entries only in TestMode. `probeexc.go:baselineProbeModel`
uses the same config/exemption rule. Config.DatapathValue supplies SNAT.
No dynamic on-link/address discovery populates these fields despite the
model's comment describing their intended content. Lab/other library callers
can supply explicit Model prefixes; the verifier must honor the supplied model.

No normal production source changes between Apply and a later live read:
`serving.go:verifyCurrentServing` uses saved `appliedModel`; config is captured
at Start, not watched, and resolver reload does not update these fields.
`SpecDeniedV4` constructs a fresh slice from compiled constants. Kernel sets
can change out of band, and startup can supply a different candidate without
Apply. The check will reject any coverage mismatch, including additional
restrictions. It will not silently adopt a new policy or infer host addresses.
No mutable global cache of expected ranges will be introduced.

Agent-core 36 separately compares retained and candidate prefixes. That does
not cover an unrecorded successful kernel Apply followed by configuration
rollback: both records can agree while the kernel differs. An owned agent
fixture will exercise actual Boot against that state and require baseline.

Resolver/probe element parsers, all existing reason values, rule ordering,
RELATED ICMP source guards and deadlines stay unchanged. Existing isolation
reason identifies set drift. Real arm64 stays unmeasured.
