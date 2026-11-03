# Revised C1 eligibility, transactions and prerequisites

[Entry point](revision.md). This is replacement design text, not implemented
commands. Retain all deadline, identity, isolation, metering, failure-reporting
and compatibility conditions from [C1](../G2-followup31-design/candidates.md).

## H-2: eligibility and shaping

An otherwise eligible device remains eligible when its member's class ID is
unchanged, each directional ceiling does not tighten, and only its guaranteed
fair-share rate changes because another member joins or leaves. A guarantee
is scheduler allocation, not an authorization lifetime or identity. Require
valid candidate rates (`0 < rate <= ceil`), signed/current candidate authority,
unchanged parent and host-reserve configuration, stable root/leaf attachment
and the old/new identity and deadline predicates. Global policy/path/meter/probe
changes still require unconditional fallback. A lower ceiling or changed class
makes the affected member ineligible; it must never inherit the old gate merely
because its address matches. Initial C1 excludes non-nil probes entirely.

Reason: [buildQoS/classRate](../../../node/agent/decide.go) computes
`fair = (parent - host) / admittedMemberCount`, `rate = min(fair, ceil)` for
each direction. Revoking an independent member can increase a sibling's rate;
adding one can reduce it. A member already capped below fair may have no rate
change. Removing one of two devices of the same member does not change this
count. r24's categorical “every revocation/addition including AT13” overstates
the code, but the old unchanged-rate rule would exclude real churn cases.

Inside the window a rate replacement can change contention, borrowing and queue
delay, and can cause congestion loss. It cannot change nft admission, class
identity, timeout, meter namespace or bypass the class/parent ceilings. A
non-tightening ceiling increase must itself be authorized by the candidate;
as with today's pre-install shaping, it can take effect before the nft commit.
There is no claim of atomic nft+tc or of keeping the previous smaller ceiling
until commit. If that stronger ceiling-transition contract is required, restrict
C1 to equal ceilings or stage that increase after the commit in a separately
reviewed protocol. Rate-only eligibility with equal ceilings does not need it.

C1.1 performs additions and replacements, preserving old classes and fq_codel
leaves. C1.2 commits the full candidate plus guard, removing old classification
references. Retire unreferenced qdiscs/classes **after that commit and before
release/verification**, as in the enforcement fix. A crash between commit and
retirement leaves a class with no nft producer; it is not authority to forward.
Queued packets already admitted before commit are a separate egress observation.
The next Apply reserves occupied minors, checks ownership, and retires residue
before exact serving verification; baseline removes safe owned roots after nft
closure. Foreign attachments require quarantine. No serving-verifier exception
for arbitrary orphans is introduced. If cleanup cannot complete, do not report
serving. This extra tc phase can lengthen the guard-present bracket and must be
included in future cost measurement, not hidden in install-only statistics.

AT13 must prove BOTH same-member device removal (A2 revoked, A1 continuous) and
independent-member removal/addition (B removed/added, A1's guaranteed rate
changes). Assert actual eligibility and unchanged class/ceilings for A1, no
revoked admission at/after the candidate cut, no member bytes into `1:fffe`,
counter continuity, and exact final tc readback. Exercise rate increases and
decreases, a capped-rate no-change case and a tightening-ceiling negative control.
Traffic for the zero-loss continuity proof must fit both rates; congestion
changes are reported separately, never relabeled as guard losses or waived.

## H-3: agent-core requirements, not an appliedModel shortcut

[decide.go](../../../node/agent/decide.go) currently folds all a member's binding
windows into one `MemberLease.DeadlineMono`; the first binding also supplies
`EffectiveServiceEnd` and `NodeLeaseEnd`. [deadline.go](../../../node/enforce/deadline.go)
DOES cap by each binding.ValidUntil. That cap cannot undo the earlier collapse.
Require independent per-device monotonic binding deadlines and independent
member/node/service/policy limits. A sibling's shorter window must not change
another device's D_old/D_new comparison. Test either iteration order, removal
and re-addition, differing validity, expiry and replay/boot transitions. This is
agent-core work, not a model migration in this follow-up.

Agent-core must supply a separate **verified-serving record**, established only
after Apply and agent readback succeed. It contains boot ID, active bank and
operation identity, the exact post-render `NowMono` (Apply's RenderStart), the
corresponding wall snapshot, authenticated model/deadline inputs, and final
class assignments. All comparisons and live checks run under the same lock.
`retained_applied` is still useful for containment attribution; it is not that
record. It survives withdrawals and may contain another boot's monotonic epoch.
Apply takes Model by value, so the caller's pre-lock sample is not automatically
its installed sample.

Clear the verified-serving record on ANY loss of serving: baseline, quarantine,
failed Apply/reverification, expired/missing authority, withdrawn/handoff state,
unsafe meter/member path, boot mismatch or failed adoption. Persistence alone
cannot reinstate it. Same-boot adoption must reconstruct and fully verify exact
bank, timers, assignments and provenance; interrupted guard/token state closes.
A verified ConntrackDeleteError keeps the new verified record while the caller
still legitimately serves; retry-bound withdrawal clears it. Failure to persist
a truthful serving result cannot promote a merely installed candidate.

## M-1 and L-1: exact release and fencing corrections

The shared lock path is **`/var/lib/vpn-node/update-journal/update.lock`**, from
[agent configuration](../../../node/agent/config.go) and
[nodelock](../../../node/internal/nodelock/lock.go).

In C1.2, retain `fv_allow_A` without refreshing its elements. Build `fv_allow_N`,
`fv_token_g`, the candidate's complete ordinary policy, and the guard atomically.
With a nonempty eligible set, create/fill `fv_eligible_g` and use the two exact
old-bank/eligible lookup return rules in F31. With an **empty** eligible set,
including cold start, baseline recovery and unconditional fallback, omit both
return rules AND `fv_eligible_g`. The guard contains only:

```text
add chain inet fvpn fv_deadline_guard { type filter hook forward priority -151; policy accept; }
flush chain inet fvpn fv_deadline_guard
add rule inet fvpn fv_deadline_guard iifname M drop
add rule inet fvpn fv_deadline_guard oifname M drop
```

The fresh token still exists. No nonexistent old bank is referenced. If a known
old bank exists during fallback, retire it as part of the install once the new
chains remove its last references; do not strand it awaiting a later operation.
Cold start has no old-bank deletion. Unknown bank state requires baseline first.

C1.4 release, for a nonempty intersection, is one transaction:

```text
delete set inet fvpn fv_token_g
delete chain inet fvpn fv_deadline_guard
delete set inet fvpn fv_eligible_g
delete set inet fvpn fv_allow_A
```

The last reference to A disappears with the guard, so retire A here. Empty-set
fallback release contains only the first two lines. There is no post-release
retired-bank allowance: C1.5 verifies exactly the active N bank, zero retired
banks, and no guard/token/eligibility sets. A crash before metadata persistence
still requires readback/adoption or baseline; deleting A removes unnecessary
ambiguity, not the obligation to persist/reconstruct truthful state.

Replace F31 candidates.md's delayed-install row: **nodelock remains held until
any timed-out nft child is killed and reaped**. [System](../../../node/enforce/executor.go)
uses CommandContext and synchronous Run; the enabled
[lifecycle observer](../../../node/enforce/command_lifecycle.go) uses Start/Wait
and preserves cancellation. The owner of the lock cannot release on timeout
while a child can still write. A late release fails atomically on its missing
unique token after baseline. An intersection install also references A, but an
unconditional fallback install does NOT; missing A cannot fence that install.
Process-death/orphan-child behavior and rollout writers need explicit kernel
fault tests before C1 approval. A Go context deadline is not evidence of reaping.

C1 failure baseline must atomically close ordinary chains, invalidate tokens,
and remove every candidate/old/eligible bank after dropping references, even
when state refresh fails. This needs the complete finite operation object set
and a bank-aware baseline protocol; today's fixed-name baseline fix does not
implement it. Preserve counters/quarantine markers and detached cleanup budgets.

## M-3: blocking campaign-tool prerequisites

The copied `TestR24_M3_*` was removed: it never invoked the real tooling. Direct
inspection of [trace.go](../../../tests/netlab/cmd/fvpn-netlab/trace.go) confirms
that one-line-only release classification and the two-drop decorator reject C1.
Before a future campaign:

- Exercise the REAL trace executor/classifier with legacy release and both C1
  release shapes, including old-bank retirement. Emit guard_release only for
  the complete valid deletion transaction; reject ambiguous or malformed shapes.
  A substring match alone is insufficient. Keep producer/analyzer brackets real.
- Exercise the REAL guard-log decorator with the complete approved two-rule and
  four-rule guards. Leave the two exact return rules unchanged and decorate
  exactly the terminal drops. Reject extra permits, changed gates and duplicates.
- Require **ctunku = 0 and ctunkd = 0 deltas alongside zero eligible NFLOG drops**
  in the dedicated eligible-stream reload campaign. NFLOG alone misses ordinary
  forward-chain membership drops. Run deliberate unknown/revoked negative probes
  in separately bracketed controls so their expected counter increments do not
  contaminate this zero-delta assertion. No unexplained subtraction or tolerance.
- Keep B1's TCP_RTO_MIN/socket-pacing transport follow-up **disabled in every C1
  acceptance run and both comparison arms**. It is a separate experiment and
  cannot mask enforcement loss. Keep the original 1.5-second timeout, cadence,
  D-120 analyzer, 100-reload inputs and all old failures.

These are prerequisites only. Neither trace.go nor its decorator changes here.
Future unit tests must call the real functions; future kernel tests must prove
bank TTL retention, return-hook ordering, atomic token failure, baseline cleanup
and crash cuts on supported amd64 and arm64. F32's fixed-name baseline check is
not a substitute for C1's bank protocol or native arm64 qualification.
