//go:build e2e

package enrollment

import "testing"

func TestAT06_ReadClassMatchesControl252(t *testing.T) {
	for _, tc := range []struct{ method, path, want string }{
		{"GET", "/api/v1/admin/audit-events?limit=100", "read"},
		{"GET", "/api/v1/admin/nodes", "read"},
		{"GET", "/api/v1/admin/nodes/test-id/incarnations", "read"},
		{"GET", "/api/v1/admin/members", "read"},
		{"GET", "/api/v1/admin/devices", "read"},
		{"GET", "/api/v1/admin/nodes/test-id", "browser"},
		{"GET", "/api/v1/admin/operations/test-id", "browser"},
		{"POST", "/api/v1/admin/enrollments", "browser"},
		{"POST", "/api/v1/agent/enroll", "anonymous"},
	} {
		if got, _ := classifyLimited(tc.method, tc.path); got != tc.want {
			t.Errorf("%s %s: got %s, want %s", tc.method, tc.path, got, tc.want)
		}
	}
}
