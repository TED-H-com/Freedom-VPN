package isolation

import (
	"os/exec"
	"testing"
)

func TestD164_ServingReadExporter(t *testing.T) {
	if out, err := exec.Command("python3", "-B", "-m", "unittest", "-v", "serving_read_test.py").CombinedOutput(); err != nil {
		t.Fatalf("serving read exporters: %v\n%s", err, out)
	}
}
