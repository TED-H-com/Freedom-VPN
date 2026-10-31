package operations_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
)

func TestSECOPS11_BoundedFailureCausesMatchExitsVocabulary(t *testing.T) {
	fset := token.NewFileSet()
	exitsFile, err := parser.ParseFile(fset, "../exits/evidence_error.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var evidenceLiterals []string
	ast.Inspect(exitsFile, func(n ast.Node) bool {
		clause, ok := n.(*ast.CaseClause)
		if !ok {
			return true
		}
		for _, expr := range clause.List {
			lit, ok := expr.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			val, err := strconv.Unquote(lit.Value)
			if err == nil && val != "publish" && val != "readback" {
				evidenceLiterals = append(evidenceLiterals, val)
			}
		}
		return true
	})
	if len(evidenceLiterals) == 0 {
		t.Fatal("failed to extract PublicationEvidenceError literals from ../exits/evidence_error.go")
	}

	nbFile, err := parser.ParseFile(fset, "../netbird/client.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var errorKindLiterals []string
	ast.Inspect(nbFile, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		ident, ok := spec.Type.(*ast.Ident)
		if !ok || ident.Name != "ErrorKind" {
			return true
		}
		for _, v := range spec.Values {
			lit, ok := v.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			val, err := strconv.Unquote(lit.Value)
			if err == nil {
				errorKindLiterals = append(errorKindLiterals, val)
			}
		}
		return true
	})
	if len(errorKindLiterals) == 0 {
		t.Fatal("failed to extract netbird.ErrorKind literals from ../netbird/client.go")
	}

	produced := map[string]bool{}
	for _, cause := range evidenceLiterals {
		got := exits.PublicationFailureCause(exits.PublicationEvidenceError(cause), "readback")
		if got != cause {
			t.Fatalf("exits.PublicationFailureCause(%q) = %q, want %q", cause, got, cause)
		}
		if !operations.IsBoundedFailureCause(got) {
			t.Fatalf("operations.IsBoundedFailureCause(%q) = false; vocabulary diverged from exits.PublicationFailureCause", got)
		}
		produced[got] = true
	}

	for _, kind := range errorKindLiterals {
		for _, phase := range []string{"publish", "readback"} {
			got := exits.PublicationFailureCause(&netbird.Error{Kind: netbird.ErrorKind(kind)}, phase)
			want := phase + "_failed:" + kind
			if got != want {
				t.Fatalf("exits.PublicationFailureCause(kind=%q, phase=%q) = %q, want %q", kind, phase, got, want)
			}
			if !operations.IsBoundedFailureCause(got) {
				t.Fatalf("operations.IsBoundedFailureCause(%q) = false; vocabulary diverged from exits.PublicationFailureCause", got)
			}
			produced[got] = true
		}
	}

	for status := 400; status <= 599; status++ {
		got := exits.PublicationFailureCause(&netbird.Error{Kind: netbird.Upstream, Status: status}, "publish")
		want := "publish_rejected:" + strconv.Itoa(status)
		if got != want {
			t.Fatalf("exits.PublicationFailureCause(status=%d) = %q, want %q", status, got, want)
		}
		if !operations.IsBoundedFailureCause(got) {
			t.Fatalf("operations.IsBoundedFailureCause(%q) = false; vocabulary diverged from exits.PublicationFailureCause", got)
		}
		produced[got] = true
	}

	for _, extra := range []string{"member_forwarding_unconfirmed", "desired_state_unavailable"} {
		if !operations.IsBoundedFailureCause(extra) {
			t.Fatalf("operations.IsBoundedFailureCause(%q) = false, want true", extra)
		}
	}

	// Reverse check: verify that known allowlisted causes in operations are either
	// the two node causes or producible by exits.PublicationFailureCause.
	knownOperationsCauses := []string{
		"route_missing",
		"route_disabled",
		"route_ambiguous",
		"route_contract_mismatch",
		"route_peer_mismatch",
		"route_groups_mismatch",
		"group_evidence_unavailable",
		"no_distribution_groups",
		"identity_unconfirmed",
		"adapter_unavailable",
		"member_forwarding_unconfirmed",
		"desired_state_unavailable",
	}
	for _, c := range knownOperationsCauses {
		if c == "member_forwarding_unconfirmed" || c == "desired_state_unavailable" {
			continue
		}
		if !produced[c] || !slices.Contains(evidenceLiterals, c) {
			t.Fatalf("operations allowlist contains %q which is no longer in exits.PublicationFailureCause", c)
		}
	}

	for _, bad := range []string{
		"",
		"Step failed; quote the operation ID.",
		"central NetBird route withdrawn; node agent withdrawal unconfirmed",
		"publish_rejected:200",
		"publish_rejected:399",
		"publish_rejected:600",
		"publish_rejected:0400",
		"publish_rejected:+400",
		"publish_rejected:503:extra",
		"readback_rejected:503",
		"readback_failed:unknown",
		"publish_failed:unknown",
		"other_failed:unavailable",
		"TEST ONLY upstream response body",
	} {
		if operations.IsBoundedFailureCause(bad) {
			t.Fatalf("operations.IsBoundedFailureCause(%q) = true, want false", bad)
		}
	}
}

func TestD107_PublicationGateMessageRejectsUnboundedInputs(t *testing.T) {
	for _, tc := range []struct{ message, cause string }{
		{"member_forwarding_unconfirmed", ""},
		{"member_forwarding_unconfirmed", "lease_expired"},
		{"member_forwarding_unconfirmed", "publication_observed=absent"},
		{"member_forwarding_unconfirmed", "lease_unconfirmed\nTEST ONLY upstream body"},
		{"member_forwarding_unconfirmed", strings.Repeat("x", 1001)},
		{"TEST ONLY upstream body", "lease_unconfirmed"},
		{strings.Repeat("界", 1001), "lease_unconfirmed"},
		{"", "lease_unconfirmed"},
	} {
		if message, ok := operations.PublicationGateMessage(tc.message, tc.cause); ok || message != "" {
			t.Fatalf("unbounded pair produced a diagnostic: %q", message)
		}
	}
}
