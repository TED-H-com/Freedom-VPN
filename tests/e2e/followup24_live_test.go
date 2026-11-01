//go:build e2e

package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestE2E_Followup24Native(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP24") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP24=1 with a fresh owned stack")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	if err = exec.CommandContext(t.Context(), "docker", "exec", l.member, "netbird", "debug", "log", "level", "debug").Run(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FVPN_E2E_NATIVE18", "1")
	t.Setenv("FVPN_E2E_KERNEL_OBSERVE", "1")
	TestE2E_Followup18Native(t)
}
func (r *runner) exportNative24(ctx context.Context, s *Stack) {
	rec := Record{ID: "native-debug-export", Title: "Private native debug export", Status: StatusPass}
	for _, role := range []string{"member", "node"} {
		receipt, err := s.ExportContainerFile(ctx, role, "/var/lib/vpn-node/e2e/netbird-service.log", filepath.Join(r.lab.root, ".agent/tmp/followup24", r.runID, role+"-debug.log"))
		noteJSON(&rec, role, receipt)
		if err != nil {
			rec.Status = StatusFail
		}
	}
	r.record(rec)
}
