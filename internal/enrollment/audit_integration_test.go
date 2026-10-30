//go:build integration

package enrollment_test

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestD029_EnrollmentAuditAndOperationsRoundTrip(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	created, _, enrolled, cert := enroll(t, f)
	id, err := f.auth.Authenticate(ctx, cert)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.s.StoreBootstrap(ctx, id, enrolled.OperationID, "test-audit-key", "TEST ONLY audit bootstrap", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.TakeBootstrap(ctx, cert); err != nil {
		t.Fatal(err)
	}
	unused := create(t, f)
	if _, err = f.s.RevokeEnrollment(ctx, command(), unused.Enrollment.EnrollmentId, "TEST ONLY unused enrollment"); err != nil {
		t.Fatal(err)
	}
	version := decstr.U64(2)
	revocation, err := f.s.RevokeNode(ctx, command(), id.NodeID, &version, "TEST ONLY revoke")
	if err != nil {
		t.Fatal(err)
	}
	retirement, err := f.s.RetireNode(ctx, command(), id.NodeID, &revocation.ResourceVersion, "TEST ONLY retire")
	if err != nil {
		t.Fatal(err)
	}
	store := audit.Store{Pool: f.pool}
	page, err := store.List(ctx, api.ListAuditEventsParams{})
	if err != nil || len(page.Items) != 7 || store.AnomalyCount() != 0 || count(t, f, "audit_events") != len(page.Items) {
		t.Fatalf("audit round-trip: rows=%d anomalies=%d err=%v", len(page.Items), store.AnomalyCount(), err)
	}
	resources := map[string]api.ResourceRef{
		"enrollment.consumed": {Type: api.ResourceTypeEnrollment, Id: created.Enrollment.EnrollmentId},
		"enrollment.revoked":  {Type: api.ResourceTypeEnrollment, Id: unused.Enrollment.EnrollmentId},
		"bootstrap.claimed":   {Type: api.ResourceTypeNode, Id: id.NodeID},
		"node.revoked":        {Type: api.ResourceTypeNode, Id: id.NodeID},
		"node.retired":        {Type: api.ResourceTypeNode, Id: id.NodeID},
	}
	for _, event := range page.Items {
		if nullableValue(event.Actor.Id) == uuid.Nil || event.Result != api.AuditEventResultSucceeded || nullableValue(event.RequestId) == uuid.Nil {
			t.Fatal("audit identity/result missing")
		}
		if event.Action == "enrollment.created" {
			if event.Resource.Type != api.ResourceTypeEnrollment || (event.Resource.Id != created.Enrollment.EnrollmentId && event.Resource.Id != unused.Enrollment.EnrollmentId) {
				t.Fatal("wrong enrollment creation resource")
			}
			after := nullableValue(event.Changes.After)
			wantDigest := *created.Enrollment.TokenSha256
			if event.Resource.Id == unused.Enrollment.EnrollmentId {
				wantDigest = *unused.Enrollment.TokenSha256
			}
			if after["token_sha256"] != wantDigest || after["reason"] != "" {
				t.Fatalf("enrollment.created after change set mismatch: got %+v, want token_sha256=%s", after, wantDigest)
			}
		} else if event.Resource != resources[event.Action] {
			t.Fatal("wrong audit resource", event.Action)
		}
		if event.Action == "enrollment.consumed" {
			after := nullableValue(event.Changes.After)
			if _, hasDigest := after["token_sha256"]; hasDigest || after["reason"] != "" {
				t.Fatalf("enrollment.consumed after change set changed unexpectedly: %+v", after)
			}
		}
		if event.Action == "enrollment.consumed" || event.Action == "bootstrap.claimed" {
			if event.Actor.Type != api.AuditEventActorTypeNode || nullableValue(event.Actor.Id) != id.NodeID {
				t.Fatal("node actor lost stable UUID")
			}
		} else if event.Actor.Type != api.AuditEventActorTypeAdmin {
			t.Fatal("wrong administrator actor type")
		}
	}
	nodeType := api.ResourceTypeNode
	filtered, err := store.List(ctx, api.ListAuditEventsParams{ResourceType: &nodeType, ResourceId: &id.NodeID})
	if err != nil || len(filtered.Items) != 3 || store.AnomalyCount() != 0 {
		t.Fatal("node audit SQL filters lost bootstrap/revocation/retirement", err)
	}
	ops := operations.Store{Pool: f.pool}
	for operation, kind := range map[uuid.UUID]api.OperationKind{enrolled.OperationID: api.OperationKindNodeEnroll, revocation.OperationID: api.OperationKindNodeRevoke, retirement.OperationID: api.OperationKindNodeRetire} {
		op, err := ops.Get(ctx, operation, auth.Principal{Role: auth.Admin, ID: uuid.New()})
		if err != nil || op.Kind != kind || !op.Kind.Valid() || op.Resource.Type != api.ResourceTypeNode || op.Resource.Id != id.NodeID || op.Status != api.OperationStatusPending || op.RequestId == uuid.Nil {
			t.Fatal("operation does not round-trip through the contract reader", err)
		}
		for _, event := range page.Items {
			if event.Action != "bootstrap.claimed" && !event.OperationId.IsNull() && nullableValue(event.OperationId) == operation && nullableValue(event.RequestId) != op.RequestId {
				t.Fatal("operation and command audit lost request correlation")
			}
		}
	}
}

func TestD029_InvalidAdministratorCannotPersistRevocation(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	_, _, enrolled, cert := enroll(t, f)
	version := decstr.U64(2)
	for _, actor := range []string{"test-admin", "admin:" + uuid.Nil.String(), "member:" + uuid.NewString(), "node:" + uuid.NewString()} {
		cmd := command()
		cmd.Principal = actor
		_, err := f.s.RevokeNode(ctx, cmd, enrolled.Response.NodeId, &version, "TEST ONLY invalid principal")
		expectCode(t, err, apierr.ValidationFailed)
		if _, err = f.auth.Authenticate(ctx, cert); err != nil {
			t.Fatal("invalid principal persisted a revocation tombstone", err)
		}
	}
}

func TestENR02_EnrollmentTokenDigestObservableInReadAndAudit(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	hexDigestPattern := regexp.MustCompile(`^[0-9a-f]{64}$`)

	cmd := command()
	req := api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller}
	created, err := f.s.CreateEnrollment(ctx, cmd, req)
	if err != nil {
		t.Fatal(err)
	}
	token := nullableValue(created.Token)
	sum := sha256.Sum256([]byte(token))
	wantDigest := hex.EncodeToString(sum[:])
	if !hexDigestPattern.MatchString(wantDigest) {
		t.Fatalf("invalid expected sha256 hex digest: %q", wantDigest)
	}
	if created.Enrollment.TokenSha256 == nil || *created.Enrollment.TokenSha256 != wantDigest {
		t.Fatalf("create response token_sha256=%v, want %s", created.Enrollment.TokenSha256, wantDigest)
	}

	// Idempotent replay returns no plaintext token but preserves token_sha256.
	replay, err := f.s.CreateEnrollment(ctx, cmd, req)
	if err != nil {
		t.Fatal(err)
	}
	if replay.SecretAvailable || !replay.Token.IsNull() {
		t.Fatal("idempotent replay leaked plaintext token")
	}
	if replay.Enrollment.TokenSha256 == nil || *replay.Enrollment.TokenSha256 != wantDigest {
		t.Fatalf("replay response token_sha256=%v, want %s", replay.Enrollment.TokenSha256, wantDigest)
	}

	// Pending read exposes the same digest.
	pending, err := f.s.GetEnrollment(ctx, created.Enrollment.EnrollmentId)
	if err != nil {
		t.Fatal(err)
	}
	if pending.State != api.EnrollmentStatePending || pending.TokenSha256 == nil || *pending.TokenSha256 != wantDigest {
		t.Fatalf("pending get token_sha256=%v state=%s, want %s", pending.TokenSha256, pending.State, wantDigest)
	}

	// Consume the enrollment and verify the consumed read still carries the same token_sha256,
	// distinct from bound_public_key_sha256.
	enrollReq, _ := pkitest.Request(t)
	if _, err = f.s.Enroll(ctx, token, enrollReq); err != nil {
		t.Fatal(err)
	}
	consumed, err := f.s.GetEnrollment(ctx, created.Enrollment.EnrollmentId)
	if err != nil {
		t.Fatal(err)
	}
	if consumed.State != api.EnrollmentStateConsumed || consumed.TokenSha256 == nil || *consumed.TokenSha256 != wantDigest {
		t.Fatalf("consumed get token_sha256=%v state=%s, want %s", consumed.TokenSha256, consumed.State, wantDigest)
	}
	if nullableValue(consumed.BoundPublicKeySha256) == wantDigest {
		t.Fatal("token_sha256 must be the token digest, not bound_public_key_sha256")
	}

	// Revoked enrollment also preserves its token_sha256 on revoke and get.
	unused := create(t, f)
	unusedToken := nullableValue(unused.Token)
	unusedSum := sha256.Sum256([]byte(unusedToken))
	unusedDigest := hex.EncodeToString(unusedSum[:])
	revoked, err := f.s.RevokeEnrollment(ctx, command(), unused.Enrollment.EnrollmentId, "TEST ONLY revoke")
	if err != nil {
		t.Fatal(err)
	}
	if revoked.State != api.EnrollmentStateRevoked || revoked.TokenSha256 == nil || *revoked.TokenSha256 != unusedDigest {
		t.Fatalf("revoked enrollment token_sha256=%v state=%s, want %s", revoked.TokenSha256, revoked.State, unusedDigest)
	}

	// Verify audit event changes.after for enrollment.created carries token_sha256,
	// while enrollment.consumed remains unchanged (reason only).
	store := audit.Store{Pool: f.pool}
	enrollmentType := api.ResourceTypeEnrollment
	page, err := store.List(ctx, api.ListAuditEventsParams{
		ResourceType: &enrollmentType,
		ResourceId:   &created.Enrollment.EnrollmentId,
	})
	if err != nil || len(page.Items) != 2 || store.AnomalyCount() != 0 {
		t.Fatalf("enrollment audit list: items=%d anomalies=%d err=%v", len(page.Items), store.AnomalyCount(), err)
	}
	seenActions := map[string]bool{}
	for _, ev := range page.Items {
		seenActions[ev.Action] = true
		after := nullableValue(ev.Changes.After)
		switch ev.Action {
		case "enrollment.created":
			if after["token_sha256"] != wantDigest || after["reason"] != "" || len(after) != 2 {
				t.Fatalf("enrollment.created changes.after=%+v, want reason=\"\" and token_sha256=%s", after, wantDigest)
			}
		case "enrollment.consumed":
			if _, exists := after["token_sha256"]; exists || after["reason"] != "" || len(after) != 1 {
				t.Fatalf("enrollment.consumed changes.after=%+v, want only reason=\"\"", after)
			}
		default:
			t.Fatalf("unexpected enrollment audit action %q", ev.Action)
		}
	}
	if !seenActions["enrollment.created"] || !seenActions["enrollment.consumed"] {
		t.Fatalf("missing expected enrollment audit actions: %+v", seenActions)
	}

	// Verify no database column (including audit_events) or log contains plaintext tokens.
	scanSecrets(t, f, []string{token, unusedToken})
}
