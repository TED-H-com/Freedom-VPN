# Follow-up 17 — one AT29 transport run

Use `FVPN_E2E_PREFIX=w4-secops`, port base `30000`, and only
`fvpn-w4-secops-*` Docker objects. Never take a reservation. Scratch is disk
`.agent/tmp` for both TMPDIR and GOTMPDIR. Go uses the cache from `go env GOCACHE`.

Before every test, make, Python test driver or Docker command, run
`bash "$FVPN_HOLD_SH"`; proceed only on `free`, otherwise re-check every 60s.
The script is supplied by the orchestrator's private repository. `admitted.py`
checks nested commands too and keeps admission output in a separate private
stream. Exit 125 is an admission refusal, never a test result.

D-098: `followup17_archive.py` freezes all existing root records byte-identically
at `history/before-follow-up-17/`, verified against the committed source with
SHA-256 pins. It refuses an existing archive. Every earlier archive is retained.
Commit the setup and its offline proofs first, then this archive, before running.

`followup17_host.py` checks the required inventory:

```sh
ps -eo pid,etimes,args | grep -E '[g]ates\.sh|[m]k/at-|[n]ode-test-netlab|[n]etlab-test|[e]2e-test'
```

It waits for host admission and an empty foreign-suite inventory, up to 120
minutes including earlier waits supplied in `FVPN_SECOPS_PRIOR_WAIT_SECONDS`.
Before dispatch it records the baseline commit and `vmstat 5 7`, then rechecks.
Only pid, comm and ticks of foreign processes are retained. A foreground parent
monitors the run every five seconds and joins its child. Any observed foreign
run overlap marks D-133 confounding; it never triggers a retry.

```sh
bash "$FVPN_HOLD_SH" && python3 -B tests/e2e/at/secops/followup17_host.py
```

This dispatches the admitted `make -f mk/at-secops.mk at-secops-followup17`,
using the existing fresh-stack transport driver. One rerun is allowed only for
a harness, wrapper or admission stop before the subject; a failed sub-case is
retained and never rerun. Archive any allowed prior attempt before replacement.

The setup design is **cap before member admission**. The public fleet cap PUT
and read-back precede this case's existing finite grant and native device join.
Its new binding produces a configuration generation and prompts agent renewal.
The 90s wait requires a locally accepted lease issued strictly after the cap
write, signed cap 60, lifetime within 60s, matching serving heartbeat and the
existing remaining-window/member-coverage predicates. Timeout names the lease.
See [diagnosis and results](2026-11-04-follow-up-17.md).

Continuity predicates from Follow-up 16 are unchanged: same live agent identity,
original TCP/9446 and authenticated UDP/9445 sockets, independent senders,
D from the accepted lease's request CLOCK_MONOTONIC stamp plus signed duration,
continuity through D−101ms−1s for at least three Step intervals, agent request
timeout logs, and quiet by D+3s through the next ten seconds. The 101ms is
DeadlineInstallBound 100ms plus DeadlineGranularity 1ms. This one-second sampler
does not claim D-140/D-142's packet-level isolation qualification. The original
killed-agent case and its three-second diagnostic remain unchanged.

Restoration is registered before mutations: restart control, unpause worker,
restore the original fleet cap and finite member standing through the existing
public API/read-back paths, including early returns. Log snapshots at each
sub-case boundary retain watchdog decisions and agent startup/adoption evidence
before a later restart overwrites agent.log. The driver tears down all owned
Docker objects and generated bind state. Verify a second zero inventory after it.

After cleanup, run the admitted `followup17_records.py`. Record SECOPS-F16-01 as
answered only when this run reached the outage with a qualifying lease. Any
post-setup failure is retained and routed as a product finding. Earlier run
`1bfd5cff` remains failed. Evidence SHA-256 pins use immutable history paths only.
New evidence over 20 MB must use lossless gzip with both form digests.

Run the nine gates sequentially after admission: area lint, test, build;
`make -f mk/e2e.mk e2e-lint`; `go test -count=1 ./tests/handover/...`;
`make go-lint` (with disk TMPDIR/GOTMPDIR and `go-with-disk-tmp=` to avoid the
root shell cleanup trap); empty whole-module gofmt; `git diff --check`; the
committed-evidence hygiene test. Retain any out-of-scope failure and continue.
