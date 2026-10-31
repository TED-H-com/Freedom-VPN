package installer

import (
	"os"
	"strings"
	"testing"
)

func markerDoc(objects ...string) []byte {
	return []byte(`{"nftables":[` + strings.Join(objects, ",") + `]}`)
}

func TestReviewR17_LegacyMarkerDeleteCannotBeRedirected(t *testing.T) {
	l := DefaultLayout()
	exact := `{"chain":{"family":"inet","table":"fvpn","name":"owner","handle":7,"comment":"managed-by=fvpn-node-install"}}`
	st, err := parseMarker(markerDoc(exact), l)
	if err != nil || !st.LegacyOwner || st.Present || st.Table != "fvpn_install" {
		t.Fatalf("legacy marker %+v %v", st, err)
	}

	hooked := `{"chain":{"family":"inet","table":"fvpn","name":"owner","handle":7,"hook":"forward","comment":"managed-by=fvpn-node-install"}}`
	if _, err = parseMarker(markerDoc(hooked), l); err == nil || AsError(err).Code != "NFT_MARKER" {
		t.Fatalf("hooked legacy chain was eligible for deletion: %v", err)
	}
	ruled := `{"rule":{"family":"inet","table":"fvpn","chain":"owner","handle":8,"comment":"managed-by=fvpn-node-install","expr":[]}}`
	if _, err = parseMarker(markerDoc(exact, ruled), l); err == nil || AsError(err).Code != "NFT_MARKER" {
		t.Fatalf("a rule on the legacy chain was eligible for deletion: %v", err)
	}
	other := `{"chain":{"family":"inet","table":"evil","name":"owner","handle":1,"comment":"managed-by=fvpn-node-install"}}`
	st, err = parseMarker(markerDoc(other), l)
	if err != nil || st.LegacyOwner || st.Present {
		t.Fatalf("foreign table became the delete target: %+v %v", st, err)
	}
	ip := `{"chain":{"family":"ip","table":"fvpn","name":"owner","handle":1,"comment":"managed-by=fvpn-node-install"}}`
	st, err = parseMarker(markerDoc(ip), l)
	if err != nil || st.LegacyOwner {
		t.Fatalf("ip family became the delete target: %+v %v", st, err)
	}
	forward := `{"chain":{"family":"inet","table":"fvpn","name":"forward","handle":1,"comment":"managed-by=fvpn-node-install"}}`
	st, err = parseMarker(markerDoc(forward), l)
	if err != nil || st.LegacyOwner {
		t.Fatalf("a non-owner chain became the delete target: %+v %v", st, err)
	}

	raw, err := os.ReadFile("isolation.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	needle := `env.run(ctx, "nft", "delete", "chain", "inet", env.layout().NFTTable, "owner")`
	if !strings.Contains(text, needle) {
		t.Fatal("legacy delete argv is not pinned to the layout table and the owner chain")
	}
	if strings.Contains(text, "delete\", \"chain\", \"inet\", v.") || strings.Contains(text, "nft delete chain inet "+`"`) {
		t.Fatal("delete argv interpolates a listing field")
	}
}

func TestReviewR17_HookedInstallTableIsRefusedBeforeDelete(t *testing.T) {
	l := DefaultLayout()
	doc := markerDoc(
		`{"table":{"family":"inet","name":"fvpn_install","handle":1}}`,
		`{"chain":{"family":"inet","table":"fvpn_install","name":"owner","handle":2,"hook":"forward","comment":"managed-by=fvpn-node-install"}}`,
	)
	_, err := parseMarker(doc, l)
	if err == nil || AsError(err).Code != "NFT_MARKER" {
		t.Fatalf("hooked fvpn_install: %v", err)
	}
}
