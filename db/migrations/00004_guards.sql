-- +goose Up
-- §15.1: retain stable historical identities, even before they have dependents.
-- +goose StatementBegin
CREATE FUNCTION forbid_delete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 RAISE EXCEPTION 'resource must be closed or tombstoned' USING ERRCODE='23514';
END $$;
-- +goose StatementEnd
CREATE TRIGGER members_keep BEFORE DELETE OR TRUNCATE ON members FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER nodes_keep BEFORE DELETE OR TRUNCATE ON nodes FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER member_devices_keep BEFORE DELETE OR TRUNCATE ON member_devices FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();
CREATE TRIGGER fleet_config_keep BEFORE DELETE OR TRUNCATE ON fleet_config FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();

-- §15.1: historical rows and singleton configuration cannot be removed.
-- +goose StatementBegin
DO $$
DECLARE t TEXT;
BEGIN
 FOREACH t IN ARRAY ARRAY['usage_checkpoints','update_jobs','operations','retention_config','subscriptions','entitlement_grants','node_certificates','node_observations','inbox_events','exits','device_bindings','node_incarnations'] LOOP
  EXECUTE format('CREATE TRIGGER %I BEFORE DELETE OR TRUNCATE ON %I FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete()',t||'_keep',t);
 END LOOP;
END $$;
-- An opt-in GUC is insufficient without the schema function owner's identity.
-- Use SET LOCAL only, inside a reviewed migration or isolated restore (§15.5, §17.2).
CREATE FUNCTION owner_setting(setting TEXT) RETURNS BOOLEAN LANGUAGE sql STABLE
SET search_path=pg_catalog AS $$
 SELECT coalesce(current_setting(setting,true)='on' AND current_user=pg_get_userbyid(proowner),false)
 FROM pg_proc WHERE oid='public.owner_setting(text)'::regprocedure
$$;
-- +goose StatementEnd

-- §15.1: resource versions must advance on every UPDATE; generations cannot regress.
-- +goose StatementBegin
CREATE FUNCTION monotonic_fields() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE field TEXT;
BEGIN
 FOREACH field IN ARRAY TG_ARGV LOOP
  IF (to_jsonb(NEW)->>field)::numeric < (to_jsonb(OLD)->>field)::numeric
     OR (field='resource_version' AND (to_jsonb(NEW)->>field)::numeric = (to_jsonb(OLD)->>field)::numeric) THEN
   RAISE EXCEPTION 'version regression: %', field USING ERRCODE='23514';
  END IF;
 END LOOP;
 RETURN NEW;
END $$;
CREATE FUNCTION touch_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at := clock_timestamp(); RETURN NEW; END $$;
DO $$
DECLARE t TEXT;
BEGIN
 FOREACH t IN ARRAY ARRAY['members','plans','subscriptions','member_devices','nodes','exits','policy_definitions','fleet_config'] LOOP
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION monotonic_fields(''resource_version'')',t||'_version',t);
 END LOOP;
 FOREACH t IN ARRAY ARRAY['members','plans','subscriptions','member_devices','nodes','exits','fleet_config','usage_checkpoints'] LOOP
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION touch_updated_at()',t||'_updated',t);
 END LOOP;
END $$;
-- +goose StatementEnd
CREATE TRIGGER nodes_generations BEFORE UPDATE ON nodes FOR EACH ROW EXECUTE FUNCTION monotonic_fields('desired_config_generation','applied_config_generation','update_generation');
CREATE TRIGGER fleet_generation BEFORE UPDATE ON fleet_config FOR EACH ROW EXECUTE FUNCTION monotonic_fields('generation');
CREATE TRIGGER netbird_user_bindings_revision BEFORE UPDATE ON netbird_user_bindings FOR EACH ROW EXECUTE FUNCTION monotonic_fields('revision');
CREATE TRIGGER entitlement_revision BEFORE UPDATE ON entitlements FOR EACH ROW EXECUTE FUNCTION monotonic_fields('revision');

-- §13.3, §15.2–4: immutable content and snapshot identity.
-- +goose StatementBegin
CREATE FUNCTION immutable_row() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='UPDATE' AND public.owner_setting('fvpn.migration') THEN RETURN NEW; END IF;
 RAISE EXCEPTION 'immutable record' USING ERRCODE='23514';
END $$;
DO $$
DECLARE t TEXT;
BEGIN
 FOREACH t IN ARRAY ARRAY['plan_revisions','policy_revisions','releases','artifacts','desired_states','serving_leases','serving_lease_members','update_batches'] LOOP
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION immutable_row()',t||'_immutable',t);
  EXECUTE format('CREATE TRIGGER %I BEFORE TRUNCATE ON %I FOR EACH STATEMENT EXECUTE FUNCTION immutable_row()',t||'_no_truncate',t);
 END LOOP;
END $$;
CREATE FUNCTION guard_update_target() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP <> 'UPDATE' THEN RAISE EXCEPTION 'immutable snapshot' USING ERRCODE='23514'; END IF;
 IF (to_jsonb(NEW)-'execution_incarnation_id'-'target_state') IS DISTINCT FROM (to_jsonb(OLD)-'execution_incarnation_id'-'target_state')
    OR (OLD.execution_incarnation_id IS NOT NULL AND NEW.execution_incarnation_id IS DISTINCT FROM OLD.execution_incarnation_id) THEN
  RAISE EXCEPTION 'immutable snapshot or execution binding' USING ERRCODE='23514';
 END IF;
 IF OLD.target_state<>'active' AND (NEW.target_state,NEW.execution_incarnation_id) IS DISTINCT FROM (OLD.target_state,OLD.execution_incarnation_id) THEN
  RAISE EXCEPTION 'cancelled target is frozen' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER update_targets_snapshot BEFORE UPDATE OR DELETE ON update_targets FOR EACH ROW EXECUTE FUNCTION guard_update_target();
CREATE TRIGGER update_targets_no_truncate BEFORE TRUNCATE ON update_targets FOR EACH STATEMENT EXECUTE FUNCTION immutable_row();

-- Snapshot membership must be inserted in the batch creation transaction (§15.5.6).
ALTER TABLE update_batches ADD COLUMN snapshot_xid XID8 NOT NULL DEFAULT pg_current_xact_id();
-- +goose StatementBegin
CREATE FUNCTION lock_snapshot_nodes() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NOT public.owner_setting('fvpn.restore') THEN
  -- Freeze the all_nodes population, including concurrent enrollment (§13.1).
  LOCK TABLE nodes IN SHARE MODE;
 END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION guard_snapshot_insert() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE n nodes;
BEGIN
 IF public.owner_setting('fvpn.restore') THEN RETURN NEW; END IF;
 IF (SELECT snapshot_xid FROM update_batches WHERE id=NEW.batch_id) <> pg_current_xact_id() THEN
  RAISE EXCEPTION 'batch snapshot is sealed' USING ERRCODE='23514';
 END IF;
 SELECT * INTO STRICT n FROM nodes WHERE id=NEW.node_id FOR SHARE;
 IF n.lifecycle_state IN ('revoked','retired') OR NEW.incarnation_at_snapshot IS DISTINCT FROM n.current_incarnation_id THEN
  RAISE EXCEPTION 'target is not an eligible snapshot member' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION check_snapshot_count() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.target_count <> (SELECT count(*) FROM update_targets WHERE batch_id=NEW.id) THEN
  RAISE EXCEPTION 'batch target count does not match snapshot' USING ERRCODE='23514';
 END IF;
 IF NOT public.owner_setting('fvpn.restore') AND (
    NEW.target_count <> (SELECT count(*) FROM nodes WHERE lifecycle_state NOT IN ('revoked','retired'))
    OR EXISTS(SELECT 1 FROM update_targets t JOIN nodes n ON n.id=t.node_id WHERE t.batch_id=NEW.id
      AND (n.lifecycle_state IN ('revoked','retired') OR t.incarnation_at_snapshot IS DISTINCT FROM n.current_incarnation_id))) THEN
  RAISE EXCEPTION 'all_nodes snapshot does not match eligible nodes' USING ERRCODE='23514';
 END IF;
 RETURN NULL;
END $$;
CREATE FUNCTION guard_job_identity() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE phases TEXT[] := ARRAY['queued','waiting_online','downloading','verifying','applying','checking'];
BEGIN
 IF (NEW.id,NEW.target_id,NEW.attempt,NEW.execution_incarnation_id,NEW.generation)
    IS DISTINCT FROM (OLD.id,OLD.target_id,OLD.attempt,OLD.execution_incarnation_id,OLD.generation)
    OR (OLD.result_id IS NOT NULL AND (NEW.result_id,NEW.result_hash) IS DISTINCT FROM (OLD.result_id,OLD.result_hash)) THEN
  RAISE EXCEPTION 'immutable job identity or result' USING ERRCODE='23514';
 END IF;
 IF (OLD.result_id IS NOT NULL OR NOT (OLD.state=ANY(phases))) AND
    (NEW.state,NEW.actual_release_id,NEW.last_good_release_id,NEW.completed_at,NEW.error_code,NEW.observed_evidence,NEW.schema_version)
    IS DISTINCT FROM (OLD.state,OLD.actual_release_id,OLD.last_good_release_id,OLD.completed_at,OLD.error_code,OLD.observed_evidence,OLD.schema_version) THEN
  RAISE EXCEPTION 'completed job evidence is frozen' USING ERRCODE='23514';
 END IF;
 IF NEW.state IS DISTINCT FROM OLD.state AND (
    (OLD.state IN ('queued','waiting_online') AND NEW.state IN ('queued','waiting_online'))
    OR (OLD.state IN ('queued','waiting_online') AND NEW.state='already_current')
    OR array_position(phases,NEW.state)>array_position(phases,OLD.state)
    OR (OLD.state=ANY(phases) AND NEW.state IN ('succeeded','failed','superseded_incarnation','cancelled_revoked','cancelled_retired'))
    OR (OLD.state IN ('queued','waiting_online','downloading','verifying') AND NEW.state='superseded')
    OR (OLD.state IN ('applying','checking') AND NEW.state='rolled_back')) IS NOT TRUE THEN
  RAISE EXCEPTION 'invalid job transition' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER update_batches_lock BEFORE INSERT ON update_batches FOR EACH ROW EXECUTE FUNCTION lock_snapshot_nodes();
CREATE TRIGGER update_targets_insert BEFORE INSERT ON update_targets FOR EACH ROW EXECUTE FUNCTION guard_snapshot_insert();
CREATE CONSTRAINT TRIGGER update_batches_count AFTER INSERT ON update_batches DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION check_snapshot_count();
CREATE TRIGGER update_jobs_identity BEFORE UPDATE ON update_jobs FOR EACH ROW EXECUTE FUNCTION guard_job_identity();

-- §16.3: validate signed member deadlines at issuance; revocation never edits old leases.
-- +goose StatementBegin
CREATE FUNCTION check_member_lease() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE l serving_leases; e entitlements; member_status TEXT;
BEGIN
 IF public.owner_setting('fvpn.restore') THEN RETURN NEW; END IF;
 SELECT * INTO STRICT l FROM serving_leases WHERE id=NEW.lease_id FOR SHARE;
 IF l.snapshot_xid <> pg_current_xact_id() THEN
  RAISE EXCEPTION 'signed lease membership is sealed' USING ERRCODE='23514';
 END IF;
 SELECT status INTO STRICT member_status FROM members WHERE id=NEW.member_id FOR SHARE;
 SELECT * INTO STRICT e FROM entitlements WHERE member_id=NEW.member_id FOR SHARE;
 IF member_status<>'active' OR e.admin_suspended OR e.revision <> NEW.entitlement_revision
    OR NEW.valid_from < greatest(l.issued_at,e.valid_from)
    OR NEW.expires_at > least(l.expires_at,e.effective_service_end) THEN
  RAISE EXCEPTION 'lease exceeds entitlement or node deadline' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER serving_lease_members_deadline BEFORE INSERT ON serving_lease_members FOR EACH ROW EXECUTE FUNCTION check_member_lease();

-- §10.6, §17.3: advancing a contiguous watermark requires durable receipts.
-- +goose StatementBegin
CREATE FUNCTION check_usage_checkpoint() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE base NUMERIC := 0; saved NUMERIC; seen NUMERIC;
BEGIN
 IF TG_OP='UPDATE' THEN
  IF (NEW.node_id,NEW.incarnation_id) IS DISTINCT FROM (OLD.node_id,OLD.incarnation_id) THEN
   RAISE EXCEPTION 'immutable stream' USING ERRCODE='23514';
  END IF;
  IF NEW.ledger_epoch=OLD.ledger_epoch THEN
   IF NEW.highest_contiguous_seq < OLD.highest_contiguous_seq OR NEW.max_seen_seq < OLD.max_seen_seq THEN
    RAISE EXCEPTION 'checkpoint regression within ledger epoch' USING ERRCODE='23514';
   END IF;
   base := OLD.highest_contiguous_seq;
  END IF;
 END IF;
 SELECT count(*) INTO saved FROM usage_events
 WHERE node_id=NEW.node_id AND incarnation_id=NEW.incarnation_id AND event_seq>base AND event_seq<=NEW.highest_contiguous_seq;
 SELECT coalesce(max(event_seq),0) INTO seen FROM usage_events WHERE node_id=NEW.node_id AND incarnation_id=NEW.incarnation_id;
 IF saved <> NEW.highest_contiguous_seq-base OR NEW.max_seen_seq <> seen
    OR EXISTS(SELECT 1 FROM usage_events WHERE node_id=NEW.node_id AND incarnation_id=NEW.incarnation_id AND event_seq=NEW.highest_contiguous_seq+1) THEN
  RAISE EXCEPTION 'checkpoint lacks durable contiguous receipts' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER usage_checkpoints_receipts BEFORE INSERT OR UPDATE ON usage_checkpoints FOR EACH ROW EXECUTE FUNCTION check_usage_checkpoint();

-- §10.6: ACK requires recoverable event contents, not just a receipt/hash.
-- Deferred so receipt, payload/samples and checkpoint can share one transaction.
-- +goose StatementBegin
CREATE FUNCTION check_usage_event_contents() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.record_count <> (SELECT count(*) FROM usage_samples WHERE event_id=NEW.id)
    AND NOT EXISTS(SELECT 1 FROM usage_event_payloads WHERE event_id=NEW.id) THEN
  RAISE EXCEPTION 'usage event lacks recoverable records' USING ERRCODE='23514';
 END IF;
 RETURN NULL;
END $$;
CREATE FUNCTION check_idempotency_reference() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 -- Re-read final state: INSERT's NEW image predates CompleteIdempotent.
 IF EXISTS(SELECT 1 FROM idempotency_requests WHERE id=NEW.id
           AND (response_status IS NULL OR (resource_id IS NULL AND operation_id IS NULL))) THEN
  RAISE EXCEPTION 'idempotency request lacks replay reference' USING ERRCODE='23514';
 END IF;
 RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER usage_events_contents AFTER INSERT ON usage_events DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION check_usage_event_contents();
CREATE CONSTRAINT TRIGGER idempotency_requests_reference AFTER INSERT OR UPDATE ON idempotency_requests DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION check_idempotency_reference();

-- §15.4, §16.4: only a tightly scoped SECURITY DEFINER path can purge.
-- A custom GUC alone is NOT authorization: the trigger also requires the function owner.
-- No cluster roles are created; deployment grants EXECUTE to a dedicated retention role.
-- +goose StatementBegin
CREATE FUNCTION guard_append_only() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE owner_name TEXT;
BEGIN
 SELECT pg_get_userbyid(proowner) INTO owner_name FROM pg_proc WHERE oid='public.purge_retained(text,timestamptz,text)'::regprocedure;
 IF TG_OP='DELETE' AND TG_TABLE_NAME IN ('audit_events','usage_event_payloads','usage_rollups')
    AND current_user=owner_name AND current_setting('fvpn.retention_purge',true)='on' THEN
  RETURN OLD;
 END IF;
 RAISE EXCEPTION 'append-only record; use authorized retention function' USING ERRCODE='23514';
END $$;
CREATE FUNCTION purge_retained(target TEXT, cutoff TIMESTAMPTZ, reason TEXT) RETURNS BIGINT
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public AS $$
DECLARE n BIGINT; earliest TIMESTAMPTZ; column_name TEXT; policy public.retention_config; fleet public.fleet_config; previous TEXT;
BEGIN
 IF reason IS NULL OR length(trim(reason))=0 OR cutoff IS NULL OR NOT isfinite(cutoff) THEN
  RAISE EXCEPTION 'finite cutoff and reason required' USING ERRCODE='22023';
 END IF;
 SELECT * INTO STRICT policy FROM public.retention_config WHERE id='00000000-0000-0000-0000-000000000002' FOR SHARE;
 CASE target
  WHEN 'usage_event_payloads' THEN earliest:=clock_timestamp()-make_interval(days=>policy.raw_usage_days); column_name:='created_at';
  WHEN 'audit_events' THEN earliest:=clock_timestamp()-make_interval(days=>policy.audit_days); column_name:='occurred_at';
  WHEN 'usage_rollups' THEN earliest:=clock_timestamp()-make_interval(months=>policy.rollup_months); column_name:='period_end';
  ELSE RAISE EXCEPTION 'retention target is not approved' USING ERRCODE='22023';
 END CASE;
 IF cutoff > earliest THEN RAISE EXCEPTION 'cutoff violates retention policy' USING ERRCODE='22023'; END IF;
 previous := current_setting('fvpn.retention_purge',true);
 PERFORM set_config('fvpn.retention_purge','on',true);
 IF target='usage_event_payloads' THEN
  -- Every record needs a complete interval covered by receipts and rollups in
  -- one projection generation/granularity. A single side of a month split is
  -- insufficient. The projector validates the allocation itself (§10.9).
  DELETE FROM public.usage_event_payloads p WHERE p.created_at < cutoff
   AND EXISTS(SELECT 1 FROM public.usage_events e WHERE e.id=p.event_id
    AND e.record_count=(SELECT count(*) FROM public.usage_samples s WHERE s.event_id=e.id)
    AND (e.record_count=0 OR EXISTS(
     SELECT 1 FROM public.usage_samples candidate
     JOIN public.projection_receipts cr ON cr.sample_id=candidate.id
     JOIN public.usage_rollups cu ON cu.member_id=candidate.member_id AND cu.period=cr.period
       AND cu.projection_generation=cr.generation AND cu.metering_version=candidate.metering_version
     WHERE candidate.event_id=e.id AND NOT EXISTS(
      SELECT 1 FROM public.usage_samples s WHERE s.event_id=e.id AND NOT EXISTS(
       SELECT 1 FROM public.projection_receipts r JOIN public.usage_rollups u
        ON u.member_id=s.member_id AND u.period=r.period AND u.projection_generation=r.generation AND u.metering_version=s.metering_version
       WHERE r.sample_id=s.id AND r.projection_name=cr.projection_name AND r.generation=cr.generation
        AND u.granularity=cu.granularity AND NOT u.reconciliation_in_progress AND NOT u.gap
       HAVING range_agg(tstzrange(u.period_start,u.period_end,'[)')) @> tstzrange(s.interval_start,s.interval_end,'[)')
      )
     )
    )));

 ELSE
  EXECUTE format('DELETE FROM public.%I WHERE %I < $1',target,column_name) USING cutoff;
 END IF;
 GET DIAGNOSTICS n=ROW_COUNT;
 PERFORM set_config('fvpn.retention_purge',coalesce(previous,''),true);
 -- §15.4, §16.4: use the contract's fleet_config ResourceRef; retention_config
 -- is not a ResourceType. Preserve the exact retention settings in the diff.
 SELECT * INTO STRICT fleet FROM public.fleet_config WHERE id='00000000-0000-0000-0000-000000000001' FOR SHARE;
 INSERT INTO public.audit_events(actor,actor_type,action,resource,after_diff,result)
 VALUES(session_user,'system','retention.purge',jsonb_build_object('type','fleet_config','id',fleet.id)::text,
  jsonb_build_object('schema_version',1,'table',target,'fleet_generation',fleet.generation::text,
   'retention_config',to_jsonb(policy),'cutoff',cutoff,'deleted_rows',n::text,'reason',reason),'succeeded');
 RETURN n;
END $$;
REVOKE ALL ON FUNCTION purge_retained(TEXT,TIMESTAMPTZ,TEXT) FROM PUBLIC;
DO $$
DECLARE t TEXT;
BEGIN
 FOREACH t IN ARRAY ARRAY['audit_events','billing_records','usage_adjustments','usage_events','usage_samples','projection_receipts','usage_event_payloads'] LOOP
  EXECUTE format('CREATE TRIGGER %I BEFORE UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION guard_append_only()',t||'_append_only',t);
  EXECUTE format('CREATE TRIGGER %I BEFORE TRUNCATE ON %I FOR EACH STATEMENT EXECUTE FUNCTION guard_append_only()',t||'_no_truncate',t);
 END LOOP;
END $$;
-- +goose StatementEnd
CREATE TRIGGER usage_rollups_delete_guard BEFORE DELETE ON usage_rollups FOR EACH ROW EXECUTE FUNCTION guard_append_only();
CREATE TRIGGER usage_rollups_no_truncate BEFORE TRUNCATE ON usage_rollups FOR EACH STATEMENT EXECUTE FUNCTION guard_append_only();

COMMENT ON TABLE members IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE identity_bindings IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE netbird_user_bindings IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE plans IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE plan_revisions IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE subscriptions IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE inbox_events IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE billing_records IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE entitlement_grants IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE entitlements IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE member_devices IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE device_bindings IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE member_groups IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE assignments IS 'Project Spec §15.2; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE releases IS 'Project Spec §13.3, §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE artifacts IS 'Project Spec §13.3, §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE nodes IS 'Project Spec §15.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE node_incarnations IS 'Project Spec §15.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE node_enrollments IS 'Project Spec §15.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE node_certificates IS 'Project Spec §15.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE exits IS 'Project Spec §15.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE exit_groups IS 'Project Spec §15.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE policy_definitions IS 'Project Spec §15.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE policy_revisions IS 'Project Spec §15.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE desired_states IS 'Project Spec §15.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE node_observations IS 'Project Spec §15.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE serving_leases IS 'Project Spec §15.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE serving_lease_members IS 'Project Spec §16.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE external_bindings IS 'Project Spec §15.3; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE usage_events IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE usage_event_payloads IS 'Project Spec §16.4, §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE usage_samples IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE projection_receipts IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE usage_checkpoints IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE usage_rollups IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE usage_adjustments IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE operations IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE fleet_config IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE update_batches IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE update_targets IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE update_jobs IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE outbox_events IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE idempotency_requests IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE audit_events IS 'Project Spec §15.4; see docs/db/README.md for constraints and transaction boundaries.';
COMMENT ON TABLE retention_config IS 'Project Spec §16.4; see docs/db/README.md for constraints and transaction boundaries.';
