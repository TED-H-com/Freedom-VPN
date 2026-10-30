# SECOPS-11: component verified; live halves blocked

On tested revision `f7281ec`, the live publication command returned **422
NODE_NOT_SERVICEABLE before creating an operation**. Its current immutable
observation reports applied/desired generation 2, `provisioning=applying`,
`safety=failed` and `dns_policy=failed / DEPENDENCY_UNAVAILABLE`. Forwarding is
disabled despite a reported unexpired lease. Consequently there is no live
`data_path_probe` row, and the requested `member_forwarding_unconfirmed` operation
assertion remains blocked. The earlier admission refusal is not substituted for it.

The worker integration test
`TestSECOPS11_PublicationWorkerRequiresArmedMemberDataPath` passed both rejection
of an unarmed report and success with valid forwarding evidence after #109.
The separate owner unit test `TestSECOPS11_RecordedBaselineDoesNotProveServing`
failed because it reads mutable acceptance evidence and requires the historical
buggy `succeeded` operation. SECOPS-13 requests an immutable owner fixture.

Watchdog remains `serving=0`, `reason=recovery_required`, deadline 0. The persistent
recovery config is absent; no device admission counter advances. Both the live
publication positive and AT29/AT31 serving matrix require the shared lab owner's
resolver baseline and recovery fix. #118 arrived after the instructed initial
dependency check and is not part of this run; see [dependencies](dependencies.json).
No fake-agent edit or fixture row was used to make the node ready.

[Record](publication.json), [live observations](live.json), [supporting results](components.json)
and [requests](requests.md) retain the expected/actual result and exact owners.
