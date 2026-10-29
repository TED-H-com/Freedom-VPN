package operations

import (
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"testing"
)

func TestAPI01_OperationStateDerivedFromSteps(t *testing.T) {
	for _, tc := range []struct {
		steps []api.OperationStepStatus
		want  api.OperationStatus
	}{{[]api.OperationStepStatus{"pending"}, "pending"}, {[]api.OperationStepStatus{"running", "pending"}, "running"}, {[]api.OperationStepStatus{"succeeded", "pending"}, "running"}, {[]api.OperationStepStatus{"failed", "pending"}, "running"}, {[]api.OperationStepStatus{"succeeded", "failed"}, "partially_succeeded"}, {[]api.OperationStepStatus{"failed"}, "failed"}, {[]api.OperationStepStatus{"cancelled"}, "superseded"}, {[]api.OperationStepStatus{"succeeded"}, "succeeded"}} {
		steps := map[string]Step{"optional": {Status: "not_required", Reason: "explicit"}}
		for i, s := range tc.steps {
			steps[string(rune('a'+i))] = Step{Required: true, Status: s}
		}
		if got := derive(steps); got != tc.want {
			t.Fatalf("%v: got %s want %s", tc.steps, got, tc.want)
		}
	}
}
