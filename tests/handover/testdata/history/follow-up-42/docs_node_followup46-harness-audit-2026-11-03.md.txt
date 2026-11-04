# Follow-up 46 harness audit — 2026-11-03

TEST ONLY. A and B are the unchanged 43/44/45 arms. The JSON companion pins
all frozen fixtures, reused archives, overlays, inherited tools and new tools.
A's 18,889 and B's 18,939 extracted tree blobs match their declared git objects;
archive and overlay SHA-256s match 43. Every frozen fixture also matches 45.
No production file changes. The exact 45 production diff was rerun once:

```sh
git diff 38420dadef6ae9d0236288eef533bc1ddee0da53 28e9e41c -- '*.go' ':!**/*_test.go' ':!tests/**' ':!docs/**'
```

Exit 0, stdout empty (zero bytes). B equals #429 for this production-Go scope.

## Frozen output consumers

File references below are relative to **each frozen arm**, not current main.
Both arms have identical reader locations unless explicitly shown in the full
call-site inventory in the JSON. All calls through the three Go Docker helpers
use the capture modes defined here; the inventory also includes discarded
outputs so no call is hidden. Shell captures inside containers run container
commands, not the host admission wrappers.

| File and line (both arms) | Mode | Expected bytes or contract |
| --- | --- | --- |
| tests/netlab/agent/harness_test.go:57 | stdout only, Go Output | Exact artifact bytes: operations JSONL, agent/watchdog logs; saved unchanged |
| tests/netlab/agent/harness_test.go:80 | combined, Go CombinedOutput | Whitespace-trimmed output exactly `true` |
| tests/netlab/agent/harness_test.go:90 | combined, Go CombinedOutput | Whitespace-trimmed output exactly `true`; this failed in 45 |
| tests/netlab/agent/harness_test.go:228 (buffers 229–234) | stdout only for data; separate stderr for errors | dockerExec: stdout returned unchanged, stderr diagnostic only |
| tests/netlab/agent/harness_test.go:237 (buffers 239–244) | stdout only for data; separate stderr for errors | dockerExecStdin: tee echoes input or command output; never admission text |
| tests/netlab/agent/harness_test.go:247 (249–252) | stderr only diagnostic; stdout discarded | dockerDetach: exit status; stderr only included on error |
| tests/netlab/agent/scenario_ac_timing_test.go:132 | stdout only, Go Output | Exact assembled operations/packet/log artifacts, saved unchanged |
| tests/netlab/agent/scenario_ac_timing_test.go:170 | combined, Go CombinedOutput; Python is not wrapped | Analyzer summary or assertion traceback, diagnostic plus exit status |
| tests/netlab/agent/run-installer.py:29–30 | stdout only, Python PIPE | Dormant installer adapter maps object names in Docker stdout; not invoked by this leg |
| mk/node.mk:64–84 | inherited combined suite log; status-only shells | Build/make/Docker output logged; exit statuses control progression and cleanup |
| mk/netlab.mk:46–71,74–76 | inherited combined suite log or explicit discard | Build/image/up/down status; Docker network inspect output discarded; no parsed capture |

The helper callers expecting data are: harness_test.go:114 (control URL),
133/141 (nft text and owner comment; optional D116), 158/168 (matching key
hashes), 258/275 (caller-selected command/file bytes), 288 (redacted logs),
346 (probe JSON), 407 (HTTP response JSON/text), 474 (nft diagnostic);
scenario_ac_timing_test.go:143 (decimal monotonic deadline through mustExec);
scenario_at_usage_test.go:263 (usage JSON); scenario_az_external_test.go:97
(nft rule text), 139 (epoch JSON); scenario_health_test.go:97 (attestation JSON);
scenario_f31_test.go:29/41 (snapshot JSON/artifact bytes);
scenario_f36_test.go:44/49 (restart proof diagnostic/result JSON).
The complete per-arm inventory below includes every helper call and indirect
readScratch/mustExec consumer, including calls that only inspect exit status.

No host `$(docker ...)`, `$(go ...)` or `$(make ...)` capture occurs on the
node-test-netlab path. Shell capture is still tested explicitly. Container
shell captures in topology.sh:136–140,149 read public fixture keys, and :233
reads a handshake count; scenario shell PID captures read local pidfiles.
They do not run host wrappers. run-live.py is dormant and inherits streams or
discards cleanup output; it does not parse captured Docker output.

## Driver and wrapper consumers

The 46 wrapper additionally reads Docker stdout for the image ID, clock JSON,
span JSONL and interrupted artifacts. The driver reads Docker stdout for
container/image/network/volume inventories and Docker event JSONL. These all
require unmodified stdout and retain separate diagnostics. The driver captures
admission stdout/stderr itself and writes command, start/end UTC, output, exit
and admitted flag into each leg's admission.jsonl. Successful wrappers emit no
admission bytes on either stream. Refusal writes a receipt and exits 125; the
leg monitor also stops on any refused receipt even if a consumer ignores exit.
Real tool stderr and real tool exits remain unchanged.

## Synthetic evidence

Actual executable wrapper composition uses TEST ONLY fake admission and Docker
executables under repository disk scratch. For combined reads 45 yields
`free\ntrue\n`, failing the frozen exact-true contract; 46 yields `true\n`.
For stdout-only and shell captures both versions preserve the data bytes, but
45 leaks `free\n` on stderr while 46 is silent. Tests cover true, JSON, multiline
bytes, preserved tool diagnostics, and refusal for Docker, make and Go test.
The JSON stores actual probe bytes. These probes use no Docker or network.

The ancestor test excludes runner, parent, grandparent and PID 1 from suite
matching. An unrelated process with identical suite-literal text is still
foreign and disqualifies. CPU classification is untouched, including Codex
ancestor CPU. All 65 new/inherited synthetic tests pass; the 24 new tests also
pass when launched from a different working directory, using file-relative
.agent/tmp scratch. Clock, marks/top-five, stop conditions, five span filters,
half-Step threshold and D087/D104 tests are retained. Additional dry tests
reject missing spans/scenarios/receipts, an old hash, a missing arm or a failure;
dry rows can never pool.

## Exact standalone invocation

Set F46_RUN to dry1 (or dry2/dry3 only after a harness fix), then window1 or
window2. The same foreground command is used for every run:

```sh
python3 -B tests/netlab/agent/followup46_timing.py run > .agent/f46/timing/driver-"$F46_RUN".stdout 2> .agent/f46/timing/driver-"$F46_RUN".stderr
```

Its command text contains no suite-pattern literal. Dry and measured paths
both call run_leg with the same full node-test-netlab, wrappers, frozen trees,
overlays, clock and monitoring; dry merely has no reservation deadline and
cannot pool. Never reserve until this runner hash passes both full dry legs.
Dry runs, including failures, stay under fresh .agent/f46/timing/ paths.

## Complete per-arm consumer call inventory

Each `dockerExec`, `dockerExecStdin`, `mustExec` and `readScratch` call below
uses stdout-only data and separate diagnostic stderr. `dockerDetach` reads
stderr only on error. Direct exec calls use the modes listed above. Calls
assigning `_` to both streams require only status; remaining expectations are
the exact command/file content shown, never an admission token.

| Arm | File:line | Consumer |
| --- | --- | --- |
| A | tests/netlab/agent/harness_test.go:57 | `raw, err := exec.Command("docker", "exec", prefix(), "cat", labDir+"/"+name).Output()` |
| A | tests/netlab/agent/harness_test.go:80 | `out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", prefix()).CombinedOutput()` |
| A | tests/netlab/agent/harness_test.go:90 | `out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", prefix()).CombinedOutput()` |
| A | tests/netlab/agent/harness_test.go:94 | `_, _, _ = dockerExec("sh", "-c", "mkdir -p "+labDir+"/state "+labDir+"/identity "+labDir+"/journal")` |
| A | tests/netlab/agent/harness_test.go:95 | `if _, _, err := dockerExec("chmod", "700", labDir+"/journal"); err != nil {` |
| A | tests/netlab/agent/harness_test.go:98 | `if _, _, err := dockerExec("python3", "/work/tests/netlab/agent/topology-owner.py", "capture"); err != nil {` |
| A | tests/netlab/agent/harness_test.go:101 | `if err := dockerDetach("sh", "-c", "exec fvpn-labcontrol -dir "+labDir+" -admin 127.0.0.1:9091 >"+labDir+"/labcontrol.log 2>&1"); err != nil {` |
| A | tests/netlab/agent/harness_test.go:106 | `if _, _, err := dockerExec("test", "-f", labDir+"/ready"); err == nil {` |
| A | tests/netlab/agent/harness_test.go:111 | `if _, _, err := dockerExec("test", "-f", labDir+"/ready"); err != nil {` |
| A | tests/netlab/agent/harness_test.go:114 | `urlBytes, errOut, err := dockerExec("cat", labDir+"/url")` |
| A | tests/netlab/agent/harness_test.go:127 | `if _, errOut, err := dockerExecStdin([]string{"nft", "-f", "-"}, marker); err != nil {` |
| A | tests/netlab/agent/harness_test.go:133 | `baseline, errOut, err := dockerExec("nft", "list", "table", "inet", "fvpn")` |
| A | tests/netlab/agent/harness_test.go:138 | `if _, errOut, err := dockerExecStdin([]string{"nft", "-f", "-"}, []byte("delete table inet fvpn\n"+baseline)); err != nil {` |
| A | tests/netlab/agent/harness_test.go:141 | `text, errOut, err := dockerExec("nft", "list", "table", "inet", "fvpn")` |
| A | tests/netlab/agent/harness_test.go:145 | `if _, errOut, err := dockerExecStdin([]string{"tee", labDir + "/before-enrollment.txt"}, []byte(text)); err != nil {` |
| A | tests/netlab/agent/harness_test.go:148 | `if _, errOut, err := dockerExec("sh", "-c", "nft -j list ruleset >"+labDir+"/before-enrollment.json"); err != nil {` |
| A | tests/netlab/agent/harness_test.go:152 | `if out, errOut, err := dockerExec("fvpn-agent", "enroll", "-config", labDir+"/agent.json", "-token-file", labDir+"/token"); err != nil {` |
| A | tests/netlab/agent/harness_test.go:155 | `if _, _, err := dockerExec("test", "!", "-f", labDir+"/token"); err != nil {` |
| A | tests/netlab/agent/harness_test.go:158 | `keyBefore, errOut, err := dockerExec("sha256sum", labDir+"/identity/key.pem")` |
| A | tests/netlab/agent/harness_test.go:162 | `if _, errOut, err := dockerExecStdin([]string{"tee", labDir + "/token2"}, []byte("other-token\n")); err != nil {` |
| A | tests/netlab/agent/harness_test.go:165 | `if out, errOut, err := dockerExec("fvpn-agent", "enroll", "-config", labDir+"/agent.json", "-token-file", labDir+"/token2"); err != nil {` |
| A | tests/netlab/agent/harness_test.go:168 | `keyAfter, _, _ := dockerExec("sha256sum", labDir+"/identity/key.pem")` |
| A | tests/netlab/agent/harness_test.go:213 | `if _, errOut, err := dockerExecStdin([]string{"tee", labDir + "/netbird-status.json"}, status); err != nil {` |
| A | tests/netlab/agent/harness_test.go:220 | `_, errOut, err := dockerExecStdin([]string{"tee", labDir + "/agent.json"}, append(raw, '\n'))` |
| A | tests/netlab/agent/harness_test.go:228 | `cmd := exec.Command("docker", append([]string{"exec", prefix()}, args...)...)` |
| A | tests/netlab/agent/harness_test.go:237 | `cmd := exec.Command("docker", append([]string{"exec", "-i", prefix()}, args...)...)` |
| A | tests/netlab/agent/harness_test.go:247 | `cmd := exec.Command("docker", append([]string{"exec", "-d", prefix()}, args...)...)` |
| A | tests/netlab/agent/harness_test.go:258 | `out, errOut, err := dockerExec(args...)` |
| A | tests/netlab/agent/harness_test.go:267 | `out, errOut, err := dockerExecStdin([]string{"tee", path}, body)` |
| A | tests/netlab/agent/harness_test.go:275 | `out, errOut, err := dockerExec("cat", path)` |
| A | tests/netlab/agent/harness_test.go:283 | `_, _, err := dockerExec("test", "-f", path)` |
| A | tests/netlab/agent/harness_test.go:288 | `out, errOut, err := dockerExec("sh", "-c", "tail -n 60 "+labDir+"/agent.log "+labDir+"/labcontrol.log 2>/dev/null &#124;&#124; true")` |
| A | tests/netlab/agent/harness_test.go:306 | `body := readScratch(t, labDir+"/agent.log")` |
| A | tests/netlab/agent/harness_test.go:346 | `out, errOut, err := dockerExec("python3", "/opt/netlab/scripts/probe.py", labDir+"/cases.json")` |
| A | tests/netlab/agent/harness_test.go:398 | `mustExec(t, "python3", labDir+"/admin.py")` |
| A | tests/netlab/agent/harness_test.go:407 | `out, _, err := dockerExec("python3", labDir+"/admin_get.py")` |
| A | tests/netlab/agent/harness_test.go:420 | `_, _, _ = dockerExec("sh", "-c", fmt.Sprintf(` |
| A | tests/netlab/agent/harness_test.go:430 | `if err := dockerDetach("sh", "-c", script); err != nil {` |
| A | tests/netlab/agent/harness_test.go:437 | `_, _, _ = dockerExec("python3", "/work/tests/netlab/agent/topology-owner.py", "stop")` |
| A | tests/netlab/agent/harness_test.go:438 | `if err := dockerDetach("python3", "/work/tests/netlab/agent/topology-owner.py", "run"); err != nil {` |
| A | tests/netlab/agent/harness_test.go:447 | `_, _, err := dockerExec("python3", "-c", import time` |
| A | tests/netlab/agent/harness_test.go:462 | `_, _, _ = dockerExec("python3", "/work/tests/netlab/agent/topology-owner.py", "stop")` |
| A | tests/netlab/agent/harness_test.go:468 | `_, _, _ = dockerExec("conntrack", "-F")` |
| A | tests/netlab/agent/harness_test.go:469 | `_, _, _ = dockerExec("conntrack", "-F", "-f", "ipv6")` |
| A | tests/netlab/agent/harness_test.go:474 | `out, _, _ := dockerExec("nft", "list", "table", "inet", "fvpn")` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:56 | `if err := json.Unmarshal([]byte(mustExec(t, "ip", "-j", "addr", "show", "dev", "fv-fh")), &addresses); err != nil {` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:63 | `if err := json.Unmarshal([]byte(mustExec(t, "ip", "-j", family, "route", "show", "dev", "fv-fh")), &rows); err != nil {` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:72 | `mustExec(t, "ip", "-6", "addr", "replace", fmt.Sprintf("%s/%d", a.Local, a.PrefixLen), "dev", "fv-fh", "nodad")` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:79 | `mustExec(t, "ip", family, "route", "replace", route.Dst, "via", route.Gateway, "dev", "fv-fh")` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:88 | `mustExec(t, "ip", "link", "set", "fv-fh", "up")` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:92 | `mustExec(t, "tc", "qdisc", "add", "dev", "fv-fh", "clsact")` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:93 | `mustExec(t, "tc", "filter", "add", "dev", "fv-fh", "ingress", "pref", "1", "matchall", "action", "mirred", "egress", "redirect", "dev", "fv-ih")` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:94 | `defer func() { _, _, _ = dockerExec("tc", "qdisc", "del", "dev", "fv-fh", "clsact") }()` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:107 | `link := mustExec(t, "ip", "-j", "link", "show", "dev", "fv-fh")` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:119 | `mustExec(t, "tc", "filter", "del", "dev", "fv-fh", "ingress", "pref", "1")` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:120 | `mustExec(t, "tc", "qdisc", "del", "dev", "fv-fh", "clsact")` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:124 | `if strings.Contains(mustExec(t, "ip", "-j", "link", "show", "dev", "fv-fh"), "UP") {` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:129 | `if !strings.Contains(mustExec(t, "ip", "-j", "link", "show", "dev", "fv-fh"), "UP") {` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:139 | `t.Log("member addresses", mustExec(t, "ip", "-j", "addr", "show", "dev", "fv-fh"))` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:140 | `t.Log("member routes", mustExec(t, "ip", "-j", "-6", "route", "show", "table", "all", "dev", "fv-fh"))` |
| A | tests/netlab/agent/scenario_ab_recovery_test.go:157 | `leaseGen := strings.TrimSpace(mustExec(t, "python3", "-c", import sqlite3,json` |
| A | tests/netlab/agent/scenario_ac_timing_test.go:23 | `original := readScratch(t, labDir+"/agent.json")` |
| A | tests/netlab/agent/scenario_ac_timing_test.go:132 | `body, err := exec.Command("docker", "exec", prefix(), "cat", labDir+"/"+name).Output()` |
| A | tests/netlab/agent/scenario_ac_timing_test.go:145 | `mustExec(t, "python3", "/work/tests/netlab/agent/timing-probes.py", "trace", "renewal", "0", labDir+"/renewal.json")` |
| A | tests/netlab/agent/scenario_ac_timing_test.go:148 | `d := mustExec(t, "python3", "-c", import sqlite3,json` |
| A | tests/netlab/agent/scenario_ac_timing_test.go:160 | `mustExec(t, "python3", "/work/tests/netlab/agent/timing-probes.py", "trace", "expiry_restart", fmt.Sprint(mono), labDir+"/expiry.json")` |
| A | tests/netlab/agent/scenario_ac_timing_test.go:170 | `summary, err := exec.Command("python3", "timing-analysis.py", outDir).CombinedOutput()` |
| A | tests/netlab/agent/scenario_at_usage_test.go:57 | `mustExec(t, "rm", "-f", labDir+"/journal/forwarding.blocked")` |
| A | tests/netlab/agent/scenario_at_usage_test.go:136 | `_, _, _ = dockerExec("sh", "-c", script)` |
| A | tests/netlab/agent/scenario_at_usage_test.go:149 | `mustExec(t, "ip", "netns", "exec", "fv-a1", "python3", labDir+"/udpburst.py")` |
| A | tests/netlab/agent/scenario_at_usage_test.go:157 | `return len(readScratch(t, labDir+"/agent.log"))` |
| A | tests/netlab/agent/scenario_at_usage_test.go:168 | `body := readScratch(t, labDir+"/agent.log")` |
| A | tests/netlab/agent/scenario_at_usage_test.go:199 | `out := strings.TrimSpace(mustExec(t, "python3", labDir+"/nft_usage.py"))` |
| A | tests/netlab/agent/scenario_at_usage_test.go:263 | `out, errOut, err := dockerExec("python3", labDir+"/usage_read.py")` |
| A | tests/netlab/agent/scenario_az_external_test.go:69 | `mustExec(t, "nft", "flush", "chain", "inet", "fvpn", "forward")` |
| A | tests/netlab/agent/scenario_az_external_test.go:97 | `out, _, err := dockerExec("nft", "list", "chain", "inet", "fvpn", "forward")` |
| A | tests/netlab/agent/scenario_az_external_test.go:139 | `out, errOut, err := dockerExec("python3", labDir+"/usage_epoch.py")` |
| A | tests/netlab/agent/scenario_d116_test.go:64 | `saveD116(t, "before-enrollment.json", readScratch(t, labDir+"/before-enrollment.json"))` |
| A | tests/netlab/agent/scenario_d116_test.go:65 | `saveD116(t, "before-enrollment.txt", readScratch(t, labDir+"/before-enrollment.txt"))` |
| A | tests/netlab/agent/scenario_d116_test.go:69 | `if err := dockerDetach("sh", "-c", script); err != nil {` |
| A | tests/netlab/agent/scenario_d116_test.go:75 | `saveD116(t, "agent.txt", readScratch(t, labDir+"/agent.log"))` |
| A | tests/netlab/agent/scenario_d116_test.go:83 | `mustExec(t, "nft", "delete", "table", "inet", "fvpn_install")` |
| A | tests/netlab/agent/scenario_d116_test.go:91 | `mustExec(t, "nft", "add", "table", "inet", "fvpn_install")` |
| A | tests/netlab/agent/scenario_d116_test.go:92 | `mustExec(t, "nft", "add", "chain", "inet", "fvpn_install", "owner", { comment "managed-by=fvpn-node-install"; })` |
| A | tests/netlab/agent/scenario_d116_test.go:93 | `mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "start", "/scratch/agent-at10")` |
| A | tests/netlab/agent/scenario_d116_test.go:94 | `if err := dockerDetach("ip", "netns", "exec", "fv-a1", "python3", "/opt/netlab/scripts/stream.py", "203.0.113.10", "9001", "9000", "/scratch/agent-stream.log"); err != nil {` |
| A | tests/netlab/agent/scenario_d116_test.go:99 | `mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "finish-stream", "/scratch/agent-stream.log")` |
| A | tests/netlab/agent/scenario_d116_test.go:100 | `mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "stop", "/scratch/agent-at10")` |
| A | tests/netlab/agent/scenario_d116_test.go:103 | `saveD116(t, strings.ReplaceAll(name, ".log", ".txt"), readScratch(t, "/scratch/"+name))` |
| A | tests/netlab/agent/scenario_d116_test.go:107 | `for !scratchExists("/scratch/agent-stream.log") &#124;&#124; !strings.Contains(readScratch(t, "/scratch/agent-stream.log"), " ok") {` |
| A | tests/netlab/agent/scenario_d116_test.go:125 | `raw := mustExec(t, "nft", "-j", "list", "ruleset")` |
| A | tests/netlab/agent/scenario_d116_test.go:147 | `text := mustExec(t, "nft", "list", "table", "inet", "fvpn")` |
| A | tests/netlab/agent/scenario_f31_test.go:29 | `raw, errOut, err := dockerExec("python3", "-c", followup31RestartSnapshot)` |
| A | tests/netlab/agent/scenario_f31_test.go:41 | `raw, _, err := dockerExec("cat", labDir+"/"+name)` |
| A | tests/netlab/agent/scenario_f36_test.go:20 | `original := readScratch(t, labDir+"/agent.json")` |
| A | tests/netlab/agent/scenario_f36_test.go:44 | `out, errOut, err := dockerExec("python3", "/work/tests/netlab/agent/restart-sets.py")` |
| A | tests/netlab/agent/scenario_f36_test.go:49 | `raw, _, readErr := dockerExec("cat", labDir+"/f36-result.json")` |
| A | tests/netlab/agent/scenario_health_test.go:31 | `_, _, _ = dockerExec("rm", "-f", labDir+"/journal/health-challenge.json", labDir+"/journal/health-attest.json")` |
| A | tests/netlab/agent/scenario_health_test.go:57 | `bootID := strings.ToLower(strings.TrimSpace(mustExec(t, "cat", "/proc/sys/kernel/random/boot_id")))` |
| A | tests/netlab/agent/scenario_health_test.go:86 | `mustExec(t, "chmod", "600", challengePath)` |
| A | tests/netlab/agent/scenario_health_test.go:87 | `if mode := strings.TrimSpace(mustExec(t, "stat", "-c", "%a", challengePath)); mode != "600" {` |
| A | tests/netlab/agent/scenario_health_test.go:97 | `out, _, err := dockerExec("cat", attestPath)` |
| A | tests/netlab/agent/scenario_health_test.go:108 | `if mode := strings.TrimSpace(mustExec(t, "stat", "-c", "%a", attestPath)); mode != "600" {` |
| A | tests/netlab/agent/scenario_health_test.go:158 | `out := strings.TrimSpace(mustExec(t, "python3", labDir+"/identity_ids.py"))` |
| A | tests/netlab/agent/scenario_test.go:41 | `mode := strings.Fields(mustExec(t, "stat", "-c", "%a", labDir+"/identity/setup.key"))` |
| A | tests/netlab/agent/scenario_test.go:115 | `_, _, _ = dockerExec("sh", "-c", fmt.Sprintf(` |
| A | tests/netlab/agent/scenario_test.go:139 | `t.Log("TEST ONLY graceful restart stop:", mustExec(t, "python3", "/work/tests/netlab/agent/timing-probes.py", "stop", labDir+"/restart-stop.json", "10"))` |
| A | tests/netlab/agent/scenario_test.go:140 | `mustExec(t, "python3", "/work/tests/netlab/agent/topology-owner.py", "stop")` |
| A | tests/netlab/agent/scenario_test.go:171 | `_, _, _ = dockerExec("sh", "-c", "rm -f /scratch/denyloop.stop /scratch/deny-hits.log "+labDir+"/deny-cases.json")` |
| A | tests/netlab/agent/scenario_test.go:184 | `if err := dockerDetach("python3", "/opt/netlab/scripts/denyloop.py", "/scratch/deny-cases.json"); err != nil {` |
| A | tests/netlab/agent/scenario_test.go:188 | `_, _, _ = dockerExec("sh", "-c", "touch /scratch/denyloop.stop; if [ -f /scratch/denyloop.pid ]; then kill $(cat /scratch/denyloop.pid) 2>/dev/null &#124;&#124; true; fi")` |
| A | tests/netlab/agent/scenario_test.go:195 | `_, _, _ = dockerExec("sh", "-c", "touch /scratch/denyloop.stop")` |
| A | tests/netlab/agent/scenario_test.go:198 | `hits := readScratch(t, "/scratch/deny-hits.log")` |
| B | tests/netlab/agent/harness_test.go:57 | `raw, err := exec.Command("docker", "exec", prefix(), "cat", labDir+"/"+name).Output()` |
| B | tests/netlab/agent/harness_test.go:80 | `out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", prefix()).CombinedOutput()` |
| B | tests/netlab/agent/harness_test.go:90 | `out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", prefix()).CombinedOutput()` |
| B | tests/netlab/agent/harness_test.go:94 | `_, _, _ = dockerExec("sh", "-c", "mkdir -p "+labDir+"/state "+labDir+"/identity "+labDir+"/journal")` |
| B | tests/netlab/agent/harness_test.go:95 | `if _, _, err := dockerExec("chmod", "700", labDir+"/journal"); err != nil {` |
| B | tests/netlab/agent/harness_test.go:98 | `if _, _, err := dockerExec("python3", "/work/tests/netlab/agent/topology-owner.py", "capture"); err != nil {` |
| B | tests/netlab/agent/harness_test.go:101 | `if err := dockerDetach("sh", "-c", "exec fvpn-labcontrol -dir "+labDir+" -admin 127.0.0.1:9091 >"+labDir+"/labcontrol.log 2>&1"); err != nil {` |
| B | tests/netlab/agent/harness_test.go:106 | `if _, _, err := dockerExec("test", "-f", labDir+"/ready"); err == nil {` |
| B | tests/netlab/agent/harness_test.go:111 | `if _, _, err := dockerExec("test", "-f", labDir+"/ready"); err != nil {` |
| B | tests/netlab/agent/harness_test.go:114 | `urlBytes, errOut, err := dockerExec("cat", labDir+"/url")` |
| B | tests/netlab/agent/harness_test.go:127 | `if _, errOut, err := dockerExecStdin([]string{"nft", "-f", "-"}, marker); err != nil {` |
| B | tests/netlab/agent/harness_test.go:133 | `baseline, errOut, err := dockerExec("nft", "list", "table", "inet", "fvpn")` |
| B | tests/netlab/agent/harness_test.go:138 | `if _, errOut, err := dockerExecStdin([]string{"nft", "-f", "-"}, []byte("delete table inet fvpn\n"+baseline)); err != nil {` |
| B | tests/netlab/agent/harness_test.go:141 | `text, errOut, err := dockerExec("nft", "list", "table", "inet", "fvpn")` |
| B | tests/netlab/agent/harness_test.go:145 | `if _, errOut, err := dockerExecStdin([]string{"tee", labDir + "/before-enrollment.txt"}, []byte(text)); err != nil {` |
| B | tests/netlab/agent/harness_test.go:148 | `if _, errOut, err := dockerExec("sh", "-c", "nft -j list ruleset >"+labDir+"/before-enrollment.json"); err != nil {` |
| B | tests/netlab/agent/harness_test.go:152 | `if out, errOut, err := dockerExec("fvpn-agent", "enroll", "-config", labDir+"/agent.json", "-token-file", labDir+"/token"); err != nil {` |
| B | tests/netlab/agent/harness_test.go:155 | `if _, _, err := dockerExec("test", "!", "-f", labDir+"/token"); err != nil {` |
| B | tests/netlab/agent/harness_test.go:158 | `keyBefore, errOut, err := dockerExec("sha256sum", labDir+"/identity/key.pem")` |
| B | tests/netlab/agent/harness_test.go:162 | `if _, errOut, err := dockerExecStdin([]string{"tee", labDir + "/token2"}, []byte("other-token\n")); err != nil {` |
| B | tests/netlab/agent/harness_test.go:165 | `if out, errOut, err := dockerExec("fvpn-agent", "enroll", "-config", labDir+"/agent.json", "-token-file", labDir+"/token2"); err != nil {` |
| B | tests/netlab/agent/harness_test.go:168 | `keyAfter, _, _ := dockerExec("sha256sum", labDir+"/identity/key.pem")` |
| B | tests/netlab/agent/harness_test.go:213 | `if _, errOut, err := dockerExecStdin([]string{"tee", labDir + "/netbird-status.json"}, status); err != nil {` |
| B | tests/netlab/agent/harness_test.go:220 | `_, errOut, err := dockerExecStdin([]string{"tee", labDir + "/agent.json"}, append(raw, '\n'))` |
| B | tests/netlab/agent/harness_test.go:228 | `cmd := exec.Command("docker", append([]string{"exec", prefix()}, args...)...)` |
| B | tests/netlab/agent/harness_test.go:237 | `cmd := exec.Command("docker", append([]string{"exec", "-i", prefix()}, args...)...)` |
| B | tests/netlab/agent/harness_test.go:247 | `cmd := exec.Command("docker", append([]string{"exec", "-d", prefix()}, args...)...)` |
| B | tests/netlab/agent/harness_test.go:258 | `out, errOut, err := dockerExec(args...)` |
| B | tests/netlab/agent/harness_test.go:267 | `out, errOut, err := dockerExecStdin([]string{"tee", path}, body)` |
| B | tests/netlab/agent/harness_test.go:275 | `out, errOut, err := dockerExec("cat", path)` |
| B | tests/netlab/agent/harness_test.go:283 | `_, _, err := dockerExec("test", "-f", path)` |
| B | tests/netlab/agent/harness_test.go:288 | `out, errOut, err := dockerExec("sh", "-c", "tail -n 60 "+labDir+"/agent.log "+labDir+"/labcontrol.log 2>/dev/null &#124;&#124; true")` |
| B | tests/netlab/agent/harness_test.go:306 | `body := readScratch(t, labDir+"/agent.log")` |
| B | tests/netlab/agent/harness_test.go:346 | `out, errOut, err := dockerExec("python3", "/opt/netlab/scripts/probe.py", labDir+"/cases.json")` |
| B | tests/netlab/agent/harness_test.go:398 | `mustExec(t, "python3", labDir+"/admin.py")` |
| B | tests/netlab/agent/harness_test.go:407 | `out, _, err := dockerExec("python3", labDir+"/admin_get.py")` |
| B | tests/netlab/agent/harness_test.go:420 | `_, _, _ = dockerExec("sh", "-c", fmt.Sprintf(` |
| B | tests/netlab/agent/harness_test.go:430 | `if err := dockerDetach("sh", "-c", script); err != nil {` |
| B | tests/netlab/agent/harness_test.go:437 | `_, _, _ = dockerExec("python3", "/work/tests/netlab/agent/topology-owner.py", "stop")` |
| B | tests/netlab/agent/harness_test.go:438 | `if err := dockerDetach("python3", "/work/tests/netlab/agent/topology-owner.py", "run"); err != nil {` |
| B | tests/netlab/agent/harness_test.go:447 | `_, _, err := dockerExec("python3", "-c", import time` |
| B | tests/netlab/agent/harness_test.go:462 | `_, _, _ = dockerExec("python3", "/work/tests/netlab/agent/topology-owner.py", "stop")` |
| B | tests/netlab/agent/harness_test.go:468 | `_, _, _ = dockerExec("conntrack", "-F")` |
| B | tests/netlab/agent/harness_test.go:469 | `_, _, _ = dockerExec("conntrack", "-F", "-f", "ipv6")` |
| B | tests/netlab/agent/harness_test.go:474 | `out, _, _ := dockerExec("nft", "list", "table", "inet", "fvpn")` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:56 | `if err := json.Unmarshal([]byte(mustExec(t, "ip", "-j", "addr", "show", "dev", "fv-fh")), &addresses); err != nil {` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:63 | `if err := json.Unmarshal([]byte(mustExec(t, "ip", "-j", family, "route", "show", "dev", "fv-fh")), &rows); err != nil {` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:72 | `mustExec(t, "ip", "-6", "addr", "replace", fmt.Sprintf("%s/%d", a.Local, a.PrefixLen), "dev", "fv-fh", "nodad")` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:79 | `mustExec(t, "ip", family, "route", "replace", route.Dst, "via", route.Gateway, "dev", "fv-fh")` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:88 | `mustExec(t, "ip", "link", "set", "fv-fh", "up")` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:92 | `mustExec(t, "tc", "qdisc", "add", "dev", "fv-fh", "clsact")` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:93 | `mustExec(t, "tc", "filter", "add", "dev", "fv-fh", "ingress", "pref", "1", "matchall", "action", "mirred", "egress", "redirect", "dev", "fv-ih")` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:94 | `defer func() { _, _, _ = dockerExec("tc", "qdisc", "del", "dev", "fv-fh", "clsact") }()` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:107 | `link := mustExec(t, "ip", "-j", "link", "show", "dev", "fv-fh")` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:119 | `mustExec(t, "tc", "filter", "del", "dev", "fv-fh", "ingress", "pref", "1")` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:120 | `mustExec(t, "tc", "qdisc", "del", "dev", "fv-fh", "clsact")` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:124 | `if strings.Contains(mustExec(t, "ip", "-j", "link", "show", "dev", "fv-fh"), "UP") {` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:129 | `if !strings.Contains(mustExec(t, "ip", "-j", "link", "show", "dev", "fv-fh"), "UP") {` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:139 | `t.Log("member addresses", mustExec(t, "ip", "-j", "addr", "show", "dev", "fv-fh"))` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:140 | `t.Log("member routes", mustExec(t, "ip", "-j", "-6", "route", "show", "table", "all", "dev", "fv-fh"))` |
| B | tests/netlab/agent/scenario_ab_recovery_test.go:157 | `leaseGen := strings.TrimSpace(mustExec(t, "python3", "-c", import sqlite3,json` |
| B | tests/netlab/agent/scenario_ac_timing_test.go:23 | `original := readScratch(t, labDir+"/agent.json")` |
| B | tests/netlab/agent/scenario_ac_timing_test.go:132 | `body, err := exec.Command("docker", "exec", prefix(), "cat", labDir+"/"+name).Output()` |
| B | tests/netlab/agent/scenario_ac_timing_test.go:145 | `mustExec(t, "python3", "/work/tests/netlab/agent/timing-probes.py", "trace", "renewal", "0", labDir+"/renewal.json")` |
| B | tests/netlab/agent/scenario_ac_timing_test.go:148 | `d := mustExec(t, "python3", "-c", import sqlite3,json` |
| B | tests/netlab/agent/scenario_ac_timing_test.go:160 | `mustExec(t, "python3", "/work/tests/netlab/agent/timing-probes.py", "trace", "expiry_restart", fmt.Sprint(mono), labDir+"/expiry.json")` |
| B | tests/netlab/agent/scenario_ac_timing_test.go:170 | `summary, err := exec.Command("python3", "timing-analysis.py", outDir).CombinedOutput()` |
| B | tests/netlab/agent/scenario_at_usage_test.go:65 | `mustExec(t, "rm", "-f", labDir+"/journal/forwarding.blocked")` |
| B | tests/netlab/agent/scenario_at_usage_test.go:144 | `_, _, _ = dockerExec("sh", "-c", script)` |
| B | tests/netlab/agent/scenario_at_usage_test.go:157 | `mustExec(t, "ip", "netns", "exec", "fv-a1", "python3", labDir+"/udpburst.py")` |
| B | tests/netlab/agent/scenario_at_usage_test.go:165 | `return len(readScratch(t, labDir+"/agent.log"))` |
| B | tests/netlab/agent/scenario_at_usage_test.go:176 | `body := readScratch(t, labDir+"/agent.log")` |
| B | tests/netlab/agent/scenario_at_usage_test.go:207 | `out := strings.TrimSpace(mustExec(t, "python3", labDir+"/nft_usage.py"))` |
| B | tests/netlab/agent/scenario_at_usage_test.go:271 | `out, errOut, err := dockerExec("python3", labDir+"/usage_read.py")` |
| B | tests/netlab/agent/scenario_az_external_test.go:69 | `mustExec(t, "nft", "flush", "chain", "inet", "fvpn", "forward")` |
| B | tests/netlab/agent/scenario_az_external_test.go:97 | `out, _, err := dockerExec("nft", "list", "chain", "inet", "fvpn", "forward")` |
| B | tests/netlab/agent/scenario_az_external_test.go:139 | `out, errOut, err := dockerExec("python3", labDir+"/usage_epoch.py")` |
| B | tests/netlab/agent/scenario_d116_test.go:64 | `saveD116(t, "before-enrollment.json", readScratch(t, labDir+"/before-enrollment.json"))` |
| B | tests/netlab/agent/scenario_d116_test.go:65 | `saveD116(t, "before-enrollment.txt", readScratch(t, labDir+"/before-enrollment.txt"))` |
| B | tests/netlab/agent/scenario_d116_test.go:69 | `if err := dockerDetach("sh", "-c", script); err != nil {` |
| B | tests/netlab/agent/scenario_d116_test.go:75 | `saveD116(t, "agent.txt", readScratch(t, labDir+"/agent.log"))` |
| B | tests/netlab/agent/scenario_d116_test.go:83 | `mustExec(t, "nft", "delete", "table", "inet", "fvpn_install")` |
| B | tests/netlab/agent/scenario_d116_test.go:91 | `mustExec(t, "nft", "add", "table", "inet", "fvpn_install")` |
| B | tests/netlab/agent/scenario_d116_test.go:92 | `mustExec(t, "nft", "add", "chain", "inet", "fvpn_install", "owner", { comment "managed-by=fvpn-node-install"; })` |
| B | tests/netlab/agent/scenario_d116_test.go:93 | `mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "start", "/scratch/agent-at10")` |
| B | tests/netlab/agent/scenario_d116_test.go:94 | `if err := dockerDetach("ip", "netns", "exec", "fv-a1", "python3", "/opt/netlab/scripts/stream.py", "203.0.113.10", "9001", "9000", "/scratch/agent-stream.log"); err != nil {` |
| B | tests/netlab/agent/scenario_d116_test.go:99 | `mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "finish-stream", "/scratch/agent-stream.log")` |
| B | tests/netlab/agent/scenario_d116_test.go:100 | `mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "stop", "/scratch/agent-at10")` |
| B | tests/netlab/agent/scenario_d116_test.go:103 | `saveD116(t, strings.ReplaceAll(name, ".log", ".txt"), readScratch(t, "/scratch/"+name))` |
| B | tests/netlab/agent/scenario_d116_test.go:107 | `for !scratchExists("/scratch/agent-stream.log") &#124;&#124; !strings.Contains(readScratch(t, "/scratch/agent-stream.log"), " ok") {` |
| B | tests/netlab/agent/scenario_d116_test.go:125 | `raw := mustExec(t, "nft", "-j", "list", "ruleset")` |
| B | tests/netlab/agent/scenario_d116_test.go:147 | `text := mustExec(t, "nft", "list", "table", "inet", "fvpn")` |
| B | tests/netlab/agent/scenario_f31_test.go:29 | `raw, errOut, err := dockerExec("python3", "-c", followup31RestartSnapshot)` |
| B | tests/netlab/agent/scenario_f31_test.go:41 | `raw, _, err := dockerExec("cat", labDir+"/"+name)` |
| B | tests/netlab/agent/scenario_f36_test.go:20 | `original := readScratch(t, labDir+"/agent.json")` |
| B | tests/netlab/agent/scenario_f36_test.go:44 | `out, errOut, err := dockerExec("python3", "/work/tests/netlab/agent/restart-sets.py")` |
| B | tests/netlab/agent/scenario_f36_test.go:49 | `raw, _, readErr := dockerExec("cat", labDir+"/f36-result.json")` |
| B | tests/netlab/agent/scenario_health_test.go:31 | `_, _, _ = dockerExec("rm", "-f", labDir+"/journal/health-challenge.json", labDir+"/journal/health-attest.json")` |
| B | tests/netlab/agent/scenario_health_test.go:57 | `bootID := strings.ToLower(strings.TrimSpace(mustExec(t, "cat", "/proc/sys/kernel/random/boot_id")))` |
| B | tests/netlab/agent/scenario_health_test.go:86 | `mustExec(t, "chmod", "600", challengePath)` |
| B | tests/netlab/agent/scenario_health_test.go:87 | `if mode := strings.TrimSpace(mustExec(t, "stat", "-c", "%a", challengePath)); mode != "600" {` |
| B | tests/netlab/agent/scenario_health_test.go:97 | `out, _, err := dockerExec("cat", attestPath)` |
| B | tests/netlab/agent/scenario_health_test.go:108 | `if mode := strings.TrimSpace(mustExec(t, "stat", "-c", "%a", attestPath)); mode != "600" {` |
| B | tests/netlab/agent/scenario_health_test.go:158 | `out := strings.TrimSpace(mustExec(t, "python3", labDir+"/identity_ids.py"))` |
| B | tests/netlab/agent/scenario_test.go:41 | `mode := strings.Fields(mustExec(t, "stat", "-c", "%a", labDir+"/identity/setup.key"))` |
| B | tests/netlab/agent/scenario_test.go:115 | `_, _, _ = dockerExec("sh", "-c", fmt.Sprintf(` |
| B | tests/netlab/agent/scenario_test.go:139 | `t.Log("TEST ONLY graceful restart stop:", mustExec(t, "python3", "/work/tests/netlab/agent/timing-probes.py", "stop", labDir+"/restart-stop.json", "10"))` |
| B | tests/netlab/agent/scenario_test.go:140 | `mustExec(t, "python3", "/work/tests/netlab/agent/topology-owner.py", "stop")` |
| B | tests/netlab/agent/scenario_test.go:171 | `_, _, _ = dockerExec("sh", "-c", "rm -f /scratch/denyloop.stop /scratch/deny-hits.log "+labDir+"/deny-cases.json")` |
| B | tests/netlab/agent/scenario_test.go:184 | `if err := dockerDetach("python3", "/opt/netlab/scripts/denyloop.py", "/scratch/deny-cases.json"); err != nil {` |
| B | tests/netlab/agent/scenario_test.go:188 | `_, _, _ = dockerExec("sh", "-c", "touch /scratch/denyloop.stop; if [ -f /scratch/denyloop.pid ]; then kill $(cat /scratch/denyloop.pid) 2>/dev/null &#124;&#124; true; fi")` |
| B | tests/netlab/agent/scenario_test.go:195 | `_, _, _ = dockerExec("sh", "-c", "touch /scratch/denyloop.stop")` |
| B | tests/netlab/agent/scenario_test.go:198 | `hits := readScratch(t, "/scratch/deny-hits.log")` |
