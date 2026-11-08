package agent

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestSEC02_ApplyFailureLogDoesNotInventVerification(t *testing.T) {
	for _, cause := range []error{context.DeadlineExceeded, context.Canceled, errors.New("TEST ONLY private error")} {
		wrapped := &enforce.CmdError{Cmd: "TEST ONLY private argv", Out: "TEST ONLY private stderr", Err: cause}
		m := enforce.Model{Datapath: enforce.Datapath{MemberIngress: "fvmb0", Uplink: "fvup0", SNAT: netip.MustParseAddr("192.0.2.1")}}
		_, err := enforce.Apply(t.Context(), tcFailureLogExecutor{wrapped}, m, nil)
		if d, ok := enforce.ReadApplyFailure(err); !ok || d.Stage != "load_nft" {
			t.Fatalf("wrong fixture boundary: %+v %v", d, err)
		}
		var buf bytes.Buffer
		e := &Engine{clock: timeanchor.NewManual(time.Now().UTC(), time.Hour), log: slog.New(RedactHandler{Next: slog.NewTextHandler(&buf, nil)})}
		e.noteApply(err)
		want := "apply failed"
		class := "operation_error"
		if cause == context.DeadlineExceeded {
			want = "apply timed out"
			class = "deadline"
		} else if cause == context.Canceled {
			want = "apply canceled"
			class = "canceled"
		}
		if e.applyError.Message != want || e.applyError.Code != api.ErrorCodeHEALTHCHECKFAILED || e.applyError.Stage != "apply" {
			t.Fatalf("wrong error: %+v", e.applyError)
		}
		line := strings.TrimSpace(buf.String()) + " "
		for _, field := range []string{"apply_failure_stage=load_nft", "apply_failure=" + class, "apply_context_state_at_return=none"} {
			if !strings.Contains(line, " "+field+" ") {
				t.Fatalf("lost %s: %s", field, line)
			}
		}
		if strings.Contains(line, "private") || strings.Contains(line, "verification") || strings.Contains(line, "apply_exit_code") {
			t.Fatalf("unsafe or invented diagnosis: %s", line)
		}
		buf.Reset()
		e.noteApply(cause)
		if strings.Contains(buf.String(), "apply_failure") || strings.Contains(buf.String(), "private") {
			t.Fatalf("invented diagnosis: %s", &buf)
		}
	}
}
