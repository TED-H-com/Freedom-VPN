package agent

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestReviewD061_CredentialAbsentFromLogsObservationAndAttestation(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	const secret = "pw-d061-do-not-leak"
	path := t.TempDir() + "/creds.json"
	if err := os.WriteFile(path, []byte(`{"username":"e2e","password":"`+secret+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r.eng.cfg.DNS = &DNSConfig{
		Resolvers:               []DNSResolver{{ID: "lab", AdminURL: "http://127.0.0.1:8080", DNSAddr: "127.0.0.1:53"}},
		ResolverCredentialsFile: path,
	}
	r.eng.reloadResolvers()
	body := []byte(exampleResolverBody)
	ref := exampleRef(body)
	r.srv.PutResolverBody(ref.DocumentId, body)
	r.srv.SetResolverRef(&ref)
	enrollBoot(t, r)
	ch := issueChallenge(t, r, nil)
	stepOK(t, r)
	att := requireAttest(t, r, ch)
	rawAtt, err := os.ReadFile(attestPath(r))
	if err != nil {
		t.Fatal(err)
	}
	rep, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("no observation")
	}
	rawObs, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	ev := mustEvidence(t, r)
	if ev.Outcome != api.DnsPolicyEvidenceOutcomeFailed || att.Checks.DNSPolicy.Status == api.HealthCheckStatusPassed {
		t.Fatalf("bad credential did not fail closed: outcome %s attest %+v", ev.Outcome, att.Checks.DNSPolicy)
	}
	for _, surface := range []struct{ name, text string }{
		{"log", r.log.String()},
		{"observation", string(rawObs)},
		{"attestation", string(rawAtt)},
		{"detail", deref(mustDNS(t, r.eng.decision.Checks).Detail)},
	} {
		if strings.Contains(surface.text, secret) || strings.Contains(surface.text, path) {
			t.Fatalf("%s contains the credential or its path", surface.name)
		}
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
