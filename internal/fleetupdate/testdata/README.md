# D-062 result vectors

TEST ONLY - NOT A SECRET. All IDs, versions and hashes describe fictional test
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
