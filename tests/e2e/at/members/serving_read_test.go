package members

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

func TestD164_NodeDiagnosticSummaryPresence(t *testing.T) {
	node := api.NodeDetail{SafetyStatus: api.SafetyStatusFailed}
	before := summarizeNodeDetail(node)
	if _, ok := before["serving_read"]; ok {
		t.Fatal("absent field became present")
	}
	node.ServingRead.SetNull()
	if v, ok := summarizeNodeDetail(node)["serving_read"]; !ok || v != nil {
		t.Fatal("explicit null lost")
	}
	for _, verified := range []bool{false, true} {
		value := api.ServingReadEvidence{ReadAt: time.Date(2026, 11, 6, 1, 2, 3, 123456789, time.UTC), Source: api.ServingReadEvidenceSourceLive, CandidateGeneration: decstr.U64(^uint64(0)), Verified: verified, Diagnostic: strings.Repeat("x", 499) + "{"}
		node.ServingRead = presence.NewNullableWithValue(value)
		summary := summarizeNodeDetail(node)
		raw, err := json.Marshal(summary["serving_read"])
		if err != nil {
			t.Fatal(err)
		}
		want, err := json.Marshal(value)
		if err != nil || !bytes.Equal(raw, want) {
			t.Fatal("diagnostic changed", err)
		}
		delete(summary, "serving_read")
		if !reflect.DeepEqual(before, summary) {
			t.Fatal("diagnostic changed health/check summary")
		}
	}
}
