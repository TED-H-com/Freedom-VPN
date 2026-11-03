# Follow-up 29: nightly `stack` job red on `portal-billing-return` and `billing`

Date: 2026-11-03 (UTC). Lane `w3-devstack-e2e`. Read-only CI excerpts: [ci-log-excerpts.md](ci-log-excerpts.md).

## Cause

**Failed check: `browser_error`** (the Playwright probe), not `billing_return_http` or `built_portal_spa`. The probe could not launch a browser. The nightly runner has no Playwright Chromium, and the `stack` job never installs one. `billing()` calls `portal()` first, so `billing` fails on the same fault with `portal: return page probe failed`.

| Question | Answer and evidence |
|---|---|
| Which check? | `browser_error` at probe stage `launch`. Reproduced on this host under runner-like conditions: `billing_return_http 200`, `built_portal_spa true`, `browser_probe {"probe_error":{"stage":"launch","message":"browserType.launch: Executable doesn't exist at <worktree>/.agent/tmp/followup29/pw-before/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell"}}` ([before-fail record](before-fail/portal-billing-return.md)). |
| Why the runner differs | `deploy/devstack/probe-portal.mjs` imports Playwright from `apps/portal/node_modules` and calls `chromium.launch({headless: true})`. Playwright 1.64.1 (the Portal lockfile pin) then needs `chromium_headless_shell-1248` under `PLAYWRIGHT_BROWSERS_PATH` or `<home>/.cache/ms-playwright`. `npm ci` (run by `up.sh`) installs the npm package only, never a browser. This host's cache already holds `chromium_headless_shell-1248` from earlier Portal work, so every local run passed. A fresh `ubuntu-24.04` runner has none: the Portal workflow on the same image class downloads `chromium_headless_shell-1248` into `<home>/.cache/ms-playwright` on every run. The `stack` job has no `setup-node`, no `playwright install` step, and its log contains no Playwright line. `at-members.yml` added `npx playwright install chromium` for this same probe on 2026-10-30; `e2e.yml` never did. |
| What the runner did have | The Portal build (`added 482 packages`, `vite build`, `✓ built in 1.09s`; `up.sh` then requires `dist/index.html`), Node from the runner image, the gateway, the `/billing/return` route and the member cookie jar: `stack-health`, `admin-login`, `threat-feed-central` and `serving-catalogue` pass immediately before. |
| Did the CI logs alone settle it? | **Not directly.** The record that names the check was written inside the runner workspace and never uploaded, and the probe's stderr was discarded (`browser.Output()` kept stdout only). Supporting facts from the logs: all **four** scheduled runs (2026-10-30, -31, 11-01, 11-02; not two) fail exactly these two scenarios, 0.41–0.71 s after `serving-catalogue` passes. That window includes a member OIDC login, the HTTP fetch and a Node start. A real page render takes longer (the passing focused run on this host takes 6.82 s for six scenarios); a launch that fails on a missing executable does not. The local reproduction under the same missing-browser condition gives the same two failures. |

Best supported conclusion: the nightly failure is the missing Playwright headless shell. The diagnostics added here make the next CI run state it directly.

## Fix

- `mk/e2e.mk`: new `e2e-browser` target. It runs `apps/portal/node_modules/.bin/playwright install --only-shell --no-remove --no-progress chromium`. It installs exactly the revision pinned by the Portal lockfile, does nothing when that revision is cached, honours `PLAYWRIGHT_BROWSERS_PATH`, keeps all other cached revisions (`--no-remove`, the host cache is shared) and never uses sudo. `e2e-test` now runs `e2e-down`, `e2e-up E2E_RESET=1`, `e2e-browser`, then the tests.
- `.github/workflows/e2e.yml` `stack` job: `actions/setup-node@v4` with Node 24 (same as `portal.yml` and this host's v24.22.1). The job installs the headless shell with its system libraries (`npm ci` then `npx playwright install --with-deps --only-shell chromium`; sudo is the runner's, not this host's) and sets `FVPN_E2E_RUN_LABEL=nightly`.
- No assertion is weakened. A page that does not render, an API error or a non-200 status still fails, and so does a missing browser.

## Diagnostics for the next failure

- `probe-portal.mjs` reports `{"probe_error":{"stage","message"}}` on stdout for any failure. Stages are `input`, `import`, `launch`, `context`, `navigate` and `render`. The message is one line of at most 400 characters. Cookie values, the worktree and the home directory are removed before it is printed. The success output is unchanged; `tests/e2e/at/members` decodes the same fields.
- `portal()` records `billing_return_error` when the HTTP fetch fails, the process cause in `browser_error` (`exit status 1`, `probe timed out after 40s`, or a missing `node`), and `browser_stderr` for a probe that could not report. That line is first scrubbed of cookie values and paths (`ProbeFailureDetail`, unit-tested in `TestSEC02_ProbeFailureDetailRedactsCookiesAndPaths`).
- The acceptance runner writes `acceptance-summary.md` (summary line and every scenario status) next to the records.
- The `stack` job uploads `docs/e2e/runs/*/nightly/*.json` and `*.md` with `if: always()` as `e2e-stack-records-<run_id>-<attempt>`, retention 7 days. That directory holds only `Write()` records (observations already pass `Redact`) and the summary. Stack state, secrets, the CA and the cookie jar live under `.agent/` and are not matched. No committed directory is named `nightly`, so the artifact is that run's files only.

## Proof on this host

Runner-like conditions: `PLAYWRIGHT_BROWSERS_PATH` pointed at an empty directory under `.agent/tmp/followup29`, `npm_config_cache` likewise empty, a fresh `E2E_RESET=1` stack, and a fresh Portal `npm ci` plus build from `up.sh` (`added 482 packages`, `✓ built`).

| Step | Commit | Result |
|---|---|---|
| before-fail: fresh stack, empty browser dir, focused `TestE2E_PortalBilling` without `e2e-browser` | `0e315f90` (diagnostics only, no fix) | `pass=4 blocked=0 fail=2`; `portal-billing-return` fail at stage `launch`, `billing` fail (`portal: return page probe failed`). [records](before-fail/) |
| after-focus: new fresh stack, same (still empty) browser dir, `make -f mk/e2e.mk e2e-browser`, focused rerun | `e956aa97` | `e2e-browser` downloaded `chromium_headless_shell-1248` into the empty directory; `pass=6 blocked=0 fail=0`; `browser_probe {"http_status":200,"billing_page_visible":true,"api_error":false,"pending_confirmation":true}`; billing `invoice_rows_after_duplicate_deliveries 1`, `immutable_refund_rows 2`. [records](after-focus/) |
| after-full: one `make -f mk/e2e.mk e2e-test`, another empty browser dir and npm cache | `f52fc938` | `e2e-test` itself downloaded `chromium_headless_shell-1248` into the empty directory between `e2e-up` and the tests. Result `pass=31 blocked=2 fail=0`: `portal-billing-return` and `billing` pass with the same observations as after-focus, and `member-ipv6` / `lease-expiry` stay blocked. The make exit is 2 because blocked is non-green. Teardown left 0 owned containers, networks, volumes and `:run` tags. [summary](after-full/acceptance-summary.md) |

The after-full blocked owners are unchanged: `member-ipv6` is "w3-devstack-e2e / platform operators: dual-stack native path and per-platform leak measurements"; `lease-expiry` is "w2-agent-core: device admission counters absent after bounded readiness poll".

Not reproduced here: system-library gaps. This host already has Chromium's shared libraries, and a fresh runner may lack some. The workflow's `--with-deps` covers that. `e2e-browser` deliberately does not, because it must never use sudo on a shared host.

## What still keeps the nightly red

The two `fail` rows are fixed. **`member-ipv6` and `lease-expiry` stay `blocked` by design** until their owner rows exist. A blocked scenario still fails `TestE2E_Acceptance`, so the `stack` job remains red on those two. That semantics is unchanged here.
