//go:build integration

package releases_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type memoryFetcher struct{ data []byte }

func (f memoryFetcher) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(f.data)), nil
}
func TestUPD02_RegisterImmutableAndVerify(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	f := releasetest.New(t)
	s := &releases.Service{Pool: pool, Validator: f.Validator, VerifyArtifacts: true}
	data := []byte("TEST ONLY artifact bytes")
	sum := sha256.Sum256(data)
	for n := range f.Manifest.Artifacts {
		f.Manifest.Artifacts[n].SizeBytes = decstr.U64(len(data))
		f.Manifest.Artifacts[n].Sha256 = hex.EncodeToString(sum[:])
	}
	cmd := releasetest.Command()
	req := f.Request(t, f.Manifest)
	got, e := s.Register(ctx, cmd, req)
	if e != nil {
		t.Fatal(e)
	}
	if got.Status != 202 || got.OperationID == nil {
		t.Fatal(got)
	}
	replay, e := s.Register(ctx, cmd, req)
	if e != nil || *replay.OperationID != *got.OperationID {
		t.Fatal(replay, e)
	}
	f.Manifest.BuildId = "different"
	_, e = s.Register(ctx, releasetest.Command(), f.Request(t, f.Manifest))
	if !errors.Is(e, apierr.New(apierr.StateConflict, "")) {
		t.Fatal("same ID different content", e)
	}
	if e = s.VerifyRelease(ctx, got.ReleaseID, memoryFetcher{data}); e != nil {
		t.Fatal(e)
	}
	if e = s.VerifyRelease(ctx, got.ReleaseID, memoryFetcher{[]byte("bad")}); e != nil {
		t.Fatal(e)
	}
	verified, e := s.Register(ctx, releasetest.Command(), req)
	if e != nil || verified.Status != 201 {
		t.Fatal(verified, e)
	}
	list, e := s.List(ctx, "", 1)
	if e != nil || len(list.Items) != 1 || list.Items[0].VerificationState != "verified" {
		t.Fatal(list, e)
	}
	// D-013 / §15.3: one display version names exactly one immutable release.
	f.Manifest.ReleaseId = uuid.Must(uuid.NewV7())
	_, e = s.Register(ctx, releasetest.Command(), f.Request(t, f.Manifest))
	if !errors.Is(e, apierr.New(apierr.StateConflict, "")) {
		t.Fatal("same version different release", e)
	}
	if apierr.StateConflict.HTTPStatus() != 409 {
		t.Fatal("version conflict must be HTTP 409")
	}
	f.Manifest.ProductVersion = "1.2.0-test"
	got2, e := s.Register(ctx, releasetest.Command(), f.Request(t, f.Manifest))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.VerifyRelease(ctx, got2.ReleaseID, memoryFetcher{[]byte("bad")}); e != nil {
		t.Fatal(e)
	}
	var state string
	if e = pool.QueryRow(ctx, `SELECT state FROM release_verifications WHERE release_id=$1`, got2.ReleaseID).Scan(&state); e != nil || state != "failed" {
		t.Fatal(state, e)
	}
	e = db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error { _, _, _, e := s.Eligible(ctx, tx, got2.ReleaseID); return e })
	if !errors.Is(e, apierr.New(apierr.ReleaseNotApproved, "")) {
		t.Fatal(e)
	}
	releasetest.AssertLedger(t, pool, f.Validator)
}
func TestUPD02_RejectedRegistrationsPersistIneligible(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	f := releasetest.New(t)
	s := &releases.Service{Pool: pool, Validator: f.Validator}
	good := f.Request(t, f.Manifest)
	invalid := [][]byte{bytes.Replace(good, []byte("build-test-42"), []byte("build-test-99"), 1)}
	m := f.Manifest
	m.Artifacts = m.Artifacts[:1]
	invalid = append(invalid, f.Request(t, m))
	var copyManifest = f.Manifest
	raw, _ := json.Marshal(f.Manifest)
	_ = json.Unmarshal(raw, &copyManifest)
	copyManifest.Artifacts[0].Url = "https://evil.example.invalid/a"
	invalid = append(invalid, f.Request(t, copyManifest))
	for _, r := range invalid {
		if _, e := s.Register(ctx, releasetest.Command(), r); e == nil {
			t.Fatal("accepted bad release")
		}
	}
	var count int
	if e := pool.QueryRow(ctx, `SELECT count(*) FROM release_registration_rejections WHERE NOT eligible`).Scan(&count); e != nil || count != len(invalid) {
		t.Fatal(count, e)
	}
	if e := pool.QueryRow(ctx, `SELECT count(*) FROM releases`).Scan(&count); e != nil || count != 0 {
		t.Fatal(count, e)
	}
}

func TestUPD02_ListContractAndVerificationErrors(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	f := releasetest.New(t)
	s := &releases.Service{Pool: pool, Validator: f.Validator, VerifyArtifacts: true}
	got, e := s.Register(ctx, releasetest.Command(), f.Request(t, f.Manifest))
	if e != nil {
		t.Fatal(e)
	}
	list, e := s.List(ctx, "", 0)
	if e != nil || list.Items[0].VerificationState != "verifying" {
		t.Fatal(list, e)
	}
	raw, _ := json.Marshal(list)
	if e = f.Validator.ValidateBody("ReleaseList", raw); e != nil {
		t.Fatal(e, string(raw))
	}
	if e = s.VerifyRelease(ctx, got.ReleaseID, memoryFetcher{[]byte("bad")}); e != nil {
		t.Fatal(e)
	}
	list, e = s.List(ctx, "", 0)
	if e != nil || list.Items[0].VerificationState != "rejected" || (!list.Items[0].VerificationError.IsSpecified() || list.Items[0].VerificationError.IsNull()) {
		t.Fatal(list, e)
	}
}
