package members

import (
	"strings"
	"testing"
)

func TestAT09_MemberErrorReasonScrubbed(t *testing.T) {
	got := memberResponseExcerpt([]byte(`{"error":{"code":"UNAUTHENTICATED","message":"session expired","retryable":false,"details":{"credential":"TEST ONLY"}}}`))
	if !strings.Contains(got, "UNAUTHENTICATED") || !strings.Contains(got, "session expired") || strings.Contains(got, "credential") {
		t.Fatal(got)
	}
	got = memberResponseExcerpt([]byte("denied Bearer TEST_ONLY_CREDENTIAL and TEST_ONLY_PASSWORD"), "TEST_ONLY_PASSWORD")
	if strings.Contains(got, "TEST_ONLY") || !strings.Contains(got, "denied") {
		t.Fatal(got)
	}
}
