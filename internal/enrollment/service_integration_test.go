//go:build integration

package enrollment_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/pki"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fixture struct {
	pool  *pgxpool.Pool
	s     *enrollment.Service
	auth  *pki.Service
	ca    *pki.CA
	vault *signing.FileProvider
	logs  *bytes.Buffer
}

func setup(t *testing.T) fixture {
	t.Helper()
	pool := dbtest.New(t)
	vault, ca := pkitest.Authority(t)
	auth, err := pki.NewService(pool, ca, vault, pki.TestRenewalOverlap)
	if err != nil {
		t.Fatal(err)
	}
	logs := new(bytes.Buffer)
	s, err := enrollment.NewService(pool, ca, auth, vault, vault, enrollment.Config{BootstrapValidity: enrollment.TestBootstrapValidity, Logger: slog.New(slog.NewTextHandler(logs, nil))})
	if err != nil {
		t.Fatal(err)
	}
	return fixture{pool, s, auth, ca, vault, logs}
}
func command() enrollment.Command {
	return enrollment.Command{Principal: "test-admin", Key: uuid.New()}
}
func create(t *testing.T, f fixture) api.EnrollmentCreated {
	t.Helper()
	r, err := f.s.CreateEnrollment(context.Background(), command(), api.EnrollmentCreateRequest{Delivery: "installer"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Token == nil || !r.SecretAvailable {
		t.Fatal("first token missing")
	}
	return r
}
func enroll(t *testing.T, f fixture) (api.EnrollmentCreated, api.EnrollRequest, enrollment.EnrollResult, *x509.Certificate) {
	t.Helper()
	c := create(t, f)
	r, _ := pkitest.Request(t)
	out, err := f.s.Enroll(context.Background(), *c.Token, r)
	if err != nil {
		t.Fatal(err)
	}
	return c, r, out, pkitest.Certificate(t, out.Response.CertificatePem)
}
func expectCode(t *testing.T, err error, code apierr.Code) {
	t.Helper()
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
}
func count(t *testing.T, f fixture, table string) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), "SELECT count(*) FROM "+pgx.Identifier{table}.Sanitize()).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func exec(t *testing.T, f fixture, sql string, args ...any) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func TestAT06_IdempotentCreationLostResponseAndRace(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	cmd := command()
	req := api.EnrollmentCreateRequest{Delivery: "installer"}
	first, err := f.s.CreateEnrollment(ctx, cmd, req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.s.CreateEnrollment(ctx, cmd, req)
	if err != nil {
		t.Fatal(err)
	}
	if second.SecretAvailable || second.Token != nil || second.Enrollment.EnrollmentId != first.Enrollment.EnrollmentId {
		t.Fatal("creation replay leaked secret or changed resource")
	}
	changed := req
	changed.Delivery = "preinstalled"
	_, err = f.s.CreateEnrollment(ctx, cmd, changed)
	expectCode(t, err, apierr.IdempotencyConflict)
	r, key := pkitest.Request(t)
	original, err := f.s.Enroll(ctx, *first.Token, r)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := f.s.Enroll(ctx, *first.Token, r)
	if err != nil {
		t.Fatal(err)
	}
	if !retry.Replay || retry.Response != original.Response || retry.OperationID != original.OperationID {
		t.Fatal("lost-response replay changed identity, certificate or operation")
	}
	// CSR signatures are randomized. A fresh CSR from the same key, with a
	// fresh valid proof and the same bound IDs, still retrieves the first result.
	resigned := r
	resigned.CsrPem = pkitest.CSRWith(t, key, &x509.CertificateRequest{})
	pkitest.SignRequest(t, &resigned, key)
	resignedResult, err := f.s.Enroll(ctx, *first.Token, resigned)
	if err != nil || resignedResult.Response != original.Response {
		t.Fatal("same-key CSR retry rejected", err)
	}
	if count(t, f, "nodes") != 1 || count(t, f, "node_incarnations") != 1 || count(t, f, "node_certificates") != 1 || count(t, f, "operations") != 1 || count(t, f, "outbox_events") != 1 {
		t.Fatal("replay duplicated state")
	}
	other, _ := pkitest.Request(t)
	_, err = f.s.Enroll(ctx, *first.Token, other)
	expectCode(t, err, apierr.EnrollmentKeyMismatch)
	noProof := r
	noProof.Proof = ""
	_, err = f.s.Enroll(ctx, *first.Token, noProof)
	expectCode(t, err, apierr.ValidationFailed)
	changedID := r
	changedID.EnrollmentRequestId = uuid.New()
	pkitest.SignRequest(t, &changedID, key)
	_, err = f.s.Enroll(ctx, *first.Token, changedID)
	expectCode(t, err, apierr.EnrollmentKeyMismatch)
	_, err = f.s.RevokeEnrollment(ctx, command(), first.Enrollment.EnrollmentId, "test consumed")
	expectCode(t, err, apierr.StateConflict)
	// Two valid proof holders race for one row: exactly one wins.
	racing := create(t, f)
	a, _ := pkitest.Request(t)
	b, _ := pkitest.Request(t)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, request := range []api.EnrollRequest{a, b} {
		wg.Go(func() { _, e := f.s.Enroll(ctx, *racing.Token, request); results <- e })
	}
	wg.Wait()
	close(results)
	winners := 0
	for e := range results {
		if e == nil {
			winners++
		} else {
			expectCode(t, e, apierr.EnrollmentKeyMismatch)
		}
	}
	if winners != 1 {
		t.Fatal("race did not have exactly one winner")
	}
	if count(t, f, "node_incarnations") != 2 || count(t, f, "operations") != 2 {
		t.Fatal("race duplicated identities or work")
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	scanSecrets(t, f, []string{*first.Token, *racing.Token, base64.StdEncoding.EncodeToString(privateDER)})
	assertNoPrivateDER(t, f, privateDER)
	// Stored public recovery remains available after the issuing key retires.
	if _, err = f.vault.Rotate(ctx, signing.NodeCA, 24*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	retry, err = f.s.Enroll(ctx, *first.Token, r)
	if err != nil || retry.Response != original.Response {
		t.Fatal("replay unnecessarily requires CA signing", err)
	}
}

func TestAT06_ExpiryRevocationAndTransactionalRollback(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	r, _ := pkitest.Request(t)
	// Expiry is immutable. Seed a genuinely expired row without disabling any
	// production guard or waiting on wall time.
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	expiredToken := enrollment.TokenPrefix + base64.RawURLEncoding.EncodeToString(random)
	hash := sha256.Sum256([]byte(expiredToken))
	nodeID, enrollmentID := uuid.New(), uuid.New()
	exec(t, f, `INSERT INTO nodes(id) VALUES($1)`, nodeID)
	exec(t, f, `INSERT INTO node_enrollments(id,node_id,token_hash,created_at,expiry) VALUES($1,$2,$3,clock_timestamp()-interval '2 minutes',clock_timestamp()-interval '1 minute')`, enrollmentID, nodeID, hash[:])
	state, err := f.s.GetEnrollment(ctx, enrollmentID)
	if err != nil || state.State != "expired" {
		t.Fatal("expired read model incorrect", err)
	}
	_, err = f.s.Enroll(ctx, expiredToken, r)
	expectCode(t, err, apierr.EnrollmentTokenExpired)
	revoked := create(t, f)
	cmd := command()
	if _, err = f.s.RevokeEnrollment(ctx, cmd, revoked.Enrollment.EnrollmentId, "test revocation"); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.RevokeEnrollment(ctx, cmd, revoked.Enrollment.EnrollmentId, "test revocation"); err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Enroll(ctx, *revoked.Token, r)
	expectCode(t, err, apierr.EnrollmentTokenRevoked)
	// Inject failure after the identity and certificate inserts. The entire
	// consumption transaction must roll back, then retry successfully.
	rollback := create(t, f)
	exec(t, f, `CREATE FUNCTION test_reject_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'TEST ONLY injected outbox failure'; END $$`)
	exec(t, f, `CREATE TRIGGER test_reject_outbox BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION test_reject_outbox()`)
	_, err = f.s.Enroll(ctx, *rollback.Token, r)
	expectCode(t, err, apierr.DependencyUnavailable)
	got, err := f.s.GetEnrollment(ctx, rollback.Enrollment.EnrollmentId)
	if err != nil || got.State != "pending" {
		t.Fatal("failed transaction consumed token")
	}
	if count(t, f, "node_incarnations") != 0 || count(t, f, "node_certificates") != 0 {
		t.Fatal("partial enrollment persisted")
	}
	exec(t, f, `DROP TRIGGER test_reject_outbox ON outbox_events`)
	if _, err = f.s.Enroll(ctx, *rollback.Token, r); err != nil {
		t.Fatal(err)
	}
}

func TestAT28_AuthenticationRenewalAndPermanentRevocation(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	c, r, out, cert := enroll(t, f)
	id, err := f.auth.Authenticate(ctx, cert)
	if err != nil {
		t.Fatal(err)
	}
	if id.NodeID != c.Enrollment.NodeId || id.IncarnationID != r.NodeIncarnationId {
		t.Fatal("wrong identity")
	}
	exec(t, f, `UPDATE nodes SET current_incarnation_id=NULL WHERE id=$1`, id.NodeID)
	_, err = f.auth.Authenticate(ctx, cert)
	expectCode(t, err, apierr.IdentityRevoked)
	exec(t, f, `UPDATE nodes SET current_incarnation_id=$2 WHERE id=$1`, id.NodeID, id.IncarnationID)
	exec(t, f, `UPDATE exits SET publication_intent='disabled' WHERE node_id=$1`, id.NodeID)
	if _, err = f.auth.Authenticate(ctx, cert); err != nil {
		t.Fatal("disabled node lost control access", err)
	}
	_, newCSR := pkitest.CSR(t)
	renewal := api.CertificateRenewRequest{NodeId: id.NodeID, NodeIncarnationId: id.IncarnationID, RenewalRequestId: uuid.New(), CsrPem: newCSR}
	newCert, err := f.auth.Renew(ctx, cert, renewal)
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.auth.Renew(ctx, cert, renewal)
	if err != nil || again != newCert {
		t.Fatal("renewal retry changed result", err)
	}
	if count(t, f, "node_certificates") != 2 {
		t.Fatal("renewal duplicated serial history")
	}
	if _, err = f.auth.Authenticate(ctx, cert); err != nil {
		t.Fatal("old cert overlap missing", err)
	}
	current := pkitest.Certificate(t, newCert.CertificatePem)
	if _, err = f.auth.Authenticate(ctx, current); err != nil {
		t.Fatal(err)
	}
	badRenew := renewal
	badRenew.NodeId = uuid.New()
	_, err = f.auth.Renew(ctx, cert, badRenew)
	expectCode(t, err, apierr.IncarnationMismatch)
	_, differentCSR := pkitest.CSR(t)
	badRenew = renewal
	badRenew.CsrPem = differentCSR
	_, err = f.auth.Renew(ctx, cert, badRenew)
	expectCode(t, err, apierr.IdempotencyConflict)
	// A valid CA signature alone is not a database authorization.
	wrongInc, err := f.ca.Issue(ctx, pki.NodeIdentity{NodeID: id.NodeID, IncarnationID: uuid.New()}, r.CsrPem)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.auth.Authenticate(ctx, wrongInc.Certificate)
	expectCode(t, err, apierr.Unauthenticated)
	_, otherCA := pkitest.Authority(t)
	wrongCA, err := otherCA.Issue(ctx, id, r.CsrPem)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.auth.Authenticate(ctx, wrongCA.Certificate)
	expectCode(t, err, apierr.Unauthenticated)
	// Controlled overlap has a hard cutoff, independent of X.509 notAfter.
	exec(t, f, `UPDATE node_certificates SET usable_until=clock_timestamp()-interval '1 second' WHERE serial=$1`, out.Response.CertificateSerial)
	_, err = f.auth.Authenticate(ctx, cert)
	expectCode(t, err, apierr.IdentityRevoked)
	_, err = f.s.RevokeNode(ctx, command(), id.NodeID, nil, "test")
	expectCode(t, err, apierr.PreconditionRequired)
	wrongVersion := decstr.U64(1)
	_, err = f.s.RevokeNode(ctx, command(), id.NodeID, &wrongVersion, "test")
	expectCode(t, err, apierr.ResourceVersionConflict)
	if _, err = f.auth.Authenticate(ctx, current); err != nil {
		t.Fatal("failed precondition revoked identity", err)
	}
	version := decstr.U64(2)
	cmd := command()
	revoked, err := f.s.RevokeNode(ctx, cmd, id.NodeID, &version, "test")
	if err != nil {
		t.Fatal(err)
	}
	replay, err := f.s.RevokeNode(ctx, cmd, id.NodeID, &version, "test")
	if err != nil || replay != revoked {
		t.Fatal("revocation replay differs", err)
	}
	_, err = f.auth.Authenticate(ctx, current)
	expectCode(t, err, apierr.IdentityRevoked)
	_, err = f.auth.Renew(ctx, current, renewal)
	expectCode(t, err, apierr.IdentityRevoked)
	_, err = f.s.Enroll(ctx, *c.Token, r)
	expectCode(t, err, apierr.IdentityRevoked)
	if count(t, f, "outbox_events") != 3 || count(t, f, "operations") != 2 {
		t.Fatal("revocation work missing or duplicated")
	}
	if _, err = f.pool.Exec(ctx, `UPDATE nodes SET lifecycle_state='registered' WHERE id=$1`, id.NodeID); err == nil {
		t.Fatal("terminal node revived")
	}
	if _, err = f.pool.Exec(ctx, `UPDATE node_certificates SET revoked_at=NULL WHERE incarnation_id=$1`, id.IncarnationID); err == nil {
		t.Fatal("certificate revived")
	}
}

func assertNoPrivateDER(t *testing.T, f fixture, der []byte) {
	t.Helper()
	ctx := context.Background()
	rows, err := f.pool.Query(ctx, `SELECT table_name,column_name FROM information_schema.columns WHERE table_schema='public' AND data_type='bytea'`)
	if err != nil {
		t.Fatal(err)
	}
	var columns [][2]string
	for rows.Next() {
		var c [2]string
		if err = rows.Scan(&c[0], &c[1]); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, c)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, c := range columns {
		var found bool
		query := "SELECT EXISTS(SELECT 1 FROM " + pgx.Identifier{c[0]}.Sanitize() + " WHERE position($1::bytea in " + pgx.Identifier{c[1]}.Sanitize() + ")>0)"
		if err = f.pool.QueryRow(ctx, query, der).Scan(&found); err != nil {
			t.Fatal(err)
		}
		if found {
			t.Fatal("private DER persisted", c)
		}
	}
}

func TestAT28_BootstrapOnceOverAuthenticatedPathAndNoSecrets(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	c, _, out, cert := enroll(t, f)
	id, err := f.auth.Authenticate(ctx, cert)
	if err != nil {
		t.Fatal(err)
	}
	secret := "TEST ONLY bootstrap-" + uuid.NewString()
	expiry := time.Now().Add(5 * time.Minute)
	if err = f.s.StoreBootstrap(ctx, id, out.OperationID, "external-test-key", secret, expiry); err != nil {
		t.Fatal(err)
	}
	if err = f.s.StoreBootstrap(ctx, id, out.OperationID, "external-test-key", secret, expiry); err != nil {
		t.Fatal(err)
	}
	_, err = f.s.TakeBootstrap(ctx, nil)
	expectCode(t, err, apierr.Unauthenticated)
	_, _, _, otherCert := enroll(t, f)
	_, err = f.s.TakeBootstrap(ctx, otherCert)
	expectCode(t, err, apierr.StateConflict)
	scanSecrets(t, f, []string{*c.Token, secret})
	var wg sync.WaitGroup
	results := make(chan error, 2)
	deliveries := make(chan enrollment.BootstrapDelivery, 2)
	for range 2 {
		wg.Go(func() { delivery, e := f.s.TakeBootstrap(ctx, cert); results <- e; deliveries <- delivery })
	}
	wg.Wait()
	close(results)
	close(deliveries)
	winners := 0
	for e := range results {
		if e == nil {
			winners++
		} else {
			expectCode(t, e, apierr.StateConflict)
		}
	}
	if winners != 1 {
		t.Fatal("one-time delivery had wrong number of winners")
	}
	for delivery := range deliveries {
		if delivery.SetupKey != "" && delivery.SetupKey != secret {
			t.Fatal("wrong secret")
		}
	}
	var ciphertext []byte
	var keyID *string
	if err = f.pool.QueryRow(ctx, `SELECT ciphertext,encryption_key_id FROM node_bootstrap_secrets WHERE incarnation_id=$1`, id.IncarnationID).Scan(&ciphertext, &keyID); err != nil || ciphertext != nil || keyID != nil {
		t.Fatal("delivered secret retained", err)
	}
	if err = f.s.StoreBootstrap(ctx, id, out.OperationID, "external-test-key", secret, expiry); err != nil {
		t.Fatal(err)
	}
	_, err = f.s.TakeBootstrap(ctx, cert)
	expectCode(t, err, apierr.StateConflict)
	err = f.s.StoreBootstrap(ctx, id, out.OperationID, "different-test-key", secret, expiry)
	expectCode(t, err, apierr.StateConflict)
	scanSecrets(t, f, []string{*c.Token, secret})
}

func TestENR02_BootstrapExpiryAndConflictFreeze(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	_, _, out, cert := enroll(t, f)
	id, err := f.auth.Authenticate(ctx, cert)
	if err != nil {
		t.Fatal(err)
	}
	err = f.s.StoreBootstrap(ctx, id, out.OperationID, "external-expiry-test", "TEST ONLY setup value", time.Now().Add(-time.Minute))
	expectCode(t, err, apierr.StateConflict)
	sealed, err := f.vault.Seal(ctx, []byte("TEST ONLY setup value"), []byte("expired test fixture"))
	if err != nil {
		t.Fatal(err)
	}
	exec(t, f, `INSERT INTO node_bootstrap_secrets(incarnation_id,operation_id,external_key_id,encryption_key_id,ciphertext,expires_at) VALUES($1,$2,'external-expiry-test',$3,$4,clock_timestamp()-interval '1 second')`, id.IncarnationID, out.OperationID, sealed.KeyID, sealed.Ciphertext)
	_, err = f.s.TakeBootstrap(ctx, cert)
	expectCode(t, err, apierr.StateConflict)
	var erased bool
	if err = f.pool.QueryRow(ctx, `SELECT ciphertext IS NULL FROM node_bootstrap_secrets WHERE incarnation_id=$1`, id.IncarnationID).Scan(&erased); err != nil || !erased {
		t.Fatal("expired ciphertext retained", err)
	}
	allowed, err := f.s.ObserveEndpoint(ctx, cert, "endpoint-domain-a", time.Minute)
	if err != nil || !allowed {
		t.Fatal(err)
	}
	allowed, err = f.s.ObserveEndpoint(ctx, cert, "endpoint-domain-b", time.Minute)
	if err != nil || allowed {
		t.Fatal("conflict did not freeze", err)
	}
	allowed, err = f.s.ObserveEndpoint(ctx, cert, "endpoint-domain-a", time.Minute)
	if err != nil || allowed {
		t.Fatal("mapping unfroze", err)
	}
	if count(t, f, "node_identity_conflicts") != 1 {
		t.Fatal("conflict evidence missing")
	}
	if _, err = f.pool.Exec(ctx, `UPDATE node_identity_endpoints SET mapping_frozen=false WHERE incarnation_id=$1`, id.IncarnationID); err == nil {
		t.Fatal("mapping freeze undone")
	}
}

func TestAT05_UniformDeliveryMetadataAndPlatforms(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	group := uuid.New()
	exec(t, f, `INSERT INTO member_groups(id,code,name) VALUES($1,'test-group','Test group')`, group)
	name, region, notes := "測試節點", "test-region", "TEST ONLY source note"
	groups := []uuid.UUID{group}
	for _, delivery := range []api.EnrollmentDelivery{"installer", "preinstalled"} {
		for _, arch := range []api.Architecture{"amd64", "arm64"} {
			created, err := f.s.CreateEnrollment(ctx, command(), api.EnrollmentCreateRequest{Delivery: delivery, Name: &name, Region: &region, Notes: &notes, MemberGroupIds: &groups})
			if err != nil {
				t.Fatal(err)
			}
			want := enrollment.DefaultInstallerValidity
			if delivery == "preinstalled" {
				want = enrollment.DefaultPreinstalledValidity
			}
			delta := created.Enrollment.ExpiresAt.Sub(created.Enrollment.CreatedAt) - want
			if delta < 0 || delta > time.Second {
				t.Fatal("default lifetime differs", delta)
			}
			if created.Enrollment.Name == nil || *created.Enrollment.Name != name || created.Enrollment.MemberGroupIds[0] != group {
				t.Fatal("metadata lost")
			}
			r, key := pkitest.Request(t)
			r.Platform.Arch = arch
			pkitest.SignRequest(t, &r, key)
			out, err := f.s.Enroll(ctx, *created.Token, r)
			if err != nil {
				t.Fatal(err)
			}
			if out.Response.PublicationIntent != "unpublished" {
				t.Fatal("enrollment published node")
			}
			var provisioning string
			if err = f.pool.QueryRow(ctx, `SELECT provisioning_state FROM nodes WHERE id=$1`, out.Response.NodeId).Scan(&provisioning); err != nil || provisioning != "applying" {
				t.Fatal("provisioning falsely complete", err)
			}
		}
	}
	if count(t, f, "exit_groups") != 4 {
		t.Fatal("group assignments missing")
	}
}

func TestAT07_RevocationJournalFencesFailedCommitAndNewEnrollment(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	old, _, out, cert := enroll(t, f)
	// Failure after journal persistence leaves the DB looking like its old
	// snapshot. The independent journal must still reject this identity.
	exec(t, f, `CREATE FUNCTION test_reject_revocation_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'TEST ONLY injected failure'; END $$`)
	exec(t, f, `CREATE TRIGGER test_reject_revocation_outbox BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION test_reject_revocation_outbox()`)
	version := decstr.U64(2)
	cmd := command()
	_, err := f.s.RevokeNode(ctx, cmd, out.Response.NodeId, &version, "TEST ONLY revoke failure")
	expectCode(t, err, apierr.DependencyUnavailable)
	var state string
	if err = f.pool.QueryRow(ctx, `SELECT lifecycle_state FROM nodes WHERE id=$1`, out.Response.NodeId).Scan(&state); err != nil || state != "registered" {
		t.Fatal("failure injection did not roll back DB", err)
	}
	_, err = f.auth.Authenticate(ctx, cert)
	expectCode(t, err, apierr.IdentityRevoked)
	exec(t, f, `DROP TRIGGER test_reject_revocation_outbox ON outbox_events`)
	if _, err = f.s.RevokeNode(ctx, cmd, out.Response.NodeId, &version, "TEST ONLY revoke failure"); err != nil {
		t.Fatal(err)
	}
	_, _, fresh, freshCert := enroll(t, f)
	if fresh.Response.NodeId == old.Enrollment.NodeId || fresh.Response.NodeIncarnationId == out.Response.NodeIncarnationId {
		t.Fatal("new enrollment revived old identity")
	}
	if _, err = f.auth.Authenticate(ctx, freshCert); err != nil {
		t.Fatal(err)
	}
	_, err = f.auth.Authenticate(ctx, cert)
	expectCode(t, err, apierr.IdentityRevoked)
}

// Inspect every text and bytea column (plus JSON, arrays and other text casts),
// including outbox/idempotency/audit. Error output never prints searched secrets.
func scanSecrets(t *testing.T, f fixture, secrets []string) {
	t.Helper()
	ctx := context.Background()
	rows, err := f.pool.Query(ctx, `SELECT table_name,column_name,data_type FROM information_schema.columns WHERE table_schema='public'`)
	if err != nil {
		t.Fatal(err)
	}
	type column struct{ table, name, kind string }
	var columns []column
	for rows.Next() {
		var c column
		if err = rows.Scan(&c.table, &c.name, &c.kind); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, c)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, c := range columns {
		table := pgx.Identifier{c.table}.Sanitize()
		field := pgx.Identifier{c.name}.Sanitize()
		for _, secret := range append(append([]string{}, secrets...), "PRIVATE KEY") {
			var found bool
			query := "SELECT EXISTS(SELECT 1 FROM " + table + " WHERE strpos(" + field + "::text,$1)>0)"
			needle := secret
			if c.kind == "bytea" {
				query = "SELECT EXISTS(SELECT 1 FROM " + table + " WHERE strpos(encode(" + field + ",'hex'),$1)>0)"
				needle = hex.EncodeToString([]byte(secret))
			}
			if err = f.pool.QueryRow(ctx, query, needle).Scan(&found); err != nil {
				t.Fatal(err)
			}
			if found {
				t.Fatalf("secret in table %s column %s", c.table, c.name)
			}
		}
	}
	for _, secret := range secrets {
		if strings.Contains(f.logs.String(), secret) {
			t.Fatal("secret in captured service log")
		}
	}
}
