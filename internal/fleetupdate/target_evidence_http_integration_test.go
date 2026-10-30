//go:build integration

package fleetupdate_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestUPD05_TargetResultEvidenceAdminHTTP(t *testing.T) {
	for _, scenario := range []string{"rolled_back", "rolled_back_observed", "succeeded", "failed", "no_result"} {
		t.Run(scenario, func(t *testing.T) {
			service, batchID, result := fleetupdate.NewTargetEvidenceFixture(t, scenario)
			// Exercise the production handler, router and admin auth boundary;
			// only the session authenticator is replaced by a TEST ONLY identity.
			composite, err := httpapi.Compose(&httpwire.Handlers{Runtime: &httpwire.Runtime{Fleet: service}})
			if err != nil {
				t.Fatal(err)
			}
			c := config.Defaults()
			c.PublicOrigin = "https://portal.example.invalid"
			admin := auth.Principal{Role: auth.Admin, ID: uuid.New(), MFAVerified: true, AuthenticatedAt: time.Now()}
			h, err := httpapi.New(composite, httpapi.Options{Config: c, Sessions: auth.SessionAuthenticatorFunc(func(*http.Request) (auth.Principal, error) { return admin, nil })})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "https://control.example.invalid/api/v1/admin/update-batches/"+batchID.String(), nil)
			response := httptest.NewRecorder()
			h.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatal(response.Code, response.Body.String())
			}
			var batch struct {
				Targets []map[string]json.RawMessage `json:"targets"`
			}
			if err = json.Unmarshal(response.Body.Bytes(), &batch); err != nil || len(batch.Targets) != 1 {
				t.Fatal(response.Body.String(), err)
			}
			want := map[string]any{"previous_release_id": nil, "health": nil, "candidate_health": nil}
			if result != nil {
				if v, err := result.PreviousReleaseId.Get(); err == nil {
					want["previous_release_id"] = v
				}
				if v, err := result.Health.Get(); err == nil {
					want["health"] = v
				}
				if v, err := result.CandidateHealth.Get(); err == nil && result.State == api.UpdateResultStateRolledBack {
					want["candidate_health"] = v
				}
			}
			for field, value := range want {
				raw, ok := batch.Targets[0][field]
				if !ok {
					t.Fatalf("%s omitted instead of explicit value/null: %s", field, response.Body.String())
				}
				var got, expected any
				encoded, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				if err = json.Unmarshal(encoded, &expected); err != nil {
					t.Fatal(err)
				}
				if err = json.Unmarshal(raw, &got); err != nil || !reflect.DeepEqual(got, expected) {
					t.Fatalf("%s = %s, want %s (%v)", field, raw, encoded, err)
				}
			}
			t.Logf("ADMIN_BATCH_%s=%s", scenario, response.Body.String())
		})
	}
}
