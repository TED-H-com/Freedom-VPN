-- =============================================================================
-- Freedom VPN Post-Restore Operator Checklist Queries (`post-restore-checklist.sql`)
-- Spec references: §15.4, §16.2, §17.2, §17.3, §17.4, AT30 (SECOPS-05)
--
-- Strictly read-only inspection queries executed after a point-in-time restore
-- before resuming `fvpn-control` and `fvpn-worker`.
--
-- Usage:
--   psql -v ON_ERROR_STOP=1 -v restore_point='2026-10-29T18:02:00Z' \
--     -f deploy/backup/post-restore-checklist.sql
-- =============================================================================

SET SESSION CHARACTERISTICS AS TRANSACTION READ ONLY;

\if :{?restore_point}
\else
SELECT to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS restore_point \gset
\endif

\echo '=== [1/6] USAGE LEDGER EPOCH, MODE & HIGHEST ACKNOWLEDGED SEQUENCE PER NODE (§17.3) ==='
SELECT
    id AS ledger_state_id,
    current_epoch AS ledger_epoch,
    mode AS ledger_mode,
    mode_reason,
    changed_by,
    updated_at
FROM usage_ledger_state;

SELECT
    n.id AS node_id,
    COALESCE(c.incarnation_id, n.current_incarnation_id) AS incarnation_id,
    n.lifecycle_state,
    c.ledger_epoch,
    COALESCE(c.highest_contiguous_seq, 0)::text AS highest_contiguous_seq,
    COALESCE(c.max_seen_seq, 0)::text AS max_seen_seq,
    COALESCE(ev.committed_event_count, 0) AS committed_event_count,
    COALESCE(ev.max_committed_seq, 0)::text AS max_committed_seq,
    c.updated_at AS checkpoint_updated_at
FROM nodes n
LEFT JOIN usage_checkpoints c
    ON c.node_id = n.id
LEFT JOIN (
    SELECT
        node_id,
        incarnation_id,
        COUNT(*) AS committed_event_count,
        MAX(event_seq) AS max_committed_seq
    FROM usage_events
    GROUP BY node_id, incarnation_id
) ev
    ON ev.node_id = n.id
   AND ev.incarnation_id = COALESCE(c.incarnation_id, n.current_incarnation_id)
ORDER BY n.id, incarnation_id;

SELECT
    member_id,
    period,
    granularity,
    projection_generation::text AS projection_generation,
    upload_bytes::text AS upload_bytes,
    download_bytes::text AS download_bytes,
    total_bytes::text AS total_bytes,
    gap,
    reconciliation_in_progress,
    aggregate_as_of
FROM usage_rollups
WHERE reconciliation_in_progress = true
   OR gap = true
ORDER BY period DESC, member_id ASC;

\echo '=== [2/6] ENTITLEMENTS ENDING AFTER RESTORE POINT & SUBSCRIPTIONS TO RECONCILE (§9.3, §17.4) ==='
SELECT
    e.member_id,
    m.status AS member_status,
    m.admin_suspended,
    e.state AS entitlement_state,
    e.revision::text AS entitlement_revision,
    e.plan_revision_id,
    e.device_limit,
    e.valid_from,
    e.effective_service_end,
    e.computed_at
FROM entitlements e
JOIN members m
    ON m.id = e.member_id
WHERE e.effective_service_end IS NOT NULL
  AND e.effective_service_end > (:'restore_point')::timestamptz
ORDER BY e.effective_service_end ASC, e.member_id ASC;

SELECT
    s.id AS subscription_id,
    s.member_id,
    s.provider,
    s.external_subscription_id,
    s.provider_status,
    s.is_active,
    s.paid_from,
    s.paid_through,
    s.grace_until,
    s.last_reconciled_at,
    s.reconciliation_anomalies
FROM subscriptions s
WHERE s.is_active = true
   OR (s.paid_through IS NOT NULL AND s.paid_through > (:'restore_point')::timestamptz)
   OR (s.grace_until IS NOT NULL AND s.grace_until > (:'restore_point')::timestamptz)
ORDER BY s.updated_at DESC, s.id ASC;

SELECT
    id AS inbox_event_id,
    source,
    external_event_id,
    event_type,
    processing_status,
    attempt,
    last_error_code,
    received_at
FROM inbox_events
WHERE processing_status <> 'succeeded'
ORDER BY received_at ASC;

\echo '=== [3/6] PUBLICATION INTENTS & EXTERNAL SYNC OPERATIONS IN FLIGHT (§5.2, §12.5, §17.4) ==='
SELECT
    e.id AS exit_id,
    e.node_id,
    e.display_name,
    e.region,
    e.publication_intent,
    e.publication_observed,
    n.lifecycle_state,
    n.provisioning_state,
    n.control_status,
    n.data_health,
    n.safety_status,
    e.resource_version::text AS resource_version,
    e.updated_at
FROM exits e
JOIN nodes n
    ON n.id = e.node_id
WHERE e.publication_intent <> 'unpublished'
   OR e.publication_observed IN ('applying', 'active', 'error')
   OR e.publication_intent <> CASE
        WHEN e.publication_observed = 'active' THEN 'published'
        WHEN e.publication_observed = 'withdrawn' THEN 'disabled'
        ELSE e.publication_intent
      END
ORDER BY e.updated_at DESC, e.node_id ASC;

SELECT
    id AS binding_id,
    resource_type,
    resource_id,
    provider,
    deployment_id,
    external_id,
    sync_state,
    last_observed_at,
    last_verified_at,
    last_error
FROM external_bindings
WHERE sync_state NOT IN ('synced', 'in_sync')
   OR last_error IS NOT NULL
ORDER BY last_observed_at DESC NULLS FIRST;

SELECT
    definition_id,
    profile_revision_id,
    policy_generation::text AS policy_generation,
    document_sha256,
    status,
    resource_version::text AS resource_version,
    updated_at
FROM dns_publication_heads
ORDER BY updated_at DESC;

SELECT
    id AS operation_id,
    kind,
    resource_type,
    resource_id,
    desired_generation::text AS desired_generation,
    status,
    error_code,
    created_at,
    completed_at
FROM operations
WHERE status IN ('pending', 'running', 'failed', 'partially_succeeded')
ORDER BY created_at DESC;

SELECT
    id AS outbox_event_id,
    aggregate_type,
    aggregate_id,
    event_type,
    attempt,
    available_at,
    lease_owner,
    lease_until
FROM outbox_events
WHERE delivered_at IS NULL
ORDER BY available_at ASC, id ASC;

\echo '=== [4/6] UPDATE BATCHES & NODE UPDATE JOBS IN FLIGHT (§13.8, §17.4) ==='
SELECT
    fc.id AS fleet_config_id,
    fc.desired_release_id,
    r.version AS desired_release_version,
    fc.supported_release_policy,
    fc.generation::text AS fleet_generation,
    fc.support_generation::text AS support_generation,
    fc.updated_at
FROM fleet_config fc
LEFT JOIN releases r
    ON r.id = fc.desired_release_id;

SELECT
    b.id AS batch_id,
    b.operation_id,
    o.status AS operation_status,
    b.target_release_id,
    r.version AS target_version,
    b.scope,
    b.target_count,
    b.online_at_snapshot,
    b.generation::text AS generation,
    p.state AS batch_progress_state,
    p.counts AS progress_counts,
    b.created_at
FROM update_batches b
JOIN operations o
    ON o.id = b.operation_id
JOIN releases r
    ON r.id = b.target_release_id
LEFT JOIN update_batch_progress p
    ON p.batch_id = b.id
WHERE o.status IN ('pending', 'running', 'failed', 'partially_succeeded')
   OR COALESCE(p.state, '') NOT IN ('succeeded', 'cancelled')
ORDER BY b.created_at DESC;

SELECT
    j.id AS job_id,
    j.node_id,
    j.execution_incarnation_id,
    j.generation::text AS generation,
    j.attempt,
    j.state,
    j.stage,
    j.needs_attention,
    j.recovery_required,
    j.actual_release_id,
    j.last_good_release_id,
    j.error_code,
    j.created_at,
    j.completed_at
FROM update_jobs j
WHERE j.state IN ('queued', 'waiting_online', 'downloading', 'verifying', 'applying', 'checking', 'failed', 'rolled_back')
   OR j.needs_attention = true
   OR j.recovery_required = true
ORDER BY j.created_at DESC;

\echo '=== [5/6] REVOCATIONS, RETIREMENTS & SUSPENSIONS TO PRESERVE (SECOPS-05, §6.4, §9.2, §17.4) ==='
-- Operator must cross-check these records against the out-of-band signing vault
-- (/var/lib/fvpn/vault/state.json) and operator revocation logs so no revocation
-- or administrative suspension newer than the restore point is lost.
SELECT
    entity_type,
    entity_id,
    state,
    effective_at,
    reason,
    (effective_at >= (:'restore_point')::timestamptz) AS at_or_after_restore_point
FROM (
    SELECT
        'node' AS entity_type,
        id::text AS entity_id,
        lifecycle_state AS state,
        COALESCE(retired_at, updated_at) AS effective_at,
        notes AS reason
    FROM nodes
    WHERE lifecycle_state IN ('revoked', 'retired')
       OR retired_at IS NOT NULL
    UNION ALL
    SELECT
        'node_incarnation' AS entity_type,
        id::text AS entity_id,
        'revoked' AS state,
        revoked_at AS effective_at,
        'node_id=' || node_id::text AS reason
    FROM node_incarnations
    WHERE revoked_at IS NOT NULL
    UNION ALL
    SELECT
        'node_certificate' AS entity_type,
        id::text AS entity_id,
        'revoked' AS state,
        revoked_at AS effective_at,
        'serial=' || serial AS reason
    FROM node_certificates
    WHERE revoked_at IS NOT NULL
    UNION ALL
    SELECT
        'node_enrollment' AS entity_type,
        id::text AS entity_id,
        state,
        COALESCE(revoked_at, expiry) AS effective_at,
        'node_id=' || node_id::text AS reason
    FROM node_enrollments
    WHERE state = 'revoked'
       OR revoked_at IS NOT NULL
    UNION ALL
    SELECT
        'member' AS entity_type,
        id::text AS entity_id,
        status AS state,
        COALESCE(suspended_at, closed_at, updated_at) AS effective_at,
        COALESCE(suspension_reason, '') AS reason
    FROM members
    WHERE admin_suspended = true
       OR status IN ('suspended', 'closed')
       OR suspended_at IS NOT NULL
       OR closed_at IS NOT NULL
    UNION ALL
    SELECT
        'member_device' AS entity_type,
        id::text AS entity_id,
        status AS state,
        COALESCE(revoked_at, closed_at, updated_at) AS effective_at,
        'member_id=' || member_id::text AS reason
    FROM member_devices
    WHERE status IN ('revoked', 'closed')
       OR revoked_at IS NOT NULL
       OR closed_at IS NOT NULL
    UNION ALL
    SELECT
        'entitlement_grant' AS entity_type,
        id::text AS entity_id,
        'revoked' AS state,
        revoked_at AS effective_at,
        COALESCE(revocation_reason, '') AS reason
    FROM entitlement_grants
    WHERE revoked_at IS NOT NULL
    UNION ALL
    SELECT
        'revoked_release' AS entity_type,
        unnest(revoked_release_ids)::text AS entity_id,
        'revoked' AS state,
        updated_at AS effective_at,
        'fleet_config.revoked_release_ids' AS reason
    FROM fleet_config
) revocations
ORDER BY effective_at DESC NULLS LAST, entity_type ASC, entity_id ASC;

\echo '=== [6/6] REQUIRED USAGE LEDGER FREEZE & EPOCH ROTATION COMMANDS (fvpn-usage-admin, §17.3) ==='
SELECT
    s.current_epoch::text AS pre_recovery_ledger_epoch,
    s.mode AS current_ledger_mode,
    (:'restore_point')::timestamptz AS target_restore_point,
    format(
        '/usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json freeze --actor <ADMIN_UUID> --reason %L',
        'PITR restore to ' || (:'restore_point'::text)
    ) AS step_1_freeze_command,
    format(
        '/usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json begin-recovery --actor <ADMIN_UUID> --restored-to %L --affected-since %L --reason %L',
        to_char((:'restore_point')::timestamptz AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
        to_char(date_trunc('month', (:'restore_point')::timestamptz AT TIME ZONE 'UTC'), 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
        'PITR restore to ' || (:'restore_point'::text)
    ) AS step_2_begin_recovery_command,
    '/usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json complete-recovery --actor <ADMIN_UUID>' AS step_3_complete_recovery_command
FROM usage_ledger_state s;
