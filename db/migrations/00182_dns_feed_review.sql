-- +goose Up
-- D-054 review / §12.3, §14.5: separate an in-flight lease from completed state
-- and fence reactivation of the same immutable revision (including ABA races).
ALTER TABLE dns_feed_refresh_state
    ADD COLUMN activation_generation NUMERIC(20,0) NOT NULL DEFAULT 0 CHECK (activation_generation >= 0 AND activation_generation <= 18446744073709551615),
    ADD COLUMN last_receipt TEXT;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enqueue_dns_feed_refresh() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.event_type = 'policy.activated' AND NEW.payload->>'type' = 'threat_feed' THEN
        INSERT INTO outbox_events (aggregate_id, aggregate_type, aggregate_version, event_type, schema_version, payload, available_at)
        VALUES ((NEW.payload->>'revision_id')::uuid, 'policy_revision', NEW.aggregate_version,
                'dns.feed.refresh', 1, jsonb_build_object('revision_id', NEW.payload->>'revision_id',
                    'operation_id', NEW.payload->>'operation_id', 'generation', NEW.payload->>'generation', 'activation', true), NEW.available_at);
        -- Pinned profiles are never silently repointed to another revision.
        -- Re-evaluate every referencing profile in the feed definition's lineage.
        INSERT INTO outbox_events (aggregate_id, aggregate_type, aggregate_version, event_type, schema_version, payload, available_at)
        SELECT p.id, 'policy_definition', p.generation, 'dns.publish', 1,
            jsonb_build_object('type','dns_profile','definition_id',p.id,'revision_id',r.id,'generation',p.generation::text,'action','feed_lifecycle'), NEW.available_at
        FROM policy_definitions p JOIN policy_revisions r ON r.id=p.active_revision_id
        WHERE r.type='dns_profile' AND EXISTS (
            SELECT 1 FROM policy_revisions f WHERE f.definition_id=NEW.aggregate_id
            AND r.payload->'threat_feed_revision_ids' @> jsonb_build_array(f.id::text));
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd

-- Use the same lifecycle check for publication repair and authenticated delivery.
-- +goose StatementBegin
CREATE FUNCTION dns_profile_feeds_eligible(profile UUID, at_time TIMESTAMPTZ) RETURNS BOOLEAN
LANGUAGE sql STABLE AS $$
 SELECT NOT EXISTS (
  SELECT 1 FROM policy_revisions r,
   LATERAL jsonb_array_elements_text(r.payload->'threat_feed_revision_ids') ids(id)
   LEFT JOIN policy_revisions f ON f.id::text=ids.id
   LEFT JOIN policy_definitions d ON d.id=f.definition_id
  WHERE r.id=profile AND (f.id IS NULL OR f.type<>'threat_feed'
   OR d.active_revision_id IS DISTINCT FROM f.id
   OR at_time < (f.payload->>'valid_from')::timestamptz
   OR at_time >= (f.payload->>'valid_until')::timestamptz)
 );
$$;
-- +goose StatementEnd

-- Distinguish recoverable feed invalidation from explicit profile withdrawal.
ALTER TABLE dns_publication_heads ADD COLUMN feed_invalidated BOOLEAN NOT NULL DEFAULT false;

-- In-flight activations upgraded from earlier workers also require domain
-- read-back. Completed historical operations are not rewritten.
UPDATE operations o SET steps=o.steps || jsonb_build_object(
    CASE d.type WHEN 'dns_profile' THEN 'dns_document' ELSE 'dns_feed_refresh' END,
    jsonb_build_object('required',true,'status','pending'))
FROM policy_definitions d WHERE o.kind='policy.activate' AND o.resource_id=d.id
 AND o.status IN ('pending','running','failed') AND d.type IN ('dns_profile','threat_feed')
 AND NOT o.steps ? CASE d.type WHEN 'dns_profile' THEN 'dns_document' ELSE 'dns_feed_refresh' END;

INSERT INTO outbox_events(aggregate_id,aggregate_type,aggregate_version,event_type,schema_version,payload)
SELECT CASE d.type WHEN 'dns_profile' THEN d.id ELSE r.id END,
 CASE d.type WHEN 'dns_profile' THEN 'policy_definition' ELSE 'policy_revision' END,
 d.generation,CASE d.type WHEN 'dns_profile' THEN 'dns.publish' ELSE 'dns.feed.refresh' END,1,
 jsonb_build_object('definition_id',d.id,'revision_id',r.id,'generation',d.generation::text,'operation_id',o.id,'type',d.type)
FROM policy_definitions d JOIN policy_revisions r ON r.id=d.active_revision_id
 JOIN operations o ON o.resource_id=d.id AND o.desired_generation=d.generation
WHERE o.kind='policy.activate' AND o.status IN ('pending','running','failed')
 AND d.type IN ('dns_profile','threat_feed');
