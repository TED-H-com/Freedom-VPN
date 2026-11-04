# Follow-up 15 — one AT29 transport run

Use the assigned `FVPN_E2E_PREFIX=w4-secops`, port base `30000`, and only
`fvpn-w4-secops-*` Docker objects. Never take a reservation, change the host
clock/network, use sudo, or touch the shared PostgreSQL or another lane's objects.

Before every test, make, Python test driver or Docker command, run
`bash "$FVPN_HOLD_SH"`. Start only on `free`; otherwise re-check every 60 seconds.
The script is supplied by the orchestrator's private repository. Never write its
location into evidence. `admitted.py` gives nested Docker, Go, make and Python
commands the same check, with admission output captured separately under
`.agent/secops/admission.log`. It preserves the inherited Go wrapper and uses
the original PATH for admission itself, avoiding interpreter recursion.

Before the live run, check this exact process inventory and wait while another
assembled run or network lab is active; record elapsed waiting time:

```sh
ps -eo pid,etimes,args | grep -E '[g]ates\.sh|[m]k/at-|[n]ode-test-netlab|[n]etlab-test'
```

D-098 comes first. `followup15_archive.py` freezes all current root records
byte-identically, verifies them against the committed source, and writes SHA-256
pins. It refuses to replace an existing archive. The Follow-up 15 archive is
already committed at `history/before-follow-up-15/`; do not rerun its creation.

The authorized run command is:

```sh
bash "$FVPN_HOLD_SH" && python3 -B tests/e2e/at/secops/admitted.py make -f mk/at-secops.mk at-secops-followup15
```

The foreground driver creates a fresh stack, runs all AT29 transport sub-cases,
then removes its containers, images, networks, volumes and generated bind state.
It also emits the existing bounded AT31 relay row if execution reaches it.
`guest_clock_rollback` remains blocked, owned by `w3-devstack-e2e / operator`.
A maximum of one rerun is allowed only for a harness/setup/admission failure
before the test reaches its subject. A failed sub-case stays a result and must
not be rerun. Archive every attempt before any permitted replacement.

Manual withdrawal uses public `revokeMemberGrant` (#297), records 202 and actual
operation completion, and samples the original TCP/9446 and authenticated
UDP/9445 sockets with independent senders. The bound is the earlier of the
§16.5 online 60-second target and the actual observed lease/member deadline
(§9.4, §16.3, §19 AT29). Setup polling (90 s), recovery (120 s), sample cadence
(1 s), positive window (4 s), and quiet window (10 s) are the existing transport
harness limits. They are not additional production promises. Recovery keeps the
original finite grant end, uses explicit administrator device authorization
under D-077, and requires the applied ACK and fresh counted TCP/UDP traffic.
A detached cleanup context restores standing even after an early return.

Render this run only after cleanup:

```sh
bash "$FVPN_HOLD_SH" && python3 -B tests/e2e/at/secops/followup15_records.py
```

Run these gates sequentially after admission, with disk TMPDIR/GOTMPDIR under
`.agent/tmp` (the admission runner provides them): area lint, area test, area
build, `go vet -tags e2e ./tests/e2e/...`, `go test -count=1 ./tests/handover/...`,
`make go-lint`, the whole-module `gofmt -l` empty check, `git diff --check`, and
the committed-evidence hygiene test. For `make go-lint`, use `diskexec.py` and
`go-with-disk-tmp=` to avoid the root makefile's shell cleanup trap. Exit 125 is
not admission and is never a test verdict.
