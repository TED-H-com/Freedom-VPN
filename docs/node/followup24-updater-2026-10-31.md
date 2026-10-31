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

## Candidate identity and final health

At process start, the agent resolves configurable `current_release_path`
(default `/opt/vpn-node/current`). Its target release UUID, actual running
executable inode and read-only updater journal current pointer must agree.
The immutable installed-release record supplies the node-agent version after
its canonical envelope digest is checked. This resolves once per process: an
old process cannot adopt the candidate merely because the symlink moved.
`/etc/vpn-node` remains read-only; the install-time release ID is a development
or pre-seed fallback, not the candidate's health identity.

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
