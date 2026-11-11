package installer

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// §6.3: copy the chosen input bytes without merging stores or rewriting JSON.
func TestPKG01_DesiredTrustCopies(t *testing.T) {
	now := func() time.Time { return time.Unix(0, 0).UTC() }
	release := installerTrustBytes(t, envelope.PurposeRelease, now())
	desired := installerTrustBytes(t, envelope.PurposeDesiredState, now())
	control := installerTrustBytes(t, envelope.PurposeLease, now())
	for _, explicit := range []bool{false, true} {
		name := "omitted_copies_release"
		if explicit {
			name = "explicit_copies_desired"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			layout := DefaultLayout().WithRoot(filepath.Join(root, "node"))
			env := &Env{Layout: layout, Now: now}
			journal, err := openJournal(env)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(layout.EtcDir, 0o755); err != nil {
				t.Fatal(err)
			}
			input := func(name string, raw []byte) string {
				t.Helper()
				path := filepath.Join(root, name)
				if err := os.WriteFile(path, raw, 0o600); err != nil {
					t.Fatal(err)
				}
				return path
			}
			req := InstallRequest{
				ServerCA:    input("ca.pem", []byte("TEST ONLY public CA input\n")),
				TrustPath:   input("release.json", release),
				TrustBundle: input("control.json", control),
			}
			want := release
			if explicit {
				req.DesiredTrustPath = input("desired.json", desired)
				want = desired
			}
			if err := writePublic(journal, layout, req); err != nil {
				t.Fatal(err)
			}
			for name, want := range map[string][]byte{
				"desired-trust.json": want,
				"release-trust.json": release,
				"trust.json":         control,
			} {
				got, err := os.ReadFile(filepath.Join(layout.EtcDir, name))
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Errorf("%s did not preserve the selected input bytes", name)
				}
			}
		})
	}
}

func installerTrustBytes(t *testing.T, purpose envelope.Purpose, now time.Time) []byte {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	trust, err := envelope.NewNodeTrustStore(envelope.TrustedKey{
		KeyID: "test-" + string(purpose), PublicKey: pub, Purposes: []envelope.Purpose{purpose},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(trust, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}
