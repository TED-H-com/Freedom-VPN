# Follow-up 29: how to run

From the worktree root, after the host admission check prints `free` before each heavy command. No shell cleanup traps; foreground only.

```sh
mkdir -p .agent/tmp/followup29
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp" GOTOOLCHAIN=local CGO_ENABLED=0
export GOCACHE="$(go env GOCACHE)"
# Runner-like: no cached browser, no npm cache.
mkdir -p .agent/tmp/followup29/pw-before .agent/tmp/followup29/npm-before
export PLAYWRIGHT_BROWSERS_PATH="$PWD/.agent/tmp/followup29/pw-before"
export npm_config_cache="$PWD/.agent/tmp/followup29/npm-before"
export FVPN_E2E_PREFIX=e2e FVPN_E2E_PORT_BASE=18000 FVPN_E2E_STATE="$PWD/.agent/e2e-state-e2e"

# before-fail (commit 0e315f90): fresh stack and fresh Portal build, no browser install.
make -f mk/e2e.mk e2e-up E2E_RESET=1
FVPN_E2E=1 FVPN_E2E_PORTAL_BILLING=1 FVPN_E2E_RUN_LABEL=followup29/before-fail \
  go test -v -tags e2e -count=1 -timeout 15m ./tests/e2e -run '^TestE2E_PortalBilling$'
make -f mk/e2e.mk e2e-down

# after-focus (commit e956aa97): new fresh stack, the fix, same still-empty browser dir.
make -f mk/e2e.mk e2e-up E2E_RESET=1
make -f mk/e2e.mk e2e-browser
FVPN_E2E=1 FVPN_E2E_PORTAL_BILLING=1 FVPN_E2E_RUN_LABEL=followup29/after-focus \
  go test -v -tags e2e -count=1 -timeout 15m ./tests/e2e -run '^TestE2E_PortalBilling$'

# after-full (commit f52fc938): another empty browser dir and npm cache.
mkdir -p .agent/tmp/followup29/pw-full .agent/tmp/followup29/npm-full
PLAYWRIGHT_BROWSERS_PATH="$PWD/.agent/tmp/followup29/pw-full" \
npm_config_cache="$PWD/.agent/tmp/followup29/npm-full" \
FVPN_E2E_RUN_LABEL=followup29/after-full make -f mk/e2e.mk e2e-test
make -f mk/e2e.mk e2e-down
docker ps -a --format '{{.Names}}' | grep -cE '^fvpn-e2e($|-)'   # 0
```

`e2e-test` exits non-zero while any scenario is blocked. Check the summary line, not only the exit code. The nightly job sets `FVPN_E2E_RUN_LABEL=nightly` and uploads `docs/e2e/runs/*/nightly/*.{json,md}` as artifact `e2e-stack-records-<run_id>-<attempt>` for 7 days.

To read a nightly failure: download the artifact, open `acceptance-summary.md`, then the failing record. For `portal-billing-return`, `browser_probe.probe_error.stage` names the step (`import`, `launch`, `navigate`, `render`, ...). `browser_error` and `browser_stderr` cover a probe that could not report.
