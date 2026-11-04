# D-160: admission and measured scenarios

The assembled acceptance runner admits each named section before calling the
scenario. The interval conservatively includes its setup, every deadline read,
all samples and its deferred cleanup. Helpers inherit that section; they do
not start another admission wait. `update-good` and `update-bad` are separate
sections. `billing-and-portal-return` and `member-traffic-forwarding-usage` each
include the related rows that share their scenario's setup. Section names and
UTC start/end timestamps are retained in the run's JSONL ledger.

With `$FVPN_HOLD_SH` configured, set `FVPN_E2E_ADMISSION_LOG` to an absolute
path in the run's records. `make -f mk/e2e.mk e2e-test` installs temporary PATH
wrappers for Go, make, Docker and npm across the entire command tree. For other
gates use `python3 -B deploy/devstack/admission.py run <command ...>`. TMPDIR
must point to disk scratch. Wrapper files are removed on foreground completion.
The script path and admission text never enter a tool's output stream or the
public ledger. Every check records its request/answer times, free/held answer
and exit code. A held answer is rechecked after 60 seconds, bounded at two hours.
No reservation is taken.

The runner exports `FVPN_E2E_TIMED_SECTION` only after admission and removes it
when the section finishes. Wrapped child commands inside that interval record
`command_in_section` and execute immediately, with no admission check or wait.
A direct attempt to admit within a section is rejected. The existing host Go
command may still reject work with exit 125; the scenario never waits or retries
that command. Outside sections the wrappers admit every command, including
recursive make (its argv[0] remains PATH-relative).

This does not prevent another lane from starting during a measurement. The
operator's separate read-only host monitor records overlap using only foreign
pid, comm and start ticks. Such an interval is confounded under D-133 and is
never delayed or retried for that reason. Host inventory admission precedes
launch, including a recheck after `vmstat 5 7`.

Without `$FVPN_HOLD_SH` (for example on a dedicated CI worker), sections still
record their boundaries and an explicit `admission_not_configured` event.
No free answer is fabricated. Admission is scheduling evidence, not a product
acceptance predicate; neither updater scenario's predicate changes.

The node launcher stores each updater-loop invocation in its own
`updater-invocations/invocation.*` directory: complete combined output, UTC
start/end, PID and exit code. The latest `updater.out` and result remain for
existing consumers. Scenario records include the complete cumulative archive;
`update-bad` therefore also retains `update-good`. An interrupted invocation
has no end/exit receipt and is never represented as completed. Archive readers
and the update-lock/heartbeat observer only read; they take no lock or gate.
