-- +goose Up
-- §16.3 recommended default is configurable, not a new entitlement source.
ALTER TABLE fleet_config ADD COLUMN max_offline_lease_seconds INTEGER NOT NULL DEFAULT 86400 CHECK(max_offline_lease_seconds > 0);
-- +goose StatementBegin
CREATE FUNCTION check_node_lease() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE maximum INTEGER; n nodes; i node_incarnations;
BEGIN
 IF public.owner_setting('fvpn.restore') THEN RETURN NEW; END IF;
 SELECT * INTO STRICT n FROM nodes WHERE id=NEW.node_id FOR SHARE;
 SELECT * INTO STRICT i FROM node_incarnations WHERE id=NEW.incarnation_id FOR SHARE;
 IF n.lifecycle_state<>'registered' OR i.revoked_at IS NOT NULL
    OR n.current_incarnation_id IS DISTINCT FROM NEW.incarnation_id OR i.node_id<>NEW.node_id THEN
  RAISE EXCEPTION 'lease requires current registered incarnation' USING ERRCODE='23514';
 END IF;
 SELECT max_offline_lease_seconds INTO STRICT maximum FROM fleet_config WHERE id='00000000-0000-0000-0000-000000000001' FOR SHARE;
 IF NEW.expires_at > NEW.issued_at + make_interval(secs=>maximum) THEN
  RAISE EXCEPTION 'lease exceeds configured offline maximum' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER serving_leases_maximum BEFORE INSERT ON serving_leases FOR EACH ROW EXECUTE FUNCTION check_node_lease();

-- §15.3: first public-key binding is irreversible; retries read it under row lock.
-- +goose StatementBegin
CREATE FUNCTION guard_enrollment_binding() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF (NEW.id,NEW.node_id,NEW.token_hash,NEW.created_at,NEW.expiry) IS DISTINCT FROM (OLD.id,OLD.node_id,OLD.token_hash,OLD.created_at,OLD.expiry)
    OR (OLD.public_key_fingerprint IS NOT NULL AND NEW.public_key_fingerprint IS DISTINCT FROM OLD.public_key_fingerprint)
    OR (OLD.consumed_at IS NOT NULL AND NEW.consumed_at IS DISTINCT FROM OLD.consumed_at)
    OR (OLD.revoked_at IS NOT NULL AND NEW.revoked_at IS DISTINCT FROM OLD.revoked_at) THEN
  RAISE EXCEPTION 'immutable enrollment binding' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER node_enrollments_binding BEFORE UPDATE ON node_enrollments FOR EACH ROW EXECUTE FUNCTION guard_enrollment_binding();

-- §14.5, §15.4: an operation succeeds only with evidence for every required step.
-- steps is an object keyed by step name; each value has required:boolean and
-- status. Required successful steps identify an observed_refs entry by name.
-- +goose StatementBegin
CREATE FUNCTION guard_operation() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE step JSONB; reference TEXT; required_count INTEGER := 0;
BEGIN
 -- Restores replay historical terminal operations, not new commands (§17.2).
 IF TG_OP='INSERT' AND public.owner_setting('fvpn.restore') THEN RETURN NEW; END IF;
 IF TG_OP='INSERT' AND NEW.status NOT IN ('pending','running') THEN
  RAISE EXCEPTION 'operation must start pending or running' USING ERRCODE='23514';
 END IF;
 IF TG_OP='UPDATE' AND OLD.status IN ('succeeded','superseded') AND NEW IS DISTINCT FROM OLD THEN
  RAISE EXCEPTION 'terminal operation is frozen' USING ERRCODE='23514';
 END IF;
 IF TG_OP='UPDATE' AND OLD.status IS DISTINCT FROM NEW.status THEN
  IF NOT ((OLD.status='pending' AND NEW.status IN ('running','failed','superseded'))
   OR (OLD.status='running' AND NEW.status IN ('succeeded','partially_succeeded','failed','superseded'))
   OR (OLD.status IN ('failed','partially_succeeded') AND NEW.status='running')) THEN
   RAISE EXCEPTION 'invalid operation transition' USING ERRCODE='23514';
  END IF;
 END IF;
 IF NEW.status='succeeded' THEN
  IF NEW.steps='{}' THEN RAISE EXCEPTION 'required-step evidence missing' USING ERRCODE='23514'; END IF;
  FOR step IN SELECT value FROM jsonb_each(NEW.steps) LOOP
   IF jsonb_typeof(step) IS DISTINCT FROM 'object' OR jsonb_typeof(step->'required') IS DISTINCT FROM 'boolean' THEN
    RAISE EXCEPTION 'invalid operation step' USING ERRCODE='23514';
   END IF;
   IF (step->>'required')::boolean THEN required_count:=required_count+1; END IF;
   IF (step->>'required')::boolean OR step->>'status'='succeeded' THEN
    reference:=step->>'observed_ref';
    IF step->>'status' IS DISTINCT FROM 'succeeded' OR reference IS NULL
       OR NOT (NEW.observed_refs ? reference) OR NEW.observed_refs->reference IN ('null'::jsonb,'{}'::jsonb,'[]'::jsonb,'""'::jsonb) THEN
     RAISE EXCEPTION 'required-step evidence missing' USING ERRCODE='23514';
    END IF;
   ELSIF step->>'status' IS DISTINCT FROM 'not_required' OR NOT coalesce(step->>'reason' ~ '[^[:space:]]',false) THEN
    RAISE EXCEPTION 'optional step needs explicit disposition' USING ERRCODE='23514';
   END IF;
  END LOOP;
  IF required_count=0 THEN RAISE EXCEPTION 'at least one required step must succeed' USING ERRCODE='23514'; END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION check_job_success() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE target_release UUID;
BEGIN
 IF NEW.state='succeeded' THEN
  SELECT b.target_release_id INTO STRICT target_release FROM update_targets t JOIN update_batches b ON b.id=t.batch_id WHERE t.id=NEW.target_id;
  IF NEW.actual_release_id IS DISTINCT FROM target_release THEN
   RAISE EXCEPTION 'actual release differs from update target' USING ERRCODE='23514';
  END IF;
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER operations_evidence BEFORE INSERT OR UPDATE ON operations FOR EACH ROW EXECUTE FUNCTION guard_operation();
CREATE TRIGGER update_jobs_success BEFORE INSERT OR UPDATE ON update_jobs FOR EACH ROW EXECUTE FUNCTION check_job_success();

-- §15.1: core entity IDs and scoped identity associations remain stable.
-- +goose StatementBegin
CREATE FUNCTION immutable_fields() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE field TEXT;
BEGIN
 IF public.owner_setting('fvpn.migration') THEN RETURN NEW; END IF;
 FOREACH field IN ARRAY TG_ARGV LOOP
  IF (to_jsonb(NEW)->field) IS DISTINCT FROM (to_jsonb(OLD)->field) THEN
   RAISE EXCEPTION 'immutable identity field: %',field USING ERRCODE='23514';
  END IF;
 END LOOP;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER members_identity BEFORE UPDATE ON members FOR EACH ROW EXECUTE FUNCTION immutable_fields('id');
CREATE TRIGGER nodes_identity BEFORE UPDATE ON nodes FOR EACH ROW EXECUTE FUNCTION immutable_fields('id');
CREATE TRIGGER member_devices_identity BEFORE UPDATE ON member_devices FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','member_id','deployment_id','peer_id');
CREATE TRIGGER device_bindings_identity BEFORE UPDATE ON device_bindings FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','device_id','member_id','overlay_scope','overlay_ip','binding_revision','valid_from','peer_ref');
CREATE TRIGGER node_incarnations_identity BEFORE UPDATE ON node_incarnations FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','node_id','public_key_fingerprint','deployment_id','peer_id','created_at');
CREATE TRIGGER identity_bindings_identity BEFORE UPDATE ON identity_bindings FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','member_id','issuer','client_scope','subject');
CREATE TRIGGER netbird_user_bindings_identity BEFORE UPDATE ON netbird_user_bindings FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','member_id','deployment_id','external_user_id','identity_binding_id');

-- §13.8: a known snapshot incarnation cannot be replaced by a reinstallation.
ALTER TABLE update_targets ADD CONSTRAINT update_targets_snapshot_execution_check
 CHECK(incarnation_at_snapshot IS NULL OR execution_incarnation_id IS NULL OR incarnation_at_snapshot=execution_incarnation_id);

-- §2, §10.6: persist the historical binding, never reinterpret by current IP.
-- +goose StatementBegin
CREATE FUNCTION check_sample_binding() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE binding device_bindings; records INTEGER;
BEGIN
 SELECT * INTO STRICT binding FROM device_bindings WHERE device_id=NEW.device_id AND binding_revision=NEW.binding_revision FOR SHARE;
 IF NEW.member_id <> binding.member_id OR NEW.interval_start < binding.valid_from OR NEW.interval_end > binding.valid_until THEN
  RAISE EXCEPTION 'sample falls outside historical binding' USING ERRCODE='23514';
 END IF;
 SELECT record_count INTO STRICT records FROM usage_events WHERE id=NEW.event_id;
 IF NEW.record_index >= records THEN RAISE EXCEPTION 'sample index exceeds envelope record count' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER usage_samples_binding BEFORE INSERT ON usage_samples FOR EACH ROW EXECUTE FUNCTION check_sample_binding();

-- §9.2, §9.4, §15.5.2–3: serialize authorization and eligibility decisions.
-- The caller locks member, then entitlement; the trigger also enforces this
-- protocol when a device writer omitted it. Existing authorizations do not
-- automatically reappear after revocation.
-- +goose StatementBegin
CREATE FUNCTION check_device_authorization() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE member_status TEXT; e entitlements;
BEGIN
 IF NEW.status<>'authorized' THEN RETURN NEW; END IF;
 SELECT status INTO STRICT member_status FROM members WHERE id=NEW.member_id FOR UPDATE;
 SELECT * INTO e FROM entitlements WHERE member_id=NEW.member_id FOR SHARE;
 IF NOT FOUND OR member_status<>'active' OR e.admin_suspended
    OR NEW.revoked_at IS NOT NULL OR NEW.closed_at IS NOT NULL
    OR clock_timestamp()<e.valid_from OR clock_timestamp()>=e.effective_service_end
    OR (SELECT count(*) FROM member_devices WHERE member_id=NEW.member_id AND status='authorized' AND id<>NEW.id)>=e.device_limit THEN
  RAISE EXCEPTION 'device authorization exceeds current entitlement' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION guard_binding_end() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 -- check_sample_binding takes FOR SHARE on this row, serializing both orders.
 IF NEW.valid_until < OLD.valid_until AND EXISTS(
    SELECT 1 FROM usage_samples WHERE device_id=NEW.device_id AND binding_revision=NEW.binding_revision AND interval_end>NEW.valid_until) THEN
  RAISE EXCEPTION 'binding end would invalidate historical usage' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION irreversible_timestamps() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE field TEXT;
BEGIN
 FOREACH field IN ARRAY TG_ARGV LOOP
  IF to_jsonb(OLD)->field <> 'null'::jsonb AND to_jsonb(NEW)->field IS DISTINCT FROM to_jsonb(OLD)->field THEN
   RAISE EXCEPTION 'irreversible lifecycle timestamp: %',field USING ERRCODE='23514';
  END IF;
 END LOOP;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER member_devices_authorization BEFORE INSERT OR UPDATE ON member_devices FOR EACH ROW EXECUTE FUNCTION check_device_authorization();
CREATE TRIGGER device_bindings_end BEFORE UPDATE ON device_bindings FOR EACH ROW EXECUTE FUNCTION guard_binding_end();
CREATE INDEX usage_samples_binding_end_idx ON usage_samples(device_id,binding_revision,interval_end);
CREATE TRIGGER subscriptions_identity BEFORE UPDATE ON subscriptions FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','member_id','provider','external_subscription_id','external_customer_id','created_at');
CREATE TRIGGER node_certificates_identity BEFORE UPDATE ON node_certificates FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','incarnation_id','issuer','serial','fingerprint','public_certificate','not_before','not_after');
CREATE TRIGGER entitlement_grants_identity BEFORE UPDATE ON entitlement_grants FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','member_id','source','valid_from','valid_until','reason','created_by','created_at');
CREATE TRIGGER inbox_events_identity BEFORE UPDATE ON inbox_events FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','source','external_event_id','payload_hash','encrypted_payload','received_at');
CREATE TRIGGER node_observations_identity BEFORE UPDATE ON node_observations FOR EACH ROW EXECUTE FUNCTION immutable_row();
CREATE TRIGGER members_closed BEFORE UPDATE ON members FOR EACH ROW EXECUTE FUNCTION irreversible_timestamps('closed_at');
CREATE TRIGGER nodes_retired BEFORE UPDATE ON nodes FOR EACH ROW EXECUTE FUNCTION irreversible_timestamps('retired_at');
CREATE TRIGGER member_devices_revoked BEFORE UPDATE ON member_devices FOR EACH ROW EXECUTE FUNCTION irreversible_timestamps('revoked_at','closed_at');
CREATE TRIGGER node_incarnations_revoked BEFORE UPDATE ON node_incarnations FOR EACH ROW EXECUTE FUNCTION irreversible_timestamps('revoked_at');
CREATE TRIGGER node_certificates_revoked BEFORE UPDATE ON node_certificates FOR EACH ROW EXECUTE FUNCTION irreversible_timestamps('revoked_at');
CREATE TRIGGER entitlement_grants_revoked BEFORE UPDATE ON entitlement_grants FOR EACH ROW EXECUTE FUNCTION irreversible_timestamps('revoked_at','revocation_reason');
