-- +goose Up
-- §6.1, §6.2, AT07: restore only the stable Node registry through a freshly
-- consumed, proof-bound enrollment of a different incarnation. Old incarnation
-- and certificate revocation timestamps remain immutable; retired stays terminal.
-- Store the incarnation, public certificate and consumed enrollment before the
-- nodes UPDATE in the same transaction. A heartbeat alone has no such evidence.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION pki_guard_terminal() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_TABLE_NAME='nodes' THEN
  -- Retirement cannot be routed through revoked to reach the reinstall exception.
  IF OLD.lifecycle_state='retired' AND NEW.lifecycle_state<>'retired' THEN
   RAISE EXCEPTION 'terminal node identity' USING ERRCODE='23514';
  END IF;
  IF OLD.lifecycle_state='revoked' AND OLD.retired_at IS NULL
     AND NEW.lifecycle_state='registered'
     AND NEW.current_incarnation_id IS DISTINCT FROM OLD.current_incarnation_id
     AND EXISTS (SELECT 1 FROM node_incarnations i JOIN node_enrollments e
       ON e.incarnation_id=i.id AND e.node_id=i.node_id
       JOIN node_certificates c ON c.id=e.certificate_id AND c.incarnation_id=i.id
       WHERE i.id=NEW.current_incarnation_id AND i.node_id=NEW.id
         AND i.revoked_at IS NULL AND c.revoked_at IS NULL
         AND e.state='consumed' AND e.revoked_at IS NULL
         AND e.consumed_at>=transaction_timestamp() AND e.expiry>e.consumed_at
         AND e.public_key_fingerprint=i.public_key_fingerprint)
     AND NOT EXISTS (SELECT 1 FROM node_incarnations WHERE node_id=NEW.id
       AND id<>NEW.current_incarnation_id AND revoked_at IS NULL) THEN
   RETURN NEW;
  END IF;
  IF OLD.lifecycle_state IN ('revoked','retired') AND NEW.lifecycle_state NOT IN ('revoked','retired') THEN
   RAISE EXCEPTION 'terminal node identity' USING ERRCODE='23514';
  END IF;
 ELSIF OLD.revoked_at IS NOT NULL AND NEW.revoked_at IS DISTINCT FROM OLD.revoked_at THEN
  RAISE EXCEPTION 'terminal revocation' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd

-- +goose Down
-- Restore the stricter pre-reinstall guard; retain all historical identity rows.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION pki_guard_terminal() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_TABLE_NAME='nodes' THEN
  IF OLD.lifecycle_state IN ('revoked','retired') AND NEW.lifecycle_state NOT IN ('revoked','retired') THEN
   RAISE EXCEPTION 'terminal node identity' USING ERRCODE='23514';
  END IF;
 ELSIF OLD.revoked_at IS NOT NULL AND NEW.revoked_at IS DISTINCT FROM OLD.revoked_at THEN
  RAISE EXCEPTION 'terminal revocation' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
