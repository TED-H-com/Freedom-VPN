package updater

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestUPD04_DecideRollback(t *testing.T) {
	ok := RollbackInput{HasLastGood: true, TreeOK: true, SignatureOK: true, InServingSet: true, SchemaReadable: true}
	cases := []struct {
		name    string
		in      RollbackInput
		restore bool
		code    apierr.Code
	}{
		{"no last good", RollbackInput{}, false, apierr.RollbackFailed},
		{"bad tree", RollbackInput{HasLastGood: true}, false, apierr.RollbackFailed},
		{"bad signature", RollbackInput{HasLastGood: true, TreeOK: true}, false, apierr.SignatureInvalid},
		{"revoked", func() RollbackInput { in := ok; in.Revoked = true; return in }(), false, apierr.ReleaseNotApproved},
		{"below minimum", func() RollbackInput { in := ok; in.InServingSet = false; return in }(), false, apierr.MinimumVersionViolation},
		{"schema", func() RollbackInput { in := ok; in.SchemaReadable = false; return in }(), false, apierr.SchemaIncompatible},
		{"restore", ok, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DecideRollback(tc.in)
			if got.Restore != tc.restore || got.Code != tc.code {
				t.Fatalf("plan %+v, want restore %v code %s", got, tc.restore, tc.code)
			}
		})
	}
}

func TestUPD_VersionAtLeastNumeric(t *testing.T) {
	if !versionAtLeast("1.10.0", "1.9.0") || !versionAtLeast("1.2.0", "1.2.0") || !versionAtLeast("1.2.0", "1.0.0") {
		t.Fatal("expected numeric greater-or-equal")
	}
	if versionAtLeast("1.9.0", "1.10.0") || versionAtLeast("", "1.0.0") || versionAtLeast("1.0.0", "") {
		t.Fatal("expected rejection")
	}
}

func TestUPD_SelectArtifact(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	m := api.ReleaseManifest{ReleaseId: id, Artifacts: []api.ManifestArtifact{
		{OsId: "ubuntu", OsVersions: []string{"24.04"}, Architecture: "arm64", Format: "tar.zst", Sha256: "aa", SizeBytes: decstr.U64(1), InstalledSizeBytes: decstr.U64(1), Url: "https://example.invalid/a"},
	}}
	if _, err := selectArtifact(m, "ubuntu", "24.04", "amd64"); err == nil {
		t.Fatal("amd64 was selected from an arm64 manifest")
	}
}
