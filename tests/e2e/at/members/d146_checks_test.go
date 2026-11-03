package members

import (
	"os/exec"
	"testing"
)

func TestAT09_D146RequiresCompleteSameRunClampEvidence(t *testing.T) {
	cmd := exec.Command("python3", "-B", "d146_evidence_test.py")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("D-146 evidence discriminator: %v\n%s", err, output)
	}
}
