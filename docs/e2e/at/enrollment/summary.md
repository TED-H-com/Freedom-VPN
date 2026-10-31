# Enrollment acceptance — follow-up 5

| Case | Result | Current checks | Prior failure review |
|---|---|---|---|
| [AT06](AT06.md) | **pass** | 344 pass / 0 blocked / 0 fail | resolved |
| [AT07](AT07.md) | **blocked** | 19 pass / 6 blocked / 0 fail | resolved |

Run [`4011dd32-73ef-4f49-b78b-6cf1ab8496c9`](history/4011dd32-73ef-4f49-b78b-6cf1ab8496c9/AT07.json),
UTC 2026-10-31T06:10:11Z–2026-10-31T06:15:49Z, source `200c027` clean at invocation.
Both live probes and all teardown commands exited 0. The make target exited 2
because six required AT07 checks remain blocked. Final owned inventory is
`containers=0, networks=0, volumes=0, images=0`.

#266's initial real installer invocation reproduced exit **1**, code `failed`,
after enrollment. The journal confirmed NetBird's read-only socket and the
`/run/vpn-node` failure. Usage hold/snapshot/transport exports all rejected
`installer-node`. These are the current ENR-05/ENR-06 blockers, with exact asks
in [requests.md](requests.md). No counted-backlog delivery, post-reinstall
monthly equality, successful installer rerun, retired directory, or installer
old-certificate denial is claimed.

The regular-node usage endpoint-scope probe passed: retryable 503 while held,
desired-state 200, release to validation 422, unchanged paused snapshots and
successful cleanup. It had zero pending events and is not ENR-05 acceptance.
The clone conflict/revocation/recovery check still passed.

AT06 observed **zero 429s in both phases**. #233 retained browser 10/s burst 30;
#252 moved admin lists to `read` 30/s burst 60 (ingress 20/s burst 60). Those are
reviewed source defaults, not a live saturation measurement. D-078 still scores
Retry-After behavior. AT07's retained `bebfe230` and `35e542f8` process rows are
now explicitly reviewed and resolved, alongside `1d5c02a4`; no history was deleted.

Full evidence: [AT06.json](AT06.json), [AT07.json](AT07.json).
Reproduction and gate output: [RUN.md](RUN.md). Prior root records:
[follow-up 4](history/follow-up-4/README.md).
