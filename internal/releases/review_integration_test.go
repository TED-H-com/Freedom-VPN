//go:build integration

package releases_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

func TestUPD02_RegistrationValidationBeforeTypedDecode(t *testing.T) {
	ctx := context.Background()
	f := releasetest.New(t)
	good := f.Request(t, f.Manifest)
	// No pool: oversize/schema rejection must finish before any database work.
	s := &releases.Service{Validator: f.Validator}
	oversized := append(bytes.Repeat([]byte(" "), canonjson.MaxBytes), good...)
	if _, err := s.Register(ctx, releasetest.Command(), oversized); !errors.Is(err, apierr.New(apierr.ValidationFailed, "")) {
		t.Fatal(err)
	}
	s.Pool = dbtest.New(t)
	// This decimal passes OpenAPI's string syntax but overflows the Go type. An
	// invalid signature must be rejected before the typed request sees that value.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(good, &raw); err != nil {
		t.Fatal(err)
	}
	var env map[string]json.RawMessage
	if err := json.Unmarshal(raw["manifest"], &env); err != nil {
		t.Fatal(err)
	}
	var signed map[string]json.RawMessage
	if err := json.Unmarshal(env["signed"], &signed); err != nil {
		t.Fatal(err)
	}
	signed["release_sequence"] = json.RawMessage(`"18446744073709551616"`)
	env["signed"], _ = json.Marshal(signed)
	raw["manifest"], _ = json.Marshal(env)
	bad, _ := json.Marshal(raw)
	if _, err := s.Register(ctx, releasetest.Command(), bad); !errors.Is(err, apierr.New(apierr.SignatureInvalid, "")) {
		t.Fatal(err)
	}
	var count int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM releases`).Scan(&count); err != nil || count != 0 {
		t.Fatal(count, err)
	}
}

func TestUPD02_FailedRegistrationAndOperationEvidence(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)
	f := releasetest.New(t)
	s := &releases.Service{Pool: pool, Validator: f.Validator, VerifyArtifacts: true}
	cmd := releasetest.Command()
	body := f.Request(t, f.Manifest)
	pending, err := s.Register(ctx, cmd, body)
	if err != nil || pending.Status != 202 {
		t.Fatal(pending, err)
	}
	another, err := s.Register(ctx, releasetest.Command(), body)
	if err != nil || another.Status != 202 || *another.OperationID != *pending.OperationID {
		t.Fatal(another, err)
	}
	if err = s.VerifyRelease(ctx, pending.ReleaseID, memoryFetcher{[]byte("TEST ONLY bad artifact")}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Register(ctx, releasetest.Command(), body)
	var apiError *apierr.Error
	if !errors.As(err, &apiError) || apiError.HTTPStatus() != 409 || apiError.Code != apierr.StateConflict || apiError.Details["verification_state"] != "failed" {
		t.Fatal(got, err)
	}
	// §14.5: exact command replay still returns its original asynchronous ACK.
	replay, err := s.Register(ctx, cmd, body)
	if err != nil || replay.Status != pending.Status || *replay.OperationID != *pending.OperationID {
		t.Fatal(replay, err)
	}
	op, err := (operations.Store{Pool: pool}).Get(ctx, *pending.OperationID, auth.Principal{Role: auth.Admin, ID: uuid.New()})
	if err != nil || op.Status != "failed" || op.Kind != api.OperationKindReleaseRegister || op.Resource.Type != api.ResourceTypeRelease || len(op.Steps) != 1 || op.Steps[0].Status != "failed" {
		t.Fatal(op, err)
	}
	failure, err := op.Error.Get()
	code, codeErr := op.Steps[0].ErrorCode.Get()
	if err != nil || codeErr != nil || failure.Code != code || failure.Message == "" || code != api.ErrorCode(apierr.SizeMismatch) {
		t.Fatal(op, err, codeErr)
	}
	releasetest.AssertLedger(t, pool, f.Validator, "release.register", "release.verify")
}
