# Local run log excerpts (follow-up 29)

Paths written as `<worktree>` / `<home>`. Full logs stay in the ignored `.agent/tmp/followup29/`.

## before-fail bring-up (fresh Portal build)

```text
added 482 packages in 3s
✓ built in 630ms
e2e stack is up
```

## before-fail focused run (commit 0e315f90)

```text
=== RUN   TestE2E_PortalBilling
    scenarios_test.go:131: scenario stack-health: pass
    scenarios_test.go:131: scenario admin-login: pass
    scenarios_test.go:131: scenario threat-feed-central: pass
    scenarios_test.go:131: scenario serving-catalogue: pass
    scenarios_test.go:131: scenario portal-billing-return: fail
    scenarios_test.go:131: scenario billing: fail
    scenarios_test.go:164: acceptance summary: pass=4 blocked=0 fail=2 run=bc302dbc-61fd-43a5-9e48-23cf8422f6c2 source=0e315f905b43695c42125e20f242655230c851bc
    scenarios_test.go:171: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_PortalBilling (3.59s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	3.601s
FAIL
```

## after-focus: e2e-browser into the same empty directory

```text
cd <worktree>/apps/portal && ./node_modules/.bin/playwright install --only-shell --no-remove chromium
Downloading Chrome Headless Shell 156.0.8078.4 (playwright chromium-headless-shell v1248) from https://cdn.playwright.dev/builds/cft/156.0.8078.4/linux64/chrome-headless-shell-linux64.zip
Chrome Headless Shell 156.0.8078.4 (playwright chromium-headless-shell v1248) downloaded to <worktree>/.agent/tmp/followup29/pw-before/chromium_headless_shell-1248
Downloading FFmpeg (playwright ffmpeg v1013) from https://cdn.playwright.dev/dbazure/download/playwright/builds/ffmpeg/1013/ffmpeg-linux.zip
FFmpeg (playwright ffmpeg v1013) downloaded to <worktree>/.agent/tmp/followup29/pw-before/ffmpeg-1013

real	0m5.521s
user	0m2.916s
sys	0m0.735s
```

## after-focus focused run (commit e956aa97)

```text
=== RUN   TestE2E_PortalBilling
    scenarios_test.go:131: scenario stack-health: pass
    scenarios_test.go:131: scenario admin-login: pass
    scenarios_test.go:131: scenario threat-feed-central: pass
    scenarios_test.go:131: scenario serving-catalogue: pass
    scenarios_test.go:131: scenario portal-billing-return: pass
    scenarios_test.go:131: scenario billing: pass
    scenarios_test.go:164: acceptance summary: pass=6 blocked=0 fail=0 run=2281ed26-1622-4630-8f42-56392b5b9db7 source=e956aa97f97715456a742f0cb1f1ba832f63da73
--- PASS: TestE2E_PortalBilling (6.82s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e	6.831s
```

## after-full e2e-test (commit f52fc938)

```text
added 482 packages in 2s
✓ built in 642ms
e2e stack is up
cd <worktree>/apps/portal && ./node_modules/.bin/playwright install --only-shell --no-remove --no-progress chromium
Downloading Chrome Headless Shell 156.0.8078.4 (playwright chromium-headless-shell v1248) from https://cdn.playwright.dev/builds/cft/156.0.8078.4/linux64/chrome-headless-shell-linux64.zip
Chrome Headless Shell 156.0.8078.4 (playwright chromium-headless-shell v1248) downloaded to <worktree>/.agent/tmp/followup29/pw-full/chromium_headless_shell-1248
Downloading FFmpeg (playwright ffmpeg v1013) from https://cdn.playwright.dev/dbazure/download/playwright/builds/ffmpeg/1013/ffmpeg-linux.zip
FFmpeg (playwright ffmpeg v1013) downloaded to <worktree>/.agent/tmp/followup29/pw-full/ffmpeg-1013
    scenarios_test.go:131: scenario stack-health: pass
    scenarios_test.go:131: scenario control-metrics: pass
    scenarios_test.go:131: scenario gateway-principals: pass
    scenarios_test.go:131: scenario admin-login: pass
    scenarios_test.go:131: scenario backup: pass
    scenarios_test.go:131: scenario threat-feed-central: pass
    scenarios_test.go:131: scenario serving-catalogue: pass
    scenarios_test.go:131: scenario portal-billing-return: pass
    scenarios_test.go:131: scenario billing: pass
    scenarios_test.go:131: scenario release-register: pass
    scenarios_test.go:131: scenario enroll-bootstrap: pass
    scenarios_test.go:131: scenario agent-supervisor: pass
    scenarios_test.go:131: scenario desired-state: pass
    scenarios_test.go:131: scenario threat-feed-node: pass
    scenarios_test.go:131: scenario node-restart: pass
    scenarios_test.go:131: scenario publish: pass
    scenarios_test.go:131: scenario forwarding: pass
    scenarios_test.go:131: scenario usage: pass
    scenarios_test.go:131: scenario member-traffic: pass
    scenarios_test.go:131: scenario established-transports: pass
    scenarios_test.go:131: scenario forced-relay: pass
    scenarios_test.go:131: scenario exit-dns: pass
    scenarios_test.go:131: scenario member-ipv6: blocked
    scenarios_test.go:131: scenario worker-restart: pass
    scenarios_test.go:131: scenario device-revoke: pass
    scenarios_test.go:131: scenario native-scale-hook: pass
    scenarios_test.go:131: scenario gateway-privacy: pass
    scenarios_test.go:131: scenario threat-feed-tamper: pass
    scenarios_test.go:131: scenario update-good: pass
    scenarios_test.go:131: scenario update-bad: pass
    scenarios_test.go:131: scenario lease-expiry: blocked
    scenarios_test.go:131: scenario revoke: pass
    scenarios_test.go:131: scenario second-exit: pass
    scenarios_test.go:164: acceptance summary: pass=31 blocked=2 fail=0 run=91eb0372-e42b-4698-a137-7cce53b263fa source=f52fc9381405d4aafd064dc0ef055977de97f418
    scenarios_test.go:171: one or more scenarios are blocked or failed; see docs/e2e/runs
--- FAIL: TestE2E_Acceptance (848.96s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/tests/e2e	857.661s
FAIL
make: *** [mk/e2e.mk:63: e2e-test] Error 1
real	15m17.152s
```

## teardown

```text
 Volume fvpn-e2e-nb Removed
 Volume fvpn-e2e-pg Removed
e2e stack is down
containers: 0
networks: 0
volumes: 0
image tags: 0
```
