-- +goose Up
-- §14.5, §17.3: durable canonical UsageBatchResult responses for keyed
-- POST /api/v1/agent/usage-batches replays across process restarts.
CREATE TABLE usage_batch_replays (
 id UUID PRIMARY KEY,
 node_id UUID NOT NULL REFERENCES nodes(id),
 incarnation_id UUID NOT NULL,
 response_body BYTEA NOT NULL CHECK(octet_length(response_body) > 0 AND octet_length(response_body) <= 1048576),
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 expiry TIMESTAMPTZ NOT NULL,
 FOREIGN KEY(node_id, incarnation_id) REFERENCES node_incarnations(node_id, id),
 CHECK(isfinite(expiry) AND created_at < expiry)
);
CREATE INDEX usage_batch_replays_expiry_idx ON usage_batch_replays(expiry);
COMMENT ON TABLE usage_batch_replays IS 'Project Spec §14.5, §17.3; durable canonical UsageBatchResult responses for idempotent POST /api/v1/agent/usage-batches replays.';

-- +goose StatementBegin
CREATE FUNCTION cleanup_usage_batch_replay() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 DELETE FROM usage_batch_replays WHERE id = OLD.id OR expiry <= clock_timestamp();
 RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER usage_batch_replays_cleanup_on_idempotency
 AFTER DELETE OR UPDATE OF id ON idempotency_requests
 FOR EACH ROW EXECUTE FUNCTION cleanup_usage_batch_replay();

-- +goose Down
DROP TRIGGER IF EXISTS usage_batch_replays_cleanup_on_idempotency ON idempotency_requests;
DROP FUNCTION IF EXISTS cleanup_usage_batch_replay();
DROP TABLE IF EXISTS usage_batch_replays;
