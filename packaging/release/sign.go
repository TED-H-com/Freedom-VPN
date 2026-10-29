package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// Sign signs the canonical signed object with a release key (§13.3, §16.2).
// The returned bytes are the canonical JSON of the whole envelope.
// EnvelopeSHA256 of those bytes is what update intents call manifest_sha256.
func Sign(signedJSON []byte, key SigningKey) (env envelope.Envelope, canonical []byte, err error) {
	if err := ValidateSigned(signedJSON); err != nil {
		return envelope.Envelope{}, nil, err
	}
	env, err = envelope.Sign(signedJSON, key.Private(), key.KeyID)
	if err != nil {
		return envelope.Envelope{}, nil, err
	}
	canonical, err = env.MarshalCanonical()
	if err != nil {
		return envelope.Envelope{}, nil, err
	}
	if err := ValidateEnvelope(canonical); err != nil {
		return envelope.Envelope{}, nil, err
	}
	return env, canonical, nil
}

// EnvelopeSHA256 is the lowercase hex SHA-256 of the canonical envelope.
// Desired-state update intents store this as manifest_sha256 (§13.10.2).
func EnvelopeSHA256(canonicalEnvelope []byte) (string, error) {
	c, err := canonjson.Canonicalize(canonicalEnvelope)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(c)
	return hex.EncodeToString(sum[:]), nil
}

// Verify checks the envelope signature, the schema, and that both baseline
// architectures are present. It returns the canonical signed object.
func Verify(raw []byte, ts *envelope.TrustStore, now time.Time) (signed []byte, keyID string, err error) {
	if err := ValidateEnvelope(raw); err != nil {
		return nil, "", err
	}
	signed, keyID, err = envelope.Verify(raw, ts, envelope.PurposeRelease, now)
	if err != nil {
		return nil, "", err
	}
	var manifest api.ReleaseManifest
	if err := json.Unmarshal(signed, &manifest); err != nil {
		return nil, "", fmt.Errorf("release: signed object: %w", err)
	}
	if err := requireBaseline(manifest.Artifacts); err != nil {
		return nil, "", err
	}
	return signed, keyID, nil
}

// ParsedManifest unmarshals a canonical signed object.
func ParsedManifest(signed []byte) (api.ReleaseManifest, error) {
	var m api.ReleaseManifest
	if err := json.Unmarshal(signed, &m); err != nil {
		return api.ReleaseManifest{}, err
	}
	return m, nil
}

// CheckArtifact confirms that path is the manifest entry for architecture:
// the file size and the raw SHA-256 match, and the architecture exists.
// A mismatch is never treated as success because the transport used TLS
// (§13.4, UPD-A04).
func CheckArtifact(manifest api.ReleaseManifest, architecture, path string) error {
	var found *api.ManifestArtifact
	for i := range manifest.Artifacts {
		a := &manifest.Artifacts[i]
		if string(a.Architecture) == architecture && string(a.OsId) == BaselineOSID {
			for _, v := range a.OsVersions {
				if v == BaselineOSVersion {
					found = a
				}
			}
		}
	}
	if found == nil {
		return fmt.Errorf("release: UNSUPPORTED_PLATFORM: no ubuntu %s %s artifact", BaselineOSVersion, architecture)
	}
	sum, size, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if size != int64(found.SizeBytes) {
		return fmt.Errorf("release: SIZE_MISMATCH: %s is %d bytes, manifest says %s", architecture, size, found.SizeBytes)
	}
	if sum != found.Sha256 {
		return fmt.Errorf("release: HASH_MISMATCH: %s", architecture)
	}
	return nil
}

// WriteFile writes data with mode 0644, replacing path.
func WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
