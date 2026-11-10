package isolation

import (
	"os/exec"
	"testing"
)

func TestBoundaryProbeErrorDiagnostics(t *testing.T) {
	// The probe command uses explicit files because it is excluded from the
	// ordinary package. Exercise its real socket/error path, not a copied helper.
	command := exec.Command("go", "test", "-count=1", "-v", "probe_main.go", "observer_main.go", "observer_diagnostics_test.go")
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("probe diagnostics: %v\n%s", err, out)
	}
	t.Log(string(out))
}
