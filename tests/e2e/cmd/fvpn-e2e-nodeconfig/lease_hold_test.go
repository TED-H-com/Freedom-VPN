package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/leasehold"
	"github.com/google/uuid"
)

func TestAT13_MatchingLeaseTransportHold(t *testing.T) {
	for _, release := range []string{"deadline", "explicit", "restart"} {
		t.Run(release, func(t *testing.T) {
			dir := t.TempDir()
			now := time.Now().UTC()
			f := &matchingLeaseHold{directory: dir, now: func() time.Time { return now }}
			call := func(method, path, generation string, status int) (*http.Response, []byte) {
				t.Helper()
				body, _ := json.Marshal(map[string]any{"signed": map[string]any{"config_generation": generation, "lease_id": uuid.NewString()}, "signatures": []string{"TEST ONLY bytes preserved"}})
				resp := &http.Response{Request: httptest.NewRequest(method, path, nil), StatusCode: status, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{"ETag": {"unchanged"}}, ContentLength: int64(len(body))}
				if err := f.modify(resp); err != nil {
					t.Fatal(err)
				}
				got, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}
				if resp.StatusCode != 503 && !bytes.Equal(body, got) {
					t.Fatal("upstream bytes changed")
				}
				return resp, body
			}
			arm := func() string {
				t.Helper()
				id := uuid.NewString()
				if err := os.Mkdir(filepath.Join(dir, "active"), 0700); err != nil {
					t.Fatal(err)
				}
				raw, _ := json.Marshal(leasehold.Arm{ID: id, BaseGeneration: "5", HoldSeconds: 45, ArmedAt: now, ExpiresAt: now.Add(3 * time.Minute)})
				if err := os.WriteFile(filepath.Join(dir, "active/arm.json"), raw, 0600); err != nil {
					t.Fatal(err)
				}
				return id
			}
			const desired = "/api/v1/agent/desired-state"
			const renewal = "/api/v1/agent/lease-renewals"
			if r, _ := call("POST", renewal, "6", 200); r.StatusCode != 200 {
				t.Fatal("default on")
			}
			id := arm()
			if r, _ := call("POST", renewal, "6", 200); r.StatusCode != 200 {
				t.Fatal("held before desired change")
			}
			call("GET", desired, "6", 200)
			if r, _ := call("POST", renewal, "5", 200); r.StatusCode != 200 {
				t.Fatal("old lease withheld")
			}
			call("POST", "/api/v1/agent/observations", "6", 202)
			for range 3 {
				r, _ := call("POST", renewal, "6", 200)
				if r.StatusCode != 503 || r.Header.Get("Cache-Control") != "no-store" {
					t.Fatal("matching lease leaked")
				}
				now = now.Add(10 * time.Second)
				call("GET", desired, "6", 200)
			}
			switch release {
			case "deadline":
				now = now.Add(16 * time.Second)
			case "explicit":
				if err := os.WriteFile(filepath.Join(dir, "active/release"), []byte(id), 0600); err != nil {
					t.Fatal(err)
				}
			case "restart":
				f = &matchingLeaseHold{directory: dir, now: func() time.Time { return now }}
			}
			if r, _ := call("POST", renewal, "6", 200); r.StatusCode != 200 {
				t.Fatal("release failed")
			}
			if r, _ := call("POST", renewal, "6", 200); r.StatusCode != 200 {
				t.Fatal("implicitly rearmed")
			}
			raw, err := os.ReadFile(filepath.Join(dir, id+".json"))
			var ev leasehold.Evidence
			if err != nil || json.Unmarshal(raw, &ev) != nil || ev.ReleasedAt == nil {
				t.Fatal("release receipt absent")
			}
			withheld := 0
			for _, e := range ev.Events {
				if e.Kind == "withheld" {
					withheld++
				}
			}
			if withheld != 3 || bytes.Contains(raw, []byte("signatures")) || bytes.Contains(raw, []byte("TEST ONLY bytes")) {
				t.Fatal("bad content-free receipts")
			}
			arm()
			call("GET", desired, "6", 200)
			if r, _ := call("POST", renewal, "6", 200); r.StatusCode != 503 {
				t.Fatal("distinct arm failed")
			}
		})
	}
}

func TestAT13_LeaseHoldExpiresWithoutDesiredAndSupersedes(t *testing.T) {
	for _, mode := range []string{"expired", "superseded", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			now := time.Now().UTC()
			if err := os.Mkdir(filepath.Join(dir, "active"), 0700); err != nil {
				t.Fatal(err)
			}
			arm := leasehold.Arm{ID: uuid.NewString(), BaseGeneration: "5", HoldSeconds: 45, ArmedAt: now.Add(-4 * time.Minute), ExpiresAt: now.Add(-time.Minute)}
			if mode != "expired" {
				arm.ArmedAt = now
				arm.ExpiresAt = now.Add(time.Minute)
			}
			if mode == "invalid" {
				arm.BaseGeneration = "05"
			}
			raw, _ := json.Marshal(arm)
			if err := os.WriteFile(filepath.Join(dir, "active/arm.json"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			f := &matchingLeaseHold{directory: dir, now: func() time.Time { return now }}
			resp := &http.Response{Request: httptest.NewRequest("GET", "/api/v1/agent/desired-state", nil), StatusCode: 304}
			err := f.modify(resp)
			if mode == "invalid" {
				if err == nil {
					t.Fatal("accepted invalid arm")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if mode == "expired" {
				if f.evidence.ReleasedAt == nil {
					t.Fatal("arm not expired")
				}
				return
			}
			f.evidence.TargetGeneration = "6"
			resp.Request = httptest.NewRequest("POST", "/api/v1/agent/lease-renewals", nil)
			resp.StatusCode = 200
			resp.Body = io.NopCloser(bytes.NewBufferString(`{"signed":{"config_generation":"7"}}`))
			if err = f.modify(resp); err != nil || f.evidence.ReleasedAt == nil || resp.StatusCode != 200 {
				t.Fatal("superseded fixture not released", err)
			}
		})
	}
}

func TestD102_SurvivorHoldContinuation(t *testing.T) {
	survivor := uuid.NewString()
	sibling := uuid.NewString()
	other := uuid.NewString()
	const desired = "/api/v1/agent/desired-state"
	const renewal = "/api/v1/agent/lease-renewals"

	newHarness := func(t *testing.T, arm leasehold.Arm, now *time.Time) (*matchingLeaseHold, func(method, path, generation string, members []string) (*http.Response, string, []byte)) {
		t.Helper()
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "active"), 0700); err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(arm)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "active/arm.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		f := &matchingLeaseHold{directory: dir, now: func() time.Time { return *now }}
		call := func(method, path, generation string, members []string) (*http.Response, string, []byte) {
			t.Helper()
			leaseID := uuid.NewString()
			memberObjs := make([]map[string]string, 0, len(members))
			for _, m := range members {
				memberObjs = append(memberObjs, map[string]string{"member_id": m})
			}
			body, _ := json.Marshal(map[string]any{
				"signed": map[string]any{
					"config_generation": generation,
					"lease_id":          leaseID,
					"members":           memberObjs,
				},
				"signatures": []string{"TEST ONLY bytes preserved"},
			})
			resp := &http.Response{
				Request:       httptest.NewRequest(method, path, nil),
				StatusCode:    200,
				Body:          io.NopCloser(bytes.NewReader(body)),
				Header:        http.Header{"ETag": {"unchanged"}},
				ContentLength: int64(len(body)),
			}
			if err := f.modify(resp); err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 503 && !bytes.Equal(body, got) {
				t.Fatal("upstream bytes modified")
			}
			return resp, leaseID, body
		}
		return f, call
	}

	t.Run("continues_one_generation_and_recovers_on_release", func(t *testing.T) {
		now := time.Now().UTC()
		arm := leasehold.Arm{
			ID:                      uuid.NewString(),
			BaseGeneration:          "9",
			HoldSeconds:             90,
			SurvivorMemberID:        survivor,
			MaxContinuedGenerations: 1,
			ArmedAt:                 now,
			ExpiresAt:               now.Add(3 * time.Minute),
		}
		f, call := newHarness(t, arm, &now)

		// Initial transition to generation 10 with both members.
		call("GET", desired, "10", nil)
		for range 3 {
			r, _, _ := call("POST", renewal, "10", []string{survivor, sibling})
			if r.StatusCode != 503 {
				t.Fatalf("expected 503 at initial target generation 10, got %d", r.StatusCode)
			}
			now = now.Add(5 * time.Second)
		}

		// Sibling suspension advances desired state to generation 12 with survivor only.
		call("GET", desired, "12", nil)
		for range 2 {
			r, _, _ := call("POST", renewal, "12", []string{survivor})
			if r.StatusCode != 503 {
				t.Fatalf("expected 503 during survivor continuation at generation 12, got %d", r.StatusCode)
			}
			now = now.Add(5 * time.Second)
		}
		if f.evidence.ContinuedGeneration != "12" || f.evidence.ContinuedAt == nil || f.evidence.ReleasedAt != nil {
			t.Fatalf("unexpected continuation state: %+v", f.evidence)
		}

		// Explicit release forwards the next genuine generation-12 survivor lease unchanged.
		if err := os.WriteFile(filepath.Join(f.directory, "active/release"), []byte(arm.ID), 0600); err != nil {
			t.Fatal(err)
		}
		r, forwardedID, _ := call("POST", renewal, "12", []string{survivor})
		if r.StatusCode != 200 || f.evidence.ReleasedAt == nil {
			t.Fatalf("expected 200 and release after explicit release, got status=%d released=%v", r.StatusCode, f.evidence.ReleasedAt)
		}
		matched := false
		continuedStarted := false
		for _, ev := range f.evidence.Events {
			if ev.Kind == "continuation_started" && ev.Generation == "12" {
				continuedStarted = true
			}
			if ev.Kind == "matching_lease_forwarded" && ev.Generation == "12" && ev.LeaseID == forwardedID {
				matched = true
			}
		}
		if !continuedStarted || !matched {
			t.Fatalf("expected continuation_started and matching_lease_forwarded at generation 12, events=%+v", f.evidence.Events)
		}
	})

	t.Run("second_generation_advance_releases_generation_superseded", func(t *testing.T) {
		now := time.Now().UTC()
		arm := leasehold.Arm{
			ID:                      uuid.NewString(),
			BaseGeneration:          "9",
			HoldSeconds:             90,
			SurvivorMemberID:        survivor,
			MaxContinuedGenerations: 1,
			ArmedAt:                 now,
			ExpiresAt:               now.Add(3 * time.Minute),
		}
		f, call := newHarness(t, arm, &now)

		call("GET", desired, "10", nil)
		if r, _, _ := call("POST", renewal, "10", []string{survivor, sibling}); r.StatusCode != 503 {
			t.Fatal("initial hold did not start")
		}
		now = now.Add(5 * time.Second)
		call("GET", desired, "12", nil)
		if r, _, _ := call("POST", renewal, "12", []string{survivor}); r.StatusCode != 503 {
			t.Fatal("first continuation did not hold")
		}
		now = now.Add(5 * time.Second)

		// A second generation advance beyond ContinuedGeneration exceeds the 1-generation bound.
		call("GET", desired, "13", nil)
		r, _, _ := call("POST", renewal, "13", []string{survivor})
		if r.StatusCode != 200 || f.evidence.ReleasedAt == nil {
			t.Fatalf("expected release on second generation advance, got status=%d released=%v", r.StatusCode, f.evidence.ReleasedAt)
		}
		last := f.evidence.Events[len(f.evidence.Events)-1]
		if last.Kind != "released" || last.Reason != "generation_superseded" {
			t.Fatalf("expected released(generation_superseded), got %+v", last)
		}
	})

	t.Run("survivor_mismatch_releases_and_forwards_unchanged", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			members []string
		}{
			{"empty", nil},
			{"both_still_present", []string{survivor, sibling}},
			{"wrong_member", []string{other}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				now := time.Now().UTC()
				arm := leasehold.Arm{
					ID:                      uuid.NewString(),
					BaseGeneration:          "9",
					HoldSeconds:             90,
					SurvivorMemberID:        survivor,
					MaxContinuedGenerations: 1,
					ArmedAt:                 now,
					ExpiresAt:               now.Add(3 * time.Minute),
				}
				f, call := newHarness(t, arm, &now)
				call("GET", desired, "10", nil)
				if r, _, _ := call("POST", renewal, "10", []string{survivor, sibling}); r.StatusCode != 503 {
					t.Fatal("initial hold did not start")
				}
				now = now.Add(5 * time.Second)
				call("GET", desired, "12", nil)
				r, _, _ := call("POST", renewal, "12", tc.members)
				if r.StatusCode != 200 || f.evidence.ReleasedAt == nil {
					t.Fatalf("expected release and 200 on survivor mismatch, got status=%d released=%v", r.StatusCode, f.evidence.ReleasedAt)
				}
				last := f.evidence.Events[len(f.evidence.Events)-1]
				if last.Kind != "released" || last.Reason != "survivor_mismatch" {
					t.Fatalf("expected released(survivor_mismatch), got %+v", last)
				}
			})
		}
	})

	t.Run("rejects_invalid_continuation_bounds", func(t *testing.T) {
		for _, arm := range []leasehold.Arm{
			{ID: uuid.NewString(), BaseGeneration: "9", HoldSeconds: 45, SurvivorMemberID: survivor, MaxContinuedGenerations: 2},
			{ID: uuid.NewString(), BaseGeneration: "9", HoldSeconds: 45, SurvivorMemberID: survivor, MaxContinuedGenerations: 0},
			{ID: uuid.NewString(), BaseGeneration: "9", HoldSeconds: 45, SurvivorMemberID: "not-a-uuid", MaxContinuedGenerations: 1},
			{ID: uuid.NewString(), BaseGeneration: "9", HoldSeconds: 45, SurvivorMemberID: "", MaxContinuedGenerations: 1},
		} {
			dir := t.TempDir()
			now := time.Now().UTC()
			arm.ArmedAt = now
			arm.ExpiresAt = now.Add(time.Minute)
			if err := os.Mkdir(filepath.Join(dir, "active"), 0700); err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(arm)
			if err := os.WriteFile(filepath.Join(dir, "active/arm.json"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			f := &matchingLeaseHold{directory: dir, now: func() time.Time { return now }}
			resp := &http.Response{Request: httptest.NewRequest("GET", desired, nil), StatusCode: 304}
			if err := f.modify(resp); err == nil {
				t.Fatalf("expected invalid continuation arm %+v to be rejected", arm)
			}
		}
	})
}
