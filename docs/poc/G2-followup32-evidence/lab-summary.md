# Follow-up 32 lab result

**PASS**, one run, no retry or reservation. Source `f9aa1811abf50490d0a4c6b485c9a13e6e1782f5`
(production/test change `240b0278`). Command:

```text
bash "$FVPN_HOLD_SH"
python3 -B tests/netlab/scripts/disktmp.py make -f mk/netlab.mk netlab-test FVPN_LAB_PREFIX=fvpn-w1-netlab-f32
```

Admission printed `free`. [Receipt](gate2.json): exit 0, 241.010 seconds including
startup/cleanup; Go suite 232.826 seconds. [Full output](gate2.txt.gz),
[all lab artifacts and digests](lab-pins.json). All 113 lab files are retained;
large text/JSON files and text containing original trailing whitespace are
gzip-compressed with uncompressed hashes, preserving those bytes. The AT12
`counters.json` Go struct dump is published as `counters.txt`, byte-identical;
the first hygiene check rejected its misleading suffix and is retained. Absolute
worktree prefixes in command output are replaced with `<worktree>`; packet and
measurement content is unchanged. No historical evidence was edited.

## H-4 exact nft batch and atomicity

[TestISO01_AT13_F32_UnknownStateBaselineAtomic](../../../tests/netlab/scenario_f32_test.go)
passed in 1.42s on **nftables 1.0.9**, lab kernel **7.0.0-38-generic**
([version output](lab/TestISO01_AT13_F32_UnknownStateBaselineAtomic/versions.txt)).
This proves these commands on the tested backend; it is not a native arm64 or
all-Ubuntu-kernels qualification.

The [actual baseline batch](lab/TestISO01_AT13_F32_UnknownStateBaselineAtomic/baseline.nft)
uses a single `nft -f -` transaction. It ensures then deletes the guard, adds
needed counters idempotently, ensures/flushes allowed_v4, ensures/deletes/recreates
each base chain, adds only baseline rules, and ensures/deletes every unused
fixed candidate set. `State{}` is deliberately stale/empty throughout rendering.
No fresh listing is needed to decide whether a deletion or flush is emitted.
Named counters and quarantine markers are not deleted; no table is flushed.

| Kernel state / injection | Observation |
|---|---|
| No owned table | Batch succeeds; [cold readback](lab/TestISO01_AT13_F32_UnknownStateBaselineAtomic/cold.json) passes VerifyBaseline, exactly four chains and only empty allowed_v4. |
| Live generation 3200; invalid statement appended at END of baseline | [nft rejects](lab/TestISO01_AT13_F32_UnknownStateBaselineAtomic/atomic-rejection.txt). The prior generation marker and positive member TCP probe remain. Chain deletions, admission flush and set removals did not partly commit. |
| Live candidate plus guard and both optional probe sets, renderer still State{} | Batch succeeds; [readback](lab/TestISO01_AT13_F32_UnknownStateBaselineAtomic/warm-guarded.json) verifies baseline; no live accepts or candidate sets remain. Usage counter values survive and member probe is denied. |
| Repeat baseline | [Repeated readback](lab/TestISO01_AT13_F32_UnknownStateBaselineAtomic/repeat.json) verifies the same closed layout. |

The test checks real kernel results, not a string simulator. The original r24
unit proof additionally forces cold-start/post-release verification failure and
checks the required delete/flush commands, with only the initial table read.
Its cancellation proof and the stricter all-outputs-honor-cancellation variant
verify cleanup context independence. The latter reaches quarantine because a
canceled tc read is a containment failure.

## H-2, M-2 and unchanged controls

The real AT18 suite passed (117.71s): shared-member shaping, contention, stable
class on rate change, removal of the member class/qdisc, host reserve and shadow
behavior. AT10/AT13 reload/revoke/baseline passed (20.20s); AT14 metering and
AT29 expiry/recovery passed. Foreign root/class/leaf filters and containment
checks also passed. Their raw evidence remains in the lab artifact inventory.

[Focused output](proofs-after.txt) covers exact prepare → nft commit → retirement
→ release → verify order, crash-residue recovery, failed retirement → baseline,
foreign attachment rejection before deletion, missing/extra final classes,
post-shape clock failure and future-anchor rollback, canceled shape/install/
release cleanup, and G7 conntrack command failure retaining verified serving.
[Before output](proofs-before.txt) retains every original failure.

At the crash cut, the old HTB class is unreferenced by candidate nft. The
unconditional guard remains closed until retirement succeeds; no new member
packet can use the retired class or the host default. Already enqueued pre-cut
packets are not a new authorization. A next Apply reserves then deletes residue
before serving verification; baseline removes owned roots after nft closure.
We do not verify transitional residue as a serving generation or relax the
foreign-state verifier. Exact final class checking also rejects an unexpected
class without a leaf.

The fixed-object baseline makes no claim about unimplemented C1 bank names.
Rollback can still fail on command/kernel errors; failed Apply is not evidence
that closure completed. The current agent's withdrawal/quarantine obligations
remain. No previous-generation restoration is added.

No D-133 A/B/A/B was run. This functional pass measures neither isolated added
cost nor quiet-host performance. Deferred retirement adds a tc command inside
the guard-present interval when there are orphans, which a later comparison
must count. Native arm64 runtime proof remains outstanding; gate 1 cross-builds
linux/arm64 with CGO disabled. C1 remains docs only, and historical B1/F30
qualifications remain unchanged.
