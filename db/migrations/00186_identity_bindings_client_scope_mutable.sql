-- +goose Up
-- D-052 / §15.2: issuer and public subject own the identity. Client scope is
-- diagnostic metadata and may follow an OIDC client rotation without rebinding.
DROP TRIGGER identity_bindings_identity ON identity_bindings;
CREATE TRIGGER identity_bindings_identity BEFORE UPDATE ON identity_bindings
FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','member_id','issuer','subject');

-- +goose Down
-- Restore the original guard without rewriting stored identity metadata.
DROP TRIGGER identity_bindings_identity ON identity_bindings;
CREATE TRIGGER identity_bindings_identity BEFORE UPDATE ON identity_bindings
FOR EACH ROW EXECUTE FUNCTION immutable_fields('id','member_id','issuer','client_scope','subject');
