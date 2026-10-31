-- +goose Up
-- D-084 / D-088, §15.3, §16.3: the stored deadline may only get shorter.
-- Signed envelopes and all other issuance evidence remain immutable.
-- +goose StatementBegin
CREATE FUNCTION guard_serving_lease_clamp() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='UPDATE' THEN
  IF public.owner_setting('fvpn.migration') THEN RETURN NEW; END IF;
  IF NEW.expires_at < OLD.expires_at
     AND (to_jsonb(NEW)-'expires_at') IS NOT DISTINCT FROM (to_jsonb(OLD)-'expires_at') THEN
   RETURN NEW;
  END IF;
 END IF;
 RAISE EXCEPTION 'immutable record' USING ERRCODE='23514';
END $$;
-- +goose StatementEnd
DROP TRIGGER serving_leases_immutable ON serving_leases;
CREATE TRIGGER serving_leases_immutable BEFORE UPDATE OR DELETE ON serving_leases
 FOR EACH ROW EXECUTE FUNCTION guard_serving_lease_clamp();
-- serving_leases_no_truncate continues to call immutable_row().

-- +goose Down
DROP TRIGGER serving_leases_immutable ON serving_leases;
CREATE TRIGGER serving_leases_immutable BEFORE UPDATE OR DELETE ON serving_leases
 FOR EACH ROW EXECUTE FUNCTION immutable_row();
DROP FUNCTION guard_serving_lease_clamp();
