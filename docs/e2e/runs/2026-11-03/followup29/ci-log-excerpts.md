# CI log excerpts (follow-up 29)

Read-only excerpts from `gh run view <id> -R TED-H-com/Freedom-VPN --log` (full logs, not `--log-failed`), read 2026-11-03. Timestamps are the runner's. ANSI colour codes are stripped and the runner home directory is written `<home>`. Every scheduled run of the `stack` job so far is listed; all four fail the same two scenarios within 0.41 to 0.71 s of `serving-catalogue`.

The stack job log has no Playwright line at all: no `playwright install`, no browser download, and the scenario record (which would name the failed check) was written inside the runner workspace and not uploaded.

## e2e stack run 41044684548

```text
2026-10-30T10:58:50.4049847Z Image: ubuntu-24.04
2026-10-30T10:58:50.4050427Z Version: 20261025.353.1
2026-10-30T11:01:17.7447032Z added 482 packages in 6s
2026-10-30T11:01:25.0959099Z ✓ built in 1.70s
2026-10-30T11:02:06.1646466Z     scenarios_test.go:103: scenario serving-catalogue: pass
2026-10-30T11:02:06.8438956Z     scenarios_test.go:103: scenario portal-billing-return: fail
2026-10-30T11:02:06.8444632Z     scenarios_test.go:103: scenario billing: fail
2026-10-30T11:08:16.6973234Z     scenarios_test.go:133: acceptance summary: pass=17 blocked=4 fail=2 run=a30e945c-94fa-493d-bd0b-df1b7d3dacdc source=ea860cc5c6076ab955adbd2e3ab590351735c29c
2026-10-30T11:08:16.7419954Z make: *** [mk/e2e.mk:51: e2e-test] Error 1
```

## e2e stack run 41193145343

```text
2026-10-31T10:32:20.3294803Z Image: ubuntu-24.04
2026-10-31T10:32:20.3295433Z Version: 20261025.353.1
2026-10-31T10:34:52.1550877Z added 482 packages in 6s
2026-10-31T10:34:59.7862888Z ✓ built in 1.60s
2026-10-31T10:35:43.8943571Z     scenarios_test.go:120: scenario serving-catalogue: pass
2026-10-31T10:35:44.5381493Z     scenarios_test.go:120: scenario portal-billing-return: fail
2026-10-31T10:35:44.5385902Z     scenarios_test.go:120: scenario billing: fail
2026-10-31T10:51:12.3280391Z     scenarios_test.go:153: acceptance summary: pass=27 blocked=4 fail=2 run=2c51c63d-e130-48b0-965d-4cae55a5ea19 source=8933ff2266079673e1e7f4473b8faff9732d279c
2026-10-31T10:51:12.4000778Z make: *** [mk/e2e.mk:51: e2e-test] Error 1
```

## e2e stack run 41301877131

```text
2026-11-01T11:26:20.5298024Z Image: ubuntu-24.04
2026-11-01T11:26:20.5298620Z Version: 20261025.353.1
2026-11-01T11:29:15.2824836Z added 482 packages in 7s
2026-11-01T11:29:23.3741380Z ✓ built in 1.98s
2026-11-01T11:30:10.3777163Z     scenarios_test.go:120: scenario serving-catalogue: pass
2026-11-01T11:30:11.0919570Z     scenarios_test.go:120: scenario portal-billing-return: fail
2026-11-01T11:30:11.0920922Z     scenarios_test.go:120: scenario billing: fail
2026-11-01T11:44:05.9630310Z     scenarios_test.go:153: acceptance summary: pass=29 blocked=2 fail=2 run=1ec5aadb-a3c1-4b42-85ca-b5feef44aded source=efa8304f24ddcdb125ce1714bbbbe755733c8a84
2026-11-01T11:44:06.0409268Z make: *** [mk/e2e.mk:51: e2e-test] Error 1
```

## e2e stack run 41447429149

```text
2026-11-02T12:46:33.1957667Z Image: ubuntu-24.04
2026-11-02T12:46:33.1958024Z Version: 20261025.353.1
2026-11-02T12:48:56.7120123Z added 482 packages in 4s
2026-11-02T12:49:01.4844962Z ✓ built in 1.09s
2026-11-02T12:49:48.8475669Z     scenarios_test.go:120: scenario serving-catalogue: pass
2026-11-02T12:49:49.2600000Z     scenarios_test.go:120: scenario portal-billing-return: fail
2026-11-02T12:49:49.2601236Z     scenarios_test.go:120: scenario billing: fail
2026-11-02T13:03:37.7497753Z     scenarios_test.go:153: acceptance summary: pass=29 blocked=2 fail=2 run=81e1def1-443b-4695-a5f4-1bae54bfce94 source=a6d56cff1f15ce4915e8664f97c27116b753ae62
2026-11-02T13:03:37.8408637Z make: *** [mk/e2e.mk:51: e2e-test] Error 1
```

## Portal workflow run 41303988690 (same runner image class)

```text
2026-11-01T11:55:33.9920125Z node: v24.22.1
2026-11-01T11:56:21.2337766Z ##[group]Run npx playwright install --with-deps chromium
2026-11-01T11:56:31.9901329Z Downloading Chrome for Testing 156.0.8078.4 (playwright chromium v1248) from https://cdn.playwright.dev/builds/cft/156.0.8078.4/linux64/chrome-linux64.zip
2026-11-01T11:56:36.5995291Z Chrome for Testing 156.0.8078.4 (playwright chromium v1248) downloaded to <home>/.cache/ms-playwright/chromium-1248
2026-11-01T11:56:37.1473985Z FFmpeg (playwright ffmpeg v1013) downloaded to <home>/.cache/ms-playwright/ffmpeg-1013
2026-11-01T11:56:37.1477224Z Downloading Chrome Headless Shell 156.0.8078.4 (playwright chromium-headless-shell v1248) from https://cdn.playwright.dev/builds/cft/156.0.8078.4/linux64/chrome-headless-shell-linux64.zip
2026-11-01T11:56:40.5084886Z Chrome Headless Shell 156.0.8078.4 (playwright chromium-headless-shell v1248) downloaded to <home>/.cache/ms-playwright/chromium_headless_shell-1248
```
