//go:build e2e

package e2e

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestE2E_PortalBilling is a focused rerun of portal-billing-return and
// billing (§18.4, §19 AT03/AT04) on a fresh stack, with only their
// prerequisites: stack-health, admin-login, threat-feed-central (the DNS
// profile the plan needs) and serving-catalogue. Assertions are the ones
// TestE2E_Acceptance uses. Records go to docs/e2e/runs/<UTC date>/<label>,
// where the label is FVPN_E2E_RUN_LABEL or "portal-billing". The stack must
// be fresh because the catalogue codes are created once.
func TestE2E_PortalBilling(t *testing.T) {
	if os.Getenv("FVPN_E2E") != "1" || os.Getenv("FVPN_E2E_PORTAL_BILLING") != "1" {
		t.Skip("set FVPN_E2E=1 FVPN_E2E_PORTAL_BILLING=1 on a fresh stack for the focused billing run")
	}
	lab, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := RunDir(lab.root, os.Getenv(RunLabelEnv), "portal-billing", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	run := &runner{t: t, lab: lab, dir: dir, summary: true, runID: uuid.NewString(), sourceCommit: strings.TrimSpace(string(head))}
	defer run.finish()
	run.health(ctx)
	admin := run.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		return
	}
	run.threatFeed(ctx, admin)
	run.catalogue(ctx, admin)
	run.billing(ctx, admin)
}
