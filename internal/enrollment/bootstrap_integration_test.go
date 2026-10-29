//go:build integration

package enrollment_test

import (
	"context"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

type delayedDecrypt struct {
	signing.Provider
	beforeOpen func(context.Context) error
}

func (p delayedDecrypt) Open(ctx context.Context, sealed signing.Sealed, aad []byte) ([]byte, error) {
	if err := p.beforeOpen(ctx); err != nil {
		return nil, err
	}
	return p.Provider.Open(ctx, sealed, aad)
}

// §6.4 / D-015: the DB verdict wins even if a slow provider finishes after the
// delivery deadline. A committed delivery cannot become an expiry error.
func TestAT28_BootstrapCommittedDeliverySurvivesDecryptDelay(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	_, _, out, cert := enroll(t, f)
	id, err := f.auth.Authenticate(ctx, cert)
	if err != nil {
		t.Fatal(err)
	}
	expires := time.Now().Add(2 * time.Second)
	secret := "TEST ONLY delayed bootstrap"
	if err = f.s.StoreBootstrap(ctx, id, out.OperationID, "test-delayed-key", secret, expires); err != nil {
		t.Fatal(err)
	}
	provider := delayedDecrypt{Provider: f.vault, beforeOpen: func(ctx context.Context) error {
		var committed bool
		if err := f.pool.QueryRow(ctx, `SELECT delivered_at IS NOT NULL AND delivered_at<expires_at AND ciphertext IS NULL FROM node_bootstrap_secrets WHERE incarnation_id=$1`, id.IncarnationID).Scan(&committed); err != nil {
			return err
		}
		if !committed {
			t.Fatal("decryption preceded the committed database delivery verdict")
		}
		timer := time.NewTimer(time.Until(expires) + time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}}
	s, err := enrollment.NewService(f.pool, f.ca, f.auth, provider, f.vault, enrollment.Config{BootstrapValidity: enrollment.TestBootstrapValidity})
	if err != nil {
		t.Fatal(err)
	}
	delivery, err := s.TakeBootstrap(ctx, cert)
	if err != nil || delivery.SetupKey != secret || time.Now().Before(delivery.ExpiresAt) {
		t.Fatal("committed first delivery must succeed after slow decryption", err)
	}
	delivery, err = s.TakeBootstrap(ctx, cert)
	expectCode(t, err, apierr.BootstrapConsumed)
	if delivery.SetupKey != "" || apierr.BootstrapConsumed.HTTPStatus() != 410 {
		t.Fatal("second delivery must be HTTP 410 with no secret")
	}
}
