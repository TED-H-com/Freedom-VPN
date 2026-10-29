package audit

import (
	"strings"
	"testing"
)

func TestSEC02_RedactTypedAndNestedDiffs(t *testing.T) {
	value := map[string]any{"state": map[string]string{"status": "pending", "setup_key": "TEST-ONLY-SECRET"}, "unapproved": "TEST-ONLY-NOT-ALLOWED", "authorization": "TEST-ONLY-TOKEN"}
	before, after, e := Diff(nil, value, "state", "authorization")
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(before)+string(after), "TEST-ONLY-") || !strings.Contains(string(after), "pending") {
		t.Fatal(string(before), string(after))
	}
}
