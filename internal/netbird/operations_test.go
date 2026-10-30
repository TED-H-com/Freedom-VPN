package netbird

import "testing"

func TestD052_F03ProofRevisionDoesNotChangeImmutableIdentity(t *testing.T) {
	_, client := newFake(t)
	d := MemberDevice{MemberID: "TEST-member", DeviceID: "TEST-device", PeerID: "TEST-device", UserID: "TEST-user", IdentityBindingID: "TEST-binding", MappingRevision: "1"}
	if _, err := client.EnsureMemberDevice(t.Context(), d); err != nil {
		t.Fatal(err)
	}
	before, _, err := client.store.Get(t.Context(), client.deployment, "users", d.MemberID)
	if err != nil {
		t.Fatal(err)
	}
	d.MappingRevision = "2"
	if _, err = client.EnsureMemberDevice(t.Context(), d); err != nil {
		t.Fatal(err)
	}
	after, _, err := client.store.Get(t.Context(), client.deployment, "users", d.MemberID)
	if err != nil || after.DesiredHash != before.DesiredHash {
		t.Fatal("revision changed immutable proof", err)
	}
	d.IdentityBindingID = "TEST-other-binding"
	if _, err = client.EnsureMemberDevice(t.Context(), d); !IsKind(err, Conflict) {
		t.Fatal("foreign identity tuple accepted", err)
	}
}
