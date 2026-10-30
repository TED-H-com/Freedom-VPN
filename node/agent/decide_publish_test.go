package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
)

func TestDecidePublicationDeadlock(t *testing.T) {
	t.Run("unpublished baseline", func(t *testing.T) {
		in := servingInput()
		ds := *in.Desired
		ds.PublicationIntent = api.PublicationIntentUnpublished
		ds.DeviceBindings = []api.DeviceBinding{oneBinding(in.Now)}
		in.Desired = &ds
		lease := *in.Lease
		lease.Members = []api.MemberLease{oneMember(in.Now, ds.DeviceBindings[0].MemberId)}
		in.Lease = &lease

		d, rep := reported(t, in)
		if d.Serve || len(d.Model.Bindings) != 0 || d.Model.Generation != 0 {
			t.Fatalf("forwarded while unpublished: serve %v model %+v", d.Serve, d.Model)
		}
		if d.Safety != api.SafetyStatusPassed || d.Provisioning != api.ProvisioningStateReady || d.DataHealth != api.DataHealthHealthy {
			t.Fatalf("safety %s provisioning %s health %s reasons %v", d.Safety, d.Provisioning, d.DataHealth, d.Reasons)
		}
		if d.Applied != uint64(ds.ConfigGeneration) {
			t.Fatalf("applied %d desired %d", d.Applied, ds.ConfigGeneration)
		}
		if !hasReason(d.Reasons, api.ForwardingBlockReasonNotPublished) || hasReason(d.Reasons, api.ForwardingBlockReasonLeaseExpired) {
			t.Fatalf("reasons %v", d.Reasons)
		}
		assertServingChecks(t, d.Checks, api.HealthCheckStatusNotRun)
		assertBaselineChecks(t, d.Checks, api.HealthCheckStatusPassed)
		assertReport(t, rep, d, false)
	})

	t.Run("unpublished empty release list", func(t *testing.T) {
		in := servingInput()
		ds := *in.Desired
		ds.PublicationIntent = api.PublicationIntentUnpublished
		ds.ServingReleaseIds = nil
		in.Desired = &ds
		d := Decide(in)
		if d.Serve || d.Safety != api.SafetyStatusPassed || hasReason(d.Reasons, api.ForwardingBlockReasonReleaseNotSupported) {
			t.Fatalf("empty release list: serve %v safety %s reasons %v", d.Serve, d.Safety, d.Reasons)
		}
		c, ok := checkBy(d.Checks, api.NodeCheckNameReleaseSupported)
		if !ok || c.Status != api.HealthCheckStatusPassed {
			t.Fatalf("release %+v", c)
		}
	})

	t.Run("disabled does not serve", func(t *testing.T) {
		in := servingInput()
		ds := *in.Desired
		ds.PublicationIntent = api.PublicationIntentDisabled
		ds.DeviceBindings = []api.DeviceBinding{oneBinding(in.Now)}
		in.Desired = &ds
		d := Decide(in)
		if d.Serve || d.Safety != api.SafetyStatusPassed {
			t.Fatalf("disabled: serve %v safety %s reasons %v", d.Serve, d.Safety, d.Reasons)
		}
		assertServingChecks(t, d.Checks, api.HealthCheckStatusNotRun)
	})

	t.Run("published serving", func(t *testing.T) {
		in := withAdmittedMember(servingInput())
		d, rep := reported(t, in)
		if !d.Serve || len(d.Model.Bindings) != 1 || d.Model.Generation != uint64(in.Desired.ConfigGeneration) {
			t.Fatalf("did not serve: serve %v gen %d bindings %d reasons %v errors %+v", d.Serve, d.Model.Generation, len(d.Model.Bindings), d.Reasons, d.Errors)
		}
		if d.Safety != api.SafetyStatusPassed || d.Provisioning != api.ProvisioningStateReady || d.DataHealth != api.DataHealthHealthy {
			t.Fatalf("safety %s provisioning %s health %s", d.Safety, d.Provisioning, d.DataHealth)
		}
		if d.Applied != uint64(in.Desired.ConfigGeneration) {
			t.Fatalf("applied %d", d.Applied)
		}
		assertServingChecks(t, d.Checks, api.HealthCheckStatusPassed)
		assertBaselineChecks(t, d.Checks, api.HealthCheckStatusPassed)
		assertReport(t, rep, d, true)
	})

	t.Run("published expired lease", func(t *testing.T) {
		in := withAdmittedMember(servingInput())
		in.LeaseFresh = false
		d, rep := reported(t, in)
		if d.Serve || d.Safety != api.SafetyStatusExpired || d.DataHealth != api.DataHealthHealthy {
			t.Fatalf("expired lease: serve %v safety %s health %s reasons %v", d.Serve, d.Safety, d.DataHealth, d.Reasons)
		}
		if !hasReason(d.Reasons, api.ForwardingBlockReasonLeaseExpired) {
			t.Fatalf("reasons %v", d.Reasons)
		}
		if d.Applied != uint64(in.Desired.ConfigGeneration) || d.Provisioning != api.ProvisioningStateReady {
			t.Fatalf("applied %d provisioning %s", d.Applied, d.Provisioning)
		}
		assertServingChecks(t, d.Checks, api.HealthCheckStatusFailed)
		assertBaselineChecks(t, d.Checks, api.HealthCheckStatusPassed)
		assertReport(t, rep, d, false)
	})

	t.Run("baseline missing", func(t *testing.T) {
		in := servingInput()
		in.BaselineOK = false
		d, rep := reported(t, in)
		if d.Serve || d.Safety != api.SafetyStatusFailed || d.Applied != 0 || d.Provisioning != api.ProvisioningStateApplying {
			t.Fatalf("baseline: serve %v safety %s applied %d provisioning %s", d.Serve, d.Safety, d.Applied, d.Provisioning)
		}
		if !hasCode(d.Errors, api.ErrorCodeISOLATIONBASELINEMISSING) || hasCode(d.Errors, api.ErrorCodeHEALTHCHECKFAILED) {
			t.Fatalf("errors %+v", d.Errors)
		}
		c, ok := checkBy(d.Checks, api.NodeCheckNameInternetOnly)
		if !ok || c.Status != api.HealthCheckStatusFailed {
			t.Fatalf("internet %+v", c)
		}
		if _, err := rep.AppliedConfigGeneration.Get(); !errors.Is(err, presence.ErrNull) {
			t.Fatal("baseline failure acknowledged a generation")
		}
		if rep.SafetyStatus != api.SafetyStatusFailed || rep.ProvisioningState != api.ProvisioningStateApplying || rep.MemberForwardingEnabled {
			t.Fatalf("report safety %s provisioning %s forwarding %v", rep.SafetyStatus, rep.ProvisioningState, rep.MemberForwardingEnabled)
		}
	})

	t.Run("release mismatch fails baseline", func(t *testing.T) {
		in := servingInput()
		ds := *in.Desired
		ds.ServingReleaseIds = []uuid.UUID{uuid.MustParse("019a0000-0000-7000-8000-0000000000c2")}
		in.Desired = &ds
		d := Decide(in)
		if d.Serve || d.Safety != api.SafetyStatusFailed || !hasReason(d.Reasons, api.ForwardingBlockReasonReleaseNotSupported) {
			t.Fatalf("release: serve %v safety %s reasons %v", d.Serve, d.Safety, d.Reasons)
		}
		c, ok := checkBy(d.Checks, api.NodeCheckNameReleaseSupported)
		if !ok || c.Status != api.HealthCheckStatusFailed {
			t.Fatalf("release check %+v", c)
		}
	})

	t.Run("relay degraded still passes safety", func(t *testing.T) {
		in := servingInput()
		in.ProbeHealth = api.DataHealthDegraded
		d := Decide(in)
		if !d.Serve || d.Safety != api.SafetyStatusPassed || d.DataHealth != api.DataHealthDegraded {
			t.Fatalf("degraded overlay: serve %v safety %s health %s reasons %v", d.Serve, d.Safety, d.DataHealth, d.Reasons)
		}
		c, ok := checkBy(d.Checks, api.NodeCheckNameTunnelProbe)
		if !ok || c.Status != api.HealthCheckStatusPassed {
			t.Fatalf("tunnel %+v", c)
		}
	})

	t.Run("probe failed", func(t *testing.T) {
		in := servingInput()
		in.ProbeCheck = api.HealthCheckStatusFailed
		in.ProbeHealth = api.DataHealthFailed
		d := Decide(in)
		if d.Serve || d.Safety != api.SafetyStatusFailed || d.DataHealth != api.DataHealthFailed {
			t.Fatalf("probe: serve %v safety %s health %s", d.Serve, d.Safety, d.DataHealth)
		}
		if !hasCode(d.Errors, api.ErrorCodeHEALTHCHECKFAILED) || !hasMessage(d.Errors, "tunnel_probe") {
			t.Fatalf("errors %+v", d.Errors)
		}
		c, ok := checkBy(d.Checks, api.NodeCheckNameTunnelProbe)
		if !ok || c.Status != api.HealthCheckStatusFailed {
			t.Fatalf("tunnel %+v", c)
		}
	})

	t.Run("probe not run", func(t *testing.T) {
		in := servingInput()
		in.ProbeRan = false
		in.ProbeCheck = ""
		in.ProbeHealth = ""
		d := Decide(in)
		if d.Serve || d.Safety != api.SafetyStatusUnknown || d.DataHealth != api.DataHealthUnknown {
			t.Fatalf("not run: serve %v safety %s health %s reasons %v", d.Serve, d.Safety, d.DataHealth, d.Reasons)
		}
		if hasCode(d.Errors, api.ErrorCodeHEALTHCHECKFAILED) {
			t.Fatalf("not_run reported a health-check failure: %+v", d.Errors)
		}
		c, ok := checkBy(d.Checks, api.NodeCheckNameTunnelProbe)
		if !ok || c.Status != api.HealthCheckStatusNotRun {
			t.Fatalf("tunnel %+v", c)
		}
	})

	t.Run("published policy deadline expires safety", func(t *testing.T) {
		in := servingInput()
		in.DesiredOK = false
		in.DesiredExpired = true
		d := Decide(in)
		if d.Serve || d.Safety != api.SafetyStatusExpired {
			t.Fatalf("policy deadline: serve %v safety %s reasons %v", d.Serve, d.Safety, d.Reasons)
		}
		c, ok := checkBy(d.Checks, api.NodeCheckNameReleaseSupported)
		if !ok || c.Status != api.HealthCheckStatusNotRun {
			t.Fatalf("release on an expired document %+v", c)
		}
	})

	t.Run("unpublished policy deadline fails safety", func(t *testing.T) {
		in := servingInput()
		ds := *in.Desired
		ds.PublicationIntent = api.PublicationIntentUnpublished
		in.Desired = &ds
		in.DesiredOK = false
		in.DesiredExpired = true
		d := Decide(in)
		if d.Safety != api.SafetyStatusFailed || d.Serve {
			t.Fatalf("unpublished deadline: safety %s serve %v", d.Safety, d.Serve)
		}
	})
}

func TestDecideNobodyAdmittedKeepsSafety(t *testing.T) {
	in := servingInput()
	ds := *in.Desired
	ds.DeviceBindings = []api.DeviceBinding{oneBinding(in.Now)}
	in.Desired = &ds
	d := Decide(in)
	if d.Serve || d.Safety != api.SafetyStatusPassed {
		t.Fatalf("serve %v safety %s reasons %v", d.Serve, d.Safety, d.Reasons)
	}
	assertProvisionalDecision(t, d)
	assertServingChecks(t, d.Checks, api.HealthCheckStatusFailed)
	_, rep := reported(t, in)
	assertProvisionalReport(t, rep)
}

func TestDecideBareIPv4BindingIsAdmitted(t *testing.T) {
	in := withAdmittedMember(servingInput())
	in.Desired.DeviceBindings[0].AllowedIpv4 = []string{"100.64.1.1"}
	d := Decide(in)
	if !d.Serve || hasReason(d.Reasons, api.ForwardingBlockReasonSafetyCheckFailed) || hasReason(d.Reasons, api.ForwardingBlockReasonLeaseExpired) {
		t.Fatalf("serve %v reasons %v", d.Serve, d.Reasons)
	}
	if len(d.Model.Bindings) != 1 || len(d.Model.Bindings[0].AllowedV4) != 1 || d.Model.Bindings[0].AllowedV4[0].String() != "100.64.1.1/32" {
		t.Fatalf("bindings %+v", d.Model.Bindings)
	}
	lease, ok := checkBy(d.Checks, api.NodeCheckNameLease)
	if !ok || lease.Status != api.HealthCheckStatusPassed {
		t.Fatalf("lease %+v", lease)
	}
}

func TestHostPrefixForms(t *testing.T) {
	got, err := hostPrefixes([]string{"100.64.1.1", "100.64.1.2/32"})
	if err != nil || len(got) != 2 || got[0].String() != "100.64.1.1/32" || got[1].String() != "100.64.1.2/32" {
		t.Fatalf("got %v err %v", got, err)
	}
	if _, err := hostPrefixes([]string{"100.64.1.0/24"}); err == nil {
		t.Fatal("prefix length other than 32 was accepted")
	}
	if _, err := hostPrefixes(nil); err == nil {
		t.Fatal("empty allowed set was accepted")
	}
}

func withAdmittedMember(in Input) Input {
	ds := *in.Desired
	ds.DeviceBindings = []api.DeviceBinding{oneBinding(in.Now)}
	in.Desired = &ds
	lease := *in.Lease
	lease.Members = []api.MemberLease{oneMember(in.Now, ds.DeviceBindings[0].MemberId)}
	in.Lease = &lease
	return in
}

func oneBinding(now time.Time) api.DeviceBinding {
	return api.DeviceBinding{
		AllowedIpv4:     []string{"100.64.1.1/32"},
		AllowedIpv6:     []string{},
		BindingRevision: 1,
		DeviceId:        uuid.MustParse("019a0000-0000-7000-8000-0000000000a1"),
		MemberId:        uuid.MustParse("019a0000-0000-7000-8000-00000000000a"),
		PeerId:          "peer-a1",
		ValidFrom:       now.Add(-time.Minute),
		ValidUntil:      now.Add(time.Hour),
	}
}

func oneMember(now time.Time, member uuid.UUID) api.MemberLease {
	end := now.Add(time.Hour)
	return api.MemberLease{MemberId: member, LeaseEnd: end, EffectiveServiceEnd: end}
}

func reported(t *testing.T, in Input) (Decision, api.ObservationReport) {
	t.Helper()
	d := Decide(in)
	rel := ""
	if in.ReleaseID != uuid.Nil {
		rel = in.ReleaseID.String()
	}
	e := &Engine{
		cfg:        Config{AgentRelease: "0.0.0-test", NetbirdBackend: in.Backend, ReleaseID: rel},
		decision:   d,
		appliedGen: d.Applied,
		desired:    in.Desired,
		serving:    d.Serve,
		nodeID:     uuid.MustParse("019a0000-0000-7000-8000-0000000000e1"),
		incID:      uuid.MustParse("019a0000-0000-7000-8000-0000000000e2"),
		bootID:     uuid.MustParse("019a0000-0000-7000-8000-0000000000e3"),
		lease:      in.Lease,
	}
	at := in.Now
	if at.IsZero() {
		at = time.Unix(0, 0).UTC()
	}
	rep, err := e.report(context.Background(), at)
	if err != nil {
		t.Fatal(err)
	}
	return d, rep
}

func assertReport(t *testing.T, rep api.ObservationReport, d Decision, forwarding bool) {
	t.Helper()
	if rep.SafetyStatus != d.Safety || rep.DataHealth != d.DataHealth || rep.ProvisioningState != d.Provisioning || rep.MemberForwardingEnabled != forwarding {
		t.Fatalf("report safety %s health %s provisioning %s forwarding %v", rep.SafetyStatus, rep.DataHealth, rep.ProvisioningState, rep.MemberForwardingEnabled)
	}
	gen, err := rep.AppliedConfigGeneration.Get()
	if d.Applied == 0 {
		if !errors.Is(err, presence.ErrNull) {
			t.Fatalf("applied generation %v %v", gen, err)
		}
		return
	}
	if err != nil || uint64(gen) != d.Applied {
		t.Fatalf("applied generation %v (%d) err %v", gen, d.Applied, err)
	}
	if _, err := rep.Netbird.PeerId.Get(); err == nil {
		t.Fatal("report invented a peer id")
	}
}

func assertServingChecks(t *testing.T, checks []api.NodeCheck, status api.HealthCheckStatus) {
	t.Helper()
	for _, name := range []api.NodeCheckName{api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
		c, ok := checkBy(checks, name)
		if !ok || c.Status != status {
			t.Fatalf("%s status %s, want %s", name, c.Status, status)
		}
		if status == api.HealthCheckStatusNotRun {
			if _, err := c.ErrorCode.Get(); !errors.Is(err, presence.ErrNull) {
				t.Fatalf("%s error %v", name, err)
			}
		}
	}
}

func assertBaselineChecks(t *testing.T, checks []api.NodeCheck, status api.HealthCheckStatus) {
	t.Helper()
	for _, name := range []api.NodeCheckName{
		api.NodeCheckNameDnsPolicy, api.NodeCheckNameInternetOnly, api.NodeCheckNameMeterPath,
		api.NodeCheckNamePolicySignature, api.NodeCheckNameReleaseSupported, api.NodeCheckNameStorage,
		api.NodeCheckNameTunnelProbe,
	} {
		c, ok := checkBy(checks, name)
		if !ok || c.Status != status {
			t.Fatalf("%s status %s, want %s", name, c.Status, status)
		}
	}
}

func checkBy(checks []api.NodeCheck, name api.NodeCheckName) (api.NodeCheck, bool) {
	for _, c := range checks {
		if c.Name == name {
			return c, true
		}
	}
	return api.NodeCheck{}, false
}
