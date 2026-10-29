//go:build adguard_integration

package dnspolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

type integrationEnv struct {
	prefix    string
	stateDir  string
	ad1       *AdGuardAdapter
	ad2       *AdGuardAdapter
	pub       *MultiResolverPublisher
	dnsCli    *DNSClient
	r1DNSAddr string
	r2DNSAddr string
}

func setupIntegrationEnv(t *testing.T) *integrationEnv {
	t.Helper()

	prefix := os.Getenv("FVPN_DNS_PREFIX")
	if prefix == "" {
		prefix = "fvpn-dev-adguard"
	}

	stateDir := os.Getenv("FVPN_DNS_STATE_DIR")
	if stateDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		stateDir = filepath.Clean(filepath.Join(wd, "..", "..", "deploy", "dev", "adguard", ".state"))
	}

	credsBytes, err := os.ReadFile(filepath.Join(stateDir, "credentials.json"))
	if err != nil {
		t.Skipf("skipping adguard_integration test: credentials not found at %s (run 'make -f mk/dns.mk dns-up'): %v", stateDir, err)
	}
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(credsBytes, &creds); err != nil {
		t.Fatalf("parse credentials.json: %v", err)
	}

	r1HTTPPort := envOrDefault("FVPN_DNS_RESOLVER1_HTTP_PORT", "18081")
	r2HTTPPort := envOrDefault("FVPN_DNS_RESOLVER2_HTTP_PORT", "18082")
	r1DNSPort := envOrDefault("FVPN_DNS_RESOLVER1_DNS_PORT", "15351")
	r2DNSPort := envOrDefault("FVPN_DNS_RESOLVER2_DNS_PORT", "15352")

	r1DNSAddr := "127.0.0.1:" + r1DNSPort
	r2DNSAddr := "127.0.0.1:" + r2DNSPort
	dnsCli := NewDNSClient(3 * time.Second)

	ad1 := NewAdGuardAdapter(ResolverConfig{
		ID:            "resolver-1",
		AdminURL:      "http://127.0.0.1:" + r1HTTPPort,
		DNSAddr:       r1DNSAddr,
		Username:      creds.Username,
		Password:      creds.Password,
		DNSClient:     dnsCli,
		DataDir:       filepath.Join(stateDir, "resolver1"),
		ContainerName: prefix + "-resolver1",
	})
	ad2 := NewAdGuardAdapter(ResolverConfig{
		ID:            "resolver-2",
		AdminURL:      "http://127.0.0.1:" + r2HTTPPort,
		DNSAddr:       r2DNSAddr,
		Username:      creds.Username,
		Password:      creds.Password,
		DNSClient:     dnsCli,
		DataDir:       filepath.Join(stateDir, "resolver2"),
		ContainerName: prefix + "-resolver2",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := ad1.GetAppliedState(ctx); err != nil {
		t.Skipf("skipping adguard_integration test: resolver-1 not reachable (%v)", err)
	}
	if _, err := ad2.GetAppliedState(ctx); err != nil {
		t.Skipf("skipping adguard_integration test: resolver-2 not reachable (%v)", err)
	}

	return &integrationEnv{
		prefix:    prefix,
		stateDir:  stateDir,
		ad1:       ad1,
		ad2:       ad2,
		pub:       NewMultiResolverPublisher(ad1, ad2),
		dnsCli:    dnsCli,
		r1DNSAddr: r1DNSAddr,
		r2DNSAddr: r2DNSAddr,
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func compileStandardTestBundle(t *testing.T, profile Profile) *CompiledBundle {
	t.Helper()
	now := time.Now().UTC()
	opts := CompileOptions{
		ValidFrom:      now.Add(-1 * time.Hour),
		ValidUntil:     now.Add(24 * time.Hour),
		UpstreamDNS:    []string{"upstream:5353"},
		BootstrapDNS:   []string{"127.0.0.11:53"},
		AllowedClients: []string{"127.0.0.0/8", "::1/128", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10"},
	}
	bundle, err := CompileBundle(profile, opts)
	if err != nil {
		t.Fatalf("CompileBundle failed: %v", err)
	}
	return bundle
}

func TestAT20_DNS01_DNS02_IntegrationVectorsAndAppliedRevision(t *testing.T) {
	env := setupIntegrationEnv(t)
	vf := loadVectors(t)
	ctx := context.Background()

	bundle := compileStandardTestBundle(t, vf.EvaluationProfile)
	rep, err := env.pub.Publish(ctx, bundle)
	if err != nil {
		t.Fatalf("Publish evaluation_profile failed: %v (report=%+v)", err, rep)
	}
	if rep.Status != PublishAllApplied || !rep.AllSucceeded {
		t.Fatalf("expected status=%s AllSucceeded=true, got %+v", PublishAllApplied, rep)
	}

	// Verify both resolvers report the exact same applied revision and bundle SHA-256.
	st1, err := env.ad1.GetAppliedState(ctx)
	if err != nil {
		t.Fatalf("ad1.GetAppliedState: %v", err)
	}
	st2, err := env.ad2.GetAppliedState(ctx)
	if err != nil {
		t.Fatalf("ad2.GetAppliedState: %v", err)
	}
	if st1.AppliedRevision != "12" || st2.AppliedRevision != "12" {
		t.Fatalf("applied revisions = (%q, %q), want (\"12\", \"12\")", st1.AppliedRevision, st2.AppliedRevision)
	}
	if st1.AppliedSHA256 != bundle.SHA256 || st2.AppliedSHA256 != bundle.SHA256 {
		t.Fatalf("applied sha256s = (%q, %q), want %q", st1.AppliedSHA256, st2.AppliedSHA256, bundle.SHA256)
	}

	// Verify idempotency: re-publishing the same revision is a no-op on both resolvers.
	repNoOp, err := env.pub.Publish(ctx, bundle)
	if err != nil {
		t.Fatalf("idempotent re-publish failed: %v", err)
	}
	if repNoOp.Status != PublishAllApplied {
		t.Fatalf("expected status=%s on re-publish, got %s", PublishAllApplied, repNoOp.Status)
	}
	for _, r := range repNoOp.Results {
		if !r.NoOp || r.AppliedRevision != "12" {
			t.Errorf("expected NoOp=true AppliedRevision=12 for %s, got %+v", r.ResolverID, r)
		}
	}

	// Prove every evaluation vector in testdata/vectors.json yields the exact
	// same decision from both resolvers (via real DNS queries and check_host)
	// as from Evaluate(profile, name).
	for _, vec := range vf.EvaluationVectors {
		t.Run(vec.Name, func(t *testing.T) {
			expected := Evaluate(vf.EvaluationProfile, vec.Input)
			if expected.Err != nil {
				t.Fatalf("Evaluate(%q) failed: %v", vec.Input, expected.Err)
			}

			for _, resolver := range []*AdGuardAdapter{env.ad1, env.ad2} {
				// 1. Check via AdGuard Home filtering/check_host API
				chk, err := resolver.CheckHost(ctx, expected.Domain)
				if err != nil {
					t.Fatalf("%s CheckHost(%q): %v", resolver.ID(), expected.Domain, err)
				}
				gotCheckAction := ActionAllow
				if chk.IsBlocked() {
					gotCheckAction = ActionBlock
				}
				if gotCheckAction != expected.Action {
					t.Errorf("%s CheckHost(%q) action = %s (reason=%s, rule=%s), Evaluate = %s (layer=%s)",
						resolver.ID(), vec.Input, gotCheckAction, chk.Reason, chk.Rule, expected.Action, expected.Layer)
				}

				// 2. Check via real UDP/TCP DNS TypeA query
				dnsResp, err := env.dnsCli.Query(ctx, resolver.Config().DNSAddr, expected.Domain, dnsmessage.TypeA)
				if err != nil {
					t.Fatalf("%s DNS Query(%q): %v", resolver.ID(), expected.Domain, err)
				}
				if expected.Action == ActionBlock {
					if !dnsResp.IsNXDOMAIN() {
						t.Errorf("%s DNS Query(%q) RCode = %v (answers=%+v), expected NXDOMAIN for blocked domain",
							resolver.ID(), vec.Input, dnsResp.RCode, dnsResp.Answers)
					}
					if len(dnsResp.Answers) != 0 {
						t.Errorf("%s DNS Query(%q) returned %d answers on blocked domain, want 0",
							resolver.ID(), vec.Input, len(dnsResp.Answers))
					}
				} else {
					if !dnsResp.IsSuccess() || len(dnsResp.Answers) == 0 {
						t.Errorf("%s DNS Query(%q) RCode = %v (answers=%d), expected NOERROR with A record for allowed domain",
							resolver.ID(), vec.Input, dnsResp.RCode, len(dnsResp.Answers))
					}
				}
			}
		})
	}
}

func TestAT20_DNS02_IntegrationRecordTypesAndCNAMEChain(t *testing.T) {
	env := setupIntegrationEnv(t)
	vf := loadVectors(t)
	ctx := context.Background()

	bundle := compileStandardTestBundle(t, vf.EvaluationProfile)
	if _, err := env.pub.Publish(ctx, bundle); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	for _, resolver := range []*AdGuardAdapter{env.ad1, env.ad2} {
		addr := resolver.Config().DNSAddr
		rID := resolver.ID()

		t.Run(rID+"_allowed_A_AAAA_HTTPS_SVCB", func(t *testing.T) {
			// Type A
			respA, err := env.dnsCli.Query(ctx, addr, "svc.example.com", dnsmessage.TypeA)
			if err != nil || !respA.IsSuccess() || len(respA.Answers) == 0 {
				t.Fatalf("%s TypeA svc.example.com failed: err=%v resp=%+v", rID, err, respA)
			}
			if respA.Answers[0].IP != netip.MustParseAddr("192.0.2.11") {
				t.Errorf("%s TypeA IP = %v, want 192.0.2.11", rID, respA.Answers[0].IP)
			}

			// Type AAAA
			respAAAA, err := env.dnsCli.Query(ctx, addr, "svc.example.com", dnsmessage.TypeAAAA)
			if err != nil || !respAAAA.IsSuccess() || len(respAAAA.Answers) == 0 {
				t.Fatalf("%s TypeAAAA svc.example.com failed: err=%v resp=%+v", rID, err, respAAAA)
			}
			if respAAAA.Answers[0].IP != netip.MustParseAddr("2001:db8::11") {
				t.Errorf("%s TypeAAAA IP = %v, want 2001:db8::11", rID, respAAAA.Answers[0].IP)
			}

			// Type HTTPS (65)
			respHTTPS, err := env.dnsCli.Query(ctx, addr, "svc.example.com", dnsmessage.TypeHTTPS)
			if err != nil || !respHTTPS.IsSuccess() || len(respHTTPS.Answers) == 0 {
				t.Fatalf("%s TypeHTTPS svc.example.com failed: err=%v resp=%+v", rID, err, respHTTPS)
			}
			if respHTTPS.Answers[0].Type != dnsmessage.TypeHTTPS || respHTTPS.Answers[0].Priority != 1 {
				t.Errorf("%s TypeHTTPS unexpected answer: %+v", rID, respHTTPS.Answers[0])
			}

			// Type SVCB (64)
			respSVCB, err := env.dnsCli.Query(ctx, addr, "svc.example.com", dnsmessage.TypeSVCB)
			if err != nil || !respSVCB.IsSuccess() || len(respSVCB.Answers) == 0 {
				t.Fatalf("%s TypeSVCB svc.example.com failed: err=%v resp=%+v", rID, err, respSVCB)
			}
			if respSVCB.Answers[0].Type != dnsmessage.TypeSVCB || respSVCB.Answers[0].Target != "svc.example.com" {
				t.Errorf("%s TypeSVCB unexpected answer: %+v", rID, respSVCB.Answers[0])
			}
		})

		t.Run(rID+"_blocked_must_block_A_AAAA_HTTPS_SVCB", func(t *testing.T) {
			blockedNames := []string{
				"corp-block.example.com",
				"sub.corp-block.example.com",
				"exact-block.example.com",
				"malware.test",
				"sub.malware.test",
			}
			qtypes := []struct {
				name string
				typ  dnsmessage.Type
			}{
				{"A", dnsmessage.TypeA},
				{"AAAA", dnsmessage.TypeAAAA},
				{"HTTPS", dnsmessage.TypeHTTPS},
				{"SVCB", dnsmessage.TypeSVCB},
			}
			for _, domain := range blockedNames {
				for _, qt := range qtypes {
					resp, err := env.dnsCli.Query(ctx, addr, domain, qt.typ)
					if err != nil {
						t.Fatalf("%s query %s (%s): %v", rID, domain, qt.name, err)
					}
					if qt.typ == dnsmessage.TypeSVCB {
						// AdGuard Home strips SVCB records on blocked domains by returning
						// either NXDOMAIN or synthetic NODATA (NOERROR with zero answers).
						if len(resp.Answers) != 0 || (!resp.IsNXDOMAIN() && !resp.IsSuccess()) {
							t.Errorf("%s query %s (SVCB) got RCode=%v answers=%+v; blocking must strip SVCB answers",
								rID, domain, resp.RCode, resp.Answers)
						}
					} else if !resp.IsNXDOMAIN() || len(resp.Answers) != 0 {
						t.Errorf("%s query %s (%s) got RCode=%v answers=%+v; blocking must return NXDOMAIN and not leave %s path open",
							rID, domain, qt.name, resp.RCode, resp.Answers, qt.name)
					}
				}
			}
		})

		t.Run(rID+"_cname_chain_blocked_target", func(t *testing.T) {
			// Allowed CNAME chain resolves to final A/AAAA target
			okResp, err := env.dnsCli.Query(ctx, addr, "cname-chain-ok.example.com", dnsmessage.TypeA)
			if err != nil || !okResp.IsSuccess() || len(okResp.Answers) == 0 {
				t.Fatalf("%s allowed CNAME chain failed: err=%v resp=%+v", rID, err, okResp)
			}

			// Blocked CNAME chain (cname-chain-blocked.example.net -> cname-hop.example.net -> cname-blocked-target.example.net)
			// and cross-zone (cname-chain-blocked.example.com -> cname-hop.example.com -> cname-blocked-target.example.net)
			// where cname-blocked-target.example.net is in src-malware-test.
			for _, cnameDomain := range []string{"cname-chain-blocked.example.net", "cname-chain-blocked.example.com"} {
				for _, qtype := range []dnsmessage.Type{dnsmessage.TypeA, dnsmessage.TypeAAAA, dnsmessage.TypeHTTPS} {
					resp, err := env.dnsCli.Query(ctx, addr, cnameDomain, qtype)
					if err != nil {
						t.Fatalf("%s blocked CNAME chain query %s (type=%v): %v", rID, cnameDomain, qtype, err)
					}
					if !resp.IsNXDOMAIN() || len(resp.Answers) != 0 {
						t.Errorf("%s blocked CNAME chain %s (type=%v) returned RCode=%v answers=%+v, want NXDOMAIN",
							rID, cnameDomain, qtype, resp.RCode, resp.Answers)
					}
				}
			}
		})
	}
}

func TestAT20_AT21_IntegrationDraftDoesNotAffectLive(t *testing.T) {
	env := setupIntegrationEnv(t)
	vf := loadVectors(t)
	ctx := context.Background()

	liveBundle := compileStandardTestBundle(t, vf.EvaluationProfile)
	if _, err := env.pub.Publish(ctx, liveBundle); err != nil {
		t.Fatalf("Publish liveBundle: %v", err)
	}

	// Create and validate an unapplied draft that flips decisions on www.example.com
	// and corp-block.example.com, without publishing it.
	draftProfile := Profile{
		ProfileID: "standard",
		Revision:  "99",
		Rules: []Rule{
			{
				RuleID:            "draft-block-www",
				ProfileID:         "standard",
				Domain:            "www.example.com",
				IncludeSubdomains: false,
				Action:            ActionBlock,
				Reason:            "draft_only",
				Revision:          "99",
			},
		},
	}
	if err := ValidateDraft(draftProfile, nil, nil); err != nil {
		t.Fatalf("ValidateDraft(draftProfile): %v", err)
	}
	if dec := Evaluate(draftProfile, "www.example.com"); dec.Action != ActionBlock {
		t.Fatalf("expected draft Evaluate(www.example.com) = block, got %+v", dec)
	}

	// Verify live resolvers still report revision "12" and still allow www.example.com
	// and block corp-block.example.com.
	for _, resolver := range []*AdGuardAdapter{env.ad1, env.ad2} {
		st, err := resolver.GetAppliedState(ctx)
		if err != nil {
			t.Fatalf("%s GetAppliedState: %v", resolver.ID(), err)
		}
		if st.AppliedRevision != "12" {
			t.Errorf("%s live AppliedRevision = %q, want 12", resolver.ID(), st.AppliedRevision)
		}
		respWWW, err := env.dnsCli.Query(ctx, resolver.Config().DNSAddr, "www.example.com", dnsmessage.TypeA)
		if err != nil || !respWWW.IsSuccess() {
			t.Errorf("%s live query www.example.com affected by draft: err=%v resp=%+v", resolver.ID(), err, respWWW)
		}
		respCorp, err := env.dnsCli.Query(ctx, resolver.Config().DNSAddr, "corp-block.example.com", dnsmessage.TypeA)
		if err != nil || !respCorp.IsNXDOMAIN() {
			t.Errorf("%s live query corp-block.example.com affected by draft: err=%v resp=%+v", resolver.ID(), err, respCorp)
		}
	}
}

func TestAT21_IntegrationBrokenBundlePartialRollback(t *testing.T) {
	env := setupIntegrationEnv(t)
	vf := loadVectors(t)
	ctx := context.Background()

	// Ensure baseline revision 12 is applied on both resolvers.
	bundleRev12 := compileStandardTestBundle(t, vf.EvaluationProfile)
	if _, err := env.pub.Publish(ctx, bundleRev12); err != nil {
		t.Fatalf("Publish rev 12: %v", err)
	}
	t.Cleanup(func() {
		env.ad1.SetFaultHook(nil)
		env.ad2.SetFaultHook(nil)
		_, _ = env.pub.Publish(context.Background(), bundleRev12)
	})

	// Build revision 13 that adds a block rule on rev13-blocked.example.com.
	prof13 := vf.EvaluationProfile
	prof13.Revision = "13"
	prof13.Rules = append(append([]Rule(nil), vf.EvaluationProfile.Rules...), Rule{
		RuleID:            "rule-rev13-block",
		ProfileID:         "standard",
		Domain:            "rev13-blocked.example.com",
		IncludeSubdomains: false,
		Action:            ActionBlock,
		Reason:            "rev13_test",
		Revision:          "13",
	})
	bundleRev13 := compileStandardTestBundle(t, prof13)

	// Case 1: Send valid bundleRev13 to resolver-1 and a corrupted/broken bundle
	// (with illegal @@...$important syntax) to resolver-2 via PublishPerResolver.
	brokenBundle := *bundleRev13
	brokenBundle.UserRules = append(append([]string(nil), bundleRev13.UserRules...), "@@||illegal-important.example.com^$important")

	rep1, err := env.pub.PublishPerResolver(ctx, bundleRev13, map[string]*CompiledBundle{
		"resolver-1": bundleRev13,
		"resolver-2": &brokenBundle,
	})
	if err == nil {
		t.Fatal("expected error when broken bundle is rejected on resolver-2")
	}
	if rep1 == nil || rep1.Status != PublishPartial || rep1.AllSucceeded {
		t.Fatalf("expected status=%s AllSucceeded=false, got %+v", PublishPartial, rep1)
	}

	// Verify resolver-1 is on revision 13 while resolver-2 stayed on last_good revision 12.
	st1, _ := env.ad1.GetAppliedState(ctx)
	st2, _ := env.ad2.GetAppliedState(ctx)
	if st1.AppliedRevision != "13" {
		t.Errorf("resolver-1 AppliedRevision = %q, want 13", st1.AppliedRevision)
	}
	if st2.AppliedRevision != "12" {
		t.Errorf("resolver-2 AppliedRevision = %q, want 12 (last_good)", st2.AppliedRevision)
	}

	// Case 2: Post-apply smoke-test failure on resolver-2 rolls back live AdGuard Home
	// state from revision 13 back to revision 12.
	env.ad2.SetFaultHook(func(ctx context.Context, resolverID string, b *CompiledBundle) error {
		return errors.New("simulated post-apply smoke test failure on resolver-2")
	})
	rep2, err := env.pub.Publish(ctx, bundleRev13)
	env.ad2.SetFaultHook(nil)
	if err == nil {
		t.Fatal("expected error from smoke test failure on resolver-2")
	}
	if rep2 == nil || rep2.Status != PublishPartial || rep2.AllSucceeded {
		t.Fatalf("expected status=%s AllSucceeded=false, got %+v", PublishPartial, rep2)
	}
	if !rep2.Results[1].RolledBack || rep2.Results[1].AppliedRevision != "12" {
		t.Errorf("expected resolver-2 RolledBack=true AppliedRevision=12, got %+v", rep2.Results[1])
	}

	// Prove via live DNS query that resolver-1 blocks rev13-blocked.example.com (rev 13)
	// while resolver-2 allows rev13-blocked.example.com (rolled back to rev 12!).
	r1Resp, err := env.dnsCli.Query(ctx, env.r1DNSAddr, "rev13-blocked.example.com", dnsmessage.TypeA)
	if err != nil || !r1Resp.IsNXDOMAIN() {
		t.Errorf("resolver-1 (rev 13) expected NXDOMAIN for rev13-blocked.example.com, got err=%v resp=%+v", err, r1Resp)
	}
	r2Resp, err := env.dnsCli.Query(ctx, env.r2DNSAddr, "rev13-blocked.example.com", dnsmessage.TypeA)
	if err != nil || !r2Resp.IsSuccess() {
		t.Errorf("resolver-2 (rolled back to rev 12) expected NOERROR for rev13-blocked.example.com, got err=%v resp=%+v", err, r2Resp)
	}
}

func TestAT21_IntegrationFailoverAndNoUnfilteredFallback(t *testing.T) {
	env := setupIntegrationEnv(t)
	vf := loadVectors(t)
	ctx := context.Background()

	bundle := compileStandardTestBundle(t, vf.EvaluationProfile)
	if _, err := env.pub.Publish(ctx, bundle); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Pause resolver-1 container to simulate a real single-resolver outage.
	r1Container := env.ad1.Config().ContainerName
	pauseCmd := exec.Command("docker", "pause", r1Container)
	if out, err := pauseCmd.CombinedOutput(); err != nil {
		t.Fatalf("docker pause %s failed: %v (%s)", r1Container, err, string(out))
	}
	unpaused := false
	unpause := func() {
		if !unpaused {
			_ = exec.Command("docker", "unpause", r1Container).Run()
			unpaused = true
		}
	}
	defer unpause()

	shortCli := NewDNSClient(600 * time.Millisecond)

	// 1. With resolver-1 down, QueryWithFailover answers from resolver-2 with
	// identical policy enforcement.
	allowResp, err := shortCli.QueryWithFailover(ctx, []string{env.r1DNSAddr, env.r2DNSAddr}, "www.example.com", dnsmessage.TypeA)
	if err != nil || !allowResp.IsSuccess() || allowResp.Resolver != env.r2DNSAddr {
		t.Fatalf("failover allowed query failed: err=%v resp=%+v", err, allowResp)
	}

	blockResp, err := shortCli.QueryWithFailover(ctx, []string{env.r1DNSAddr, env.r2DNSAddr}, "malware.test", dnsmessage.TypeA)
	if err != nil || !blockResp.IsNXDOMAIN() || blockResp.Resolver != env.r2DNSAddr {
		t.Fatalf("failover blocked query failed: err=%v resp=%+v", err, blockResp)
	}

	// Unpause resolver-1 before testing both-down scenario.
	unpause()

	// 2. With both controlled resolvers unreachable, QueryWithFailover fails
	// closed and never falls back to any unfiltered public or host DNS.
	deadResolvers := []string{"127.0.0.1:15398", "127.0.0.1:15399"}
	respBothDown, err := shortCli.QueryWithFailover(ctx, deadResolvers, "www.example.com", dnsmessage.TypeA)
	if err == nil || respBothDown != nil {
		t.Fatalf("expected fail-closed error when both controlled resolvers are down, got resp=%+v err=%v", respBothDown, err)
	}
}

func TestAT22_SEC02_IntegrationDataMinimization(t *testing.T) {
	env := setupIntegrationEnv(t)
	vf := loadVectors(t)
	ctx := context.Background()

	bundle := compileStandardTestBundle(t, vf.EvaluationProfile)
	if _, err := env.pub.Publish(ctx, bundle); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Distinct queried domain names that are NOT literal rule strings in the
	// configuration file (subdomains of blocked zones + allowed domains).
	probeQueries := []struct {
		marker         string
		qname          string
		expectedAction RuleAction
	}{
		{
			marker:         "privacy-allowed-probe-91a8b7c6",
			qname:          "privacy-allowed-probe-91a8b7c6.example.com",
			expectedAction: ActionAllow,
		},
		{
			marker:         "privacy-blocked-probe-44d3e2f1",
			qname:          "privacy-blocked-probe-44d3e2f1.malware.test",
			expectedAction: ActionBlock,
		},
		{
			marker:         "privacy-blocked-corp-77c6b5a4",
			qname:          "privacy-blocked-corp-77c6b5a4.corp-block.example.com",
			expectedAction: ActionBlock,
		},
		{
			marker:         "privacy-allowed-net-22f1e0d9",
			qname:          "privacy-allowed-net-22f1e0d9.example.net",
			expectedAction: ActionAllow,
		},
	}

	for _, pq := range probeQueries {
		for _, addr := range []string{env.r1DNSAddr, env.r2DNSAddr} {
			for _, qtype := range []dnsmessage.Type{dnsmessage.TypeA, dnsmessage.TypeAAAA, dnsmessage.TypeHTTPS} {
				resp, err := env.dnsCli.Query(ctx, addr, pq.qname, qtype)
				if err != nil {
					t.Fatalf("query %s to %s: %v", pq.qname, addr, err)
				}
				if pq.expectedAction == ActionBlock && !resp.IsNXDOMAIN() {
					t.Fatalf("expected NXDOMAIN for %s, got %v", pq.qname, resp.RCode)
				}
				if pq.expectedAction == ActionAllow && !resp.IsSuccess() {
					t.Fatalf("expected NOERROR for %s, got %v", pq.qname, resp.RCode)
				}
			}
		}
	}

	for _, resolver := range []*AdGuardAdapter{env.ad1, env.ad2} {
		rID := resolver.ID()

		// 1. Verify configuration flags via API.
		st, err := resolver.GetAppliedState(ctx)
		if err != nil {
			t.Fatalf("%s GetAppliedState: %v", rID, err)
		}
		if st.QueryLogEnabled {
			t.Errorf("%s querylog.enabled = true, want false", rID)
		}
		if st.StatisticsEnabled {
			t.Errorf("%s statistics.enabled = true, want false", rID)
		}

		// 2. Verify /control/querylog and /control/stats API responses are empty and
		// contain none of the queried test names.
		qlItems, err := resolver.GetQueryLogItems(ctx)
		if err != nil {
			t.Fatalf("%s GetQueryLogItems: %v", rID, err)
		}
		if len(qlItems) != 0 {
			t.Errorf("%s querylog returned %d items, want 0", rID, len(qlItems))
		}

		statsMap, err := resolver.GetStatsSummary(ctx)
		if err != nil {
			t.Fatalf("%s GetStatsSummary: %v", rID, err)
		}
		statsJSON, _ := json.Marshal(statsMap)
		for _, pq := range probeQueries {
			if bytes.Contains(statsJSON, []byte(pq.marker)) {
				t.Errorf("%s /control/stats response leaked queried marker %q: %s", rID, pq.marker, string(statsJSON))
			}
		}

		// 3. Verify container logs contain none of the queried test names.
		containerName := resolver.Config().ContainerName
		logOut, err := exec.Command("docker", "logs", containerName).CombinedOutput()
		if err != nil {
			t.Fatalf("docker logs %s: %v", containerName, err)
		}
		for _, pq := range probeQueries {
			if bytes.Contains(logOut, []byte(pq.marker)) {
				t.Errorf("%s container logs leaked queried marker %q", containerName, pq.marker)
			}
		}

		// 4. Walk and grep all files in the resolver's data & config volume directories
		// on disk (and inside the container volume /opt/adguardhome).
		dataDir := resolver.Config().DataDir
		err = filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return walkErr
			}
			base := filepath.Base(path)
			if strings.HasPrefix(base, "querylog") {
				t.Errorf("%s unexpected querylog file created on disk: %s", rID, path)
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}
			for _, pq := range probeQueries {
				if bytes.Contains(content, []byte(pq.marker)) {
					t.Errorf("%s file %s on disk leaked queried domain marker %q", rID, path, pq.marker)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%s walk dataDir %s: %v", rID, dataDir, err)
		}

		// Also run grep inside the container volume paths (/opt/adguardhome/work and /opt/adguardhome/conf).
		for _, pq := range probeQueries {
			grepCmd := exec.Command("docker", "exec", containerName, "grep", "-rn", pq.marker, "/opt/adguardhome/work", "/opt/adguardhome/conf")
			out, err := grepCmd.CombinedOutput()
			if err == nil || len(bytes.TrimSpace(out)) > 0 {
				t.Errorf("%s container volume grep found queried marker %q: %s", containerName, pq.marker, string(out))
			}
		}
	}
}
