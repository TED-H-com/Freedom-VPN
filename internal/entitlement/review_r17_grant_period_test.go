package entitlement

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestReviewR17_Year9999GrantFailsClosed(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	plan := Plan{ID: uuid.New(), DeviceLimit: 3, ExitGroups: []uuid.UUID{uuid.New()}}
	grant := Grant{
		ID: uuid.New(), Plan: plan, Source: "admin_grant", Reason: "TEST ONLY far grant",
		Start: now, End: time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if ValidPeriod(grant.Start, grant.End, TestConfig().MaxGrantDuration) {
		t.Fatal("year 9999 fitted inside the 366-day maximum")
	}
	_, err := Compute(Input{Now: now, MemberStatus: "active", Grants: []Grant{grant}, Config: TestConfig()})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Compute accepted a year-9999 grant: %v", err)
	}
	// A duration that saturates still fails a positive maximum. Sub does not wrap.
	if grant.End.Sub(grant.Start) <= TestConfig().MaxGrantDuration {
		t.Fatalf("Sub wrapped or fit: %s", grant.End.Sub(grant.Start))
	}
}

func TestReviewR17_NoGrantRevocationIsCleared(t *testing.T) {
	roots := []string{".", filepath.Join("..", "members")}
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(raw)
			if strings.Contains(text, "revoked_at = NULL") || strings.Contains(text, "revoked_at=NULL") || strings.Contains(text, "SET revoked_at = NULL") {
				t.Errorf("%s clears revoked_at", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
