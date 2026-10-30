# Follow-up 2 run procedure

Use the assigned `w4-isolation` / `36000` binding. The mechanism container is
`fvpn-w4-isolation-lab`; this supersedes the legacy lab prefix in README.

```sh
make -f mk/at-isolation.mk at-isolation-lint
make -f mk/at-isolation.mk at-isolation
make go-lint
```

The run target builds the mechanism executable without privilege, then starts
the isolated privileged lab. It runs the mechanism controls, removes that lab,
creates a fresh assembled stack, runs the product cases and removes the stack.
Docker operations use only the assigned prefix. Host networking is untouched;
temporary build files use `.agent/tmp` on disk. A blocked or failed product AT
makes the target exit 2 after writing all four records. Mechanism passes never
promote a blocked product result.

`stack.py` calls the shared devstack lifecycle. A temporary Docker argument
wrapper substitutes task-owned cache tags for its neutral `fvpn-devstack-*`
tags. All service images retain their shared build definitions; the records
contain the immutable IDs of the running images. Python uses only its standard
library. The existing Ubuntu workflow already has the shared stack's Python
dependency and remains unchanged.

`driver.py` uses a Go overlay to add `driver.go.txt` to the shared `tests/e2e`
test package. This executes the original private `countedFlow`, `selectExit`
and `tracedFetch` helpers. It neither edits shared source nor copies the route
selection implementation. Vet accepts the overlay; pinned staticcheck analyzes
an ephemeral source copy under the owned directory because its parser cannot
open a virtual overlay filename. The copy is deleted on exit.

Setup uses `PrepareServingBaseline`, API-created DNS rules, groups, plans and
finite grants, native admission, and publication read-back. SQL is read-only.
No projector-owned row, limiter configuration or fleet setting is written.
The controlled DNS rule is nonempty, as in the members harness. No synthetic
probe is used as a member positive control.

Each negative embeds the same device's counted, CA-verified origin request.
Protected destination cases also verify a listening target from the node and
capture headers at node ingress and target ingress. The interface-change case
keeps at least ten protected-target connection attempts per second across a
dummy-interface change and a generation produced by an API capacity update.
All interface and route changes are confined to disposable containers.

Publication excludes B while retaining A, then disables and restores the exit.
Revocation observes established TCP/UDP fixtures, conntrack deletion, the next
serving generation and an admitted sibling. Native route withdrawal limits
independent attribution of post-revoke kernel ingress. Restart waits for a new
serving heartbeat before checking recovery.

The finite-member expiry case checks the actual signed member deadline, then
stops both control and worker before it. This preserves the advertised route
while testing local expiry, with five post-deadline ingress captures and frozen
admission bytes. Both services are restarted afterward. Whole-node short-lease
expiry requires the D-065 fleet API handler; the contract-only stub is recorded
as blocked. No host clock is changed.

`AT*.json` are authoritative product records; `lab-AT*.json` and their sibling
captures/matrices are mechanism records. Each has a source hash, run ID, image
IDs and UTC timestamps. `summary.md` and `requests.md` describe the current run.
The #69 evidence lives in `history/follow-up-1/`. Diagnostic histories retain
their original dirty-source marker and are not the final acceptance run.

The final live run initially classified the finite-member expiry check as
blocked because its capture parser missed tcpdump's singular `1 packet captured`.
The original AT13 record is preserved in `history/follow-up-2-live/`. The corrected
parser accepts singular/plural summaries and rejects capture loss or an interface
disappearing. Its unit tests and an offline review establish five ingress packets,
zero egress packets, five timeouts, a retained exit route and frozen counters.
The root AT13 records this evaluation's UTC time and evaluator SHA256 separately
from the unchanged live source hash. No new live traffic is claimed by this review.

To reproduce that review, first archive the input AT13 JSON/Markdown, then run:

```sh
make -f mk/at-isolation.mk at-isolation-review-captures
```
