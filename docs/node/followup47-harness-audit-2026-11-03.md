# Follow-up 47 harness audit — 2026-11-03

TEST ONLY; no production change. A is `567080be1969bf22782617e78c7605c2c92a52a2`;
B is `38420dadef6ae9d0236288eef533bc1ddee0da53`. The two archives, every extracted
blob (18,889 A; 18,939 B), overlays and frozen fixtures match the prior pins.
No rebuild was needed. The JSON companion contains all SHA-256s and the fresh
verification receipt. All 18 inherited tool hashes match 46's audit.

45's declared production diff was rerun once:

```sh
git diff 38420dadef6ae9d0236288eef533bc1ddee0da53 28e9e41c -- '*.go' ':!**/*_test.go' ':!tests/**' ':!docs/**'
```

Exit 0; stdout empty, zero bytes. B's production Go equals #429 in this scope.

The runner derives from 46. It reads NUL-separated argv from procfs, matches
make's specified target/path strings, go's test subcommand, or an interpreter's
actual gates.sh script operand. Inline text, stdin, grep/pgrep patterns, editors
and sessions do not count by themselves. Live suite descendants count.
The runner's own ancestors and descendants are excluded from this check only.
CPU classification is unchanged, including foreign session and ancestor CPU.
Only pid, comm and ticks of matching foreign processes are retained.

At each leg's preflight, suites and competing fvpn containers cause rechecks at
15-second intervals for at most 600 seconds. Every check and elapsed wait is
receipted. A clear preflight starts the leg only with seven minutes remaining;
expiration records `not started: preflight wait expired`. No suite launch or
round is spent by a preflight refusal. The dry reservation attempt remains
recorded and subject to the two-reservation maximum. Preflight CPU sampling
rebases after its idle wait; running-leg monitoring still samples every four
seconds and rejects any interval over five seconds or unreadable state.

Dry and measured legs use the same standalone command, full frozen
node-test-netlab, wrappers, overlays, clock probes and finite reservation limits:

```sh
python3 -B tests/netlab/agent/followup47_timing.py run > .agent/f47/timing/driver-"$F47_RUN".stdout 2> .agent/f47/timing/driver-"$F47_RUN".stderr
```

Set F47_RUN to dry1, optionally dry2, then window1 and window2. Each request
uses `bash "$FVPN_HOLD_SH" --reserve 30` with `agent-core f47 D-150 dry pair`
or `agent-core f47 D-150 window N`. Refusals and request/grant times are recorded
for up to 120 minutes. Wait 60 seconds after grant; release promptly. Require
at least ten minutes after every previous release before requesting the next
reservation. No third dry or measured reservation. Never start a leg with less
than seven minutes left; stop 30 seconds before expiry. Dry legs never pool.
No declaration or measured leg is allowed until both complete dry legs pass on
the current runner hash, produce Step spans and retain admission receipts.

All other measurement, overlap filtering, half-retained-Step pooling,
D087/D104 failure handling and D-138 attribution rules remain 46's. The result
tool only adds preflight receipts to its unchanged five-metric calculations.
Admission receipts remain outside both Docker streams. The complete frozen
consumer audit is unchanged and hash-linked in the JSON companion. Fresh
actual-wrapper probes cover combined, stdout and shell reads without Docker.
The separate clock binaries match 46; zero offsets and aligned brackets remain
mandatory. Scratch resolves from the tool file, independent of working directory.

All 97 inherited/new synthetic tests pass. The 32 follow-up 47 tests also pass
from another working directory. Tests cover all requested process tables,
ancestry and CPU classification, waits clearing/expiring, no suite launch on
expiration, reservation limits and spacing hook, unchanged clock/mark/filter/
failure behavior, dry no-pooling and current-hash/receipt prerequisites.
Earlier runners, tests, records and production files are unchanged.
