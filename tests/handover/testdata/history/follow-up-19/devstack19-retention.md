# D-118 retained origin streams (TEST ONLY)

These coherent usage snapshots include durable WAL state. Adoption changes the
database representation; the predicate is exact canonical payload and origin
sequence/hash preservation plus nonregressing origin ACK. The replacement
stream is separate. This is local retention evidence, not a substitute for
usage AT 5's delivery/projector acceptance.

| Run | Origin incarnation | Ledger epoch | Last sequence | ACK before / retired after |
|---|---|---|---:|---|
| d6aaa798 | `01a19246-cb9e-70d2-9e31-e611a924c6a6` | `0376b40d-abbd-4bb0-b6f9-1e5badf32f5d` | 4 | 4 / 4 |
| 04de29d3 | `01a1924d-b2ed-7dae-875c-392d39febfa2` | `3c6e8f2a-10e3-4a46-a451-d3c7dbfda9e2` | 4 | 3 / 4 |

| Run | Origin sequence | Canonical payload bytes | Stored payload hash | Retired payload/hash comparison |
|---|---:|---:|---|---|
| d6aaa798 | 1 | 654 | `334d1c5a9898a6468424ad622b595516a81ea2cee6e158e3e3ce6d6b63076277` | identical |
| d6aaa798 | 2 | 654 | `2e2ca6a4eeefe03276986c5e2234ce77e5f986592b244b6d4722a623a252284c` | identical |
| d6aaa798 | 3 | 652 | `50d79a8d6164b2557903adec5390760db7b96a9954a112dd4d67074392ec2b41` | identical |
| d6aaa798 | 4 | 652 | `855a46e94d239eda9547817961915d7af15ab3c1af14edfb5888dc8acfb91fb7` | identical |
| 04de29d3 | 1 | 654 | `fddbf4a5ab1f20d08561a3ee1033f9426a503cf2d277e70c55d6a9ba0ddbabc5` | identical |
| 04de29d3 | 2 | 653 | `abfeb34fee6524f8d492b9dd17942db3570fc26ee22710f524a412e13705606f` | identical |
| 04de29d3 | 3 | 652 | `d77d7e7a9dd3f2f4b30279620d6c738260f2bab252acd92d5d7b25991191a86b` | identical |
| 04de29d3 | 4 | 652 | `f64552289ec90c8283a2284bab92019b36bca7aac07e51f0bb3ad0b0d79e02ee` | identical |

| Private coherent snapshot | Bytes | SHA-256 |
|---|---:|---|
| `.agent/tmp/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91-usage-origin.db` | 110592 | `5abb241579cca37a6a92cdce21b98183476c14115e2747cdc17430178f54c282` |
| `.agent/tmp/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91-usage-adopted.db` | 118784 | `a7eb9ee71a9bb6c683aa6065c0ebc1c7a1c8b958a82e6fedfe04ae8b372f33b2` |
| `.agent/tmp/followup19/04de29d3-e56b-4a32-b678-21462845ddb6-usage-origin.db` | 110592 | `aba62d95c8dc20ccc1ce4d4bae3cb477bc0ec64f8ebd7bf5f4cdb00e386dbb62` |
| `.agent/tmp/followup19/04de29d3-e56b-4a32-b678-21462845ddb6-usage-adopted.db` | 110592 | `e5c91b5b03339d065f86724eaec58d13fb7fe0c136f9b0776e946ec1124787b3` |

The serving run's checkpoint ACK is 3 at finalization, then reaches 4 before
re-enrollment; those are different observation times. The stopped-agent run
has origin ACK 3 before re-enrollment and retired ACK 4 afterward. Neither
is compared with the replacement stream's ACK. All eight canonical payloads
are retained byte-for-byte. Source snapshot files and raw payloads remain
private; the scenario records retain the CLI and identity restoration proof.
