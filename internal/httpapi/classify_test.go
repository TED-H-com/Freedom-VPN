package httpapi

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/getkin/kin-openapi/routers/legacy"
)

// middlewareOperationIDs are the operation-id literals classify and the
// bootstrap body check compare against. A literal missing from gen.GetSpec
// is dead and never selects its branch.
var middlewareOperationIDs = []string{
	"SubmitObservation",
	"RenewServingLease",
	"SubmitUsageBatch",
	"EnrollNode",
	"ReceiveProviderWebhook",
	"ConsumeAgentBootstrap",
	"ListAuditEvents",
	"ListNodes",
	"ListNodeIncarnations",
	"ListMembers",
	"ListAdminDevices",
}

func TestClassifyOperationIDsMatchLoadedSpec(t *testing.T) {
	doc, err := gen.GetSpec()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]struct{}{}
	for _, item := range doc.Paths.Map() {
		if item == nil {
			continue
		}
		for _, op := range item.Operations() {
			if op == nil || op.OperationID == "" {
				t.Fatal("loaded operation has an empty OperationID")
			}
			ids[op.OperationID] = struct{}{}
		}
	}
	var loaded []string
	for id := range ids {
		loaded = append(loaded, id)
	}
	slices.Sort(loaded)
	for _, id := range middlewareOperationIDs {
		if _, ok := ids[id]; ok {
			continue
		}
		var folded []string
		for _, got := range loaded {
			if strings.EqualFold(got, id) {
				folded = append(folded, got)
			}
		}
		t.Errorf("middleware operation id %q is not in the loaded spec; equal-fold matches %q", id, folded)
	}
	if t.Failed() {
		t.Logf("loaded OperationIDs: %s", strings.Join(loaded, ", "))
	}

	doc.Servers = nil
	router, err := legacy.NewRouter(doc)
	if err != nil {
		t.Fatal(err)
	}
	c := config.Defaults()
	cases := []struct {
		method, path, class string
		bucket              config.Bucket
		body                int64
	}{
		{http.MethodPost, "/api/v1/agent/enroll", "anonymous", c.AnonymousRate, c.AgentBodyBytes},
		{http.MethodPost, "/api/v1/webhooks/example", "anonymous", c.AnonymousRate, c.WebhookBodyBytes},
		{http.MethodPost, "/api/v1/agent/observations", "heartbeat", c.HeartbeatRate, c.AgentBodyBytes},
		{http.MethodPost, "/api/v1/agent/lease-renewals", "lease", c.LeaseRate, c.AgentBodyBytes},
		{http.MethodPost, "/api/v1/agent/usage-batches", "usage", c.UsageRate, c.UsageBodyBytes},
		{http.MethodGet, "/api/v1/agent/desired-state", "agent", c.AgentRate, c.AgentBodyBytes},
		{http.MethodGet, "/api/v1/admin/nodes", "read", c.ReadRate, c.BrowserBodyBytes},
		{http.MethodGet, "/api/v1/admin/members", "read", c.ReadRate, c.BrowserBodyBytes},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		route, _, err := findRoute(router, req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		class, bucket, body := classify(route.Operation, c)
		if class != tc.class || bucket != tc.bucket || body != tc.body {
			t.Errorf("%s %s: OperationID=%q got class=%s bucket=%+v body=%d, want class=%s bucket=%+v body=%d",
				tc.method, tc.path, route.Operation.OperationID, class, bucket, body, tc.class, tc.bucket, tc.body)
		}
	}
}
