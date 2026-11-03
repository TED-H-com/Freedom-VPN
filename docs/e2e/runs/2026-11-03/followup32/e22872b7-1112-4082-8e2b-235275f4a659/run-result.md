# run-result

Original forced-conflict invocation failed before enrollment

Status: fail

Run: e22872b7-1112-4082-8e2b-235275f4a659
Source: fff680c6c2cd539b77cc534ce9d090f8ad9a8ec3

- go_test_exit: 1
- test_only_stale_etag: true
- failure: underlay requires a running fresh node waiting for enrollment
- cause: The local admission wrapper printed free before docker stdout. lab.stage expects one JSON document and rejected the prefixed status. The wrapper was fixed after this invocation ended.
- diagnostic_readback: At 2026-11-03T23:07:56.240904Z, the corrected wrapper returned {"stage":"waiting_token"} from the same node.
- publisher_reached: false
- real_conflict_attempts: 0
- retry: false; the no-run-retry instruction is honored. The required live forced-conflict proof remains not exercised.
- original_summary_scope: acceptance-summary.md contains the one completed admin-login record only; the original go test invocation failed at its next setup step.
