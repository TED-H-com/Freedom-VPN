package updater

import (
	"errors"
	"fmt"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

func TestD155_ExitCode(t *testing.T) {
	busy := transient(apierr.UpdateBusy, "lock")
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"success", nil, 0},
		{"blocked", ErrMembersBlocked, 2},
		{"wrapped blocked", fmt.Errorf("recovery: %w", ErrMembersBlocked), 2},
		{"live holder", busy, ExitBusy},
		{"wrapped live holder", fmt.Errorf("recovery: %w", busy), ExitBusy},
		{"retry flag is not the classifier", permanent(apierr.UpdateBusy, "lock"), ExitBusy},
		{"blocked wins", errors.Join(busy, ErrMembersBlocked), 2},
		{"blocked wins reversed", errors.Join(ErrMembersBlocked, busy), 2},
		{"stale boot", transient(apierr.UpdateBusy, "lock boot id"), 1},
		{"stale pid", transient(apierr.UpdateBusy, "lock pid"), 1},
		{"stale process", transient(apierr.UpdateBusy, "lock process"), 1},
		{"stale start", transient(apierr.UpdateBusy, "lock start time"), 1},
		{"other busy", transient(apierr.UpdateBusy, "other"), 1},
		{"empty busy", transient(apierr.UpdateBusy, ""), 1},
		{"not exact detail", transient(apierr.UpdateBusy, "lock "), 1},
		{"not busy", transient(apierr.ValidationFailed, "lock"), 1},
		{"plain", errors.New("updater: UPDATE_BUSY: lock"), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.want {
				t.Fatalf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
	if ExitBusy != 3 {
		t.Fatalf("ExitBusy = %d, want 3", ExitBusy)
	}
}
