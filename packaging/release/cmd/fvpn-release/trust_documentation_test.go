package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func TestUPD02_TrustR7DocumentationOutput(t *testing.T) {
	// The virtual clock is deliberately outside the guide's 2026-2028 window.
	// Explicit -at must determine results, whatever the host calendar date.
	synctest.Test(t, testTrustDocumentationOutput)
}

func testTrustDocumentationOutput(t *testing.T) {
	// TEST ONLY real command run with runtime-generated keys. The documented
	// public hashes are one captured run; only those hashes vary between runs.
	f := &trustFixture{t: t, dir: t.TempDir(), keys: map[string]release.SigningKey{}}
	outputs := map[string]string{}
	for _, id := range []string{"release-A", "release-B"} {
		var output, diagnostic bytes.Buffer
		args := []string{"keygen", "-key-id", id, "-out", f.path(id + ".key.json"), "-trust-out", f.path(id + ".trust.json"), "-not-before", "2026-01-01T00:00:00Z", "-not-after", "2028-01-01T00:00:00Z"}
		if code := run(args, &output, &diagnostic); code != 0 {
			t.Fatalf("keygen exit=%d: %s", code, &diagnostic)
		}
		key, err := release.LoadKey(f.path(id + ".key.json"))
		if err != nil {
			t.Fatal(err)
		}
		f.keys[id] = key
		if id == "release-B" {
			outputs["keygen"], outputs["keygen-stderr"] = output.String(), diagnostic.String()
		}
	}
	initial, err := envelope.ParseNodeTrustStoreJSON(readTestFile(t, f.path("release-A.trust.json")))
	if err != nil {
		t.Fatal(err)
	}
	root, _ := initial.Lookup("release-A")
	added, err := envelope.ParseNodeTrustStoreJSON(readTestFile(t, f.path("release-B.trust.json")))
	if err != nil {
		t.Fatal(err)
	}
	newRoot, _ := added.Lookup("release-B")
	// Public snapshots are explicit test inputs, not apply receipts. This
	// documentation contract test must not consult the provider's wall clock.
	// Real-provider add/retire/revoke remains independently covered by
	// TestUPD02_TrustOperationsAndHandover (2000-2100 signer windows).
	current := []envelope.TrustedKey{root}
	for i, step := range []string{"add", "retire", "revoke"} {
		bundle, err := envelope.NewNodeTrustStore(current...)
		if err != nil {
			t.Fatal(err)
		}
		trustPath := f.path(fmt.Sprintf("centre-trust-r%d.json", i))
		writeTrustTest(t, trustPath, jsonTrustTest(t, bundle))
		signer, target, at := "release-B", "release-A", "2026-11-01T12:00:00Z"
		if step == "add" {
			signer, target = "release-A", f.path("release-B.trust.json")
		}
		if step == "revoke" {
			at = "2026-11-02T12:00:00Z"
		}
		updatePath := f.path(step + ".json")
		flags := map[string]string{"key": f.path(signer + ".key.json"), "trust": trustPath, "revision": fmt.Sprint(i + 1), "out": updatePath, step: target, "at": at}
		if step == "retire" {
			flags["not-after"] = "2026-12-01T00:00:00Z"
		}
		outputs[step] = f.success(trustArgs("trust-update", flags))
		outputs["verify-"+step] = f.success([]string{"trust-verify", "-update", updatePath, "-trust", trustPath, "-at", at, "-current-revision", fmt.Sprint(i)})
		switch step {
		case "add":
			current = append(current, newRoot)
		case "retire":
			current[0].NotAfter = trustDate("2026-12-01T00:00:00Z")
		case "revoke":
			current[0].Revoked = true
		}
	}
	doc := string(readTestFile(t, "../../../../docs/updates/release-process.md"))
	hash := regexp.MustCompile(`(?m)^update_sha256=[0-9a-f]{64}$`)
	for _, id := range []string{"keygen", "keygen-stderr", "add", "verify-add", "retire", "verify-retire", "revoke", "verify-revoke"} {
		actual := outputs[id]
		f.noSeed([]byte(actual))
		f.noPaths(actual)
		// Capture the actual run for reproducible documentation updates. These
		// lines contain only public results, never the generated seeds.
		t.Logf("trust-documentation-output %s %q", id, actual)
		marker := "<!-- trust-output:" + id + " -->\n\n```text\n"
		_, rest, ok := strings.Cut(doc, marker)
		if !ok {
			t.Errorf("missing documentation block %s", id)
			continue
		}
		want, _, ok := strings.Cut(rest, "```\n")
		if !ok {
			t.Errorf("unclosed documentation block %s", id)
			continue
		}
		if !strings.HasPrefix(id, "keygen") && (!hash.MatchString(actual) || !hash.MatchString(want)) {
			t.Errorf("invalid hash line in block %s", id)
		}
		// Exact comparison covers line names/order, target JSON quoting, record
		// field names/order/types/values and all stable command result values.
		if hash.ReplaceAllString(actual, "update_sha256=VARIABLE") != hash.ReplaceAllString(want, "update_sha256=VARIABLE") {
			t.Errorf("documentation block %s differs from command output", id)
		}
		for _, line := range strings.Split(actual, "\n") {
			if strings.HasPrefix(line, "key_record=") {
				var record release.TrustRecordResult
				if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "key_record=")), &record); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
