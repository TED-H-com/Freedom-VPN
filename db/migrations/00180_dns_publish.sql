-- +goose Up
-- §12.5: canonical resolver documents produced by the DNS publish worker.
-- Keyed by the policy revision id that desired_states.policy_refs already carries.
-- The bytes column is the RFC 8785 document; JSONB is an operator view of the same object.
CREATE TABLE dns_resolver_documents (
    profile_revision_id UUID PRIMARY KEY REFERENCES policy_revisions(id),
    definition_id UUID NOT NULL REFERENCES policy_definitions(id),
    profile_code TEXT CHECK (profile_code ~ '[^[:space:]]') NOT NULL,
    policy_generation NUMERIC(20,0) CHECK (policy_generation >= 0 AND policy_generation <= 18446744073709551615) NOT NULL,
    document_sha256 TEXT CHECK (document_sha256 ~ '^[0-9a-f]{64}$') NOT NULL,
    document JSONB CHECK (jsonb_typeof(document) = 'object') NOT NULL,
    canonical_bytes BYTEA CHECK (octet_length(canonical_bytes) > 0 AND octet_length(canonical_bytes) <= 1048576) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
COMMENT ON TABLE dns_resolver_documents IS 'Immutable RFC 8785 resolver documents for an activated dns_profile revision (§12.4, §12.5). policy_generation versions the publication; it is not part of the hashed bytes.';

CREATE TRIGGER dns_resolver_documents_immutable
    BEFORE UPDATE OR DELETE ON dns_resolver_documents
    FOR EACH ROW EXECUTE FUNCTION immutable_row();

CREATE TRIGGER dns_resolver_documents_no_truncate
    BEFORE TRUNCATE ON dns_resolver_documents
    FOR EACH STATEMENT EXECUTE FUNCTION immutable_row();

-- Current publication for a definition. Supersede moves the pointer forward.
-- Withdraw sets status without deleting the document (§12.5, §17.1).
CREATE TABLE dns_publication_heads (
    definition_id UUID PRIMARY KEY REFERENCES policy_definitions(id),
    profile_revision_id UUID NOT NULL REFERENCES dns_resolver_documents(profile_revision_id),
    policy_generation NUMERIC(20,0) CHECK (policy_generation >= 0 AND policy_generation <= 18446744073709551615) NOT NULL,
    document_sha256 TEXT CHECK (document_sha256 ~ '^[0-9a-f]{64}$') NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'withdrawn')),
    resource_version NUMERIC(20,0) CHECK (resource_version >= 1 AND resource_version <= 18446744073709551615) NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
COMMENT ON TABLE dns_publication_heads IS 'Head pointer from a dns_profile definition to the resolver document desired state references via policy_refs (§12.5).';

CREATE TRIGGER dns_publication_heads_version
    BEFORE UPDATE ON dns_publication_heads
    FOR EACH ROW EXECUTE FUNCTION monotonic_fields('resource_version', 'policy_generation');

CREATE TRIGGER dns_publication_heads_touch
    BEFORE UPDATE ON dns_publication_heads
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER dns_publication_heads_keep
    BEFORE DELETE OR TRUNCATE ON dns_publication_heads
    FOR EACH STATEMENT EXECUTE FUNCTION forbid_delete();

-- Fan a dns_profile activation out to dns.publish without consuming policy.activated.
-- Desired-state compilation still observes policy.activated (§14.5).
-- +goose StatementBegin
CREATE FUNCTION enqueue_dns_publish() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.event_type = 'policy.activated' AND NEW.payload->>'type' = 'dns_profile' THEN
        INSERT INTO outbox_events (
            aggregate_id, aggregate_type, aggregate_version, event_type, schema_version, payload
        ) VALUES (
            NEW.aggregate_id, NEW.aggregate_type, NEW.aggregate_version, 'dns.publish', NEW.schema_version, NEW.payload
        );
    END IF;
    RETURN NEW;
END $$;
-- +goose StatementEnd

CREATE TRIGGER outbox_events_dns_publish
    AFTER INSERT ON outbox_events
    FOR EACH ROW EXECUTE FUNCTION enqueue_dns_publish();
