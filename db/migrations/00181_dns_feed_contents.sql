-- +goose Up
-- D-054 / §12.3: worker projections; never member query data or raw feed bodies.
CREATE TABLE dns_feed_contents (
    feed_revision_id UUID NOT NULL REFERENCES policy_revisions(id),
    content_sha256 TEXT NOT NULL CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0 AND size_bytes <= 32000000),
    rule_count INTEGER NOT NULL CHECK (rule_count > 0 AND rule_count <= 500000),
    entries JSONB NOT NULL CHECK (jsonb_typeof(entries) = 'array' AND jsonb_array_length(entries) = rule_count),
    parse_metrics JSONB NOT NULL CHECK (jsonb_typeof(parse_metrics) = 'object'),
    fetched_at TIMESTAMPTZ NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('approved_snapshot', 'refreshed')),
    PRIMARY KEY (feed_revision_id, content_sha256)
);
CREATE TRIGGER dns_feed_contents_immutable BEFORE UPDATE OR DELETE ON dns_feed_contents
    FOR EACH ROW EXECUTE FUNCTION immutable_row();
CREATE TRIGGER dns_feed_contents_no_truncate BEFORE TRUNCATE ON dns_feed_contents
    FOR EACH STATEMENT EXECUTE FUNCTION immutable_row();
CREATE TABLE dns_feed_refresh_state (
    feed_revision_id UUID PRIMARY KEY REFERENCES policy_revisions(id),
    last_good_content TEXT,
    last_attempt_at TIMESTAMPTZ,
    last_success_at TIMESTAMPTZ,
    last_error_code TEXT CHECK (last_error_code ~ '^[a-z_]{1,64}$'),
    consecutive_failures BIGINT NOT NULL DEFAULT 0 CHECK (consecutive_failures >= 0),
    next_attempt_at TIMESTAMPTZ,
    attempt_token UUID,
    attempt_until TIMESTAMPTZ,
    last_event_id UUID,
    FOREIGN KEY (feed_revision_id, last_good_content) REFERENCES dns_feed_contents(feed_revision_id, content_sha256)
);
COMMENT ON TABLE dns_feed_refresh_state IS 'D-054 operator observations written only by dns.feed.refresh. NULL last_good_content means bootstrap has never passed. Error codes never contain source URLs or domains.';

-- The existing activation outbox remains available to other domains.
-- +goose StatementBegin
CREATE FUNCTION enqueue_dns_feed_refresh() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.event_type = 'policy.activated' AND NEW.payload->>'type' = 'threat_feed' THEN
        INSERT INTO outbox_events (aggregate_id, aggregate_type, aggregate_version, event_type, schema_version, payload)
        VALUES ((NEW.payload->>'revision_id')::uuid, 'policy_revision', NEW.aggregate_version,
                'dns.feed.refresh', 1, jsonb_build_object('revision_id', NEW.payload->>'revision_id'));
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER outbox_events_dns_feed_refresh AFTER INSERT ON outbox_events
    FOR EACH ROW EXECUTE FUNCTION enqueue_dns_feed_refresh();

-- One revision can now have several immutable content generations. Existing
-- references still identify the profile revision; the head identifies its version.
ALTER TABLE dns_publication_heads DROP CONSTRAINT dns_publication_heads_profile_revision_id_fkey;
ALTER TABLE dns_resolver_documents DROP CONSTRAINT dns_resolver_documents_pkey;
ALTER TABLE dns_resolver_documents ADD PRIMARY KEY (profile_revision_id, policy_generation);
ALTER TABLE dns_publication_heads ADD FOREIGN KEY (profile_revision_id, policy_generation)
    REFERENCES dns_resolver_documents(profile_revision_id, policy_generation);
