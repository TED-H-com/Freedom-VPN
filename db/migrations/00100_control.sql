-- +goose Up
-- §16.1: opaque browser credentials are stored only as SHA-256 hashes.
CREATE TABLE browser_sessions (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 token_hash BYTEA NOT NULL UNIQUE CHECK(octet_length(token_hash)=32),
 member_id UUID NOT NULL REFERENCES members,
 role TEXT NOT NULL CHECK(role IN ('admin','member')),
 display_label TEXT NOT NULL, mfa_verified BOOLEAN NOT NULL,
 authenticated_at TIMESTAMPTZ NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 last_seen_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 absolute_expires_at TIMESTAMPTZ NOT NULL,
 CHECK(isfinite(absolute_expires_at) AND created_at < absolute_expires_at)
);
CREATE INDEX browser_sessions_expiry_idx ON browser_sessions(absolute_expires_at);
CREATE TABLE oidc_login_flows (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 state_hash BYTEA NOT NULL UNIQUE CHECK(octet_length(state_hash)=32),
 browser_hash BYTEA NOT NULL CHECK(octet_length(browser_hash)=32),
 nonce TEXT NOT NULL, verifier TEXT NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL CHECK(isfinite(expires_at))
);
CREATE INDEX oidc_login_flows_expiry_idx ON oidc_login_flows(expires_at);
-- Additive metadata; older domain operations remain readable.
ALTER TABLE operations ADD COLUMN request_id UUID;
ALTER TABLE operations ADD COLUMN resource_version NUMERIC(20,0) CHECK(resource_version BETWEEN 0 AND 18446744073709551615);
ALTER TABLE operations ADD COLUMN observed_generation NUMERIC(20,0) CHECK(observed_generation BETWEEN 0 AND 18446744073709551615);
ALTER TABLE operations ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE INDEX operations_principal_id_idx ON operations(principal,id);
CREATE INDEX audit_events_page_idx ON audit_events(occurred_at DESC,id DESC);
CREATE INDEX audit_events_request_idx ON audit_events(request_id,occurred_at DESC,id DESC);
-- ClaimOutbox excludes infinity without confusing a dead letter with delivery.
ALTER TABLE outbox_events ADD COLUMN dead_lettered_at TIMESTAMPTZ;
ALTER TABLE outbox_events ADD COLUMN last_error_code TEXT;
ALTER TABLE outbox_events ADD COLUMN delivery_receipt TEXT;
CREATE INDEX outbox_events_dead_letter_idx ON outbox_events(dead_lettered_at) WHERE dead_lettered_at IS NOT NULL;

COMMENT ON TABLE browser_sessions IS 'Project Spec §16.1; opaque server-side sessions with idle and absolute expiry.';
COMMENT ON TABLE oidc_login_flows IS 'Project Spec §16.1; one-use browser-bound OIDC state, nonce and PKCE verifier.';
