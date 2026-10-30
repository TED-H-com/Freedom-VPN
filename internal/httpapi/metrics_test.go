package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

func TestSEC02_RequestMetricsUseBoundedRoutesAndRecoveredStatus(t *testing.T) {
	registry := prometheus.NewRegistry()
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "fvpn_test_requests_total", Help: "TEST ONLY responses."}, []string{"route", "status"})
	registry.MustRegister(requests)
	handlers, err := Compose()
	if err != nil {
		t.Fatal(err)
	}
	api, err := New(handlers, Options{Config: config.Defaults()})
	if err != nil {
		t.Fatal(err)
	}
	handler := TransportWithMetrics(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/panic" {
			panic("TEST ONLY secret panic")
		}
		api.ServeHTTP(w, r)
	}), requests)
	id := uuid.NewString()
	want := map[string]string{}
	for _, tc := range []struct {
		path, route string
		status      int
	}{
		{"/api/v1/admin/operations/" + id + "?token=TEST-ONLY-SECRET", "GET /api/v1/admin/operations/{operation_id}", 401},
		{"/unknown/" + id + "?token=TEST-ONLY-SECRET", "unmatched", 404},
		{"/panic", "unmatched", 500},
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		if w.Code != tc.status {
			t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
		}
		want[strconv.Itoa(tc.status)] = tc.route
	}
	families, err := registry.Gather()
	if err != nil || len(families) != 1 || len(families[0].Metric) != len(want) {
		t.Fatal("unexpected request metrics", families, err)
	}
	for _, metric := range families[0].Metric {
		labels := map[string]string{}
		for _, label := range metric.Label {
			labels[label.GetName()] = label.GetValue()
		}
		if len(labels) != 2 || labels["route"] != want[labels["status"]] || metric.Counter.GetValue() != 1 {
			t.Fatal("unbounded route or incorrect counter", metric)
		}
	}
}
