package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

type tcFailureLogExecutor struct{ err error }

func (tcFailureLogExecutor) NftBatch(context.Context, string) error { return nil }
func (tcFailureLogExecutor) TCBatch(context.Context, string) error  { return nil }
func (tcFailureLogExecutor) LookPath(name string) (string, error)   { return name, nil }
func (x tcFailureLogExecutor) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, x.err
}

func TestSEC02_TCReadLogProcess(t *testing.T) {
	if os.Getenv("FVPN_TC_LOG_HELPER") == "exit" {
		os.Exit(7)
	}
}

func TestSEC02_TCReadFailureLogKeepsErrorContract(t *testing.T) {
	t.Setenv("FVPN_TC_LOG_HELPER", "exit")
	exit := exec.Command(os.Args[0], "-test.run=^TestSEC02_TCReadLogProcess$").Run()
	if exit == nil {
		t.Fatal("expected real command exit")
	}
	for _, cause := range []error{context.DeadlineExceeded, context.Canceled, exit, errors.New("TEST ONLY private error")} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		wrapped := &enforce.CmdError{Cmd: "TEST ONLY private argv", Out: "TEST ONLY private stderr", Err: cause}
		_, err := enforce.VerifyServing(ctx, tcFailureLogExecutor{wrapped}, enforce.Model{Datapath: enforce.Datapath{MemberIngress: "fvmb0", Uplink: "fvup0"}}, nil)
		diagnostic, ok := enforce.ReadTCFailure(err)
		if !ok {
			t.Fatalf("missing original diagnostic: %v", err)
		}
		var buf bytes.Buffer
		e := &Engine{clock: timeanchor.NewManual(time.Now().UTC(), time.Hour), log: slog.New(RedactHandler{Next: slog.NewJSONHandler(&buf, nil)})}
		e.noteApply(errors.Join(err, context.DeadlineExceeded))
		if e.applyError.Code != api.ErrorCodeTCFOREIGNFILTER || e.applyError.Message != "TC_FOREIGN_FILTER" || len(e.decision.Errors) != 1 {
			t.Fatalf("changed API verdict: %+v", e.applyError)
		}
		var row map[string]any
		if parseErr := json.Unmarshal(buf.Bytes(), &row); parseErr != nil {
			t.Fatal(parseErr)
		}
		if row["msg"] != "apply" || row["err"] != "TC_FOREIGN_FILTER" || row["tc_read_failure"] != diagnostic.Classification || row["tc_read_context_state_at_return"] != "canceled" || row["tc_read_interface"] != "fvmb0" || row["tc_read_attachment"] != "qdisc" {
			t.Fatalf("log mapping: %s", buf.Bytes())
		}
		if cause == exit {
			if row["tc_read_exit_code"] != float64(7) {
				t.Fatalf("missing observed exit code: %s", buf.Bytes())
			}
		} else if _, ok := row["tc_read_exit_code"]; ok {
			t.Fatalf("invented exit: %s", buf.Bytes())
		}
		if strings.Contains(buf.String(), "private") {
			t.Fatalf("raw command error leaked: %s", buf.Bytes())
		}
		buf.Reset()
		e.noteApply(errors.New("TEST ONLY private generic error"))
		if strings.Contains(buf.String(), "tc_read_") || strings.Contains(buf.String(), "private") || e.applyError.Message != "apply failed" {
			t.Fatalf("invented diagnostics: %s", buf.Bytes())
		}
	}
}
