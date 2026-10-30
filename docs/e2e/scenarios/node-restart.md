# Enrolled node container restart (§5, §6, F2-05)

The entrypoint checks the durable `identity/node.json` before waiting for an
installer token. An existing identity requires its certificate and key; an
incomplete identity stops with `identity_incomplete`. A normal resume regenerates
the boot recovery configuration and boot ID, clears the stale agent PID file,
and starts the watchdog, agent service and NetBird. It never reenrolls to recover
from an error. New enrollment still consumes and neutralizes the installer token.

`node-restart` restarts the enrolled node before member admission. Within the
configured 30-second watchdog deadline it requires status `enrolled`, a readable
`inet fvpn` table and a heartbeat naming the new boot ID. The durable node and
incarnation must be byte-identical, the installer token absent, and the database's
enrollment IDs, states and consumption times unchanged. This is a process restart,
not AT07 reinstall, and it cannot substitute for renewing a member lease after
publication or member authorization changes.

The node and AdGuard both join the stable `node-netns` service namespace. This
keeps their loopback listeners together across a node-container restart. Making
AdGuard join the node's replaceable namespace stranded it after restart; the
first diagnostic run exposed this with resolver connection failure. The namespace
holder runs only `sleep infinity` and owns the existing isolated lab/origin
addresses. No host interfaces or routes change. Publication polls the latest
accepted baseline observation, not a historical pre-restart success.
