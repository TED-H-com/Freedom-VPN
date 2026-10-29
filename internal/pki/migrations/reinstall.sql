-- Proposed additive domain migration; orchestrator must install through the
-- database owner's embedded migration directory. Never run DDL in HTTP startup.
-- §6.2, AT07: a revoked incarnation stays revoked. Only newly consumed, explicit
-- enrollment of another incarnation may restore the stable Node registry row.
CREATE OR REPLACE FUNCTION pki_guard_terminal() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_TABLE_NAME='nodes' THEN
  IF OLD.lifecycle_state='revoked' AND NEW.lifecycle_state='registered'
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
