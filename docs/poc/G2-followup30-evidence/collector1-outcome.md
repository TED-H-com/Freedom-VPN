# Attempt 1 — preflight stopped, collector not launched

Declaration/source commit: `7bfb9b43`. The ten-minute reservation was granted
at 11:13:33 UTC, followed by 60 seconds settling. Eighteen host checks were
retained. No lab container, collector, packet injection or Go test was started.

The preflight classified `fvpn-pg` as a foreign lane container because it rejected
every non-owned `fvpn-*` name. The task's wording excludes **other lanes'**
containers; the identity of this persistent service needs clarification. It was
not stopped or modified. All measured foreign CPU intervals were below 0.5 cores
(maximum 0.272657); no other suite was detected. The preflight was interrupted
and its reservation released at 11:15:44 UTC. The qualification is false.

This attempt is retained and conservatively consumes the first of three slots.
It says nothing about live collector correctness. The F27-01 window remains
unspent. The next reservation must follow offline review and shared-host admission.
