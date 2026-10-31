# Agent supervisor

`agent-service.sh stop` and `start` do not fork `fvpn-agent`. `stop` creates
`/var/lib/vpn-node/e2e/pause-agent` and waits until the entrypoint supervisor
has reaped the agent. `start` writes `agent-selected`, clears the marker, and
waits until that supervisor owns exactly one matching process. Both print that
PID (`0` when stopped). The supervisor polls every 0.2 seconds and is the only
start/stop path. Harnesses may still toggle the marker file directly.

`agent-supervisor` runs after enrollment. It calls stop and start twenty times
back to back and, after each cycle, requires zero `fvpn-agent run` processes
when stopped and exactly one process whose PID equals both the start output
and `agent.pid` when started. One extra start must keep that same PID. The
scenario then leaves the boot agent running and the marker clear.

See [RUN.md](../RUN.md).
