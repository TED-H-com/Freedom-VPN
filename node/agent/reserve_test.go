package agent

import (
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
)

func TestHostReserveRequired(t *testing.T) {
	dir := t.TempDir()
	base := `{
		"control_url": "https://control.example.invalid",
		"server_ca_file": "/etc/vpn-node/ca.pem",
		"trust_bundle": "/etc/vpn-node/trust.json",
		"datapath": {"member_ingress": "fv-fh", "uplink": "fv-ih", "snat": "192.0.2.37"},
		"agent_release": "0.0.0-test",
		"netbird_backend": "kernel"
	}`
	path := filepath.Join(dir, "agent.json")
	if err := os.WriteFile(path, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(path)
	if !errors.Is(err, ErrHostReserve) {
		t.Fatalf("missing: %v", err)
	}
	if ExitStatus(err) != ExitUsage || ExitUsage != 64 {
		t.Fatalf("status %d", ExitStatus(err))
	}
	zero := `{
		"control_url": "https://control.example.invalid",
		"server_ca_file": "/etc/vpn-node/ca.pem",
		"trust_bundle": "/etc/vpn-node/trust.json",
		"datapath": {"member_ingress": "fv-fh", "uplink": "fv-ih", "snat": "192.0.2.37"},
		"agent_release": "0.0.0-test",
		"netbird_backend": "kernel",
		"host_reserve_bps": 0
	}`
	if err := os.WriteFile(path, []byte(zero), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); !errors.Is(err, ErrHostReserve) || ExitStatus(err) != 64 {
		t.Fatalf("zero: %v", err)
	}
	neg := `{
		"control_url": "https://control.example.invalid",
		"server_ca_file": "/etc/vpn-node/ca.pem",
		"trust_bundle": "/etc/vpn-node/trust.json",
		"datapath": {"member_ingress": "fv-fh", "uplink": "fv-ih", "snat": "192.0.2.37"},
		"agent_release": "0.0.0-test",
		"netbird_backend": "kernel",
		"host_reserve_bps": -1
	}`
	if err := os.WriteFile(path, []byte(neg), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil || errors.Is(err, ErrHostReserve) || ExitStatus(err) == 64 {
		t.Fatalf("negative should fail decode, got %v", err)
	}
	ok := `{
		"state_dir": "` + dir + `",
		"control_url": "https://control.example.invalid",
		"server_ca_file": "/etc/vpn-node/ca.pem",
		"trust_bundle": "/etc/vpn-node/trust.json",
		"datapath": {"member_ingress": "fv-fh", "uplink": "fv-ih", "snat": "192.0.2.37"},
		"agent_release": "0.0.0-test",
		"netbird_backend": "kernel",
		"host_reserve_bps": 1
	}`
	if err := os.WriteFile(path, []byte(ok), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HostReserveBPS != 1 || cfg.UsageDBPath() != filepath.Join(dir, "usage.db") {
		t.Fatalf("cfg %+v", cfg)
	}
	if ExitStatus(nil) != 0 || ExitStatus(errors.New("other")) != 1 {
		t.Fatal("status map")
	}
}

func TestDecideUsageCapsKeepMeterPath(t *testing.T) {
	base := servingInput()
	d := Decide(base)
	if !d.Serve {
		t.Fatalf("baseline refused: %v", d.Reasons)
	}

	zero := base
	zero.HostReserveBPS = 0
	d = Decide(zero)
	if d.Serve {
		t.Fatal("zero host reserve was substituted and served")
	}

	outbox := base
	outbox.UsageOutbox = true
	d = Decide(outbox)
	if d.Serve || !hasReason(d.Reasons, api.ForwardingBlockReasonOutboxFull) || hasReason(d.Reasons, api.ForwardingBlockReasonMeterPathInvalid) {
		t.Fatalf("outbox: serve %v reasons %v", d.Serve, d.Reasons)
	}
	if hasCode(d.Errors, api.ErrorCodeMETERPATHINVALID) {
		t.Fatal("usage outbox set METER_PATH_INVALID")
	}

	persist := base
	persist.UsagePersist = true
	d = Decide(persist)
	if d.Serve || !hasReason(d.Reasons, api.ForwardingBlockReasonMeterStorageFailed) || hasCode(d.Errors, api.ErrorCodeMETERPATHINVALID) {
		t.Fatalf("persist: %+v", d.Reasons)
	}
	if !hasMessage(d.Errors, "persist_failed") {
		t.Fatalf("persist message: %+v", d.Errors)
	}

	disk := base
	disk.UsageDisk = true
	d = Decide(disk)
	if d.Serve || !hasReason(d.Reasons, api.ForwardingBlockReasonDiskLow) || hasCode(d.Errors, api.ErrorCodeMETERPATHINVALID) {
		t.Fatalf("disk: %+v", d.Reasons)
	}
	if !hasMessage(d.Errors, "disk_space") {
		t.Fatalf("disk message: %+v", d.Errors)
	}

	path := base
	path.MeterFailed = true
	d = Decide(path)
	if d.Serve || !hasReason(d.Reasons, api.ForwardingBlockReasonMeterPathInvalid) || !hasCode(d.Errors, api.ErrorCodeMETERPATHINVALID) {
		t.Fatalf("meter path: reasons %v errors %+v", d.Reasons, d.Errors)
	}
}

func servingInput() Input {
	rel := uuid.MustParse("019a0000-0000-7000-8000-0000000000c1")
	issued := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	ds := &api.DesiredState{
		PublicationIntent: api.PublicationIntentPublished,
		ServingReleaseIds: []uuid.UUID{rel},
		ConfigGeneration:  1,
		NodeCapacity: presence.NewNullableWithValue(api.NodeCapacity{
			UpBps: 200_000_000, DownBps: 200_000_000,
		}),
		Dns: api.DnsServiceConfig{ResolverIpv4: []string{}, ResolverIpv6: []string{}},
	}
	lease := &api.ServingLease{
		ConfigGeneration: 1,
		IssuedAt:         issued,
		ExpiresAt:        issued.Add(time.Hour),
		Members:          []api.MemberLease{},
	}
	return Input{
		Now: issued, MonoNow: 2 * time.Second,
		Datapath: enforce.Datapath{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: netip.MustParseAddr("192.0.2.37")},
		Desired:  ds, DesiredOK: true,
		Lease: lease, LeaseFresh: true, LeaseMonoSent: time.Second,
		AnchorOK: true, Backend: "kernel", ReleaseID: rel,
		Enrolled: true, BaselineOK: true, StorageOK: true,
		HostReserveBPS: 1_000_000,
		ProbeRan:       true,
		ProbeCheck:     api.HealthCheckStatusPassed,
		ProbeHealth:    api.DataHealthHealthy,
	}
}

func hasReason(rs []api.ForwardingBlockReason, want api.ForwardingBlockReason) bool {
	for _, r := range rs {
		if r == want {
			return true
		}
	}
	return false
}

func hasCode(errs []api.NodeError, code api.ErrorCode) bool {
	for _, e := range errs {
		if e.Code == code {
			return true
		}
	}
	return false
}

func hasMessage(errs []api.NodeError, msg string) bool {
	for _, e := range errs {
		if e.Message == msg {
			return true
		}
	}
	return false
}
