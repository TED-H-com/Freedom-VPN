# Candidates and the closure trade

Read the [current guarantee ledger](current-guarantees.md) first. `M` and `U`
below are the validated member and uplink interfaces; `A` is the active bank,
`N` the fresh candidate bank, and `g` a unique transaction identity. All objects
are owned objects in `inet fvpn`. Command templates are design notation, **not
executable files**. No sketch or kernel change is made in this follow-up.

## C1 — retain the old deadline gate while installing the candidate

This is a restricted intersection guard. It keeps two banks of timed member
admission; it does not copy or reload old elements. The candidate's full policy
is installed immediately, but only a carefully eligible subset can traverse
the temporary guard before the existing elapsed-budget check. The remaining
members stay closed until release. Every packet, including established/related
traffic and DNS, must satisfy the same checks.

### Eligibility is authorization, not address overlap

The caller must supply a trusted, actually verified previous model and its boot,
generation, bank identity and assignments. A bare kernel generation comment or
an address in a set is not provenance. All validation is under the existing
Apply/watchdog serialization lock, with no promotion based on desired state.

An address is eligible only if its member ID, device ID, peer binding, binding
revision and class identity are unchanged, it was admitted in the prior verified
model, and its **old effective deadline is no later than the new effective
deadline**. Compare the absolute anchored limits including binding validity,
node and policy expiry, service end and every ceiling; comparing lease end alone
is insufficient. Address reuse, an absent old element, missing anchor or an
unknown comparison is ineligible. A new address cannot become eligible from a
coincidentally equal IP. Keep the eligible set static; it conveys no lifetime.

For the first implementation proposal, any global isolation/resolver/host-prefix,
SNAT, interface/route/MTU, IPv6 mode, probe configuration, node/member QoS ceiling,
class or metering-semantic change selects the current unconditional guard for
the **whole** Apply. Quarantine, recovery, baseline, failed prior Apply,
unverified state or a pending unsafe control condition do too. A non-nil probe
is excluded from C1 initially. Permit device additions/removals within unchanged
member classes: new/rebound/shorter-deadline devices are ineligible; removed
devices also disappear from candidate admission. This covers AT10's recurring
removal/re-addition of device A2 while A1 stays eligible. It is not a promise
of hitless service for every generation change.

Select the unconditional path before additional C1-specific reads when the
authenticated model already proves a restriction requiring it. A failed live
check is a rejection requiring baseline/quarantine, not a reason to quietly
retry a permissive fast path. A valid but ineligible change can use normal Apply.

### Exact batch and call structure

**C1.0 — migrate the representation while closed.** The first upgraded Apply,
or any boot without a compatible verified layout, uses the unconditional guard.
Create a unique bank such as `fv_allow_g0` instead of the serving references to
`allowed_v4`, but otherwise render the same four base chains at −150/100, full
rules, named counters and class markers. Verify the new layout and deadline
read-back before recording it. All baseline writers/readers must already
understand the format. This transition is not hitless and is not counted as a
C1 reload. There is never a rename/copy of a live timeout to simulate retention.

**C1.1 — preflight and shape.** Obtain provenance, perform existing ownership,
rule/path checks and choose eligible addresses. Keep the old bank immutable.
The existing `tc -batch -` remains ordered before nft installation: HTB root if
absent, orphan leaf/class deletes, parent/default/member class replacements,
missing fq_codel leaves, uplink then member direction
([render_tc.go:28–80](../../../node/enforce/render_tc.go)). In a C1-eligible
transition no roots/leaves/classes/rates change; any emitted idempotent class
replacements must be byte-equivalent in meaning. Do not slip a changed ceiling
or class deletion into this case. General changes keep today's guarded path and
its existing partial-tc failure behavior. No assertion of an atomic nft/tc batch.

**C1.2 — final render and one atomic install.** Sample `R`; compute candidate
timeouts with the unchanged `100 ms` reserve and `1 ms` floor. Start the existing
100 ms installation context. Submit one batch in this order (all statements
commit together):

```text
# Remove only a now-unreferenced retired bank from the preceding successful
# operation, if one exists; preserve the active A elements without modification.
delete set inet fvpn <retired-bank>              # only if recorded and present
create set inet fvpn fv_allow_N { type ipv4_addr; flags timeout; }
add element inet fvpn fv_allow_N { <new addresses with computed relative TTLs> }
create set inet fvpn fv_eligible_g { type ipv4_addr; }
add element inet fvpn fv_eligible_g { <eligible address list> } # omit if empty
create set inet fvpn fv_token_g { type ipv4_addr; }            # empty fence
# Full Render transaction: preserve counters; update ordinary sets and all four
# base chains; every former @allowed_v4 lookup, including SNAT, uses @fv_allow_N.
# No old allow-bank flush, element update, copy, delete or recreation is allowed.
add chain inet fvpn fv_deadline_guard { type filter hook forward priority -151; policy accept; }
flush chain inet fvpn fv_deadline_guard
add rule inet fvpn fv_deadline_guard iifname M oifname U meta nfproto ipv4 ip saddr @fv_eligible_g ip saddr @fv_allow_A return
add rule inet fvpn fv_deadline_guard iifname U oifname M meta nfproto ipv4 ip daddr @fv_eligible_g ip daddr @fv_allow_A return
add rule inet fvpn fv_deadline_guard iifname M drop
add rule inet fvpn fv_deadline_guard oifname M drop
```

Expand `M/U` as validated quoted names and `N/g` as locally generated identifiers
never reused across operations, including same-generation retries and restart.
`create` fails on collision. The token must be bound in the retained operation
metadata; it is not an authentication secret. Metadata records the complete
active/old/eligible bank identities and model digests, without changing the
signed generation or fabricating applied ACKs. The rule layout is canonical and
closed to arbitrary extra chains or sets.

The `return` from this **base** chain reaches its accept policy, then the −150
forward base chain still runs. That candidate chain performs all normal
isolation, timed membership, accounting and QoS classification. Nothing accepts
a packet directly into postrouting. Denied/host/resolver checks and both IPv6
drops remain exactly ordered. Reverse NAT has already restored the destination
member address before these forward-hook rules. The old gate has no usage
counter; only candidate accepts count admitted bytes once.

**C1.3 — check the install budget.** Sample installed-by and apply exactly
today's success, monotonic ordering and `InstalledBy−R <= 100 ms` requirements.
On command error, cancellation, invalid sample or overrun: do not release;
attempt the node-wide baseline, return failure and invoke agent withdrawal.
No old-generation restoration, even if it would keep an unchanged sibling up.

**C1.4 — release after success.** One nft transaction:

```text
delete set inet fvpn fv_token_g
delete chain inet fvpn fv_deadline_guard
delete set inet fvpn fv_eligible_g
```

The old bank remains unreferenced until cleanup in the next install or baseline.
Deleting the unique token makes release non-repeatable: if a baseline or newer
operation already invalidated it, the **entire** stale release must fail, leaving
any current guard intact. A release retry must read the actual state and reject
uncertainty; it may never delete an arbitrary current guard. Do not add a fresh
candidate timeout or restart any timer here. Release delay only consumes time.

**C1.5 — live verify, agent reverify and persistence.** Preserve current ordering:
full live verification after release, agent's independent reverify before ACK,
then durable applied evidence. The verifier requires exactly the active bank,
at most one identified unreferenced previous bank, no outstanding token/eligible
set/guard, exact active address membership/lifetimes and current deny/host sets,
unchanged four base chains, counters/classes, tc and foreign-state checks.
Never expand `verifyChainSet` to accept arbitrary extras. Parser and precision
read APIs must accept an expected bank identity, never choose by prefix or by
largest generation. Incomplete hand-overs cannot be adopted on restart.

**C1.F — any failure/expiry/rollback.** In one nft batch first replace forward
with generation-0 drops and empty admission, replace input/output/postrouting
with the current bare baseline, then remove guard, tokens, eligible sets and
all old/candidate banks after references are gone. Preserve counter objects and
class identity markers. Then perform today's ownership-checked tc cleanup.
Foreign tc invokes quarantine with the retained member interface. Keep the
independent cleanup context, original error and partial rollback evidence.
Closure is only proven when read back; a failed command is not a successful
baseline. Never switch to a retained bank as recovery.

### Deadline and revocation proof

While the guard exists, a passing packet must satisfy
`eligible(address) AND live_old(address) AND candidate_policy(packet)`.
For an eligible identity `E_old <= D_old <= D_new`, so even a candidate installed
arbitrarily late cannot forward that identity beyond `D_new`. Ineligible members
cannot traverse the guard. After C1.3, `E_new <= D_new` follows from the current
reserve proof; release changes no expiry. The old gate handles a SIGKILL/SIGSTOP
without a live agent or watchdog. A removed member fails candidate admission
at the same candidate-commit boundary as today; conntrack does not bypass it.
G1, G2, G3, G7–G10's authorization properties therefore have a concrete proof
target, subject to the kernel and provenance obligations below.

There is **no extra transaction between the candidate's revocation and today's
revocation boundary**. From final render, an accepted install still has the
100 ms bound. From Apply receipt, C1 can add eligibility/read/parse work and
larger nft input; its latency must be measured. A revoke arriving while a previous
C1 Apply holds the shared lock can wait for that operation. No finite absolute
scheduler bound exists today or in this proposal. Require the agent to use its
existing finite local I/O budgets, no additional retries in the lock, and publish
`extra revoke wait <= remaining C1 operation + next poll/control pass + revoke
Apply`, additionally capped for packet authorization by `D_old`. Missing any
finite configured budget blocks implementation review; do not advertise the
100 ms install budget as this end-to-end bound. The comparison demands no later
observed revoke cutoff than paired A, including a revoke arriving mid-handover.

### Crash matrix and the guarantee it does not preserve

| Cut | C1 outcome compared with the current ledger |
|---|---|
| Before shape / each tc command / before atomic install | Same as main: old nft with already running timers; partial tc remains a possible failure. |
| Install rejected atomically | Old state. Caller rejection still requires baseline, as on main. |
| Install commits; kill before budget check | Candidate is capped by immutable old admission for eligible identities; everyone else closed. **G5 all-member closure is weaker.** No candidate-only overlong lifetime is reachable. |
| Install arbitrarily late; kill/stop before detection | Same intersection until old expiry. **G6 immediate all-member closure is weaker.** A stopped lock holder can prevent watchdog action; only old kernel expiry is unconditional. |
| Budget fails; baseline pending/fails | Caller reports failure and attempts baseline, but eligible traffic may continue until old expiry or successful independent closure. Main's guard would already close it. This is the associated G4/G8 failure-containment difference. |
| Budget passes; kill before release | Intersection remains, bounded by old expiry. This differs from main's blackout, even though authorization is valid. |
| Release ambiguously commits | Either intersection or candidate with proven safe TTL; never an unbounded timer. Token fences a stale release after a successful baseline. |
| Release commits; verify fails / kill before rollback | Same class of post-release exposure as main; baseline/quarantine required. No old bank restoration. |
| Baseline commits; delayed install/release arrives | Release token is absent; install references a removed old bank. Such transactions must fail atomically. If they linearized before baseline, baseline wins. Test this ordering, including orphaned `nft` children. |
| Restart / rollback to an older binary | Interrupted or unknown bank state is not adoptable; close first. Old readers/writers are incompatible unless upgraded to understand the layout. Rollout compatibility is a blocking cross-lane obligation. |

This is not semantic sleight of hand: **C1 does not keep every guarantee in the
current ledger**. Its valid-old-authority continuation is the owner's explicit
trade, not a reason to quietly weaken G5/G6 or turn a failed Apply green. With
strict retention of those exact cuts, keep the unconditional guard. A userspace
health check just before release cannot detect a death that occurs immediately
after that check, and a new relative watchdog element has the same late-install
problem. Adding either does not eliminate this trade.

### New risks, platform needs and cost

Risks are wrong identity/deadline comparison, accidentally refreshing the old
bank, active-bank ambiguity, stale release deleting a newer guard, a baseline
leaving referenced banks, parser acceptance of orphan admission, meter double
counting and old-binary rollback incompatibility. Reject rather than repair
uncertain provenance into a success. A global restriction incorrectly classified
as compatible is a safety failure even if the stream never loses a packet.

C1 needs existing nft transactions, timeout sets, regular set lookups, base-chain
`return`, named counters and atomic deletion failure; no new kernel ABI. These
are available in the Ubuntu 24.04 nftables 1.0.9 family on amd64 and arm64, whose
packages are listed by [Ubuntu](https://packages.ubuntu.com/noble/nftables).
Availability is not qualification: exercise the exact timeout/backend, hook
ordering, stale-reference and delete-token semantics on each supported kernel
and physical arm64. Main's live lab evidence uses a newer host kernel, so it
does not certify every Noble kernel or suspend/resume combination.

Expected ordinary cost: still one install and one release transaction; two
additional lookup rules before the terminal drops; one retained timed bank,
one temporary eligibility set and one empty fence set; O(addresses) model
comparison and extra rendered bytes. At most active plus previous bank after
success, with one new bank and one retired-bank deletion during preparation.
No duplicated usage counters or tc tree. Current bank storage can approach twice
member-element memory. Failed cleanup can grow it unless operations fail closed
instead of staging more. Every command, byte and parser cost is unmeasured.
Expected packet benefit is no intentional guard loss for eligible traffic;
new members wait for release, revoked members stop immediately at install.

## C2 — stage a wholly inactive generation, then atomically switch

This more general alternative leaves the old **full** policy serving. In
`inet fvpn`, keep four permanent base chains, with forward/postrouting each
jumping to an active generation's ordinary chain. Each generation has its own
timed allowset and private deny/host/resolver sets, and references the same
stable device counters. Input/output safety remains permanent. Bootstrap the
dispatcher layout under today's unconditional guard and verify it first.

1. Preflight provenance and ownership. For identical QoS, tc is empty or the same
   idempotent batch as C1. For changed QoS/classes, close under the current guard;
   there is no atomic cross-subsystem switch. Do not stage a second HTB root and
   pretend both roots can own the same interface.
2. At `R`, one `nft -f` transaction creates `fv_allow_N`, private policy sets,
   `fv_forward_N`, `fv_nat_N`, and a unique release token. The new forward chain
   is the complete current ordered policy, ending in explicit drop; the NAT
   chain carries the corresponding SNAT lookup. No hooks or dispatcher changes
   occur. Its relative timers start while unreachable.
3. Check `InstalledBy−R <= 100 ms`, then read/verify the staged generation,
   current active generation, tc, foreign hooks and exact references. Failure
   takes node-wide baseline, never successful old-state fallback. An expired
   candidate cannot be promoted as live authority.
4. One activation batch deletes the unique token, flushes/rebuilds **both**
   forward and postrouting dispatchers with new generation/class markers and
   jumps to N. The permanent forward base policy stays drop. No element is
   replaced, copied or refreshed. A delayed switch consumes N's safe TTL.
5. Verify active state, agent reverify/ACK; only later delete unreferenced A
   chains and sets. Baseline atomically disconnects dispatchers, drops members
   and removes all generation objects/tokens before tc cleanup.

G1: budget-checked staged timers remain safe at activation, even with a delayed
switch. G3/G7/G8: closure, verification and no old restoration are still required.
G9: each traversable bank must enforce the complete policy; never switch NAT
alone, never use an accept-only staging shortcut. G10: shared counters avoid
reset/double count, but tc changes force the slow path. **G2 is not preserved
for unrestricted changes:** a revoked member continues on A between staging
commit and activation. Added latency is verification plus switch delay, with
no hard userspace scheduling ceiling; if killed before switch, it can last to
`D_old`. G5/G6/G4 also have C1's old-authority-versus-immediate-closure trade.

A revocation-first batch could prune A before staging, but then there are three
transactions, an additional failure state and the pre-prune commit can itself
be delayed. It cannot be silently counted as the same no-later cutoff. A
restricted C2 that accepts **only** exactly nonrestrictive, unchanged-QoS
generations avoids that particular revoke, but falls back whenever AT10 removes
A2. A later incoming revoke still waits for staging/verification. Requiring a
finite activation timeout does not cause the kernel to close on process death.

New risks include mixed-bank NAT, dispatcher fall-through, verification of an
unreachable bank mistaken for live evidence, partial cleanup, stale activation
after baseline, duplicate counter paths and bank proliferation. Atomic token
fencing and baseline cleanup are mandatory as in C1. Stock nft chains/jumps and
sets exist on both target architectures; the verifier currently accepts exactly
four chains ([verify.go:832–842](../../../node/enforce/verify.go)), so this is a
larger parser, model, adoption and compatibility change. Expected cost is a full
second policy in memory, at least stage+activation+eventual cleanup transactions,
and an additional staged-state verification. No measured benefit/cost is claimed.
Not recommended over C1: its revoke delay and larger representation change buy
no necessary advantage for the observed unchanged-member loss.

## C3 — use an absolute deadline in the kernel

The attractive structure is: shape as today; render absolute elapsed deadline
`D` for each element; atomically replace the complete policy; verify and ACK.
The intended primitive would install `expiry = D` (or
`timeout = max(0,D−kernel_now_at_commit)`) and reject/omit an already expired
element. It would perform that calculation in the kernel, with a boot/clock
identity and conservative rounding, before making the rule reachable. No guard
release is needed because no unchecked relative budget exists. Baseline,
revocation-before-established, isolation, tc failure and verification rollback
must otherwise remain as in main. A kill after commit leaves safely bounded
candidate forwarding, just as a kill after today's release does.

**This is not implementable by writing `expires D` into current nft syntax.**
The [nft manual](https://netfilter.org/projects/nftables/manpage.html) defines
`expires` as remaining time. Linux 6.8's
[nft_set_elem_init and element attribute decoding](https://raw.githubusercontent.com/torvalds/linux/v6.8/net/netfilter/nf_tables_api.c)
add a supplied duration to a kernel time sample; they do not accept the agent's
absolute monotonic instant. Computing `D−now` just before `nft -f`, or even in a
userspace netlink sender, still leaves delay before kernel processing. `nft -c`
validates a batch; it does not reserve a commit time. This conclusion is about
the inspected supported interface, not a claim that all future kernels lack it.

Using `meta time >= wall_deadline drop` instead is also unsuitable: the
[kernel implementation](https://kernel.googlesource.com/pub/scm/linux/kernel/git/netdev/net/+/b85966adbf5de0668a815c6e3527f87e0c387fb4/net/netfilter/nft_meta.c)
uses realtime. A wall-clock rollback can extend forwarding, violating §16.3.
Adding a relative set alongside that check does not fix delayed initialization.

With a **new, reviewed kernel absolute-monotonic expiry/commit predicate**, G1
and G2 could hold without the relative-time guard; G3/G4/G7–G10 retain their
current limits and cleanup obligations. The exact G5/G6 all-member-blackout cut
would still be replaced by atomic safe admission, so the owner's required
meaning must be explicit. Failure cases would include wrong boot/clock domain,
time namespace translation, wrap/overflow, rounding, suspend and kernel ABI
compatibility. It adds no deliberate revocation delay at commit, but inherits
normal command and control delivery latency. If that primitive is absent, all
guarantees cannot be proven and it must fail closed with the current mechanism.

No such primitive is established for main's supported amd64 or arm64 platforms
by the reviewed interfaces. It would require kernel/userspace work, packaging,
support and verification outside this lane, not a small renderer change.
The likely userspace command saving is one release invocation; kernel and
per-packet cost are unknown. No kernel fork, BPF bypass, dependency upgrade or
platform change is proposed here. C3 is rejected as a current implementation
candidate, rather than inventing a production feature or running an unnecessary
privileged sketch.

## Recommendation boundary

C1 is the narrowest practical review candidate: the old timed elements cap the
only temporary permits, the candidate immediately removes revoked admission,
and the path still applies full isolation, metering and QoS. C2 adds revocation
delay; C3 needs an unavailable proven primitive. **None presently keeps the
literal G5/G6 and associated G4 failure containment while also being hitless.**
The [proof plan](proof-obligations.md) therefore marks that obligation as a
decision blocker, not a test to redefine after a good run. Until the owner
explicitly accepts a different valid-old-authority failure contract, production
recommendation remains A. Even with that decision, C1 requires every other proof,
cross-lane integration review and the [D-133 comparison](d133-comparison.md).
