package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

type failingSelf struct{ err error }

func (h failingSelf) GetMemberSelf(context.Context, gen.GetMemberSelfRequestObject) (gen.GetMemberSelfResponseObject, error) {
	return nil, h.err
}

func TestAPI01_ReportOnlyHandlerErrorsBecomeInternalErrors(t *testing.T) {
	for _, code := range []apierr.Code{apierr.ArtifactInvalid, apierr.IsolationBaselineMissing, apierr.MeterEpochChange} {
		t.Run(string(code), func(t *testing.T) {
			const secret = "TEST ONLY private handler details"
			var logs bytes.Buffer
			p := principal(auth.Member)
			err := fmt.Errorf("%s: %w", secret, apierr.New(code, secret).WithDetail("private", secret))
			h := fixture(t, p, false, slog.New(slog.NewJSONHandler(&logs, nil)), failingSelf{err: err})
			w := assertStatus(t, h, request("GET", "/api/v1/member/me", "", p), 500)
			var envelope apierr.Envelope
			if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Error.Code != apierr.InternalError || envelope.Error.Retryable || len(envelope.Error.Details) != 0 || envelope.Error.RequestID != w.Header().Get("X-Request-ID") || strings.Contains(w.Body.String(), string(code)) || strings.Contains(w.Body.String(), secret) {
				t.Fatal("report-only error escaped the HTTP boundary", w.Body.String())
			}
			anomalies := 0
			for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
				var record map[string]any
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				if record["msg"] == "report-only error rendered as INTERNAL_ERROR" {
					anomalies++
					if record["level"] != "ERROR" || record["request_id"] != envelope.Error.RequestID || record["reported_code"] != string(code) || len(record) != 5 {
						t.Fatal("missing correlation or unsafe anomaly attributes", record)
					}
				}
			}
			if anomalies != 1 || strings.Contains(logs.String(), secret) {
				t.Fatal("missing or unsafe report-only anomaly log", logs.String())
			}
		})
	}
}

func TestAPI01_NotImplementedOnlyForCompositionFallback(t *testing.T) {
	p := principal(auth.Member)
	for _, tc := range []struct {
		name   string
		sets   []any
		status int
		code   apierr.Code
	}{
		{"unregistered", nil, 501, apierr.NotImplemented},
		{"registered failure", []any{failingSelf{err: errors.New("TEST ONLY internal failure")}}, 500, apierr.InternalError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := assertStatus(t, fixture(t, p, false, nil, tc.sets...), request("GET", "/api/v1/member/me", "", p), tc.status)
			var envelope apierr.Envelope
			if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Error.Code != tc.code || envelope.Error.Retryable {
				t.Fatal("wrong composition error", envelope)
			}
		})
	}
}
