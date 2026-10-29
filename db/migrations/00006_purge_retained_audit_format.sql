-- +goose Up
-- D-035 / §15.4, §16.4: audit.Store.List consumes type:UUID resources and
-- the nil UUID for the system actor. Preserve all retention guards and evidence.
-- CREATE OR REPLACE preserves the existing owner, EXECUTE grants and function OID.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION public.purge_retained(target TEXT, cutoff TIMESTAMPTZ, reason TEXT) RETURNS BIGINT
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
 VALUES('00000000-0000-0000-0000-000000000000','system','retention.purge','fleet_config:' || fleet.id::text,
  jsonb_build_object('schema_version',1,'table',target,'fleet_generation',fleet.generation::text,
   'retention_config',to_jsonb(policy),'cutoff',cutoff,'deleted_rows',n::text,'reason',reason),'succeeded');
 RETURN n;
END $$;
-- +goose StatementEnd

-- +goose Down
-- D-035 rollback restores the previous producer body, including its legacy
-- audit encoding. It does not rewrite existing append-only audit rows.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION public.purge_retained(target TEXT, cutoff TIMESTAMPTZ, reason TEXT) RETURNS BIGINT
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
-- +goose StatementEnd
