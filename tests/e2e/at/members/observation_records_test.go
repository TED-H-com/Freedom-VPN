package members

import (
	"os/exec"
	"testing"
)

func TestD146_F24ObservationReceiptsAcceptedRetryAndOverwrite(t *testing.T) {
	if b, err := exec.Command("python3", "-B", "observation_records_test.py").CombinedOutput(); err != nil {
		t.Fatalf("observation receipts: %v\n%s", err, b)
	}
}
