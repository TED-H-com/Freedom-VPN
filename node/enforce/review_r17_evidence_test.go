package enforce

import (
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func reviewTCReads() (ServingEvidence, TCReadEvidence) {
	ev := ServingEvidence{}
	for i := 0; i < 20; i++ {
		ev.TC = append(ev.TC, TCReadEvidence{
			Interface: "idle", Attachment: "clsact",
			Objects: []TCObject{{Kind: "fq_codel", Handle: "10:", Actions: []string{"drop"}}},
		})
	}
	deciding := TCReadEvidence{
		Interface: "evil iface", Attachment: "egress",
		Objects: []TCObject{{
			Kind: "u32", Handle: "1:",
			Actions: []string{"sk_live_secret", "mirred", "police", "gact", "fifth"},
		}},
	}
	ev.TC = append(ev.TC, deciding)
	return ev, deciding
}

func TestReviewR17_DecidingTCReadSurvivesTheCap(t *testing.T) {
	ev, deciding := reviewTCReads()
	obj := deciding.Objects[0]
	err := tcCheck(deciding, &obj)
	ev.retainTC(7, "verify", err)
	got := ev.TCReadbackEvidence()
	if got == nil || len(got.Attachments) == 0 || len(got.Attachments) > 16 {
		t.Fatalf("attachments %+v", got)
	}
	if got.Attachments[0].Interface != "redacted" {
		t.Fatalf("interface leaked or deciding row moved: %+v", got.Attachments[0])
	}
	if len(got.Attachments[0].Objects) != 1 || got.Attachments[0].Objects[0].Kind != "u32" {
		t.Fatalf("deciding object was truncated out: %+v", got.Attachments[0].Objects)
	}
	actions := got.Attachments[0].Objects[0].Actions
	if actions == nil || len(*actions) != 4 || (*actions)[0] != "redacted" {
		t.Fatalf("actions %+v", actions)
	}
	code, err := got.ErrorCode.Get()
	if err != nil || code != api.ErrorCodeTCFOREIGNFILTER {
		t.Fatalf("error code %v %v", code, err)
	}
	if !got.Truncated || got.Sha256 == "" {
		t.Fatalf("digest %+v", got)
	}
	for _, text := range []string{"sk_live_secret", "evil iface", "fifth"} {
		if strings.Contains(got.Sha256, text) {
			t.Fatal(text)
		}
	}
	raw := mustEvidenceJSON(t, *got)
	for _, text := range []string{"sk_live_secret", "evil iface", "fifth"} {
		if strings.Contains(raw, text) {
			t.Fatalf("redaction leaked %s in %s", text, raw)
		}
	}

	other, deciding2 := reviewTCReads()
	obj2 := deciding2.Objects[0]
	other.retainTC(8, "verify", tcCheck(deciding2, &obj2))
	if other.TCSummary.SHA256 != ev.TCSummary.SHA256 {
		t.Fatalf("generation changed the canonical digest: %s %s", ev.TCSummary.SHA256, other.TCSummary.SHA256)
	}
	if other.TCSummary.Generation.String() == ev.TCSummary.Generation.String() {
		t.Fatal("generation was not retained beside the digest")
	}
}

func mustEvidenceJSON(t *testing.T, v api.TCReadbackEvidence) string {
	t.Helper()
	return tcJSON(v)
}
