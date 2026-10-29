package dnspublish

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func sampleInput() Input {
	return Input{
		ProfileCode:       "standard",
		BlockResponse:     "nxdomain",
		BlockedTTLSeconds: 60,
		SafeSearch:        false,
		Upstreams:         []string{"192.0.2.53", "192.0.2.54"},
		Rules: []Rule{
			{RuleID: "01900000-0000-7000-8000-0000000000b1", Domain: "Tracker.Example.com.", IncludeSubdomains: true, Action: "block", Reason: "admin_rule"},
			{RuleID: "01900000-0000-7000-8000-0000000000a1", Domain: "cdn.example.com", IncludeSubdomains: false, Action: "allow", Reason: "false_positive"},
		},
		Feeds: []Feed{{
			FeedCode:      "example-malware",
			ContentSHA256: "7777777777777777777777777777777777777777777777777777777777777777",
			Format:        "domain_list",
			Categories:    []string{"phishing", "malware"},
		}},
	}
}

func Test12_4_CanonicalDocumentDeterministic(t *testing.T) {
	first, err := Compile(sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Compile(sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes, second.Bytes) || first.Hash != second.Hash {
		t.Fatalf("same rules produced different bytes\n%s\n%s", first.Bytes, second.Bytes)
	}
	again, raw, err := Hash(first.Document)
	if err != nil || again != first.Hash || !bytes.Equal(raw, first.Bytes) {
		t.Fatalf("hash mismatch %s vs %s", again, first.Hash)
	}

	reordered := sampleInput()
	reordered.Rules = []Rule{reordered.Rules[1], reordered.Rules[0]}
	reordered.Feeds[0].Categories = []string{"malware", "phishing"}
	third, err := Compile(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes, third.Bytes) {
		t.Fatalf("rule order changed the document\n%s\n%s", first.Bytes, third.Bytes)
	}

	// A second activation of the same rules is a new generation outside the
	// hashed bytes, so the document itself does not change (§12.5).
	if strings.Contains(string(first.Bytes), "policy_generation") {
		t.Fatal("policy_generation leaked into the hashed document")
	}
	if first.Document.Privacy != (Privacy{}) || len(first.Document.FallbackDNS) != 0 {
		t.Fatalf("privacy or fallback not forced off: %+v", first.Document)
	}
	if first.Document.Rules[0].Layer != layerAdminBlock || first.Document.Rules[1].Layer != layerAdminAllow {
		t.Fatalf("rules not ordered by layer: %+v", first.Document.Rules)
	}
	if first.Document.Rules[0].Domain != "tracker.example.com" {
		t.Fatalf("domain not canonical: %s", first.Document.Rules[0].Domain)
	}
	if strings.Join(first.Document.Feeds[0].Categories, ",") != "malware,phishing" {
		t.Fatalf("categories not sorted: %v", first.Document.Feeds[0].Categories)
	}
}

func Test12_4_IDNAAndConflicts(t *testing.T) {
	in := sampleInput()
	in.Rules = []Rule{{
		RuleID: uuid.NewString(), Domain: "bücher.example.com", IncludeSubdomains: false, Action: "block", Reason: "admin_rule",
	}}
	got, err := Compile(in)
	if err != nil {
		t.Fatal(err)
	}
	if got.Document.Rules[0].Domain != "xn--bcher-kva.example.com" {
		t.Fatalf("IDNA canonical = %s", got.Document.Rules[0].Domain)
	}

	in.Rules = append(in.Rules, Rule{
		RuleID: uuid.NewString(), Domain: "xn--bcher-kva.example.com", IncludeSubdomains: false, Action: "block", Reason: "again",
	})
	if _, err := Compile(in); err == nil {
		t.Fatal("same-layer conflict was accepted")
	}

	bad := sampleInput()
	bad.Rules[0].Domain = "https://tracker.example.com/path"
	if _, err := Compile(bad); err == nil {
		t.Fatal("URL was accepted as a domain")
	}

	pub := sampleInput()
	pub.Upstreams = nil
	if _, err := Compile(pub); err == nil {
		t.Fatal("missing upstream was accepted")
	}
}

func Test12_5_SafeSearchAndBlockModeChangeBytes(t *testing.T) {
	base, err := Compile(sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	search := sampleInput()
	search.SafeSearch = true
	withSearch, err := Compile(search)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(base.Bytes, withSearch.Bytes) || !withSearch.Document.SafeSearch {
		t.Fatal("safe search did not change the document")
	}
	refused := sampleInput()
	refused.BlockResponse = "refused"
	withRefused, err := Compile(refused)
	if err != nil {
		t.Fatal(err)
	}
	if withRefused.Document.BlockResponse != "refused" || bytes.Equal(base.Bytes, withRefused.Bytes) {
		t.Fatal("block response did not change the document")
	}
}

func Test12_5_RenderRulesCarryHash(t *testing.T) {
	compiled, err := Compile(sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	lines, err := RenderRules(compiled.Document, "7", compiled.Hash)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, compiled.Hash) || !strings.Contains(text, "! fvpn-policy-generation: 7") {
		t.Fatalf("headers missing:\n%s", text)
	}
	if !strings.Contains(text, "||tracker.example.com^$important") || !strings.Contains(text, "@@|cdn.example.com^") {
		t.Fatalf("rules missing:\n%s", text)
	}
}
