# Updates follow-up 10

Run: 695f977c-b587-4768-88f1-0514f3a4f1ac

Source: 0df8599dc84d4b85e48182f20aa4b0fc741bc72a

**Installer extension: pass (23/23 checks), with an explicit reporting adapter. Live make: exit 2.** See the [deciding AT and hand-off tables](follow-up-10.md) and [two owner requests](requests.md).

These labels follow the orchestrator's follow-up scenario list. They do not redefine the broader §19 cases. Physical arm64 execution and the full original §19 population remain outside this run. Both architectures contain compiled real binaries.

| Scenario | Result | Observation |
|---|---|---|
| [AT23](AT23.md) | blocked | devstack / agent-core / updater: good candidate did not reach verified local success; inspect process_agent.log, executable_layout and journal; required follow-up 8 check incomplete: probe_inspection; required follow-up 8 check incomplete: probe_traversal; required follow-up 8 check incomplete: first_result_last_good |
| [AT24](AT24.md) | blocked | w4-node-updater: last-good prerequisite absent; required follow-up 8 check incomplete: first_result_last_good; required follow-up 8 check incomplete: recorded_versions; required follow-up 8 check incomplete: offline_snapshot; required follow-up 8 check incomplete: offline_completion; required follow-up 8 check incomplete: new_node_inheritance |
| [AT25](AT25.md) | blocked | w4-node-updater: last-good prerequisite absent; required follow-up 8 check incomplete: probe_inspection; required follow-up 8 check incomplete: lab_probe_peer; required follow-up 8 check incomplete: failed_requires_cause |
| [AT26](AT26.md) | blocked | w4-node-updater: last-good prerequisite absent; required follow-up 8 check incomplete: revision7_empty_target; required follow-up 8 check incomplete: revision7_rollback_target; required follow-up 8 check incomplete: conflict_bounds |
| [AT27](AT27.md) | blocked | w4-node-updater: last-good prerequisite absent; required follow-up 8 check incomplete: recorded_audits; required follow-up 8 check incomplete: recorded_versions; required follow-up 8 check incomplete: withheld_observation; required follow-up 8 check incomplete: recorded_last_good; required follow-up 8 check incomplete: stale_ack; required follow-up 8 check incomplete: observation_authority |

See [current requests](requests.md) and the [follow-up 9 archive](history/follow-up-9/ARCHIVE.md).
