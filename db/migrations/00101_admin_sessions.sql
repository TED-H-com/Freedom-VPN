-- +goose Up
-- D-032 / §16.1: administrators are control identities, never VPN members.
-- Keep member-session guards intact; administrators have separate auth tables.
CREATE TABLE admin_identities (
 id UUID PRIMARY KEY,
 issuer TEXT NOT NULL CHECK(issuer ~ '[^[:space:]]'),
 client_scope TEXT NOT NULL CHECK(client_scope ~ '[^[:space:]]'),
 subject TEXT NOT NULL CHECK(subject ~ '[^[:space:]]'),
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(issuer,client_scope,subject)
);
CREATE TABLE admin_sessions (
 id UUID PRIMARY KEY,
 token_hash BYTEA NOT NULL UNIQUE CHECK(octet_length(token_hash)=32),
 admin_id UUID NOT NULL REFERENCES admin_identities,
 display_label TEXT NOT NULL,
 mfa_verified BOOLEAN NOT NULL CHECK(mfa_verified),
 authenticated_at TIMESTAMPTZ NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 last_seen_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 absolute_expires_at TIMESTAMPTZ NOT NULL,
 CHECK(isfinite(absolute_expires_at) AND created_at < absolute_expires_at)
);
CREATE INDEX admin_sessions_expiry_idx ON admin_sessions(absolute_expires_at);
COMMENT ON TABLE admin_identities IS 'Project Spec §16.1; scoped OIDC administrator identities isolated from VPN membership (D-032).';
COMMENT ON TABLE admin_sessions IS 'Project Spec §16.1; opaque administrator sessions with IdP MFA and idle/absolute expiry (D-032).';
