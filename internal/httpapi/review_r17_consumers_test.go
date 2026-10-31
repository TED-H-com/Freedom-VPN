package httpapi

import (
	"os"
	"strings"
	"testing"
)

func readRepo(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestReviewR17_Revision14CodesHaveConsumers(t *testing.T) {
	portal := readRepo(t, "../../apps/portal/src/api/errors.ts")
	for _, code := range []string{"STATE_CONFLICT", "ACK_CONFLICT", "USAGE_EVENT_CONFLICT", "LEDGER_EPOCH_MISMATCH", "INCARNATION_MISMATCH", "TC_FOREIGN_FILTER", "VALIDATION_FAILED"} {
		if !strings.Contains(portal, code+":") {
			t.Fatalf("TODO(review): portal error map has no entry for %s", code)
		}
	}
	if strings.Contains(portal, "job_id_mismatch") || strings.Contains(portal, "origin_ineligible") {
		t.Fatal("portal treats a detail reason as an ErrorCode")
	}
	api := readRepo(t, "../../pkg/api/api.gen.go")
	if strings.Contains(api, "JobIdMismatch") || strings.Contains(api, "JOB_ID_MISMATCH") {
		t.Fatal("job_id_mismatch became an ErrorCode")
	}
	worker := readRepo(t, "../../apps/worker/jobs_members.go")
	if !strings.Contains(worker, "member.grant_revoked") {
		t.Fatal("TODO(review): worker batch does not handle member.grant_revoked")
	}
	command := readRepo(t, "../../internal/entitlement/command.go")
	if !strings.Contains(command, "member.grant_revoke") {
		t.Fatal("TODO(review): entitlement command does not handle member.grant_revoke")
	}
	updater := readRepo(t, "../../node/updater/result.go")
	if !strings.Contains(updater, `ack.Reason == "job_id_mismatch"`) {
		t.Fatal("TODO(review): updater does not treat job_id_mismatch as terminal")
	}
}
