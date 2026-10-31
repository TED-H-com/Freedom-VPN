# AT29 guest-clock rollback: operator-run row

Decision for follow-up 17: keep this as an operator-run row, blocked until its
evidence is supplied. The shared Docker netlab does not provide an independent
wall clock. No privileged container clock write, time namespace offset, fake
clock, or process restart counts as this case. None was executed on the shared
development host.

The operator supplies a disposable Ubuntu 24.04 VM with its own kernel and
adjustable guest wall clock, plus an isolated controller, native member and
traffic origin. Reuse the current signed release, real enrollment, publication,
finite grant, member transport and usage APIs. Restrict every traffic destination
to that test topology. Record the VM/hypervisor version and settings, release
and source digest, architecture and topology. Do not attach production networks.

1. Disable time synchronization **inside the guest**, including hypervisor guest
   time synchronization. Record the guest clock before/after and an independent
   controller/observer clock. The host/controller clock must remain unchanged.
2. Obtain a valid short TEST ONLY lease with an earlier effective service end.
   Record signed lease/grant deadlines, guest monotonic/boottime readings,
   heartbeat, watchdog status, nft timeout values and a counted positive path.
   Start persistent TCP and authenticated UDP with sender-attempt counters and
   independent origin receipts. A stopped sender is not evidence of withdrawal.
3. Cut only the guest's control transport in the disposable topology, kill the
   guest agent, and roll the **guest wall clock** back far enough to cross the
   short lease's issue time. Record before/after wall and monotonic time, guest
   boot ID and the observer timestamp. Keep watchdog/kernel enforcement running.
4. Continue traffic beyond the original earliest monotonic expiry. Record the
   last successful TCP/UDP replies, ongoing send attempts, device counters,
   watchdog/timeout withdrawal and preserved control path. The original deadline
   must not move later because of the rollback or agent death (§16.3; §19 AT29).
5. Restore the guest time and control transport, restart the agent, and show
   forwarding resumes only after a valid new policy/lease and accepted ACK.
   Repeat the finite grace/grant subcases separately; do not infer them from one
   lease case. Destroy the VM/disks and isolated test infrastructure afterward.

Acceptance owns the continuous timing trace and scoring. Publish a sanitized
record with the real time delta, original/observed monotonic deadlines, traffic
stop points, recovery ACK, and any gaps. This procedure is not a pass and does
not replace the existing blocked guest-clock row. VM provisioning and operator
execution are additional work, outside this lane's Docker-only host authority.
