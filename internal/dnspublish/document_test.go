package dnspublish

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
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
			Entries:       []dnspolicy.SourceEntry{{Domain: "malware.example.com", IncludeSubdomains: true}},
			ContentSHA256: "7777777777777777777777777777777777777777777777777777777777777777",
			Format:        "domain_list",
			Categories:    []string{"phishing", "malware"},
		}},
	}
}

func Test12_5_ContractResolverDocumentIsCompileOutput(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/examples/valid/ResolverDocument.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	// The contract input leaves bootstrap empty. Compile copies upstreams.
	compiled, err := Compile(Input{
		ProfileCode: doc.ProfileCode, BlockResponse: doc.BlockResponse, BlockedTTLSeconds: doc.BlockedTTLSeconds,
		SafeSearch: doc.SafeSearch, Upstreams: append([]string(nil), doc.Upstreams...),
		Rules: append([]Rule(nil), doc.Rules...), Feeds: append([]Feed(nil), doc.Feeds...),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(compiled.Bytes, raw) {
		t.Fatalf("vector is not Compile output\n got %s\nwant %s", compiled.Bytes, raw)
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	if compiled.Hash != digest {
		t.Fatalf("digest %s, compiler %s", digest, compiled.Hash)
	}
	var ref struct {
		ContentSHA256 string `json:"content_sha256"`
		SizeBytes     int    `json:"size_bytes"`
	}
	refRaw, err := os.ReadFile("../../contracts/examples/valid/ResolverDocumentRef.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(refRaw, &ref); err != nil {
		t.Fatal(err)
	}
	if ref.ContentSHA256 != compiled.Hash || ref.SizeBytes != len(compiled.Bytes) {
		t.Fatalf("reference binds %s/%d, compiler %s/%d", ref.ContentSHA256, ref.SizeBytes, compiled.Hash, len(compiled.Bytes))
	}
	var evidence struct {
		ContentSHA256 string `json:"content_sha256"`
	}
	evidenceRaw, err := os.ReadFile("../../contracts/examples/valid/DnsPolicyEvidence.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(evidenceRaw, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.ContentSHA256 != compiled.Hash {
		t.Fatalf("evidence digest %s, compiler %s", evidence.ContentSHA256, compiled.Hash)
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

func TestD054_AT20_FeedContentAndCanonicalIntegrity(t *testing.T) {
	in := sampleInput()
	in.Feeds[0].Entries = []dnspolicy.SourceEntry{{Domain: "TRACKER.example.com.", IncludeSubdomains: true}, {Domain: "cdn.example.com", IncludeSubdomains: true}, {Domain: "cdn.example.com", IncludeSubdomains: true}}
	c, err := Compile(in)
	if err != nil {
		t.Fatal(err)
	}
	in.Feeds[0].Entries = []dnspolicy.SourceEntry{{Domain: "cdn.example.com", IncludeSubdomains: true}, {Domain: "tracker.example.com", IncludeSubdomains: true}}
	reordered, err := Compile(in)
	if err != nil || !bytes.Equal(c.Bytes, reordered.Bytes) {
		t.Fatalf("feed ordering: %v", err)
	}
	lines, err := RenderRules(c.Document, "9", c.Hash)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"||tracker.example.com^$important", "@@|cdn.example.com^", "||cdn.example.com^", "||tracker.example.com^"} {
		if !strings.Contains(strings.Join(lines, "\n"), want) {
			t.Fatalf("missing layer rule %s", want)
		}
	}
	invalid := c.Document
	invalid.Feeds[0].RuleCount++
	if Verify(invalid) == nil {
		t.Fatal("tampered rule count accepted")
	}
	in.Feeds[0].Entries = nil
	if _, err := Compile(in); err == nil {
		t.Fatal("metadata-only feed accepted")
	}
}
