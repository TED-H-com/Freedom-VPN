package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func TestAPI01_Revision15ServingReadEvidenceAndCorrections(t *testing.T) {
	doc := loadDoc(t)
	if doc.Info.Version != "1.11.0" || len(operations(t)) != 50 {
		t.Fatalf("revision 15 must be API 1.11.0 with 50 operations, got %s and %d", doc.Info.Version, len(operations(t)))
	}
	if !strings.Contains(doc.Info.Description, "revision 15 (2026-11-05)") {
		t.Fatal("info.description must record revision 15 (2026-11-05)")
	}

	sre := doc.Components.Schemas["ServingReadEvidence"].Value
	if sre == nil {
		t.Fatal("missing ServingReadEvidence component")
	}
	if sre.AdditionalProperties.Has == nil || *sre.AdditionalProperties.Has {
		t.Fatal("ServingReadEvidence must set additionalProperties: false")
	}
	wantReq := []string{"read_at", "source", "candidate_generation", "verified", "diagnostic"}
	if !slices.Equal(sre.Required, wantReq) {
		t.Fatalf("ServingReadEvidence required = %v, want %v", sre.Required, wantReq)
	}
	if src := sre.Properties["source"].Value; src == nil || !slices.Equal(src.Enum, []any{"apply", "live"}) {
		t.Fatal("ServingReadEvidence.source must be enum [apply, live]")
	}
	if diag := sre.Properties["diagnostic"].Value; diag == nil || diag.MaxLength == nil || *diag.MaxLength != 500 {
		t.Fatal("ServingReadEvidence.diagnostic must have maxLength 500")
	}
	if len(sre.Properties) != 5 || sre.Properties["error_code"] != nil {
		t.Fatalf("ServingReadEvidence must have exactly the 5 adopted properties without error_code, got %d", len(sre.Properties))
	}
	if strings.Contains(sre.Description, "error_code") || strings.Contains(sre.Description, "empty") {
		t.Fatalf("ServingReadEvidence description must not invent error_code or empty diagnostic on verified=true: %s", sre.Description)
	}

	for _, tc := range []struct {
		name   string
		schema *presence.Nullable[api.ServingReadEvidence]
		parent any
	}{
		{"ObservationReport", nil, api.ObservationReport{}},
		{"NodeDetail", nil, api.NodeDetail{}},
	} {
		f, ok := reflect.TypeOf(tc.parent).FieldByName("ServingRead")
		if !ok || f.Type != reflect.TypeOf(presence.Nullable[api.ServingReadEvidence]{}) || !strings.Contains(f.Tag.Get("json"), "serving_read,omitzero") {
			t.Fatalf("%s.ServingRead wrong type or tag: %+v", tc.name, f)
		}
	}
	obsSchema := doc.Components.Schemas["ObservationReport"].Value
	if contains(obsSchema.Required, "serving_read") || !obsSchema.Properties["serving_read"].Value.Nullable || obsSchema.Properties["serving_read"].Value.AllOf[0].Ref != "#/components/schemas/ServingReadEvidence" {
		t.Fatal("ObservationReport.serving_read must be optional nullable ServingReadEvidence")
	}
	nodeDetailExt := doc.Components.Schemas["NodeDetail"].Value.AllOf[1].Value
	if contains(nodeDetailExt.Required, "serving_read") || !nodeDetailExt.Properties["serving_read"].Value.Nullable || nodeDetailExt.Properties["serving_read"].Value.AllOf[0].Ref != "#/components/schemas/ServingReadEvidence" {
		t.Fatal("NodeDetail.serving_read must be optional nullable ServingReadEvidence")
	}
	nodeSRDesc := nodeDetailExt.Properties["serving_read"].Value.Description
	if !strings.Contains(nodeSRDesc, "selected") || !strings.Contains(nodeSRDesc, "no available `serving_read`") || strings.Contains(nodeSRDesc, "accepting an observation increments") {
		t.Fatalf("NodeDetail.serving_read description must describe selected-observation null and ETag semantics without claiming any accepted observation increments resource_version: %s", nodeSRDesc)
	}
}

func TestISO01_D164ServingReadEvidenceVectorsAndPresence(t *testing.T) {
	var vector struct {
		Cases []struct {
			Name     string          `json:"name"`
			Evidence json.RawMessage `json:"evidence"`
		} `json:"cases"`
	}
	readJSON(t, "vectors/serving-read-evidence.json", &vector)
	if len(vector.Cases) != 4 {
		t.Fatalf("got %d cases, want 4", len(vector.Cases))
	}
	for _, c := range vector.Cases {
		t.Run(c.Name, func(t *testing.T) {
			if err := validateComponent(t, "ServingReadEvidence", c.Evidence); err != nil {
				t.Fatal(err)
			}
			var model api.ServingReadEvidence
			assertRevision13RoundTrip(t, c.Evidence, &model)
			switch c.Name {
			case "apply-verified":
				if !model.Verified || model.Source != api.ServingReadEvidenceSourceApply || !strings.HasPrefix(model.Diagnostic, "read_at=2026-10-28T15:00:29Z source=apply candidate=9007199254740993 verified=true ") || model.CandidateGeneration.String() != "9007199254740993" {
					t.Fatalf("unexpected apply-verified: %+v", model)
				}
			case "live-verified":
				if !model.Verified || model.Source != api.ServingReadEvidenceSourceLive || !strings.HasPrefix(model.Diagnostic, "read_at=2026-10-28T15:00:30Z source=live candidate=18446744073709551615 verified=true ") || model.CandidateGeneration.String() != "18446744073709551615" {
					t.Fatalf("unexpected live-verified: %+v", model)
				}
			case "apply-rejected":
				if model.Verified || model.Source != api.ServingReadEvidenceSourceApply || !strings.HasPrefix(model.Diagnostic, "read_at=2026-10-28T15:00:10Z source=apply candidate=18446744073709551615 verified=false ") || model.CandidateGeneration.String() != "18446744073709551615" {
					t.Fatalf("unexpected apply-rejected: %+v", model)
				}
			case "live-rejected-tc":
				if model.Verified || model.Source != api.ServingReadEvidenceSourceLive || !strings.HasPrefix(model.Diagnostic, "read_at=2026-10-28T15:00:15Z source=live candidate=9007199254740993 verified=false ") || model.CandidateGeneration.String() != "9007199254740993" {
					t.Fatalf("unexpected live-rejected-tc: %+v", model)
				}
			default:
				t.Fatalf("unexpected vector case %s", c.Name)
			}
		})
	}

	// Verify 3-state presence on ObservationReport (absent, explicit null, present verified/rejected).
	var obsAbsent api.ObservationReport
	readJSON(t, "examples/valid/ObservationReport.json", &obsAbsent)
	if obsAbsent.ServingRead.IsSpecified() {
		t.Fatal("ObservationReport.json must leave serving_read unspecified")
	}

	for _, tc := range []struct {
		file      string
		wantNull  bool
		wantSrc   api.ServingReadEvidenceSource
		wantVer   bool
		wantGen   string
		wantOlder bool
	}{
		{"ObservationReport.serving-read-null.json", true, "", false, "", false},
		{"ObservationReport.serving-read-apply-verified.json", false, api.ServingReadEvidenceSourceApply, true, "9007199254740993", true},
		{"ObservationReport.serving-read-live-rejected.json", false, api.ServingReadEvidenceSourceLive, false, "18446744073709551615", true},
	} {
		t.Run(tc.file, func(t *testing.T) {
			raw := readFile(t, "examples/valid/"+tc.file)
			if err := validateComponent(t, "ObservationReport", raw); err != nil {
				t.Fatal(err)
			}
			assertObservationRoundTrip(t, raw)
			var rep api.ObservationReport
			if err := json.Unmarshal(raw, &rep); err != nil {
				t.Fatal(err)
			}
			if !rep.ServingRead.IsSpecified() {
				t.Fatal("serving_read must be specified")
			}
			if tc.wantNull {
				if !rep.ServingRead.IsNull() {
					t.Fatal("serving_read must be explicit null")
				}
				return
			}
			sr, err := rep.ServingRead.Get()
			if err != nil {
				t.Fatal(err)
			}
			if sr.Source != tc.wantSrc || sr.Verified != tc.wantVer || sr.CandidateGeneration.String() != tc.wantGen {
				t.Fatalf("serving_read = %+v", sr)
			}
			if tc.wantOlder && !rep.ObservedAt.After(sr.ReadAt) {
				t.Fatalf("observed_at %s must be after retained read_at %s", rep.ObservedAt, sr.ReadAt)
			}
			if !strings.HasPrefix(sr.Diagnostic, "read_at=") || !strings.Contains(sr.Diagnostic, "verified=") || len(sr.Diagnostic) > 500 {
				t.Fatalf("diagnostic must be producer-faithful bounded string for both verified=true and false, got %q", sr.Diagnostic)
			}
			appliedGen, err := rep.AppliedConfigGeneration.Get()
			if err != nil || appliedGen == sr.CandidateGeneration {
				t.Fatal("candidate_generation fixture must stay distinct from applied_config_generation")
			}
		})
	}

	// Verify 3-state presence on NodeDetail (absent, explicit null, present verified/rejected).
	var nodeAbsent api.NodeDetail
	readJSON(t, "examples/valid/NodeDetail.json", &nodeAbsent)
	if nodeAbsent.ServingRead.IsSpecified() {
		t.Fatal("NodeDetail.json must leave serving_read unspecified")
	}
	for _, tc := range []struct {
		file     string
		wantNull bool
		wantVer  bool
		wantGen  string
	}{
		{"NodeDetail.serving-read-null.json", true, false, ""},
		{"NodeDetail.serving-read-apply-verified.json", false, true, "9007199254740993"},
		{"NodeDetail.serving-read-live-rejected.json", false, false, "18446744073709551615"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			raw := readFile(t, "examples/valid/"+tc.file)
			if err := validateComponent(t, "NodeDetail", raw); err != nil {
				t.Fatal(err)
			}
			var detail api.NodeDetail
			assertRevision13RoundTrip(t, raw, &detail)
			if !detail.ServingRead.IsSpecified() {
				t.Fatal("serving_read must be specified")
			}
			if tc.wantNull {
				if !detail.ServingRead.IsNull() {
					t.Fatal("serving_read must be explicit null")
				}
				return
			}
			sr, err := detail.ServingRead.Get()
			if err != nil || sr.Verified != tc.wantVer || sr.CandidateGeneration.String() != tc.wantGen {
				t.Fatalf("NodeDetail.serving_read = %+v err=%v", sr, err)
			}
		})
	}

	// Invalid ServingReadEvidence mutations and uint64 overflow rejection.
	fresh := func() map[string]any {
		var v map[string]any
		readJSON(t, "examples/valid/ServingReadEvidence.apply-verified.json", &v)
		return v
	}
	for name, mutate := range map[string]func(map[string]any){
		"missing read_at":              func(v map[string]any) { delete(v, "read_at") },
		"missing source":               func(v map[string]any) { delete(v, "source") },
		"missing candidate_generation": func(v map[string]any) { delete(v, "candidate_generation") },
		"missing verified":             func(v map[string]any) { delete(v, "verified") },
		"missing diagnostic":           func(v map[string]any) { delete(v, "diagnostic") },
		"numeric generation":           func(v map[string]any) { v["candidate_generation"] = 128 },
		"leading zero generation":      func(v map[string]any) { v["candidate_generation"] = "01" },
		"negative generation":          func(v map[string]any) { v["candidate_generation"] = "-1" },
		"21 digit generation":          func(v map[string]any) { v["candidate_generation"] = "184467440737095516150" },
		"unknown source":               func(v map[string]any) { v["source"] = "rollback" },
		"non UTC read_at":              func(v map[string]any) { v["read_at"] = "2026-10-28T15:00:29+00:00" },
		"unexpected error_code":        func(v map[string]any) { v["error_code"] = "GENERATION_MISMATCH" },
		"diagnostic over 500":          func(v map[string]any) { v["diagnostic"] = strings.Repeat("x", 501) },
		"extra property":               func(v map[string]any) { v["extra"] = true },
	} {
		t.Run(name, func(t *testing.T) {
			v := fresh()
			mutate(v)
			if validateComponent(t, "ServingReadEvidence", mustMarshal(t, v)) == nil {
				t.Fatalf("invalid ServingReadEvidence (%s) was accepted", name)
			}
		})
	}
	overflow := fresh()
	overflow["candidate_generation"] = "18446744073709551616"
	var overflowModel api.ServingReadEvidence
	if err := json.Unmarshal(mustMarshal(t, overflow), &overflowModel); err == nil {
		t.Fatal("uint64 overflow 18446744073709551616 must be rejected by Go decstr.U64")
	}

	// Existing ObservationReport fixtures must remain byte-for-byte unchanged and keep their canonical hashes.
	wantCanonHashes := map[string]string{
		"ObservationReport.json":                        "1a6b69f4f9fd62f7e883918efe4213654e165a3af65eab8c4fd0023eff59ecda",
		"ObservationReport.awaiting-member-grants.json": "33b75a0352badbca8eaf1eab5cd55aa9aec696c92fc10f80f309355c1710e9a4",
		"ObservationReport.tc-verified.json":            "28e3deb637cce042daf63b47cd456ac5175c1d596499889fe7e95428a23db29a",
		"ObservationReport.tc-rejected.json":            "8f0bb971a6801e7d3f516ae673465beae06754e6019c656f2ef432eed4ef65bc",
		"ObservationReport.tc-truncated.json":           "b3518d4bf45d8a74d9cd98df405babeea7a92fa1e8cf9cdd53193d003c1ecd99",
		"ObservationReport.tc-null.json":                "d88e66faa01bbaefb24d7b4a6327d80737ce640cc394d344062aad95f2d48dd7",
	}
	wantRawHashes := map[string]string{
		"ObservationReport.json":                        "98bd6abbb84bee964b22110d60b755a5d3d227de0a87f908ed6e186b082b2def",
		"ObservationReport.awaiting-member-grants.json": "fc681d3e5dc45683d6e626b7a569133df32bc2c39636d1923d26c927ceac2f25",
		"ObservationReport.tc-verified.json":            "a962227396637000e8c7cedce3abf45159151ea59d71ee766b98220894e4b028",
		"ObservationReport.tc-rejected.json":            "13e10b12631999b0ece48ffb260d3203e9c792a9f61aea977afbf0954ed934ae",
		"ObservationReport.tc-truncated.json":           "24ac04be12fd859f2579ea8c9fda62cf4410a6bedf9f49d1ca1982542e1ec619",
		"ObservationReport.tc-null.json":                "b2b9acf15df1996160822ef709fa1d9431d1d22511fe6341a94ce6151d4a8619",
	}
	for _, file := range []string{
		"ObservationReport.json",
		"ObservationReport.awaiting-member-grants.json",
		"ObservationReport.tc-verified.json",
		"ObservationReport.tc-rejected.json",
		"ObservationReport.tc-truncated.json",
		"ObservationReport.tc-null.json",
	} {
		raw := readFile(t, "examples/valid/"+file)
		if err := validateComponent(t, "ObservationReport", raw); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		assertObservationRoundTrip(t, raw)
		if wantCanon, ok := wantCanonHashes[file]; ok {
			gotHash, err := canonjson.HashHex(raw)
			if err != nil || gotHash != wantCanon {
				t.Fatalf("%s canonical hash = %s, want %s (err %v)", file, gotHash, wantCanon, err)
			}
		}
		if wantRaw, ok := wantRawHashes[file]; ok {
			sum := sha256.Sum256(raw)
			if gotRaw := hex.EncodeToString(sum[:]); gotRaw != wantRaw {
				t.Fatalf("%s raw hash = %s, want %s", file, gotRaw, wantRaw)
			}
		}
	}
}

func TestMTR03_D112HandoffSamplingIntervalBoundWording(t *testing.T) {
	doc := loadDoc(t)
	submitDesc := doc.Paths.Value("/agent/usage-batches").Post.Description
	originDesc := doc.Components.Schemas["UsageBatchRequest"].Value.Properties["origin_incarnation_id"].Value.Description
	for name, desc := range map[string]string{
		"submitUsageBatch":                        submitDesc,
		"UsageBatchRequest.origin_incarnation_id": originDesc,
	} {
		for _, want := range []string{
			"sampling_interval_bound_seconds",
			"default 10 seconds",
			"Central stores no per-stream historical sampling interval",
		} {
			if !strings.Contains(desc, want) {
				t.Fatalf("%s missing D-112 phrase %q", name, want)
			}
		}
		if strings.Contains(desc, "in force for the retired stream") {
			t.Fatalf("%s still contains misleading legacy phrase 'in force for the retired stream'", name)
		}
	}

	var vector struct {
		Comment string `json:"_comment"`
		Context struct {
			Interval     int `json:"sampling_interval_seconds"`
			DefaultBound int `json:"sampling_interval_bound_default_seconds"`
		} `json:"context"`
	}
	readJSON(t, "vectors/usage-retired-handoff.json", &vector)
	if vector.Context.Interval != 30 || vector.Context.DefaultBound != 10 {
		t.Fatalf("handoff vector context = %+v, want interval 30 and default 10", vector.Context)
	}
	if !strings.Contains(vector.Comment, "sampling_interval_bound_seconds") || !strings.Contains(vector.Comment, "default is 10") {
		t.Fatalf("handoff vector comment missing D-112 note: %s", vector.Comment)
	}
}

func TestAT29_D117FiniteGrantWithdrawalCorrection(t *testing.T) {
	doc := loadDoc(t)
	revokeDesc := doc.Paths.Value("/admin/members/{member_id}/grants/{grant_id}/revoke").Post.Description
	for _, want := range []string{
		"all entitlement grants are finite",
		"open-ended grant creation remains forbidden",
	} {
		if !strings.Contains(revokeDesc, want) {
			t.Fatalf("revokeMemberGrant description missing %q", want)
		}
	}
	if strings.Contains(revokeDesc, "or an open-ended grant") {
		t.Fatal("revokeMemberGrant description still claims open-ended grants exist")
	}

	var vector struct {
		Comment string `json:"_comment"`
		Cases   []struct {
			Name string `json:"name"`
		} `json:"cases"`
	}
	readJSON(t, "vectors/member-grant-revoke.json", &vector)
	var names []string
	for _, c := range vector.Cases {
		names = append(names, c.Name)
	}
	wantNames := []string{"finite", "same-key-replay", "unknown-or-other-member", "already-revoked", "expired"}
	if !slices.Equal(names, wantNames) {
		t.Fatalf("member-grant-revoke cases = %v, want %v", names, wantNames)
	}
}
