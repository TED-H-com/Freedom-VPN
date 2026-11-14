# Follow-up 4: target evidence and observation authority

The current run and results are in [summary.md](summary.md). The previous
passing rollback capture is preserved under [history/follow-up-3](history/follow-up-3/README.md);
the earlier blocked capture remains under [history/follow-up-2](history/follow-up-2/README.md).

The suite still builds and registers real signed candidates through
`tests/e2e.PrepareServingBaseline`. D-053 health comes from the running agent.
The rollback candidate performs a real TCP probe against its closed loopback
port. Neither journal health nor projector-owned database fields are written.
The additional API stimuli below are TEST ONLY negative cases explicitly
requested by the orchestrator; they are separate from updater-generated evidence.

| Scenario | Follow-up 4 extension | Evidence in the AT record |
|---|---|---|
| AT25 | A failed result at stage `check` with null `error_code` must return 422 `VALIDATION_FAILED`; the newly authorized job remains queued without attention | `failed_without_cause`, `check_failed_requires_cause` |
| AT26 | The three revision 7 fields are present and null before a current receipt; after rollback they exactly match the accepted result | `batch_api_without_receipt`, `revision7_batch_api_response`, `check_revision7_*` |
| AT26 | Two changed payloads reuse the accepted result ID; the second is replayed three more times. All five requests must return 409 `ACK_CONFLICT`, with exactly one conflict row and one audit per hash | `conflict_*`, `conflict_persistence_counts`, `accepted_receipt_unchanged` |
| AT27 | A disagreeing heartbeat cannot replace the accepted release or recovery flag and must withhold confirmation; an agreeing heartbeat confirms | `authority_before`, `disagreeing_heartbeat`, `agreeing_heartbeat` |
| AT27 | A report with a generation 100 above the accepted generation and null job ID records arrival without advancing the high-water or changing the serving release | `jobless_generation_report`, `restored_confirmation`, `check_observation_authority` |

The observation probes pause the real agent and copy its last real report,
retaining its checks. They change only report ID, time, and the requested
release/update fields. After the arrival-only probe an additional agreeing
report restores confirmation, then the real restored agent resumes. The node's
private key stays on the node: the probe invokes its HTTP client inside the
owned container. Only request evidence, response bodies and HTTP status return
to the harness. Database queries are read-only.

The missing-cause probe copies a real result's restored health and binds the
negative request to a new API-authorized job. It does not assert that an updater
produced this malformed result. Conflict probes change only `boot_id` and
recompute the canonical payload hash, leaving the immutable accepted receipt
untouched. No progress result is posted to make rollback pass.

The records-first exit gate also requires every extension's evidence. An older
base-scenario pass cannot hide a skipped or failed follow-up check. Run with
`make -f mk/at-updates.mk at-updates`; it tears down the owned stack even when
any assigned scenario remains blocked. See [scope mapping](follow-up-1.md) for
the broader §19 cases that these narrower scenarios do not claim to complete.

The [initial follow-up 4 run](history/follow-up-4-rate-limit/README.md) was
blocked by one 429 during an unpaced conflict burst. The probe now waits two
seconds before each node POST to stay within the existing rate buckets; no
product limit is changed and no response is hidden by an automatic retry.
