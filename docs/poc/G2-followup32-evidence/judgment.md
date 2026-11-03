# r24 enforcement judgments before implementation

2026-11-03. Task-start main: `dff6bdb0ddcf6dfd8392a8db364c0120783a48f3`; original review
`ba5d3004`, reviewed base `52a7e59b`. The exact enforcement proof file was imported
and committed alone as `cdb384edcd309df7873cb20693d1d27b3e420522`.
[Before output](proofs-before.txt) records the requested command on that commit:

```text
bash "$FVPN_HOLD_SH"
python3 -B tests/netlab/scripts/disktmp.py go test -count=1 -v ./node/enforce -run '^TestR24_(H2|H4|M2)_'
```

Admission was `free`; the test exited 1, with both H-4 subtests, H-2 and M-2
failing as reported. No production edits preceded this judgment. Following the
task's D-141 judgment requirement, distinguish main's defects from unimplemented
design claims; D-141 itself keeps option (c) in production and supplies no C1
approval. Findings are hypotheses until checked, not automatic patch orders.

| Finding | Judgment and evidence | Patch boundary |
|---|---|---|
| H-4 | **Confirmed.** `installBaselineModel` silently retains cold State after a failed read; `RenderBaseline` omits chain replacement and set flush. `add rule` can follow live accepts. Apply also passes canceled contexts to rollback. Both original tests fail. A real canceled tc read can select quarantine instead, so that cleanup needs the same independence. | Self-contained baseline transaction for every fixed candidate chain/set, preserving counters; bounded detached cleanup context for baseline and quarantine inside Apply. Prove actual nft semantics and atomic rejection in the single final lab gate. |
| H-2 tc | **Confirmed.** `renderTC` puts orphan deletes in the pre-install batch. Old nft still classifies the removed member into the deleted HTB class, permitting fallback to the host default reserve. Original ordering proof fails. | Separate additions/replacements from retirement; retire only after candidate commit and before verification. Keep ownership checks and exact serving verification. Test crash residue/recovery and foreign attachments. |
| M-2 post-shape | **Confirmed.** Monotonic sampling/error and future-anchor rejection follow TCBatch without rollback. The original future-anchor proof fails. | Keep the final sample after shaping to avoid consuming the install budget with tc; route both errors through independent rollback. |
| M-2 fatal verified return | **Rejected as a currently reachable path**, with a dormant-code caveat. `Samples` always returns `(out, nil)`; `deleteConntrack`'s only non-command error is an invalid address, rejected by `validateIPv4Sets` on initial Render, final Render and verification. No ordinary immutable input reaches either alleged fatal return. | Keep a regression for invalid-address rejection before mutation and valid/failed conntrack controls. Do not claim an exploited or reproduced fatal verified result. Overall M-2 is **confirmed in part**. |

The original proof names are retained; fixed assertions lose their review TODOs.
The copied M-3 classifier/decorator test is removed after the before record: it
did not invoke the tooling. M-3 is a documented prerequisite in the
[C1 revision](../G2-followup32-design/revision.md), not a tooling fix here.

Spec: §8.1–§8.3, §10.3/§10.7, §11.4–§11.5, §16.3; AT10/AT13/AT29.
QoS and host reserve are in §11 (the review's §8.3 QoS reference is inaccurate).
No historical acceptance result is upgraded and no C1 implementation is authorized.
