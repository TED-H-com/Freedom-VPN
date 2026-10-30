package agent

import (
	"context"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestReviewD056_TruncatedAndSizeClaimDoNotApply(t *testing.T) {
	good := []byte(exampleResolverBody)
	short := good[:len(good)-1]

	t.Run("truncated", func(t *testing.T) {
		r := newRig(t)
		dnsApply := &scriptDNS{}
		r.eng.dnsApply = dnsApply
		ref := exampleRef(good)
		r.srv.PutResolverBody(ref.DocumentId, short)
		r.srv.SetResolverRef(&ref)
		enrollBoot(t, r)
		stepOK(t, r)
		if dnsApply.calls != 0 {
			t.Fatalf("apply calls %d", dnsApply.calls)
		}
		c := mustDNS(t, r.eng.decision.Checks)
		if c.Status != api.HealthCheckStatusFailed || c.Detail == nil || *c.Detail != "Resolver document size differs from the signed reference." {
			t.Fatalf("%+v", c)
		}
		if codeOf(t, c) != api.ErrorCodeSIZEMISMATCH {
			t.Fatalf("code %s", codeOf(t, c))
		}
		ev := mustEvidence(t, r)
		if ev.Outcome != api.DnsPolicyEvidenceOutcomeFailed || ev.DocumentId != ref.DocumentId {
			t.Fatalf("%+v", ev)
		}
	})

	t.Run("size claim with digest of the short body", func(t *testing.T) {
		r := newRig(t)
		dnsApply := &scriptDNS{}
		r.eng.dnsApply = dnsApply
		ref := exampleRef(short)
		ref.SizeBytes = len(good)
		r.srv.PutResolverBody(ref.DocumentId, short)
		r.srv.SetResolverRef(&ref)
		enrollBoot(t, r)
		stepOK(t, r)
		if dnsApply.calls != 0 {
			t.Fatalf("apply calls %d", dnsApply.calls)
		}
		c := mustDNS(t, r.eng.decision.Checks)
		if c.Status == api.HealthCheckStatusPassed || codeOf(t, c) != api.ErrorCodeSIZEMISMATCH {
			t.Fatalf("status %s code %s detail %v", c.Status, codeOf(t, c), c.Detail)
		}
	})
}

func TestReviewD061_MissingResolverStaysNotRun(t *testing.T) {
	r := newRig(t)
	body := []byte(exampleResolverBody)
	ref := exampleRef(body)
	r.srv.PutResolverBody(ref.DocumentId, body)
	r.srv.SetResolverRef(&ref)
	enrollBoot(t, r)
	stepOK(t, r)
	if r.eng.dnsApply != nil {
		t.Fatal("applier was constructed without a resolver")
	}
	c := mustDNS(t, r.eng.decision.Checks)
	if c.Status != api.HealthCheckStatusFailed || codeOf(t, c) != api.ErrorCodeDEPENDENCYUNAVAILABLE {
		t.Fatalf("%+v", c)
	}
	ev := mustEvidence(t, r)
	if ev.Outcome != api.DnsPolicyEvidenceOutcomeNotRun || ev.DocumentId != ref.DocumentId {
		t.Fatalf("%+v", ev)
	}
	if r.eng.serving {
		t.Fatal("missing resolver opened serving")
	}
}

func TestReviewD056_ForgedETagDoesNotHideTheNewDocument(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)
	if r.eng.desired == nil || r.eng.desiredETag == "" {
		t.Fatal("no desired state")
	}
	staleGen := r.eng.desired.ConfigGeneration
	r.eng.desiredETag = `"forged-etag-not-the-body"`
	body := []byte(exampleResolverBody)
	ref := exampleRef(body)
	dnsApply := &scriptDNS{}
	r.eng.dnsApply = dnsApply
	r.srv.PutResolverBody(ref.DocumentId, body)
	r.srv.SetResolverRef(&ref)
	ch := issueChallenge(t, r, nil)
	stepOK(t, r)
	if r.eng.desired == nil || r.eng.desired.ConfigGeneration == staleGen || r.eng.desiredETag == `"forged-etag-not-the-body"` {
		t.Fatalf("forged etag hid the new document: gen %s etag %q", r.eng.desired.ConfigGeneration, r.eng.desiredETag)
	}
	if _, err := r.eng.desired.Dns.ResolverDocument.Get(); err != nil {
		t.Fatal(err)
	}
	if dnsApply.calls != 1 {
		t.Fatalf("apply calls %d", dnsApply.calls)
	}
	att := requireAttest(t, r, ch)
	if att.Checks.DNSPolicy.Status != api.HealthCheckStatusPassed {
		t.Fatalf("verified document did not pass after apply: %+v", att.Checks.DNSPolicy)
	}
}

func TestReviewD056_HealthAnswerBeforeDNSApply(t *testing.T) {
	// A named resolver document is dns_policy passed only after this process
	// records that id and digest as applied. A revoked step does not Apply
	// again, and a lease revoke before evaluate has not Applied at all.
	t.Run("revoked step", func(t *testing.T) {
		r := newRig(t)
		r.eng.prober = &verProbe{version: "0.76.2"}
		body := []byte(exampleResolverBody)
		ref := exampleRef(body)
		dnsApply := &scriptDNS{}
		r.eng.dnsApply = dnsApply
		r.srv.PutResolverBody(ref.DocumentId, body)
		r.srv.SetResolverRef(&ref)
		enrollBoot(t, r)
		stepOK(t, r)
		if dnsApply.calls != 1 {
			t.Fatalf("first apply %d", dnsApply.calls)
		}
		if _, err := r.eng.desired.Dns.ResolverDocument.Get(); err != nil {
			t.Fatal(err)
		}
		ch := issueChallenge(t, r, nil)
		r.eng.revoked = true
		stepOK(t, r)
		att := requireAttest(t, r, ch)
		if dnsApply.calls != 1 || att.Checks.DNSPolicy.Status != api.HealthCheckStatusFailed || att.Checks.DNSPolicy.Reason != "policy not in force" {
			t.Fatalf("revoked step dns_policy %s (%q) calls=%d", att.Checks.DNSPolicy.Status, att.Checks.DNSPolicy.Reason, dnsApply.calls)
		}
		ev := mustEvidence(t, r)
		if ev.Outcome != api.DnsPolicyEvidenceOutcomeFailed || ev.DocumentId != ref.DocumentId || ev.ContentSha256 != ref.ContentSha256 {
			t.Fatalf("evidence %+v", ev)
		}
	})

	t.Run("lease revoke before evaluate", func(t *testing.T) {
		r := newRig(t)
		r.eng.prober = &verProbe{version: "0.76.2"}
		enrollBoot(t, r)
		stepOK(t, r)
		body := []byte(exampleResolverBody)
		ref := exampleRef(body)
		dnsApply := &scriptDNS{}
		r.eng.dnsApply = dnsApply
		r.srv.PutResolverBody(ref.DocumentId, body)
		r.srv.SetResolverRef(&ref)
		ch := issueChallenge(t, r, nil)
		if err := r.eng.syncDesired(context.Background()); err != nil {
			t.Fatal(err)
		}
		if _, err := r.eng.desired.Dns.ResolverDocument.Get(); err != nil {
			t.Fatal(err)
		}
		r.eng.markRevoked(context.Background())
		r.eng.answerHealth(context.Background())
		att := requireAttest(t, r, ch)
		if dnsApply.calls != 0 || att.Checks.DNSPolicy.Status != api.HealthCheckStatusFailed || att.Checks.DNSPolicy.Reason != "policy not in force" {
			t.Fatalf("lease revoke dns_policy %s (%q) calls=%d", att.Checks.DNSPolicy.Status, att.Checks.DNSPolicy.Reason, dnsApply.calls)
		}
	})
}
