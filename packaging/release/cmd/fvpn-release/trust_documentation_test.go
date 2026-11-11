package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func TestUPD02_TrustR7DocumentationOutput(t *testing.T) {
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
	var err error
	f.provider, err = signing.InitializeFileProvider(f.path("vault"))
	if err != nil {
		t.Fatal(err)
	}
	initial, err := envelope.ParseNodeTrustStoreJSON(readTestFile(t, f.path("release-A.trust.json")))
	if err != nil {
		t.Fatal(err)
	}
	root, _ := initial.Lookup("release-A")
	if err := f.provider.PinReleaseRoot(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	for i, step := range []string{"add", "retire", "revoke"} {
		bundle, err := f.provider.TrustBundle(context.Background())
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
		f.apply(readTestFile(t, updatePath))
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
