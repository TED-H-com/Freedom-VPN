# Follow-up 12 run instructions

```sh
make -f mk/at-secops.mk at-secops-lint at-secops-test at-secops-build
FVPN_SECOPS_EVIDENCE_FOLLOWUP=12 make -f mk/at-secops.mk at-secops-followup11-transports
FVPN_SECOPS_EVIDENCE_FOLLOWUP=12 FVPN_SECOPS_FOLLOWUP11=restore make -f mk/at-secops.mk at-secops-followup10
python3 tests/e2e/at/secops/followup12_records.py
python3 tests/e2e/at/secops/followup12_guards.py
python3 tests/e2e/at/secops/diskexec.py go test -count=1 ./tests/handover/...
make go-lint
test -z "$(gofmt -l .)"
git diff --check
```

Use only the assigned w4-secops / port-base 30000 stack. Before each live run check
`pgrep -af gates.sh | grep -v grep` and
`docker ps --format '{{.Names}}' | grep -c '^fvpn-w'`.
If two other lanes' suites or labs are active, wait and retry every five minutes. Both fresh runs
use disk scratch and unconditional cleanup. The restore target runs the same
Follow-up 10 drill, with Follow-up 12 filenames and D-095 hold read-back added.
Freeze and begin-recovery run while control/worker remain stopped. Briefly stop
the retained agent for the pre-observation HTTP/node hold probe, then resume it
without altering its desired state, leases, identity or generation. The HTTP
probe after fresh observation is not an interception of the first agent response.
The dated roadmap gate controls SECOPS-21 eligibility; absent D-099 means open.

Archive guards compare every Follow-up 10 file against the merged PR #265 blob.
The record renderer never edits snapshots or runs product mutations. Re-render
after document edits to refresh current scope-review pins.
