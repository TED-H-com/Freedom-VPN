//go:build integration

package auth_test

import (
	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestAT01_AdminIdentityAndEverySessionAudited(t *testing.T) {
	pool := dbtest.New(t)
	store := auth.PostgresStore{Pool: pool}
	identity := auth.Identity{Issuer: "https://idp.example.invalid", ClientScope: "portal", Subject: "admin-subject", Role: auth.Admin, MFA: true, Label: "Test Admin", AuthTime: time.Now(), Expires: time.Now().Add(time.Hour)}
	p, err := store.Login(t.Context(), identity, "admin-old", "", time.Hour, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.Login(t.Context(), identity, "admin-new", "admin-old", time.Hour, uuid.New())
	if err != nil || p.ID != again.ID {
		t.Fatal("unstable admin identity", err)
	}
	if _, err = store.Session(t.Context(), "admin-old", time.Minute); err == nil {
		t.Fatal("old admin session remained")
	}
	var members, bindings, logins int
	if err = pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM members),(SELECT count(*) FROM identity_bindings),(SELECT count(*) FROM audit_events WHERE action='session.login')`).Scan(&members, &bindings, &logins); err != nil || members != 0 || bindings != 0 || logins != 2 {
		t.Fatal(members, bindings, logins, err)
	}
	identity.Role = auth.Member
	member, err := store.Login(t.Context(), identity, "member-token", "admin-new", time.Hour, uuid.New())
	if err != nil || member.ID == p.ID {
		t.Fatal("admin and member identities conflated", err)
	}
	if _, err = store.Session(t.Context(), "admin-new", time.Minute); err != nil {
		t.Fatal("member login clobbered admin", err)
	}
	if err = store.Logout(t.Context(), "admin-new"); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Session(t.Context(), "member-token", time.Minute); err != nil {
		t.Fatal("admin logout clobbered member", err)
	}
	if err = store.Logout(t.Context(), "member-token"); err != nil {
		t.Fatal(err)
	}
	reader := audit.Store{Pool: pool}
	page, err := reader.List(t.Context(), api.ListAuditEventsParams{})
	if err != nil || len(page.Items) != 6 || reader.AnomalyCount() != 0 {
		t.Fatal("audit conformance", len(page.Items), reader.AnomalyCount(), err)
	}
	for _, event := range page.Items {
		if event.Actor.Type == api.AuditEventActorTypeAdmin && (event.Resource.Type != api.ResourceTypeFleetConfig || event.Resource.Id != p.ID) {
			t.Fatal("admin audit reference", event.Resource)
		}
	}
	identity.Role = auth.Admin
	identity.MFA = false
	if _, err = store.Login(t.Context(), identity, "no-mfa", "", time.Hour, uuid.New()); err == nil {
		t.Fatal("admin without MFA")
	}
}
