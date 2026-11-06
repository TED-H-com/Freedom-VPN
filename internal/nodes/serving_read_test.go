package nodes

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

func TestD164_ServingReadValidationAndCorruption(t *testing.T) {
	base := api.ServingReadEvidence{ReadAt: time.Time{}, Source: api.ServingReadEvidenceSourceLive, CandidateGeneration: decstr.U64(^uint64(0)), Verified: false, Diagnostic: strings.Repeat("界", 500)}
	for _, source := range []api.ServingReadEvidenceSource{api.ServingReadEvidenceSourceLive, api.ServingReadEvidenceSourceApply} {
		base.Source = source
		raw, err := json.Marshal(base)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decodeServingRead(raw)
		value, getErr := got.Get()
		if err != nil || getErr != nil || !reflect.DeepEqual(value, base) {
			t.Fatalf("valid retained read lost: %v %v", err, getErr)
		}
	}
	for name, mutate := range map[string]func(*api.ServingReadEvidence){
		"source":  func(v *api.ServingReadEvidence) { v.Source = "invented" },
		"length":  func(v *api.ServingReadEvidence) { v.Diagnostic += "界" },
		"non_utc": func(v *api.ServingReadEvidence) { v.ReadAt = time.Now().In(time.FixedZone("TEST", 3600)) },
	} {
		t.Run(name, func(t *testing.T) {
			v := base
			mutate(&v)
			if validateServingRead(presence.NewNullableWithValue(v)) == nil {
				t.Fatal("invalid typed evidence accepted")
			}
		})
	}
	for _, raw := range []string{`{}`, `[]`, `{"read_at":null,"source":"live","candidate_generation":"1","verified":false,"diagnostic":""}`, `{"read_at":"2026-02-30T00:00:00Z","source":"live","candidate_generation":"1","verified":false,"diagnostic":""}`, `{"read_at":"2026-11-06T00:00:00Z","source":"live","candidate_generation":"18446744073709551616","verified":false,"diagnostic":""}`, `{"read_at":"2026-11-06T00:00:00Z","source":"live","candidate_generation":"1","verified":null,"diagnostic":""}`} {
		if _, err := decodeServingRead([]byte(raw)); err == nil {
			t.Fatalf("corrupt stored evidence accepted: %s", raw)
		}
	}
	if v, err := decodeServingRead([]byte("null")); err != nil || !v.IsNull() {
		t.Fatal("explicit null", err)
	}
}

func TestD164_ObservationCanonicalPresenceAndIndependentFailures(t *testing.T) {
	raw, err := os.ReadFile("testdata/d126-mixed-observation.json")
	if err != nil {
		t.Fatal(err)
	}
	var report api.ObservationReport
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	want, err := canonjson.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	safety, health, forwarding, failures := evaluateObservationEvidence(report)
	hashes := make(map[[32]byte]bool)
	for _, mode := range []string{"absent", "null", "rejected", "verified"} {
		t.Run(mode, func(t *testing.T) {
			candidate := report
			switch mode {
			case "null":
				candidate.ServingRead.SetNull()
			case "rejected", "verified":
				candidate.ServingRead.Set(api.ServingReadEvidence{ReadAt: report.ObservedAt.Add(time.Hour), Source: api.ServingReadEvidenceSourceApply, CandidateGeneration: decstr.U64(^uint64(0)), Verified: mode == "verified", Diagnostic: "TEST ONLY opaque rejected text {"})
			}
			canonical, payload, _, _, err := marshalObservationReport(candidate)
			if err != nil {
				t.Fatal(err)
			}
			stored, err := canonjson.Canonicalize(payload)
			if err != nil || !bytes.Equal(stored, canonical) {
				t.Fatal("stored canonical values differ", err)
			}
			if mode == "absent" && !bytes.Equal(canonical, want) {
				t.Fatal("old canonical bytes changed")
			}
			hash := sha256.Sum256(canonical)
			if hashes[hash] {
				t.Fatal("presence/outcomes collapsed in hash")
			}
			hashes[hash] = true
			s, h, f, e := evaluateObservationEvidence(candidate)
			if s != safety || h != health || f != forwarding || !reflect.DeepEqual(e, failures) {
				t.Fatal("diagnostic changed independent failure precedence")
			}
		})
	}
}
