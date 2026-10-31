package members

import "testing"

func TestAT02_OperationCompletionRequiresParentSuccess(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
	}{
		{`{"status":"running","steps":[{"status":"succeeded"},{"status":"pending"}]}`, false},
		{`{"status":"failed","steps":[{"status":"succeeded"}]}`, false},
		{`{"status":"succeeded","steps":[{"status":"succeeded"}]}`, true},
		{`{"steps":[{"status":"succeeded"}]}`, false},
		{`invalid`, false},
	} {
		if got := completedOperation([]byte(tc.body)); got != tc.want {
			t.Errorf("body %s: got %t, want %t", tc.body, got, tc.want)
		}
	}
}
