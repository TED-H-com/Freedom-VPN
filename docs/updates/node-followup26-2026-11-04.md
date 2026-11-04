# Node updater follow-up 26 — 2026-11-04

## Diagnosis before changes (D-141)

**Verdict: H-2 and M-1 are real. H-2 is reachable through the production lock
publication window; proceed with the clarified D-158.** One example in r30's
M-1 prose is incorrect: a context already cancelled on entry produces no wait
record today. This does not invalidate the non-busy filesystem-error finding.
This diagnosis was committed before importing tests or changing code.

References in this section use base `2d3aaeed` (the worktree's current `main`).
Review source is `origin/r30-update-fixes-review` at `da2560a9`; its
`.agent/REVIEW.md` H-2, M-1, D-158 Fidelity rows and “What held” were read with
`git show`, along with its updater proofs. Its updater line references do not
match this base; the references below are from the actual source.

### H-2: record publication and reachable holders

`node/nodelock/lock.go:59` acquires `LOCK_EX|LOCK_NB` **before** `writeHolder`
at line 74. Until `writeHolder` truncates at line 124 and writes at line 127,
the previous record remains readable. Reading the boot ID (74,86–95), reading
the current process start time (116–118), and marshaling (120) all precede that
truncate. A competing acquisition fails flock, reads the record at lines
61,98–112, and classifies it at 64,145–161. `Release` (230–240) unlocks and
closes without clearing or removing the record. Thus no stale-file fabrication
is needed in production: a new owner can hold flock while the prior owner's
metadata is still present. After truncation, empty or partial/unreadable
metadata instead returns plain busy (69,100–110), classified as live.

All these production holders can leave stale records:

- An exited updater `run`: `Run` enters `withLockFor` at
  `node/updater/engine.go:107`; the lock is released at 168, and the CLI returns
  the command status at `node/updater/cmd/fvpn-updater/main.go:139–154`.
- A restarted agent: ordinary enforcement acquires/releases at
  `node/agent/engine.go:1156,1171`; adoption and stop also do so at
  `node/agent/restart.go:71,75,178,182`. A record for the prior process survives
  its exit until another holder publishes metadata.
- A restarted watchdog: `node/watchdog/watch.go:132,139` uses the same shared
  lock and release. The old watchdog's exit leaves the same record lifetime.
- The installer: `packaging/installer/lock.go:22` uses `AcquireWithBootID`;
  install acquires/releases at `install.go:83,90–97`; upgrade does so at
  `upgrade.go:53,60,115`. Its record can likewise outlive its process.

Release alone does not make a still-running process's record stale. Once the
recorded process has exited, the process check fails (lock.go:152–157); PID
reuse is caught by start-time mismatch (159–160), and reboot by boot ID
(146–147). The flock must be held by a new owner for a contender to read that
old record; with no holder, acquisition succeeds and replaces it (59,74–79).
These are source-supported production possibilities, not observations from an
acceptance run.

In `node/updater/lock_wait.go:45–52`, once `staleAttempts` becomes nonzero,
**every** subsequent busy answer increments it. The live-budget check at
54–59 is then bypassed. Consequently:

| Answers | Current code | Clarified D-158 |
|---|---|---|
| One stale, then live | Fifth answer returns live busy at about 80 ms, exit 3, even inside the budget. | Each live answer resets the stale count; continue until acquisition/cancellation or the first live answer at/past 10 s. |
| Live, then stale, then live again | Earlier live answers do not count, but the fifth answer starting with the first stale ends the wait; a final live answer returns exit 3 prematurely. | The first stale starts a consecutive run; the later live answer clears it. The original 10 s start time is retained. |
| Four stale, live, four stale, release | Fifth answer is live but ends the command; release is never reached. | Live clears the first four; the next four are insufficient to fail, so release permits acquisition. |
| Earlier live, then five consecutive stale | Fifth stale returns its final detail after four 20 ms delays. | Same outcome: exit 1, about 80 ms after the first of those five. |

`withLockFor` wraps busy errors using `lockBusyDetail`
(`node/updater/engine.go:162–164,1131–1141`). `ExitCode` maps exact `lock` to 3,
other details to 1, and prioritizes blocked-member exit 2
(`node/updater/host.go:247–258`). The CLI prints the unchanged line at
`node/updater/cmd/fvpn-updater/main.go:151–152`. `recover`/`Retain` keep their
separate prompt path (`engine.go:144–160`); nested callers bypass acquisition
at 151–153. These agree with r30's “What held”.

Under the clarification, five consecutive stale answers may finish after the
live deadline; the first live answer at/past that deadline returns exit 3.
The nominal overrun is at most about 100 ms. Filesystem work and scheduling
can add wall time; this is not a real-time completion guarantee.

### M-1: first errors and no-attempt cancellation

`waited = true` precedes the busy check (`node/updater/lock_wait.go:41–44`),
so a non-busy first failure triggers the deferred record at 25–31. Non-busy
errors from the single acquisition include parent-directory creation (shared
lock.go:52–53), open (55–57: e.g. directory path, permissions, read-only
filesystem), flock errors other than EWOULDBLOCK (59–72), and holder-publication
failure (74–77: unreadable current process start time at 116–118, truncate,
write or sync failure at 124–130). Marshal error propagation exists at
120–122, though this fixed integer/string record has no ordinary marshal
failure. Failure to read the *contended* holder record is plain `ErrBusy`, not
a non-busy error (61–69,98–110). Boot-ID read failure uses fallback (86–95).

Cancellation on entry is different: `ctx.Err()` is checked **before the first
attempt** (`lock_wait.go:34–37`), with `waited` still false (24). It already
returns without a record. A cancellation after an initial busy answer does
log once through the defer (25–31,61–65). The clarification moves the flag
assignment after the `ErrBusy` check; all record fields remain unchanged.

### Pins: assertions, comments and documentation

No existing test asserts that a live answer consumes a previously started
stale window. No existing assertion needs changing. Specifically:

- `TestD158_StaleDetailsKeepPromptWindow`,
  `node/updater/followup25_test.go:192–212`: line 204 requires exit 1, the exact
  detail, and 80 ms–1 s for all four details on all four commands; 207 requires
  no journal. All answers remain stale, so it also fits consecutive counting.
- `TestD158_StaleWindowStartsAtFirstStaleAnswer`, same file:216–233:
  line 226 requires exit 1, exact `lock pid`, and 380 ms–1 s after metadata
  changes at 300 ms. This pins independence from *earlier* live answers, not
  counting later live answers. It remains correct unchanged.
- `TestD158_CLIWaitRecordAndExactErrorLine`,
  `node/updater/cmd/fvpn-updater/followup25_test.go:44–47,109,119–120,128,149–150`:
  persistent stale cases require exit 1, 80 ms–1 s, one correctly shaped wait
  record, and each exact error line. None interleaves live answers.
- `TestD158_CancellationAndUnreadableMetadata`,
  `node/updater/followup25_test.go:257–281`: 276 checks cancellation at
  300 ms–1 s with no journal; 279 requires one failed-wait record. Cancellation
  occurs *after* contention. It neither pins a non-busy first error record nor
  covers cancellation before the first attempt.
- The other follow-up 25 assertions (live release/budget, nested calls and
  prompt recovery), the four r29 proofs, and follow-up 24's D-155 tests pin
  exclusions, exits, records and journal ordering, not stale/live counting.
  r30's H-2/M-1 assertions express the clarified behavior and need no changes;
  its D-155 proof must remain unchanged. Its comment calling recovery a
  “single-attempt” path is imprecise: its assertion and production path use
  the prompt five-attempt window, as r30's report also states.

The following text pins or describes the previous policy:

1. `node/updater/lock_wait.go:19–21`: “Stale metadata switches to the existing
   five-attempt window, counted from the first stale answer, even if live
   contention preceded it. Later attempts may acquire a released lock;
   otherwise the final diagnostic is preserved.” Rewrite for consecutive
   stale answers and live reset/budget behavior.
2. `docs/updates/node-updater.md:36–38`: “Stale-holder diagnostics keep that
   five-attempt window, counted from the first stale answer, and exit 1;
   unreadable metadata retains the existing conservative live-busy
   classification.” Rewrite the counting; preserve classification.
3. Same guide:48–49: “A first stale answer switches to the prompt five-attempt
   policy even after earlier live contention.” Replace with consecutive
   counting, reset and the original live deadline.
4. Same guide:53–58: “a failed first acquisition produces exactly one JSON
   slog record” and “A successful first attempt emits no wait record”. Narrow
   the first condition to `ErrBusy`; also explicitly exclude no-attempt and
   non-busy first failures.
5. Historical follow-up 25:342–350: “The first stale answer starts the original
   **five-attempt** policy, including that answer; four further attempts are
   separated by 20 ms.” Also “Earlier live attempts do not consume this window,
   and a later successful acquisition still proceeds”, “As in the existing
   lock library, exhaustion returns the last error”, “Thus a retained stale
   detail stays exit 1 after its prompt window, rather than waiting out the
   remaining live budget”, and “The stale window is counted from its first
   answer even if that answer arrives near the live deadline; it is not
   restarted on each stale retry.” These collectively document the old
   counting. Lines 352–353 say “only if their first attempt fails”, the old
   logging condition. Lines 390–391 record persistent-stale and live-then-stale
   proof results, which remain valid. Lines 513–517 contain the old handover
   sentence, ending “five attempts counted from the first stale answer”.
   **This historical document will not be edited.**
6. The brief identifies that same sentence in handover's operator guide.
   On this base, `docs/handover/operator-guide.md` contains no D-158 or
   “first stale” sentence (its updater exit paragraph is at line 661).
   The replacement will still be handed to its owner, for the owner's current
   version. No handover file is changed here.

This delivery concerns §13.5 exclusion and §13.11 bounded `UPDATE_BUSY` handling,
UPD02/UPD03/UPD05 (§4) and AT25/AT26 (§19) regression paths. No live acceptance,
arm64 hardware result or F30-01 closure is claimed.
