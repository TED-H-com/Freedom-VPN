//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/internal/usage/usagetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func TestCLI_FreezeRecoveryCompleteAdjustAndAudit(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)

	uFix := usagetest.New(t)
	ps := usage.NewProviderSigner(uFix.Provider)
	svc, err := usage.NewService(usage.Config{
		Pool:              pool,
		Signer:            ps,
		LedgerEpochSigner: ps,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	adminID := uuid.New()
	memberID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO members (id, email, display_name, status)
		VALUES ($1, 'cli-member@example.com', 'CLI Member', 'active')
	`, memberID); err != nil {
		t.Fatalf("insert member: %v", err)
	}

	// 1. freeze
	var out bytes.Buffer
	if err := Run(ctx, svc, []string{"freeze", "--actor", adminID.String(), "--reason", "planned restore"}, &out); err != nil {
		t.Fatalf("Run freeze: %v", err)
	}

	// 2. begin-recovery
	out.Reset()
	if err := Run(ctx, svc, []string{"begin-recovery", "--actor", adminID.String(), "--reason", "restore snapshot"}, &out); err != nil {
		t.Fatalf("Run begin-recovery: %v", err)
	}
	var recState usage.RecoveryState
	if err := json.Unmarshal(out.Bytes(), &recState); err != nil {
		t.Fatalf("unmarshal begin-recovery output: %v", err)
	}
	rawRecEnv, err := recState.SignedRecord.MarshalCanonical()
	if err != nil {
		t.Fatalf("MarshalCanonical: %v", err)
	}
	if _, _, err := envelope.Verify(rawRecEnv, uFix.CentralTrust, envelope.PurposeLedgerEpoch, time.Now().UTC()); err != nil {
		t.Fatalf("Verify recovery record with PurposeLedgerEpoch: %v", err)
	}

	// 3. complete-recovery
	out.Reset()
	if err := Run(ctx, svc, []string{"complete-recovery", "--actor", adminID.String()}, &out); err != nil {
		t.Fatalf("Run complete-recovery: %v", err)
	}

	// 4. adjust
	out.Reset()
	if err := Run(ctx, svc, []string{
		"adjust",
		"--member-id", memberID.String(),
		"--period", "2026-10",
		"--upload-bytes", "-1024",
		"--download-bytes", "-2048",
		"--actor", adminID.String(),
		"--reason", "operator correction",
	}, &out); err != nil {
		t.Fatalf("Run adjust: %v", err)
	}

	// 5. drill-down
	out.Reset()
	if err := Run(ctx, svc, []string{
		"drill-down",
		"--member-id", memberID.String(),
		"--period", "2026-10",
	}, &out); err != nil {
		t.Fatalf("Run drill-down: %v", err)
	}

	// Verify all 4 mutating commands wrote valid audit rows with 0 anomalies.
	auditStore := &audit.Store{Pool: pool}
	page, err := auditStore.List(ctx, api.ListAuditEventsParams{})
	if err != nil {
		t.Fatalf("auditStore.List: %v", err)
	}
	if auditStore.AnomalyCount() != 0 {
		t.Fatalf("expected 0 audit anomalies, got %d", auditStore.AnomalyCount())
	}
	if len(page.Items) < 4 {
		t.Fatalf("expected at least 4 audit events, got %d", len(page.Items))
	}
}
