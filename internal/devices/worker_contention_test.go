package devices

import (
	"errors"
	"fmt"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
)

func TestRetryableContentionClassifiesWorkerOverlap(t *testing.T) {
	upstream := errors.New("netbird upstream")
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"lease lost", db.ErrLeaseLost, true},
		{"conflict", entitlement.ErrConflict, true},
		{"wrapped lease", fmt.Errorf("sync member: %w", db.ErrLeaseLost), true},
		{"joined overlap", errors.Join(db.ErrLeaseLost, entitlement.ErrConflict), true},
		{"joined with upstream", errors.Join(db.ErrLeaseLost, upstream), false},
		{"wrapped mixed join", fmt.Errorf("sync: %w", errors.Join(entitlement.ErrConflict, upstream)), false},
		{"upstream", upstream, false},
		{"invalid", entitlement.ErrInvalid, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := retryableContention(tc.err); got != tc.want {
				t.Fatalf("retryableContention(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
