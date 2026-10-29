package worker

import "testing"

func TestBILL01_SynchronousCheckoutHasNoSessionJob(t *testing.T) {
	jobs := MemberJobs()
	if _, registered := jobs["billing.session_requested"]; registered {
		t.Fatal("D-031/D-038: legacy checkout job is still registered")
	}
	if jobs["billing.inbox_received"] == nil {
		t.Fatal("payment inbox dispatch must remain registered")
	}
}
