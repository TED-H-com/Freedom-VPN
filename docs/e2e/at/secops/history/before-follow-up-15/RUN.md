# Follow-up 13 run instructions

```sh
pgrep -af 'gates.sh|mk/at-'
make -f mk/at-secops.mk at-secops-followup13
python3 tests/e2e/at/secops/followup13_records.py
make -f mk/at-secops.mk at-secops-lint at-secops-test at-secops-build
python3 tests/e2e/at/secops/diskexec.py go test -count=1 ./tests/handover/...
python3 tests/e2e/at/secops/diskexec.py make go-lint go-with-disk-tmp=
python3 tests/e2e/at/secops/followup13_guards.py
git diff --check
```

Wait while two other integration suites are active; record the preflight before
starting. Use only fvpn-w4-secops objects, network fvpn-w4-secops and port base
30000. Foreground runner, disk TMPDIR/GOTMPDIR under .agent/tmp, no shell traps.
The runner cleans its objects and generated bind state even on test failure.

The archived preparation failure preceded any capture arm; it is not a failed
first-offer result. Never repeat a delivered first offer to obtain a pass.

This target runs AT30 only, using its own freshly enrolled primary. It skips
transport, grace and bulk-peer campaigns. Stop the retained agent, arm the shared
first-offer export exactly once before the clean restore, freeze/begin-recovery
while central services are stopped, and prove D-095 through direct-control HTTP.
Resume observations with desired GETs held, wait for the retained applied
generation at central, then release and read FirstDesiredOffer. The separate
HTTP probe runs after capture so it cannot induce compilation before delivery.
A first 304/error/stale generation or missing local/receipt match stays open;
never re-arm to obtain a passing replacement. No generation or SQLite writes.

Record rendering never changes history. Follow-up 12 is SHA-256 pinned and
compared with PR #301; older handover archives remain in place. Re-render after
current document edits to refresh scope pins. Devstack owns RESPONSES.md; this
lane records export discrepancies in requests.md and the orchestrator report.
