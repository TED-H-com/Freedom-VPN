package httpwire

import (
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/jackc/pgx/v5"
)

type fleetConfigErrorRow struct{ err error }

func (r fleetConfigErrorRow) Scan(...any) error { return r.err }

func TestD065_FleetConfigMissingRowUsesIssuerDefault(t *testing.T) {
	// The singleton DELETE guard stays intact; model the anomalous absent read
	// without dropping a base guard in an integration database.
	out, err := fleetConfig(fleetConfigErrorRow{pgx.ErrNoRows})
	if err != nil || out.MaxOfflineLeaseSeconds != leases.DefaultMaxOfflineLeaseSeconds || out.DefaultMaxOfflineLeaseSeconds == nil || int(*out.DefaultMaxOfflineLeaseSeconds) != leases.DefaultMaxOfflineLeaseSeconds || out.ResourceVersion != 0 || !out.UpdatedAt.Equal(time.Unix(0, 0).UTC()) {
		t.Fatal("missing singleton must have an unpersisted default", out, err)
	}
	failure := errors.New("TEST ONLY database unavailable")
	if _, err = fleetConfig(fleetConfigErrorRow{failure}); !errors.Is(err, failure) {
		t.Fatal("database failure silently became a default", err)
	}
}
