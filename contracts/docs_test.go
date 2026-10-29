package contracts

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestDocs_EndpointTable keeps the endpoint table of docs/contracts/README.md
// in sync with the operations and their x-fvpn-spec / x-fvpn-requirements.
func TestDocs_EndpointTable(t *testing.T) {
	doc := string(readFile(t, "../docs/contracts/README.md"))
	row := regexp.MustCompile("(?m)^\\| `([A-Z]+ [^`]+)` \\| `([A-Za-z]+)` \\| [^|]+ \\| ([0-9,]+) \\| ([^|]+) \\| ([^|]+) \\|$")
	got := map[string]string{}
	for _, m := range row.FindAllStringSubmatch(doc, -1) {
		got[m[1]] = strings.Join([]string{m[2], m[3], strings.TrimSpace(m[4]), strings.TrimSpace(m[5])}, " | ")
	}
	want := map[string]string{}
	for _, o := range operations(t) {
		var succ []string
		for code := range o.op.Responses.Map() {
			if code[0] == '2' || code == "304" {
				succ = append(succ, code)
			}
		}
		sort.Strings(succ)
		var reqs []string
		for _, r := range o.op.Extensions["x-fvpn-requirements"].([]any) {
			reqs = append(reqs, r.(string))
		}
		want[o.key] = strings.Join([]string{o.op.OperationID, strings.Join(succ, ","), o.op.Extensions["x-fvpn-spec"].(string), strings.Join(reqs, " ")}, " | ")
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("docs/contracts/README.md endpoint table is out of date (-openapi +docs):\n%s", diff)
	}
}
