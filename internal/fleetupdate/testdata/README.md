# D-062 result vectors

TEST ONLY - NOT A SECRET. The `d062-*` IDs, versions and hashes describe fictional test
inputs. Each JSON file validates as the current UpdateResult contract and carries
its canonical payload hash. Integration tests replace identity, authorization
times and manifest references with a real isolated job and recompute the hash.

- `d062-pre-health-failure.json`: preflight failure, unchanged current release,
  no health evidence, precise error and completion time.
- `d062-rollback.json`: restored checks pass; previous release is the candidate
  whose health check failed.
- `d062-rejected-result.json`: the rollback shape used with a serving policy that
  excludes the restored release. That rejection must replay after policy changes.

The node side retains failed candidate checks in its journal. These fixtures do
not require the proposed candidate_health field.

`at26-direct-rollback.json` is the unchanged result captured from the acceptance
harness run at source `5cb36d0`. It is test data, contains no secrets or host paths,
and includes both health profiles: the candidate tunnel probe failed and all
restored-release checks passed. Its canonical payload hash is verified before
rebinding job/release identities and authorization times to an isolated test job.
It covers the real producer's single terminal result with no earlier progress
result. The signed candidate fixture uses an empty rollback allowlist.
