-- +goose Up

-- §13.3, §15.4: release content is immutable, including signed compatibility.
CREATE TABLE releases (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), version TEXT NOT NULL UNIQUE,
 release_sequence NUMERIC(20,0) CHECK (release_sequence >= 0 AND release_sequence <= 18446744073709551615) NOT NULL,
 manifest_schema_version INTEGER NOT NULL CHECK(manifest_schema_version > 0),
 manifest JSONB NOT NULL CHECK(jsonb_typeof(manifest) = 'object'),
 manifest_hash BYTEA NOT NULL CHECK(octet_length(manifest_hash)=32),
 signature BYTEA NOT NULL CHECK(octet_length(signature)=64), signing_key_ref TEXT NOT NULL,
 component_versions JSONB NOT NULL CHECK(jsonb_typeof(component_versions) = 'object'),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE artifacts (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), release_id UUID NOT NULL REFERENCES releases,
 platform TEXT NOT NULL, arch TEXT NOT NULL CHECK(arch IN ('amd64','arm64')),
 digest BYTEA NOT NULL CHECK(octet_length(digest)=32), size_bytes NUMERIC(20,0) CHECK (size_bytes >= 0 AND size_bytes <= 18446744073709551615) NOT NULL,
 installed_size_bytes NUMERIC(20,0) CHECK (installed_size_bytes >= 0 AND installed_size_bytes <= 18446744073709551615) NOT NULL, storage_ref TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CONSTRAINT artifacts_platform_key UNIQUE(release_id,platform,arch)
);

-- §5, §15.3: lifecycle and desired/applied fields are independent.
CREATE TABLE nodes (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 lifecycle_state TEXT NOT NULL DEFAULT 'enrolling' CHECK(lifecycle_state IN ('enrolling','registered','revoked','retired')),
 current_incarnation_id UUID, notes TEXT NOT NULL DEFAULT '',
 provisioning_state TEXT NOT NULL DEFAULT 'pending' CHECK(provisioning_state IN ('pending','applying','ready','failed')),
 desired_config_generation NUMERIC(20,0) CHECK (desired_config_generation >= 0 AND desired_config_generation <= 18446744073709551615) NOT NULL DEFAULT 0, applied_config_generation NUMERIC(20,0) CHECK (applied_config_generation >= 0 AND applied_config_generation <= 18446744073709551615) NOT NULL DEFAULT 0,
 update_generation NUMERIC(20,0) CHECK (update_generation >= 0 AND update_generation <= 18446744073709551615) NOT NULL DEFAULT 0,
 desired_release_id UUID REFERENCES releases, current_release_id UUID REFERENCES releases, last_good_release_id UUID REFERENCES releases,
 resource_version NUMERIC(20,0) CHECK (resource_version >= 0 AND resource_version <= 18446744073709551615) NOT NULL DEFAULT 1,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), retired_at TIMESTAMPTZ
);
CREATE TABLE node_incarnations (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), node_id UUID NOT NULL REFERENCES nodes,
 public_key_fingerprint BYTEA NOT NULL UNIQUE CHECK(octet_length(public_key_fingerprint)=32),
 deployment_id TEXT, peer_id TEXT, os TEXT NOT NULL, arch TEXT NOT NULL CHECK(arch IN ('amd64','arm64')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), revoked_at TIMESTAMPTZ,
 CONSTRAINT node_incarnations_peer_key UNIQUE(deployment_id,peer_id),
 CONSTRAINT node_incarnations_node_key UNIQUE(node_id,id), CHECK((deployment_id IS NULL) = (peer_id IS NULL))
);
CREATE UNIQUE INDEX node_incarnations_active_key ON node_incarnations(node_id) WHERE revoked_at IS NULL;
ALTER TABLE nodes ADD CONSTRAINT nodes_current_incarnation_fk FOREIGN KEY(id,current_incarnation_id) REFERENCES node_incarnations(node_id,id);

-- §15.3, §15.5.1: token hash only; row-lock first consumption.
CREATE TABLE node_enrollments (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), node_id UUID NOT NULL REFERENCES nodes,
 token_hash BYTEA NOT NULL UNIQUE CHECK(octet_length(token_hash)=32),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), expiry TIMESTAMPTZ NOT NULL,
 state TEXT NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','consumed','revoked','expired')),
 request_id UUID, public_key_fingerprint BYTEA CHECK(octet_length(public_key_fingerprint)=32),
 consumed_at TIMESTAMPTZ, revoked_at TIMESTAMPTZ,
 CHECK(isfinite(expiry) AND created_at < expiry),
 CHECK(state <> 'consumed' OR (public_key_fingerprint IS NOT NULL AND consumed_at IS NOT NULL))
);
CREATE TABLE node_certificates (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), incarnation_id UUID NOT NULL REFERENCES node_incarnations,
 issuer TEXT NOT NULL, serial TEXT NOT NULL, fingerprint BYTEA NOT NULL CHECK(octet_length(fingerprint)=32),
 public_certificate BYTEA NOT NULL, not_before TIMESTAMPTZ NOT NULL, not_after TIMESTAMPTZ NOT NULL, revoked_at TIMESTAMPTZ,
 CONSTRAINT node_certificates_serial_key UNIQUE(issuer,serial),
 CHECK(isfinite(not_before) AND isfinite(not_after) AND not_before < not_after)
);

-- §5.2, §15.3: only authoritative copy of publication_intent.
CREATE TABLE exits (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), node_id UUID NOT NULL UNIQUE REFERENCES nodes,
 display_name TEXT NOT NULL, region TEXT NOT NULL,
 publication_intent TEXT NOT NULL DEFAULT 'unpublished' CHECK(publication_intent IN ('unpublished','published','disabled')),
 publication_observed TEXT NOT NULL DEFAULT 'absent' CHECK(publication_observed IN ('absent','applying','active','withdrawn','error')),
 resource_version NUMERIC(20,0) CHECK (resource_version >= 0 AND resource_version <= 18446744073709551615) NOT NULL DEFAULT 1, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE exit_groups (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), exit_id UUID NOT NULL REFERENCES exits, group_id UUID NOT NULL REFERENCES member_groups,
 CONSTRAINT exit_groups_membership_key UNIQUE(exit_id,group_id)
);

-- §15.3: typed policy validation is the compiler/caller's responsibility.
CREATE TABLE policy_definitions (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), type TEXT NOT NULL, code TEXT NOT NULL,
 active_revision_id UUID, resource_version NUMERIC(20,0) CHECK (resource_version >= 0 AND resource_version <= 18446744073709551615) NOT NULL DEFAULT 1,
 CONSTRAINT policy_definitions_code_key UNIQUE(type,code), CONSTRAINT policy_definitions_type_key UNIQUE(id,type)
);
CREATE TABLE policy_revisions (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), definition_id UUID NOT NULL, type TEXT NOT NULL,
 revision NUMERIC(20,0) CHECK (revision >= 0 AND revision <= 18446744073709551615) NOT NULL, schema_version INTEGER NOT NULL CHECK(schema_version > 0),
 payload JSONB NOT NULL CHECK(jsonb_typeof(payload) = 'object'), payload_hash BYTEA NOT NULL CHECK(octet_length(payload_hash)=32),
 signing_key_ref TEXT NOT NULL, created_by TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(definition_id,type) REFERENCES policy_definitions(id,type),
 CONSTRAINT policy_revisions_revision_key UNIQUE(definition_id,revision), CONSTRAINT policy_revisions_definition_key UNIQUE(id,definition_id)
);
ALTER TABLE policy_definitions ADD CONSTRAINT policy_definitions_active_revision_fk FOREIGN KEY(active_revision_id,id) REFERENCES policy_revisions(id,definition_id);
ALTER TABLE plan_revisions ADD CONSTRAINT plan_revisions_fup_fk FOREIGN KEY(fup_policy_revision_id) REFERENCES policy_revisions;

-- §15.3: generations are immutable history; node desired pointer is monotonic.
CREATE TABLE desired_states (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), node_id UUID NOT NULL REFERENCES nodes, incarnation_id UUID NOT NULL,
 generation NUMERIC(20,0) CHECK (generation >= 0 AND generation <= 18446744073709551615) NOT NULL, policy_refs UUID[] NOT NULL, binding_revision NUMERIC(20,0) CHECK (binding_revision >= 0 AND binding_revision <= 18446744073709551615) NOT NULL,
 release_id UUID REFERENCES releases, config_hash BYTEA NOT NULL CHECK(octet_length(config_hash)=32),
 schema_version INTEGER NOT NULL CHECK(schema_version > 0), signed_at TIMESTAMPTZ NOT NULL,
 FOREIGN KEY(node_id,incarnation_id) REFERENCES node_incarnations(node_id,id),
 CONSTRAINT desired_states_generation_key UNIQUE(node_id,incarnation_id,generation)
);
CREATE TABLE node_observations (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), node_id UUID NOT NULL REFERENCES nodes, incarnation_id UUID NOT NULL,
 report_id UUID NOT NULL, payload_hash BYTEA NOT NULL CHECK(octet_length(payload_hash)=32),
 received_at TIMESTAMPTZ NOT NULL DEFAULT now(), observed_at TIMESTAMPTZ NOT NULL,
 applied_generation NUMERIC(20,0) CHECK (applied_generation >= 0 AND applied_generation <= 18446744073709551615) NOT NULL, actual_release_id UUID REFERENCES releases,
 schema_version INTEGER NOT NULL CHECK(schema_version > 0), checks JSONB NOT NULL CHECK(jsonb_typeof(checks) = 'object'),
 control_status TEXT NOT NULL CHECK(control_status IN ('online','stale','offline')),
 data_health TEXT NOT NULL CHECK(data_health IN ('unknown','healthy','degraded','failed')),
 safety_status TEXT NOT NULL CHECK(safety_status IN ('unknown','passed','failed','expired')),
 FOREIGN KEY(node_id,incarnation_id) REFERENCES node_incarnations(node_id,id),
 CONSTRAINT node_observations_report_key UNIQUE(node_id,incarnation_id,report_id)
);
CREATE INDEX node_observations_received_idx ON node_observations(node_id,received_at);

-- §16.3, §15.3: signed lease deadlines never restart on replay.
CREATE TABLE serving_leases (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), node_id UUID NOT NULL REFERENCES nodes, incarnation_id UUID NOT NULL,
 policy_generation NUMERIC(20,0) CHECK (policy_generation >= 0 AND policy_generation <= 18446744073709551615) NOT NULL, member_grant_refs UUID[] NOT NULL DEFAULT '{}',
 issued_at TIMESTAMPTZ NOT NULL, expires_at TIMESTAMPTZ NOT NULL,
 key_id TEXT NOT NULL, payload_hash BYTEA NOT NULL CHECK(octet_length(payload_hash)=32),
 FOREIGN KEY(node_id,incarnation_id) REFERENCES node_incarnations(node_id,id),
 CHECK(isfinite(issued_at) AND isfinite(expires_at) AND issued_at < expires_at)
);
-- Normalized signed member deadlines, validated at issuance against entitlement and node lease.
CREATE TABLE serving_lease_members (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), lease_id UUID NOT NULL REFERENCES serving_leases,
 member_id UUID NOT NULL REFERENCES members, entitlement_revision NUMERIC(20,0) CHECK (entitlement_revision >= 0 AND entitlement_revision <= 18446744073709551615) NOT NULL,
 valid_from TIMESTAMPTZ NOT NULL, expires_at TIMESTAMPTZ NOT NULL,
 CONSTRAINT serving_lease_members_member_key UNIQUE(lease_id,member_id),
 CHECK(isfinite(valid_from) AND isfinite(expires_at) AND valid_from < expires_at)
);

-- §14.5, §15.3: bidirectional identity prevents retry-created external duplicates.
CREATE TABLE external_bindings (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), resource_type TEXT NOT NULL, resource_id UUID NOT NULL,
 provider TEXT NOT NULL, deployment_id TEXT NOT NULL, external_id TEXT NOT NULL,
 desired_hash BYTEA NOT NULL CHECK(octet_length(desired_hash)=32), last_observed_at TIMESTAMPTZ,
 CONSTRAINT external_bindings_internal_key UNIQUE(resource_type,resource_id,provider,deployment_id),
 CONSTRAINT external_bindings_external_key UNIQUE(provider,deployment_id,external_id)
);
