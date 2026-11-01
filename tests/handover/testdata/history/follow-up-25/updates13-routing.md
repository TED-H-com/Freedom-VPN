## Requests routed by the orchestrator

1. Updates F13-01 (this lane's numbering; not members' F13-01) — AT23's first agent exit status was reaped before the 1 ms observer saw it → devstack follow-up 25 (collected): a reliable TEST ONLY agent exit-status / SIGKILL readback in `deploy/devstack/agent-service.sh` and `deploy/devstack/entrypoint-node.sh`, or an equivalent owner-supported API. The row stays unobserved; no agent or updater defect is inferred.
