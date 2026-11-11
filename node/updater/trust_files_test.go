package updater

import (
	"crypto/ed25519"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// §13.4, §16.2: a release-only store is parseable but cannot verify a
// central desired-state signature. The control export can verify that same
// document; verification alone does not grant forwarding authorization.
func TestUPD02_DesiredStateTrustFiles(t *testing.T) {
	for _, exported := range []bool{false, true} {
		name := "release_only_store"
		if exported {
			name = "central_export"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				// The provider has no exported clock injection. synctest controls
				// its time.Now as well as every verifier and fixture timestamp.
				now := time.Now().UTC()
				ctx := t.Context()
				root := t.TempDir()
				provider, err := signing.InitializeFileProvider(filepath.Join(root, "vault"))
				if err != nil {
					t.Fatal(err)
				}
				pub, _, err := ed25519.GenerateKey(nil)
				if err != nil {
					t.Fatal(err)
				}
				if err := provider.PinReleaseRoot(ctx, envelope.TrustedKey{
					KeyID: "test-release", PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease},
					NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
				}); err != nil {
					t.Fatal(err)
				}
				releaseOnly := nodeTrustExport(t, provider)
				for _, purpose := range []signing.Purpose{signing.DesiredState, signing.Lease, signing.Recovery, signing.LedgerEpoch} {
					if _, err := provider.Rotate(ctx, purpose, time.Hour, 0); err != nil {
						t.Fatal(err)
					}
				}
				centralExport := nodeTrustExport(t, provider)
				node := Node{ID: uuid.New(), Incarnation: uuid.New()}
				releaseID := uuid.New()
				desired := sampleDesired(node, releaseID, now)
				signed, err := provider.SignEnvelope(ctx, signing.DesiredState, desired)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := signed.MarshalCanonical()
				if err != nil {
					t.Fatal(err)
				}

				trustBytes := releaseOnly
				if exported {
					trustBytes = centralExport
				}
				trust, err := envelope.ParseNodeTrustStoreJSON(trustBytes)
				if err != nil {
					t.Fatal(err)
				}
				assertResult := func(work Work, err error) {
					t.Helper()
					if exported {
						if err != nil {
							t.Fatalf("central export rejected desired state: %v", err)
						}
						if work.Intent.ReleaseId != releaseID || work.PolicyGeneration != desired.ConfigGeneration {
							t.Fatal("verified desired state lost its update or policy generation")
						}
					} else if op := asOp(err); op == nil || op.Code != apierr.KeyUntrusted || op.Retry {
						t.Fatalf("release-only trust: got %v, want permanent KeyUntrusted", err)
					}
				}
				assertResult(WorkFromEnvelope(raw, trust, node, now, nil))
				path := filepath.Join(root, "desired-state.json")
				if err := os.WriteFile(path, raw, 0o600); err != nil {
					t.Fatal(err)
				}
				gate := FileGate{Path: filepath.Join(root, "forwarding.blocked")}
				source := GuardSource{
					Inner: FileSource{Path: path, Trust: trust, Node: node, Now: func() time.Time { return now }},
					Gate:  gate,
				}
				assertResult(source.Current(ctx))
				if blocked := gate.Blocked(); blocked != !exported {
					t.Fatalf("guard blocked=%v, want %v", blocked, !exported)
				}
			})
		})
	}
}

func nodeTrustExport(t *testing.T, provider *signing.FileProvider) []byte {
	t.Helper()
	// fvpn-vault's prepareExport serializes Snapshot.NodeTrust with exactly
	// this indentation and newline. Use its production producer here: the
	// command package is not importable. TestSEC01_NormalInspectExportAndNoLeakage
	// in that package separately pins these bytes against actual CLI output.
	snapshot, err := provider.Snapshot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(snapshot.NodeTrust, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}
