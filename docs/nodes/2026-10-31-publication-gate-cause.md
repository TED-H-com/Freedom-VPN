# Publication gate diagnostics — 2026-10-31

Follow-up 30, D-101 / D-107. Contracts revision 13 already carries the
publication gate diagnostic in `Operation.steps[].message` (nullable free text,
maximum 1000 characters). The admin operation reader and the node operation
reader preserve the stored error message and append its bounded gate cause:

```json
{
  "name": "data_path_probe",
  "status": "pending",
  "required": true,
  "observed_at": null,
  "error_code": "NODE_NOT_SERVICEABLE",
  "message": "member_forwarding_unconfirmed (lease_unconfirmed)"
}
```

Read `GET /api/v1/admin/operations/{operation_id}` using the operation ID or
`Location` returned by the publication command. During a publication timeout,
find the pending `agent_policy` or `data_path_probe` step by name and display
its `message` alongside `error_code`. Display the text directly; this field
remains a diagnostic string, not a structured reason enum. A missing message
does not establish that the gate passed.

D-074 remains unchanged (§5.2, §14.5, §14.8): while the gate holds, the operation
is `running`, the observe step is `pending`, its `observed_at` is null, and the
operation's `error` and `completed_at` are null. A portal polling timeout does
not turn this into a failed or completed publication. Continue reading for
fresh observation or a terminal outcome; the existing worker retry budget
still governs actual exhaustion.

The cause vocabulary comes from `internal/nodes/publication_evidence.go`:

| Cause | Meaning |
| --- | --- |
| `stale_generation` | Applied generation or current-incarnation evidence is missing or behind the required generation. |
| `check_failed` | A reported check failed. |
| `forwarding_disabled` | The report does not confirm member forwarding enabled. |
| `tunnel_not_passed` | The tunnel probe has not passed. |
| `isolation_unconfirmed` | Safety, isolation, meter-path or isolation-apply evidence is insufficient. |
| `readiness_unconfirmed` | Publication intent/lifecycle or node readiness/health is insufficient. |
| `serving_checks_unconfirmed` | Lease, QoS or source-binding checks lack the required result. |
| `lease_unconfirmed` | A required reported lease ID/expiry is missing or expired, or the stored lease is not currently valid for this node/incarnation. |

Checks have precedence: an expired lease reported as a failed check may yield
`check_failed`. This helper does not emit `lease_expired`; an expired reported
or centrally clamped lease yields `lease_unconfirmed` when earlier checks pass.
`publication_observed=absent` is an observed publication state, not a gate cause.
The existing exception for a confirmed empty binding set remains unchanged.

Only the existing bounded error-message vocabulary and the gate causes above
are combined. Their full text fits the existing length limit without truncation.
Unlisted provider text is not added to the admin diagnostic, and member
operation responses retain their existing redaction. `NodeDetail` has no new
publication-cause field; its checks, drift and usage-outbox projection keep
their existing meanings.
