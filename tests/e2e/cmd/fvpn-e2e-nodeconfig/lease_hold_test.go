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
