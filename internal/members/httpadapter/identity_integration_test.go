//go:build integration

package httpadapter_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

// D-050: login and native users/peers are inputs. The real discovery job writes
// mappings and device bindings; HTTP reads and SQL assertions only observe them.
func (f *fixture) discover() {
	f.t.Helper()
	err := db.InTx(f.t.Context(), f.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		return db.EnqueueOutbox(f.t.Context(), tx, db.OutboxEvent{AggregateID: uuid.New(), AggregateType: "member_maintenance", AggregateVersion: "1", EventType: "devices.discover", SchemaVersion: 1, Payload: []byte(`{}`)})
	})
	if err != nil {
		f.t.Fatal(err)
	}
	f.drain()
}

func (f *fixture) identityCount(result string) float64 {
	f.t.Helper()
	families, err := f.identityMetrics.Gather()
	if err != nil {
		f.t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != "fvpn_netbird_identity_discovery_total" {
			continue
		}
		for _, metric := range family.Metric {
			if len(metric.Label) != 1 || metric.Label[0].GetName() != "result" {
				f.t.Fatal("identity metric contains an identity label")
			}
			if metric.Label[0].GetValue() == result {
				return metric.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func TestAT01_D052PublicSubjectDiscoveryAuthorizesNativePeer(t *testing.T) {
	f := newFixture(t)
	const subject = "TEST-PRIVATE-public-subject"
	member := f.login(subject, "member")
	checkout, _ := f.checkout(member)
	f.pay(member, checkout, "TEST-discovery-paid", f.now.Add(time.Hour))
	f.seedPeer(member, "TEST-native-peer", "100.64.0.21")
	if f.count(`SELECT count(*) FROM netbird_user_bindings`) != 0 || f.count(`SELECT count(*) FROM device_bindings`) != 0 {
		t.Fatal("fixture seeded a projection")
	}
	// GET /users must run with the member row unlocked: external calls never
	// occur in the mapping/admission transaction (§15.5).
	f.netbird.onUsersRead = func() error {
		tx, err := f.pool.Begin(t.Context())
		if err != nil {
			return err
		}
		defer tx.Rollback(t.Context())
		_, err = tx.Exec(t.Context(), `SELECT id FROM members WHERE id=$1 FOR UPDATE NOWAIT`, member.id)
		return err
	}
	f.discover()
	var owner, identity uuid.UUID
	var proof string
	var revision decstr.U64
	var verified time.Time
	err := f.pool.QueryRow(t.Context(), `SELECT member_id,identity_binding_id,mapping_proof,revision::text,verified_at FROM netbird_user_bindings WHERE deployment_id='TEST-ONLY' AND external_user_id=$1`, subject).Scan(&owner, &identity, &proof, &revision, &verified)
	if err != nil || owner != member.id || identity == uuid.Nil || proof != "shared_issuer_public_subject:"+f.config.OIDCIssuer || revision != 1 || verified.IsZero() {
		t.Fatal("incorrect public-subject proof", revision, err)
	}
	if f.count(`SELECT count(*) FROM identity_bindings WHERE id=$1 AND member_id=$2 AND subject=$3 AND issuer=$4`, identity, member.id, subject, f.config.OIDCIssuer) != 1 {
		t.Fatal("binding did not reference the authenticated login identity")
	}
	w := f.call(member, "GET", "/api/v1/member/devices", "", nil)
	requireStatus(t, w, 200)
	list := decode[api.MemberDeviceList](t, w)
	if len(list.Items) != 1 || list.Items[0].Status != api.DeviceStatusAuthorized || list.AuthorizedCount != 1 {
		t.Fatal("native device did not reach observed authorization", list)
	}
	device := list.Items[0].DeviceId
	if f.count(`SELECT count(*) FROM device_bindings WHERE device_id=$1 AND member_id=$2 AND overlay_ip='100.64.0.21'::inet`, device, member.id) != 1 {
		t.Fatal("device binding was not projected")
	}
	f.discover()
	var again time.Time
	var againRevision decstr.U64
	if err = f.pool.QueryRow(t.Context(), `SELECT revision::text,verified_at FROM netbird_user_bindings WHERE deployment_id='TEST-ONLY' AND external_user_id=$1`, subject).Scan(&againRevision, &again); err != nil || againRevision != revision || !again.Equal(verified) {
		t.Fatal("unchanged discovery bumped proof or verification timestamp", err)
	}
	if f.count(`SELECT count(*) FROM member_devices`) != 1 || f.count(`SELECT count(*) FROM device_bindings`) != 1 || f.identityCount("created") != 1 || f.identityCount("unchanged") != 1 {
		t.Fatal("discovery replay duplicated projection work")
	}
}

func TestAT02_D052UnmappedNativeUsersFailClosed(t *testing.T) {
	for _, tc := range []struct {
		name, user, outcome string
		service, blocked    bool
	}{
		{"pairwise", "TEST-PRIVATE-netbird-pairwise-subject", "identity_unmapped", false, false},
		{"unknown", "TEST-PRIVATE-unknown-subject", "identity_unmapped", false, false},
		{"service", "TEST-PRIVATE-portal-subject", "service_user", true, false},
		{"blocked", "TEST-PRIVATE-portal-subject", "blocked_user", false, true},
		{"wrong issuer", "TEST-PRIVATE-portal-subject", "identity_unmapped", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			member := f.login("TEST-PRIVATE-portal-subject", "member")
			if tc.name == "wrong issuer" {
				f.identityMetrics = prometheus.NewRegistry()
				var err error
				f.services.Adapter.Identity, err = devices.NewIdentityDiscovery("https://different-issuer.example.invalid", f.identityMetrics, slog.New(slog.NewTextHandler(&f.logs, nil)))
				if err != nil {
					t.Fatal(err)
				}
			}
			f.seedNative(tc.user, "TEST-native-peer", "100.64.0.22", tc.service, tc.blocked)
			f.discover()
			if f.count(`SELECT count(*) FROM netbird_user_bindings`) != 0 || f.count(`SELECT count(*) FROM device_bindings`) != 0 || f.count(`SELECT count(*) FROM member_devices`) != 0 {
				t.Fatal("unverified identity projected a binding")
			}
			_, err := f.services.HTTP.Devices.ObserveAndAuthorize(t.Context(), entitlement.Command{Actor: entitlement.Actor{ID: uuid.New(), Role: "system", RequestID: uuid.New()}, Key: uuid.New()}, "TEST-ONLY", "TEST-native-peer")
			if !errors.Is(err, devices.ErrUnmapped) {
				t.Fatal("unmapped admission did not fail closed", err)
			}
			w := f.call(member, "GET", "/api/v1/member/devices", "", nil)
			requireStatus(t, w, 200)
			if page := decode[api.MemberDeviceList](t, w); len(page.Items) != 0 || page.AuthorizedCount != 0 {
				t.Fatal("unmapped peer exposed as a device", page)
			}
			if f.identityCount(tc.outcome) != 1 {
				t.Fatal("missing counted discovery outcome", tc.outcome)
			}
			if tc.outcome == "identity_unmapped" && !strings.Contains(f.logs.String(), "reason=identity_unmapped") {
				t.Fatal("unmapped identity not logged")
			}
			if strings.Contains(f.logs.String(), tc.user) {
				t.Fatal("identity discovery logged a provider identifier")
			}
		})
	}
}

func TestAT01_D052ChangedProofAdvancesRevisionOnce(t *testing.T) {
	f := newFixture(t)
	member := f.login("TEST-proof-subject", "member")
	f.seedPeer(member, "TEST-native-peer", "100.64.0.25")
	f.discover()
	// Fault input: an earlier proof must be reverified without changing identity.
	f.exec(`UPDATE netbird_user_bindings SET mapping_proof='TEST ONLY previous proof',revision=revision+1 WHERE member_id=$1`, member.id)
	f.discover()
	f.discover()
	if f.count(`SELECT count(*) FROM netbird_user_bindings n JOIN identity_bindings i ON i.id=n.identity_binding_id AND i.member_id=n.member_id WHERE n.member_id=$1 AND n.revision=3 AND n.mapping_proof=$2 AND i.issuer=$3 AND i.subject=n.external_user_id`, member.id, "shared_issuer_public_subject:"+f.config.OIDCIssuer, f.config.OIDCIssuer) != 1 || f.identityCount("proof_updated") != 1 || f.identityCount("unchanged") != 1 {
		t.Fatal("proof update was not monotonic and idempotent")
	}
}

func TestAT04_D052BlockedUserCanStillBeWithdrawn(t *testing.T) {
	f := newFixture(t)
	const subject = "TEST-blocked-after-admission"
	member := f.login(subject, "member")
	checkout, _ := f.checkout(member)
	f.pay(member, checkout, "TEST-before-block", f.now.Add(time.Hour))
	device := f.authorize(member, "TEST-native-peer", "100.64.0.26")
	f.netbird.mu.Lock()
	f.netbird.objects["users"][subject]["is_blocked"] = true
	f.netbird.mu.Unlock()
	if err := f.services.HTTP.Devices.SyncDevice(t.Context(), device); !errors.Is(err, devices.ErrUnmapped) {
		t.Fatal("blocked user was not denied", err)
	}
	observed, err := f.services.Adapter.ObservePeer(t.Context(), "TEST-ONLY", "TEST-native-peer")
	if err != nil || observed.Authorized || !observed.IdentityUnmapped {
		t.Fatal("blocked user's paid memberships were not withdrawn", err)
	}
	if f.count(`SELECT count(*) FROM device_authorization_observations WHERE device_id=$1 AND observed_authorized`, device) != 0 {
		t.Fatal("blocked user retained an authorized observation")
	}
}

func TestAT02_D052ConflictingBindingIsNotReassigned(t *testing.T) {
	f := newFixture(t)
	member := f.login("TEST-PRIVATE-expected-subject", "member")
	other := f.login("TEST-PRIVATE-other-subject", "member")
	f.seedPeer(member, "TEST-native-peer", "100.64.0.23")
	// Fault input only: simulate an existing incompatible mapping. Positive
	// cases never seed this projection; discovery must not overwrite history.
	f.exec(`INSERT INTO netbird_user_bindings(member_id,deployment_id,external_user_id,identity_binding_id,mapping_proof,revision,verified_at) SELECT id,'TEST-ONLY','TEST-PRIVATE-expected-subject',canonical_identity_ref,'TEST ONLY incompatible historical proof',7,clock_timestamp() FROM members WHERE id=$1`, other.id)
	snapshot := func() string {
		t.Helper()
		var raw string
		if err := f.pool.QueryRow(t.Context(), `SELECT row_to_json(n)::text FROM netbird_user_bindings n`).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	before := snapshot()
	for range 2 {
		f.discover()
	}
	if after := snapshot(); after != before || f.identityCount("identity_conflict") != 2 || !strings.Contains(f.logs.String(), "reason=identity_conflict") {
		t.Fatal("conflicting binding was overwritten or anomaly not recorded")
	}
	if f.count(`SELECT count(*) FROM member_devices`) != 0 || f.count(`SELECT count(*) FROM device_bindings`) != 0 {
		t.Fatal("conflicting mapping authorized a peer")
	}
	_, err := f.services.HTTP.Devices.ObserveAndAuthorize(t.Context(), entitlement.Command{Actor: entitlement.Actor{ID: uuid.New(), Role: "system", RequestID: uuid.New()}, Key: uuid.New()}, "TEST-ONLY", "TEST-native-peer")
	if !errors.Is(err, devices.ErrUnmapped) {
		t.Fatal("direct admission trusted the conflicting legacy proof", err)
	}
}

func TestAT02_D052ConcurrentDiscoveryHasOneStableProof(t *testing.T) {
	f := newFixture(t)
	member := f.login("TEST-concurrent-subject", "member")
	f.seedPeer(member, "TEST-native-peer", "100.64.0.24")
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for range 2 {
		wg.Go(func() { errors <- f.services.Adapter.Discover(context.Background(), f.services.HTTP.Devices) })
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if f.count(`SELECT count(*) FROM netbird_user_bindings WHERE member_id=$1 AND revision=1`, member.id) != 1 || f.identityCount("created") != 1 || f.identityCount("unchanged") != 1 {
		t.Fatal("concurrent discoveries duplicated or bumped the binding")
	}
	if f.count(`SELECT count(*) FROM device_bindings`) != 0 {
		t.Fatal("pending-payment member was admitted")
	}
}
