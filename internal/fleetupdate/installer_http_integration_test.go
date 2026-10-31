//go:build integration

package fleetupdate_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestD089_InstallerAdminNodeHTTP(t *testing.T) {
	service, result := fleetupdate.NewInstallerNodeViewFixture(t)
	composite, err := httpapi.Compose(&httpwire.Handlers{Runtime: &httpwire.Runtime{Nodes: service}})
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
	for _, path := range []string{"/api/v1/admin/nodes/" + result.NodeId.String(), "/api/v1/admin/nodes"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "https://control.example.invalid"+path, nil))
			if response.Code != http.StatusOK {
				t.Fatal(response.Code, response.Body.String())
			}
			var detail api.NodeDetail
			if path == "/api/v1/admin/nodes" {
				var list api.NodeList
				if err = json.Unmarshal(response.Body.Bytes(), &list); err != nil || len(list.Items) != 1 {
					t.Fatal(response.Body.String(), err)
				}
				detail.Update = list.Items[0].Update
				detail.Releases = list.Items[0].Releases
			} else if err = json.Unmarshal(response.Body.Bytes(), &detail); err != nil {
				t.Fatal(err)
			}
			summary, e := detail.Update.Get()
			if e != nil || summary.Source == nil || *summary.Source != api.UpdateResultSourceInstaller || summary.JobId != result.JobId || !summary.BatchId.IsNull() {
				t.Fatal(response.Body.String(), e)
			}
			current, e := detail.Releases.CurrentReleaseId.Get()
			if e != nil || current != result.RequestedReleaseId {
				t.Fatal(response.Body.String(), e)
			}
		})
	}
}
