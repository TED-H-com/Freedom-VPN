package contracts

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func TestAT29_ManualGrantWithdrawalContract(t *testing.T) {
	doc := loadDoc(t)
	if doc.Info.Version != "1.10.0" || len(operations(t)) != 50 {
		t.Fatal("revision 14 must be API 1.10.0 with 50 operations")
	}
	op := doc.Paths.Value("/admin/members/{member_id}/grants/{grant_id}/revoke").Post
	if op.OperationID != "revokeMemberGrant" || !slices.Equal(securityNames(op), []string{"adminSession+csrfToken"}) || !hasParam(op, "header", "Idempotency-Key", true) {
		t.Fatal("withdrawal must be an authenticated idempotent admin command")
	}
	for _, phrase := range []string{"member.grant_revoke", "audit", "one heartbeat", "D-050 / D-081", "open-ended", "expiry remains a separate case", "RESOURCE_NOT_FOUND", "STATE_CONFLICT"} {
		if !strings.Contains(op.Description, phrase) {
			t.Fatalf("missing withdrawal semantics %q", phrase)
		}
	}
	var vector struct {
		Request  json.RawMessage `json:"request"`
		Response json.RawMessage `json:"response"`
		Cases    []struct {
			Name   string `json:"name"`
			Status int    `json:"http_status"`
			Code   string `json:"error_code"`
		} `json:"cases"`
	}
	readJSON(t, "vectors/member-grant-revoke.json", &vector)
	if err := validateComponent(t, "ReasonRequest", vector.Request); err != nil {
		t.Fatal(err)
	}
	if err := validateComponent(t, "Operation", vector.Response); err != nil {
		t.Fatal(err)
	}
	for _, c := range vector.Cases {
		if c.Name == "expired" || c.Name == "already-revoked" {
			if c.Status != 409 || c.Code != "STATE_CONFLICT" {
				t.Fatal("expired/revoked must conflict")
			}
		}
	}
	create := doc.Components.Schemas["GrantCreateRequest"].Value
	if create.Properties["valid_until"].Value.Nullable || !contains(create.Required, "valid_until") {
		t.Fatal("withdrawal must not allow open-ended creation")
	}
}

func TestMTR03_D105RetiredStreamVectors(t *testing.T) {
	var vector struct {
		Context struct {
			Retired    string    `json:"retired_incarnation_id"`
			Current    string    `json:"presenting_incarnation_id"`
			RetiredAt  time.Time `json:"retired_at"`
			EnrolledAt time.Time `json:"presenting_enrolled_at"`
			Interval   int       `json:"sampling_interval_seconds"`
		} `json:"context"`
		Cases []struct {
			Name     string          `json:"name"`
			Request  json.RawMessage `json:"request"`
			Response json.RawMessage `json:"response"`
			Audited  bool            `json:"audited"`
		} `json:"cases"`
	}
	readJSON(t, "vectors/usage-retired-handoff.json", &vector)
	if !vector.Context.EnrolledAt.After(vector.Context.RetiredAt) {
		t.Fatal("current incarnation must be enrolled after retirement")
	}
	var accepted []byte
	for _, c := range vector.Cases {
		t.Run(c.Name, func(t *testing.T) {
			for name, raw := range map[string][]byte{"UsageBatchRequest": c.Request, "UsageBatchResult": c.Response} {
				if err := validateComponent(t, name, raw); err != nil {
					t.Fatal(err)
				}
			}
			var req api.UsageBatchRequest
			var ack api.UsageBatchResult
			assertRevision13RoundTrip(t, c.Request, &req)
			assertRevision13RoundTrip(t, c.Response, &ack)
			if req.OriginIncarnationId == nil || req.OriginIncarnationId.String() != vector.Context.Retired || req.NodeIncarnationId.String() != vector.Context.Current || ack.NodeIncarnationId != *req.OriginIncarnationId {
				t.Fatal("ACK must name the retired original stream; request identity stays current")
			}
			var wrapper struct {
				Events []json.RawMessage `json:"events"`
			}
			if err := json.Unmarshal(c.Request, &wrapper); err != nil {
				t.Fatal(err)
			}
			for _, ev := range wrapper.Events {
				if err := checkHash(ev, "payload_sha256"); err != nil {
					t.Fatal(err)
				}
			}
			end := req.Events[0].IntervalEnd
			late := end.After(vector.Context.RetiredAt.Add(time.Duration(vector.Context.Interval) * time.Second))
			if c.Name == "after-retirement-window" {
				if !late || !c.Audited || ack.HighestContiguousSeq.String() != "1233" || ack.Results[0].Status != api.UsageEventResultStatusRejected {
					t.Fatal("late event must be rejected/audited without ACK advance")
				}
			} else if late {
				t.Fatal("accepted/replayed event exceeds the retirement window")
			}
			if req.Events[0].NodeIncarnationId != ack.NodeIncarnationId {
				t.Fatal("event was re-minted under another identity")
			}
			canonical, err := canonjson.Canonicalize(c.Request)
			if err != nil {
				t.Fatal(err)
			}
			if c.Name == "accepted" {
				accepted = canonical
			}
			if c.Name == "idempotent-replay" && (!bytes.Equal(canonical, accepted) || ack.Results[0].Status != api.UsageEventResultStatusDuplicate) {
				t.Fatal("hand-off replay must preserve original bytes and dedup")
			}
		})
	}
	request := loadDoc(t).Components.Schemas["UsageBatchRequest"].Value
	if contains(request.Required, "origin_incarnation_id") || request.Properties["origin_incarnation_id"].Value.Nullable {
		t.Fatal("origin must be optional non-null UUID")
	}
	for _, value := range []string{`null`, `"not-a-uuid"`, `42`} {
		var req map[string]json.RawMessage
		readJSON(t, "examples/valid/UsageBatchRequest.json", &req)
		req["origin_incarnation_id"] = json.RawMessage(value)
		if validateComponent(t, "UsageBatchRequest", mustMarshal(t, req)) == nil {
			t.Fatalf("invalid origin %s accepted", value)
		}
	}
}

func TestISO01_D106TCReadbackBoundsAndDigest(t *testing.T) {
	var vector struct {
		Cases []struct {
			Name      string          `json:"name"`
			Evidence  json.RawMessage `json:"evidence"`
			Canonical string          `json:"canonical_projection"`
		} `json:"cases"`
	}
	readJSON(t, "vectors/tc-readback-evidence.json", &vector)
	for _, c := range vector.Cases {
		if err := validateComponent(t, "TCReadbackEvidence", c.Evidence); err != nil {
			t.Fatal(err)
		}
		var v api.TCReadbackEvidence
		assertRevision13RoundTrip(t, c.Evidence, &v)
		hash, err := canonjson.HashHex([]byte(c.Canonical))
		if err != nil || hash != v.Sha256 {
			t.Fatalf("%s digest: %v", c.Name, err)
		}
		if v.CandidateGeneration.String() != "18446744073709551615" {
			t.Fatal("generation lost precision")
		}
		if c.Name == "verified" && (!v.Verified || !v.Rejected.IsNull()) {
			t.Fatal("clean verified read has no deciding rejection")
		}
		code, codeErr := v.ErrorCode.Get()
		if c.Name == "rejected" && (v.Verified || codeErr != nil || code != api.ErrorCodeTCFOREIGNFILTER) {
			t.Fatal("rejection must carry complete outcome and specific code")
		}
		if c.Name == "truncated" && (!v.Truncated || len(v.Attachments) != 16 || v.ReadsTotal != 17) {
			t.Fatal("truncated read must retain bounded rows and total counts")
		}
	}
	fresh := func() map[string]any {
		var v map[string]any
		readJSON(t, "examples/valid/TCReadbackEvidence.verified.json", &v)
		return v
	}
	attachment := func(v map[string]any) map[string]any { return v["attachments"].([]any)[0].(map[string]any) }
	object := func(v map[string]any) map[string]any { return attachment(v)["objects"].([]any)[0].(map[string]any) }
	for name, mutate := range map[string]func(map[string]any){
		"missing verified":       func(v map[string]any) { delete(v, "verified") },
		"missing rejected":       func(v map[string]any) { delete(v, "rejected") },
		"missing error":          func(v map[string]any) { delete(v, "error_code") },
		"numeric generation":     func(v map[string]any) { v["candidate_generation"] = 1 },
		"leading zero":           func(v map[string]any) { v["candidate_generation"] = "01" },
		"non UTC":                func(v map[string]any) { v["read_at"] = "2026-10-28T15:00:29+00:00" },
		"unknown phase":          func(v map[string]any) { v["phase"] = "rollback" },
		"unknown code":           func(v map[string]any) { v["error_code"] = "TC_UNKNOWN" },
		"negative reads":         func(v map[string]any) { v["reads_total"] = -1 },
		"too many objects total": func(v map[string]any) { v["objects_total"] = 2147483648 },
		"float count":            func(v map[string]any) { v["reads_total"] = 0.5 },
		"uppercase digest":       func(v map[string]any) { v["sha256"] = strings.Repeat("A", 64) },
		"17 attachments": func(v map[string]any) {
			row := attachment(v)
			rows := make([]any, 17)
			for i := range rows {
				rows[i] = row
			}
			v["attachments"] = rows
		},
		"5 objects":         func(v map[string]any) { o := object(v); attachment(v)["objects"] = []any{o, o, o, o, o} },
		"5 actions":         func(v map[string]any) { object(v)["actions"] = []string{"a", "b", "c", "d", "e"} },
		"missing kind":      func(v map[string]any) { delete(object(v), "kind") },
		"unsafe token":      func(v map[string]any) { object(v)["kind"] = "match all" },
		"long token":        func(v map[string]any) { attachment(v)["interface"] = strings.Repeat("a", 33) },
		"raw stderr":        func(v map[string]any) { attachment(v)["error"] = "command failed: details" },
		"action parameters": func(v map[string]any) { object(v)["parameters"] = "forbidden" },
		"raw matches":       func(v map[string]any) { object(v)["matches"] = []any{} },
		"extra root field":  func(v map[string]any) { v["routes"] = []any{} },
		"two deciding objects": func(v map[string]any) {
			row := attachment(v)
			o := object(v)
			row["objects"] = []any{o, o}
			v["rejected"] = row
		},
	} {
		t.Run(name, func(t *testing.T) {
			v := fresh()
			mutate(v)
			if validateComponent(t, "TCReadbackEvidence", mustMarshal(t, v)) == nil {
				t.Fatal("invalid bounded read accepted")
			}
		})
	}
	v := fresh()
	v["candidate_generation"] = "18446744073709551616"
	var model api.TCReadbackEvidence
	if err := json.Unmarshal(mustMarshal(t, v), &model); err == nil {
		t.Fatal("uint64 overflow accepted by Go")
	}
	// Observation linkage and nullable presence must preserve the legacy error rows.
	for _, name := range []string{"verified", "rejected", "truncated", "null"} {
		raw := readFile(t, "examples/valid/ObservationReport.tc-"+name+".json")
		if err := validateComponent(t, "ObservationReport", raw); err != nil {
			t.Fatal(err)
		}
		assertObservationRoundTrip(t, raw)
	}
}
