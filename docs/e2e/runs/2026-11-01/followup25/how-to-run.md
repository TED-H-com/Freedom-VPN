# Reproducing the follow-up 25 measurement

TEST ONLY. Use a new archive/run ID, never overwrite these records or pins.
Check `pgrep -af 'gates.sh|mk/at-'` before startup and again after it; wait while
two other integration suites are running. The campaign's declared counts and
order are in [plan.md](plan.md). Each attempt uses a fresh reset stack and the
same first-selection runner, including its passive two-second tail.

```sh
mkdir -p .agent/tmp/followup25-new
export TMPDIR="$PWD/.agent/tmp/followup25-new" GOTMPDIR="$PWD/.agent/tmp/followup25-new"
export GOCACHE="$(go env GOCACHE)" GOTOOLCHAIN=local CGO_ENABLED=0
export GOMAXPROCS=4 GOFLAGS=-p=4
make -f mk/e2e.mk e2e-up E2E_RESET=1
FVPN_E2E_FOLLOWUP25=1 FVPN_E2E_FOLLOWUP25_ARM=kernel FVPN_E2E_FOLLOWUP25_FAMILY=ipv4 \
  go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup25Native$'
make -f mk/e2e.mk e2e-down
```

Use `ipv6` for the separate family constraint. For the diagnostic arm use
`FVPN_E2E_FOLLOWUP25_ARM=userspace`; the runner arms the node marker before its
enrollment and the entrypoint sets `NB_WG_KERNEL_DISABLED=true` only on that
lab daemon command. The one-use marker is consumed at launch and cleaned up on
test exit; a later ordinary node boot keeps the kernel baseline. Confirm `uses_kernel_interface` in the captured node and
member status samples; a requested setting alone is not implementation proof.
The normal fixture and NetBird image pin are unchanged.

Raw captures/debug/trace files are private under `.agent/tmp/followup25/<id>/`.
The runner publishes only decoded headers and structured observations. On a
new completed run, verify receipts and derive the bounded analysis once:

```sh
python3 tests/e2e/analyze_kernel_capture.py \
  --run docs/e2e/runs/2026-11-01/followup25/<new-id> \
  --private .agent/tmp/followup25/<new-id>
python3 tests/e2e/summarize_first_selection.py \
  docs/e2e/runs/2026-11-01/followup25/<new-id>
```

Both derived records refuse to overwrite an existing result. Preserve startup,
instrumentation, transfer and analysis failures separately; do not replace a
failed attempt with a successful retry. After exporting evidence, tear down
and confirm zero objects owned by the harness prefix. Delete scratch with
Python, without shell cleanup traps; retain intentionally private raw receipts.

To test the persistent exit export on an already enrolled stack, **after** the
first-selection measurement and **before** teardown:

```sh
FVPN_E2E_EXIT25=1 go test -tags e2e -count=1 -v -timeout 5m \
  ./tests/e2e -run '^TestE2E_AgentExitReadback$'
```

This separately records a real graceful stop and an explicit agent SIGKILL,
verifies the wait receipts after reaping and after repeated stop, and restores
the agent. Unit tests additionally prove normal exit 137 differs from SIGKILL.
See the [updates answer](../../../RESPONSES.md#updates-13--2026-11-01-follow-up-25-d-102)
for the exact consumer read sequence and time semantics.
