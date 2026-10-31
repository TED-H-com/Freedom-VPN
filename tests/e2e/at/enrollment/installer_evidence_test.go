//go:build e2e

package enrollment

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
)

func TestAT07_InstallerEvidenceUsesDigestLabels(t *testing.T) {
	// TEST ONLY: disclosure sentinel, never a real credential.
	const withheld = "test-only-private-diagnostic"
	h := e2e.InstallerHashes{Key: strings.Repeat("a", 64), Cert: strings.Repeat("b", 64),
		NodeJSON: "absent", UsageDB: "absent", UsageWAL: "absent", MachineID: "absent", Current: "absent"}
	out := e2e.InstallerOutcome{Exit: 1, Message: withheld, Before: h, Hashes: h, RetiredHashes: h}
	raw, err := json.Marshal(installerOutcomeView(out, errors.New(withheld)))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, bad := range []string{withheld, `"Key":`, `"Cert":`, `"Message":`} {
		if strings.Contains(text, bad) {
			t.Fatalf("unsafe evidence field: %s", bad)
		}
	}
	for _, name := range []string{"before", "after", "retired"} {
		var view map[string]json.RawMessage
		if err := json.Unmarshal(raw, &view); err != nil {
			t.Fatal(err)
		}
		var hashes map[string]string
		if err := json.Unmarshal(view[name], &hashes); err != nil {
			t.Fatal(err)
		}
		if len(hashes) != 7 || hashes["key_sha256"] != h.Key || hashes["cert_sha256"] != h.Cert || hashes["usage_wal_sha256"] != "absent" {
			t.Fatalf("digest/sentinel loss in %s", name)
		}
	}
}
