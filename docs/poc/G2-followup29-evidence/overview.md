# Follow-up 29 — AT12 isolation fixed; guard observation startup failed

**F28-01:** the first reserved full-suite run reproduced exactly **97 upload bytes / two packets**, attributed to AT10's old TCP flow: a **57-byte data retransmission plus a 40-byte RST**. Both were forwarded to the responder. AT12 had begun with an orphaned `FIN-WAIT-1` socket after process cleanup; flushing node conntrack did not close it. This demonstrates a test-isolation defect. These bytes were admitted under the current policy and counted as required by §10.1. [Packet/counter evidence and history](at12-findings.md).

AT12 now closes and verifies prior connected IPv4 TCP sockets in its dedicated member namespace before probes and baseline sampling. Its zero-byte assertion is unchanged. Both corrected full-suite runs encountered a pending earlier socket and passed, with **zero deny-phase admitted bytes**. The final source run held **14,349 bytes / 306 packets** unchanged. Production enforcement, rendering/install paths and consumers are byte-identical to task-start main; no production D-133 cost is introduced.

**F27-01 remains unresolved, B1 remains FAIL.** The one predeclared guard diagnostic failed before its 100-reload sequence because NFLOG rejected nanosecond pcap timestamps. The failed attempt, executed observer and error are retained. The parser now supports default microsecond pcap timestamps; that correction has failing-before/passing-after offline coverage and **no live retry**. No exact guard-drop attribution or measured diagnostic D-133 cost was obtained. [Declaration](guard-declaration.md), [failed window and limits](guard-outcome.md).

Follow-up 28 gate 2 stays FAIL. Its original 97 packet bytes cannot be individually recovered because no all-traffic capture exists for that run; this follow-up proves the reproduced mechanism without fabricating historical packet identities. Its publication rewrite is recorded additively in [the note](publication-mapping.md) and [JSON](publication-mapping.json), verified with local Git diffs. Earlier evidence trees are never edited.

## Retained runs

| Run | Result | Purpose |
|---|---|---|
| [full1](full1.txt) | FAIL: AT12 +97 only | Original assertion, full-suite order, all-traffic observation before choosing a fix |
| [gate2](gate2.txt) | PASS | Same orphan-socket antecedent; socket isolation; +0 deny bytes |
| [guard-window](guard-window.txt) | FAIL: observer startup | The sole guard diagnostic attempt; no reload sequence or attribution |
| [final-gate2](final-gate2.txt) | PASS | Final tree after offline NFLOG correction; +0 deny bytes; guard logging disabled |

Exactly **three full-suite runs**, **zero AT12-only runs**, **one guard attempt**. Every run is retained. The last full suite was required to validate the final parser source; it does not replace the failed original measurement. Every measured run used its own reservation and 60-second settling interval, with required preflight both before and after topology setup. [Admission summary](admission-summary.json) retains all 13 quiet checks, including five rejections. Reservations were separated by independent work and released promptly. [Final lab inventory](final-cleanup-inventory.json): zero owned containers, images, networks and volumes.

## Reproduction and boundaries

Read [the code-first candidates, sizes and suite history](at12-declaration.md). Offline replay accepts plain or lossless-gzip inputs:

```sh
python3 tests/netlab/scripts/disktmp.py python3 -B tests/netlab/scripts/at12_analysis.py \
  docs/poc/G2-followup29-evidence/full1/TestAT12_ISO01_ISO04_AllowDeny \
  .agent/f29-at12-replay.json
python3 tests/netlab/scripts/disktmp.py python3 -B -m unittest discover \
  -s tests/netlab/scripts -p '*_test.py'
```

Admission precedes every make, Docker and go-test command via the retained PATH wrapper. Scripts read the admission location only from `FVPN_HOLD_SH`. `run.py` uses disk TMPDIR/GOTMPDIR through `disktmp.py`; GOCACHE is resolved by Go and recorded only as a digest. Delivered destinations must never be reused. The guard attempt is spent; another live run needs a new assignment/window and a new evidence directory.

Artifacts over 1 MB of JSON/JSONL are losslessly compressed with both hashes in the final manifest. Only derived synthetic packet headers are published; raw NFLOG/TCP pcaps remain in disposable lab scratch and are removed. Capture completeness and counter/read-boundary checks fail on missing evidence; they never subtract or excuse bytes. Physical arm64 and corrected live NFLOG remain unmeasured; arm64 Go cross-build is included in gate 1.

## Final verification

Every mandatory final gate passed: netlab lint/unit/integration/build (including arm64), the privileged full suite, all unchanged consumers, committed-evidence hygiene, handover, root pinned lint, whole-module vet, tracked Go formatting and diff checks. [Gate receipts](final-gates-summary.json). The separate original AT12 measurement and guard startup attempt remain failed.

D-098 passed at the [start](d098-initial.json) and [completion](d098-final.json): **994 payload files and four manifests** in follow-ups 25–28 remain byte-identical, including compressed and uncompressed hashes. Follow-up 28 uses the published `dda5a42a…` manifest. [Source review](source-review.json) also verifies unchanged production/consumer paths and the exact original AT12 deny assertion. The final manifest pins every artifact in this directory.
