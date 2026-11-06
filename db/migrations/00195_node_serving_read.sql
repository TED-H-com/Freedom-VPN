-- +goose Up
-- D-164: diagnostic projection shares the selected checks' atomic node update.
-- The incarnation binding prevents a reinstall from exposing the previous read.
ALTER TABLE nodes
 ADD COLUMN serving_read_incarnation_id UUID,
 ADD COLUMN serving_read JSONB NOT NULL DEFAULT 'null'::jsonb,
 ADD CONSTRAINT nodes_serving_read_incarnation_fk FOREIGN KEY(id,serving_read_incarnation_id) REFERENCES node_incarnations(node_id,id);

-- Nodes is locked by ALTER TABLE before inspecting legacy observations. No
-- accepted-report ordering can be reconstructed from received_at or UUIDs.
-- Omitted and explicit null both mean no available read in the detail view;
-- the original report payload and canonical hash remain unchanged.
-- +goose StatementBegin
DO $$
BEGIN
 IF EXISTS (
  SELECT 1 FROM nodes n JOIN node_observations o
   ON o.node_id=n.id AND o.incarnation_id=n.current_incarnation_id
   AND o.observed_at=n.last_observed_at
  GROUP BY n.id
  HAVING count(DISTINCT coalesce(o.report_payload->'serving_read','null'::jsonb)) > 1
 ) THEN
  RAISE EXCEPTION 'Ambiguous legacy serving_read: accept a fresh observation with a strictly later observed_at before retrying migration 00195';
 END IF;
 UPDATE nodes n SET resource_version=n.resource_version+1,
  serving_read_incarnation_id=n.current_incarnation_id,
  serving_read=coalesce((SELECT coalesce(o.report_payload->'serving_read','null'::jsonb)
    FROM node_observations o WHERE o.node_id=n.id
     AND o.incarnation_id=n.current_incarnation_id
     AND o.observed_at=n.last_observed_at LIMIT 1),'null'::jsonb);
END $$;
-- +goose StatementEnd

ALTER TABLE nodes ADD CONSTRAINT nodes_serving_read_shape CHECK (
 serving_read='null'::jsonb OR (
  jsonb_typeof(serving_read)='object'
  AND serving_read ?& ARRAY['read_at','source','candidate_generation','verified','diagnostic']
  AND serving_read - ARRAY['read_at','source','candidate_generation','verified','diagnostic']='{}'::jsonb
  AND jsonb_typeof(serving_read->'read_at')='string'
  AND serving_read->>'read_at' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$'
  AND jsonb_typeof(serving_read->'source')='string'
  AND serving_read->>'source' IN ('apply','live')
  AND jsonb_typeof(serving_read->'candidate_generation')='string'
  AND serving_read->>'candidate_generation' ~ '^(0|[1-9][0-9]{0,19})$'
  AND (serving_read->>'candidate_generation')::numeric <= 18446744073709551615
  AND jsonb_typeof(serving_read->'verified')='boolean'
  AND jsonb_typeof(serving_read->'diagnostic')='string'
  AND char_length(serving_read->>'diagnostic') <= 500
  AND serving_read_incarnation_id IS NOT NULL
 ));
COMMENT ON COLUMN nodes.serving_read IS 'D-164 retained diagnostic from the same accepted observation as latest_checks; never serving permission or generation acknowledgement.';

-- +goose Down
-- Removing the representation also invalidates existing strong ETags.
UPDATE nodes SET resource_version=resource_version+1;
ALTER TABLE nodes DROP CONSTRAINT nodes_serving_read_shape;
ALTER TABLE nodes DROP COLUMN serving_read, DROP COLUMN serving_read_incarnation_id;
