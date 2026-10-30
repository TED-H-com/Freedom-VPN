-- +goose Up
-- D-052 / §15.2: public subjects identify one member across portal client rotation.
-- Never merge identities or members silently. Report only a count, not subjects.
LOCK TABLE identity_bindings IN ACCESS EXCLUSIVE MODE;
-- +goose StatementBegin
DO $$
DECLARE duplicates BIGINT;
BEGIN
 SELECT count(*) INTO duplicates FROM (
  SELECT issuer, subject FROM identity_bindings GROUP BY issuer, subject HAVING count(*) > 1
 ) conflicts;
 IF duplicates > 0 THEN
  RAISE EXCEPTION 'identity_bindings duplicate issuer/subject groups: %', duplicates USING ERRCODE='23505';
 END IF;
END $$;
-- +goose StatementEnd
ALTER TABLE identity_bindings ADD CONSTRAINT identity_bindings_issuer_subject_key UNIQUE (issuer, subject);
