//go:build integration

package httpadapter_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPI01_BinaryStartupHelpers(t *testing.T) {
	t.Run("bound_address", func(t *testing.T) {
		for _, tc := range []struct{ name, log, want string }{
			{"public", `{"msg":"control listener started","listener":"public","address":"127.0.0.1:12345"}`, "127.0.0.1:12345"},
			{"malformed", `{"msg":"control listener started"`, ""},
			{"wrong_event", `{"msg":"effective configuration","listener":"public","address":"127.0.0.1:12345"}`, ""},
			{"wrong_listener", `{"msg":"control listener started","listener":"metrics","address":"127.0.0.1:12345"}`, ""},
			{"nonloopback", `{"msg":"control listener started","listener":"public","address":"192.0.2.1:12345"}`, ""},
			{"zero_port", `{"msg":"control listener started","listener":"public","address":"127.0.0.1:0"}`, ""},
			{"hostname", `{"msg":"control listener started","listener":"public","address":"example.invalid:12345"}`, ""},
		} {
			t.Run(tc.name, func(t *testing.T) {
				if got := controlPublicAddress(tc.log); got != tc.want {
					t.Fatalf("address = %q, want %q", got, tc.want)
				}
			})
		}
	})
	t.Run("exit_is_visible_to_readiness_and_cleanup", func(t *testing.T) {
		result := errors.New("TEST ONLY exit failure")
		exit := &binaryExit{done: make(chan struct{}), err: result}
		close(exit.done)
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := awaitBinaryReady(ctx, &http.Client{}, "", &synchronizedBuffer{}, exit)
		if err == nil || !strings.Contains(err.Error(), "process exited") || ctx.Err() != nil {
			t.Fatalf("exit did not stop readiness before deadline: %v", err)
		}
		select {
		case <-exit.done:
			if exit.err != result {
				t.Fatal("readiness changed the cleanup result")
			}
		default:
			t.Fatal("readiness consumed the cleanup notification")
		}
	})
	for _, status := range []int{http.StatusOK, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
			defer server.Close()
			exit := &binaryExit{done: make(chan struct{})}
			base, err := awaitBinaryReady(ctx, server.Client(), server.URL, &synchronizedBuffer{}, exit)
			if status == http.StatusOK {
				if err != nil || base != server.URL {
					t.Fatalf("ready listener rejected: %q %v", base, err)
				}
			} else if err == nil || base != "" || !strings.Contains(err.Error(), "HTTP 503") {
				t.Fatalf("unready listener accepted or lost status: %q %v", base, err)
			}
		})
	}
	t.Run("diagnostics_are_bounded_and_allowlisted", func(t *testing.T) {
		logs := strings.Repeat(`{"msg":"control domain built","domain":"TEST-ONLY-secret"}`+"\n", 20) +
			`{"msg":"effective configuration","config":{"secret":"TEST-ONLY-secret"}}` + "\n" +
			`{"msg":"TEST-ONLY-secret","error":"TEST-ONLY-secret"}` + "\n" +
			`{"msg":"invalid startup setting","setting":"TEST-ONLY-secret"}` + "\n" +
			`{"msg":"control stopped","error_code":"STARTUP_OR_RUNTIME_FAILURE","error":"TEST-ONLY-secret"}`
		summary := binaryStartupSummary(logs)
		if strings.Contains(summary, "TEST-ONLY-secret") || len(summary) > 1024 || strings.Count(summary, "; ") > 7 || !strings.Contains(summary, "STARTUP_OR_RUNTIME_FAILURE") {
			t.Fatalf("unsafe or incomplete summary: %q", summary)
		}
	})
}
