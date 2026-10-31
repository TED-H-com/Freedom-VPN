# Original assembled AT10 before recovery correlation

Run `bf671f3a-c6ac-4824-9107-425c4c5a5f02`, image/main-driver build
`e711ebd822eda2ea1b5ba10b187818fda6777a74`. The main driver sampled the
initial bootstrap only; it did not enter the provisional condition. Its AT10
JSON/Markdown are preserved here before the explicit same-run offline review.

`provisional-recovery.raw.json` retains the independently captured read-only
recovery values from sampler build `604f4596e29572c654e2e285acad301a199ca672`.
Its descriptive `positive_reference` was a planned AT13 restart reference;
the actual extra restart is documented in the root `recovery-action.json`.
The review used the counted AT12 node-management-22 control after that recovery
and corrected the root capture's descriptive reference. It verified identity,
generation, populated lease timing and the post-transfer serving heartbeat.

Only the root AT10 provisional check changes to pass. The interface failure
and all four blocked peer controls remain. JSON sample whitespace is normalized
to one object per line; every captured value is unchanged. The root AT10 record
also identifies the review executable commit. No archived pass is imported from
a different run.
