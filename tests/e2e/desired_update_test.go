//go:build e2e

package e2e

import "testing"

func TestUPD05_AllowlistedReleaseIsNotAnUpdateIntent(t *testing.T) {
	const release = "00000000-0000-4000-8000-000000000001"
	for _, raw := range []string{
		`{"signed":{"serving_release_ids":["` + release + `"]}}`,
		`{"signed":{"serving_release_ids":["` + release + `"],"update":null}}`,
		`{"signed":`,
	} {
		if got := desiredUpdateRelease([]byte(raw)); got != "" {
			t.Fatalf("updater would run without an instruction: %q", got)
		}
	}
	raw := `{"signed":{"serving_release_ids":[],"update":{"release_id":"` + release + `"}}}`
	if got := desiredUpdateRelease([]byte(raw)); got != release {
		t.Fatalf("update instruction lost: %q", got)
	}
}
