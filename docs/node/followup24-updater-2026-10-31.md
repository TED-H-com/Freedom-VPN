# Follow-up 24 to updater follow-up 15 / installer follow-up 4 — 2026-10-31

The agent now publishes the existing `updater.FileMeter` JSON format at the real
meter barrier. The default is `state_dir/meter.checkpoint`; optional
`meter_checkpoint` selects a matching custom updater path. No format change or
updater edit was made.

The order is sampler barrier, verified member baseline, final meter sample,
SQLite epoch close/event commit, one consistent snapshot of the closed epoch and
sequence watermarks, temporary file write/fsync, atomic rename, directory fsync.
`closed_meter_epoch_id`, `last_durable_event_seq` and `central_acked_through`
come from that snapshot; `new_meter_epoch_id` is empty. The updater owns its
subsequent journaled new-epoch note. A quiet previously non-serving node still
closes a real kernel-read epoch; it does not invent a sample or zero usage.

A failed sample, missing completed close or failed file write leaves the barrier
retryable and fails closed. Resuming forwarding invalidates the old checkpoint
under the node lock, with directory fsync. Opening the meter after restart also
invalidates stale evidence. Repeated finalization reads the same completed
close and does not generate another sequence. The unit proof uses the real
usage store/meter and the unchanged `updater.FileMeter` reader.

The full kernel gate also exposed a race between live serving verification and
the periodic sampler: a generation rejection could install the planned baseline
before the sampler saw the external clear. The agent now retains that observed
reset before withdrawal, so the real final sample commits the discontinuity gap
before publishing its checkpoint. Idle counters do not invent usage or a gap.

Closing quiet epochs exposed a second ordering defect: the first recovery bytes
could arrive before the asynchronous sampler reopened its epoch and become that
epoch's baseline. The agent now opens the next epoch durably from a real read
while forwarding and sampling remain blocked, before Apply admits any traffic.
Only a verified Apply resumes the loop. A read failure withholds Apply and retries
the barrier; the regression checks both recovery and bytes arriving inside Apply.
Installed binding metadata is also retained before later live/topology checks.
If those checks reject a restarted process, its final sample can still attribute
packets admitted by the installed model before withdrawal. A library verification
rollback retains those bindings only when its evidence identifies the installed
`verify` phase; a `pre_apply` foreign rejection does not replace them. Tests prove
both rejection paths retain their actual bytes without serving or acknowledging
the rejected state.

## Candidate identity and final health

At process start, the agent resolves configurable `current_release_path`
(default `/opt/vpn-node/current`). Its target release UUID, actual running
executable inode and read-only updater journal current pointer must agree.
The immutable installed-release record supplies the node-agent version after
its canonical envelope digest is checked. This resolves once per process: an
old process cannot adopt the candidate merely because the symlink moved.
`/etc/vpn-node` remains read-only; the install-time release ID is a development
or pre-seed fallback, not the candidate's health identity.

The first unchanged installer run exposed an initial-install ordering case:
`fvpn-updater-recover` initializes the journal before agent bootstrap, while
installer seed waits until bootstrap has supplied the NetBird setup key. The
agent permits that initialized but unseeded journal only for the configured
initial release with a matching running inode and no release, job, transaction
or result history. Missing current pointers after any history still fail closed.
The regression rejects both a different configured release and a journal with
installed history but no current pointer.

`node/internal/fakecontrol.UpgradeHealthFixture` supplies actual signed
unpublished desired-state and lease responses for the final health stage. Its
caller supplies an isolated reachable TCP target. The regression test exercises
a real local TCP connection and requires a failed health answer when that
listener stops. NetBird status in that unit test is an explicit TEST ONLY
fixture; it is not a native NetBird acceptance result.

## Remaining integration boundaries

`node/updater/host.go:FileGate.Block` writes a latch and returns immediately;
`FileMeter.FinalSample` then reads the checkpoint once. There is no handshake
that waits for the agent's final sample. While a live node is serving its old
checkpoint is intentionally absent. Updater follow-up 15 must provide a bounded
wait for genuine completed barrier evidence after confirmed forwarding withdrawal,
including a fail-closed stale-file/restart test. It must not fabricate the file,
accept a prior open epoch, or switch on a timeout. The existing format carries
no transaction/boot/close-time correlation, so a stronger handshake requires an
explicit owner decision before changing that format.

The unchanged privileged installer proof is run through
`python3 tests/netlab/agent/run-installer.py`. The adapter only translates its
hard-coded Docker object prefix to this lane's `fvpn-w2-agent-core-lab-installer`
prefix. The source, assertions, binaries and health fixtures are unchanged.
Installer follow-up 4 owns `tests/installer/cmd/fvpn-w5-control/main.go` and its
harness wiring: consume the signed desired/lease and reachable probe fixture,
then rerun final candidate health. The final report records the actual stopping
stage; no synthetic checkpoint or passing health adapter is used to promote it.

## Unchanged installer result

The final run at 09:53–09:54 UTC passed initial install/seed and consumed the
real checkpoint, then reached candidate restart and health checking. The agent
journal reports HTTP 404 for desired state and lease, so it cannot write a valid
challenge-bound health answer. Candidate and restored-release health both fail;
the updater returns 2 and seals `state=failed`, `stage=rollback`,
`journal_stage=recovery_required`, with the original release current and
`meter_checkpoint_present=true`. The Go test exits 1 as required by its unchanged
success assertion. Its `identity_and_state=passed` is the persistent-data check,
not proof of a passing agent health attestation.

The prior attempt failed before seed because of the initialized-empty-journal
case described above; that agent regression is fixed and tested. The final
failure has advanced beyond F9-03's missing file. It does not close full product
upgrade acceptance: installer follow-up 4 must supply the missing signed control
responses and real probe topology for both candidate and restored health, and
updater follow-up 15 still owns the serving-node barrier handshake. No changes
to either owner's files were made.
