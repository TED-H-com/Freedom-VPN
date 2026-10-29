package signing

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// KeyRecord is public metadata. Retired keys verify through NotAfter, but
// cannot sign. Revoked is terminal and invalidates verification immediately.
type KeyRecord struct {
	ID        string     `json:"id"`
	Purpose   Purpose    `json:"purpose"`
	Public    []byte     `json:"public,omitempty"`
	State     string     `json:"state"`
	NotBefore time.Time  `json:"not_before"`
	NotAfter  time.Time  `json:"not_after"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}
type ledger struct {
	Revision         decstr.U64  `json:"revision"`
	ReleaseRevision  decstr.U64  `json:"release_revision"`
	Keys             []KeyRecord `json:"keys"`
	DeniedIdentities []string    `json:"denied_identities,omitempty"`
}

type FileProvider struct {
	dir string
	now func() time.Time
}

// InitializeFileProvider explicitly creates a new, empty development vault.
// OpenFileProvider never recreates missing state (a restore must fail closed).
func InitializeFileProvider(dir string) (*FileProvider, error) {
	if err := os.Mkdir(dir, 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "lock"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	p := &FileProvider{dir: dir, now: time.Now}
	if err = p.save(ledger{Keys: []KeyRecord{}}); err != nil {
		return nil, err
	}
	return p, nil
}
func OpenFileProvider(dir string) (*FileProvider, error) {
	p := &FileProvider{dir: dir, now: time.Now}
	err := p.with(context.Background(), func(*ledger) error { return nil })
	return p, err
}

// with uses a cross-process lock: cached provider handles never cache key state.
func (p *FileProvider) with(ctx context.Context, fn func(*ledger) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	info, err := os.Lstat(p.dir)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return ErrUnavailable
	}
	f, err := os.OpenFile(filepath.Join(p.dir, "lock"), os.O_RDWR, 0)
	if err != nil {
		return ErrUnavailable
	}
	defer f.Close()
	for {
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if err != syscall.EWOULDBLOCK {
			return ErrUnavailable
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	raw, err := p.read("state.json")
	if err != nil {
		return err
	}
	var state ledger
	if err = json.Unmarshal(raw, &state); err != nil {
		return ErrUnavailable
	}
	if err = validateLedger(state); err != nil {
		return err
	}
	return fn(&state)
}

func validateLedger(s ledger) error {
	ids, active, pubs := map[string]bool{}, map[Purpose]bool{}, map[string]bool{}
	for _, k := range s.Keys {
		if _, err := uuid.Parse(k.ID); err != nil {
			return ErrUnavailable
		}
		if ids[k.ID] || !k.NotBefore.Before(k.NotAfter) || k.NotBefore.IsZero() {
			return ErrUnavailable
		}
		ids[k.ID] = true
		if k.State != "active" && k.State != "retired" && k.State != "revoked" {
			return ErrUnavailable
		}
		if (k.State == "revoked") != (k.RevokedAt != nil) {
			return ErrUnavailable
		}
		if k.Purpose != Release && k.State == "active" {
			if active[k.Purpose] {
				return ErrUnavailable
			}
			active[k.Purpose] = true
		}
		switch {
		case central(k.Purpose) || k.Purpose == Release:
			if len(k.Public) != ed25519.PublicKeySize {
				return ErrUnavailable
			}
		case k.Purpose == NodeCA:
			pub, err := x509.ParsePKIXPublicKey(k.Public)
			if err != nil {
				return ErrUnavailable
			}
			key, ok := pub.(*ecdsa.PublicKey)
			if !ok || key.Curve != elliptic.P256() {
				return ErrUnavailable
			}
		case k.Purpose == BootstrapSecret:
			if len(k.Public) != 0 {
				return ErrUnavailable
			}
		default:
			return ErrUnavailable
		}
		if len(k.Public) > 0 {
			if pubs[string(k.Public)] {
				return ErrUnavailable
			}
			pubs[string(k.Public)] = true
		}
	}
	return nil
}

func (p *FileProvider) read(name string) ([]byte, error) {
	path := filepath.Join(p.dir, name)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, ErrUnavailable
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, ErrUnavailable
	}
	return b, nil
}
func (p *FileProvider) save(s ledger) error {
	if err := validateLedger(s); err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(p.dir, ".state-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(f.Name(), filepath.Join(p.dir, "state.json")); err != nil {
		return err
	}
	d, err := os.Open(p.dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// Rotate atomically retires the previous signer. Its verification window is
// capped at now+overlap. The monotonic ledger is independent of PostgreSQL.
func (p *FileProvider) Rotate(ctx context.Context, purpose Purpose, validity, overlap time.Duration) (string, error) {
	if (!central(purpose) && purpose != NodeCA && purpose != BootstrapSecret) || validity <= 0 || overlap < 0 {
		return "", ErrUnavailable
	}
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	err = p.with(ctx, func(s *ledger) error {
		now := p.now().UTC()
		k := KeyRecord{ID: id.String(), Purpose: purpose, State: "active", NotBefore: now, NotAfter: now.Add(validity)}
		var raw []byte
		switch purpose {
		case BootstrapSecret:
			raw = make([]byte, 32)
			if _, err = rand.Read(raw); err != nil {
				return err
			}
		case NodeCA:
			var key *ecdsa.PrivateKey
			key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				return err
			}
			k.Public, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
			if err != nil {
				return err
			}
			raw, err = x509.MarshalPKCS8PrivateKey(key)
		default:
			var pub ed25519.PublicKey
			var key ed25519.PrivateKey
			pub, key, err = ed25519.GenerateKey(rand.Reader)
			if err != nil {
				return err
			}
			k.Public = pub
			raw, err = x509.MarshalPKCS8PrivateKey(key)
		}
		if err != nil {
			return err
		}
		f, err := os.OpenFile(filepath.Join(p.dir, id.String()+".key"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		if _, err = f.Write(raw); err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		for i := range s.Keys {
			old := &s.Keys[i]
			if old.Purpose == purpose && old.State == "active" {
				old.State = "retired"
				if now.Add(overlap).Before(old.NotAfter) {
					old.NotAfter = now.Add(overlap)
				}
				if !old.NotBefore.Before(old.NotAfter) {
					old.NotAfter = old.NotBefore.Add(time.Nanosecond)
				}
			}
		}
		s.Keys = append(s.Keys, k)
		if err = advance(s); err != nil {
			return err
		}
		return p.save(*s)
	})
	return id.String(), err
}
func advance(s *ledger) error {
	if s.Revision == decstr.U64(^uint64(0)) {
		return ErrUnavailable
	}
	s.Revision++
	return nil
}
func (p *FileProvider) Revoke(ctx context.Context, id string) error {
	return p.with(ctx, func(s *ledger) error {
		for i := range s.Keys {
			k := &s.Keys[i]
			if k.ID == id {
				if k.State == "revoked" {
					return nil
				}
				now := p.now().UTC()
				k.State = "revoked"
				k.RevokedAt = &now
				if err := advance(s); err != nil {
					return err
				}
				return p.save(*s)
			}
		}
		return ErrUnavailable
	})
}
func usable(k KeyRecord, now time.Time) bool {
	return k.State == "active" && !now.Before(k.NotBefore) && now.Before(k.NotAfter)
}
func active(s *ledger, purpose Purpose, now time.Time) (KeyRecord, error) {
	for _, k := range s.Keys {
		if k.Purpose == purpose && usable(k, now) {
			return k, nil
		}
	}
	return KeyRecord{}, ErrUnavailable
}
func (p *FileProvider) private(k KeyRecord) (crypto.Signer, error) {
	raw, err := p.read(k.ID + ".key")
	if err != nil {
		return nil, err
	}
	key, err := x509.ParsePKCS8PrivateKey(raw)
	if err != nil {
		return nil, ErrUnavailable
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, ErrUnavailable
	}
	var pub []byte
	if k.Purpose == NodeCA {
		pub, err = x509.MarshalPKIXPublicKey(signer.Public())
	} else if ed, ok := signer.Public().(ed25519.PublicKey); ok {
		pub = ed
	}
	if err != nil || !bytes.Equal(pub, k.Public) {
		return nil, ErrUnavailable
	}
	return signer, nil
}
func (p *FileProvider) SignEnvelope(ctx context.Context, purpose Purpose, v any) (out envelope.Envelope, err error) {
	if !central(purpose) {
		return out, ErrUnavailable
	}
	err = p.with(ctx, func(s *ledger) error {
		k, e := active(s, purpose, p.now())
		if e != nil {
			return e
		}
		key, e := p.private(k)
		if e != nil {
			return e
		}
		priv, ok := key.(ed25519.PrivateKey)
		if !ok {
			return ErrUnavailable
		}
		out, e = envelope.Sign(v, priv, k.ID)
		return e
	})
	return
}

type caHandle struct {
	provider *FileProvider
	ctx      context.Context
	id       string
	public   crypto.PublicKey
}

func (h *caHandle) Public() crypto.PublicKey { return h.public }
func (h *caHandle) Sign(r io.Reader, digest []byte, opts crypto.SignerOpts) (sig []byte, err error) {
	err = h.provider.with(h.ctx, func(s *ledger) error {
		for _, k := range s.Keys {
			if k.ID == h.id && k.Purpose == NodeCA && usable(k, h.provider.now()) {
				key, e := h.provider.private(k)
				if e != nil {
					return e
				}
				sig, e = key.Sign(r, digest, opts)
				return e
			}
		}
		return ErrUnavailable
	})
	return
}
func (p *FileProvider) CertificateSigner(ctx context.Context, id string) (out crypto.Signer, err error) {
	err = p.with(ctx, func(s *ledger) error {
		for _, k := range s.Keys {
			if k.ID == id && k.Purpose == NodeCA && usable(k, p.now()) {
				pub, e := x509.ParsePKIXPublicKey(k.Public)
				if e != nil {
					return ErrUnavailable
				}
				out = &caHandle{p, ctx, id, pub}
				return nil
			}
		}
		return ErrUnavailable
	})
	return
}
func (p *FileProvider) aead(k KeyRecord) (cipher.AEAD, error) {
	raw, err := p.read(k.ID + ".key")
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, ErrUnavailable
	}
	return cipher.NewGCM(block)
}
func (p *FileProvider) Seal(ctx context.Context, plain, aad []byte) (out Sealed, err error) {
	err = p.with(ctx, func(s *ledger) error {
		k, e := active(s, BootstrapSecret, p.now())
		if e != nil {
			return e
		}
		a, e := p.aead(k)
		if e != nil {
			return e
		}
		n := make([]byte, a.NonceSize())
		if _, e = rand.Read(n); e != nil {
			return e
		}
		out = Sealed{k.ID, a.Seal(n, n, plain, aad)}
		return nil
	})
	return
}
func (p *FileProvider) Open(ctx context.Context, sealed Sealed, aad []byte) (out []byte, err error) {
	err = p.with(ctx, func(s *ledger) error {
		for _, k := range s.Keys {
			if k.ID == sealed.KeyID && k.Purpose == BootstrapSecret && k.State != "revoked" {
				a, e := p.aead(k)
				if e != nil {
					return e
				}
				n := a.NonceSize()
				if len(sealed.Ciphertext) < n {
					return ErrUnavailable
				}
				out, e = a.Open(nil, sealed.Ciphertext[:n], sealed.Ciphertext[n:], aad)
				if e != nil {
					return ErrUnavailable
				}
				return nil
			}
		}
		return ErrUnavailable
	})
	return
}
func bundle(s *ledger, releaseOnly bool) (*envelope.TrustStore, error) {
	var keys []envelope.TrustedKey
	for _, k := range s.Keys {
		purpose := envelope.Purpose(k.Purpose)
		if !purpose.Valid() || (releaseOnly && k.Purpose != Release) {
			continue
		}
		keys = append(keys, envelope.TrustedKey{KeyID: k.ID, PublicKey: k.Public, Purposes: []envelope.Purpose{purpose}, NotBefore: k.NotBefore, NotAfter: k.NotAfter, Revoked: k.State == "revoked"})
	}
	return envelope.NewTrustStore(keys...)
}
func (p *FileProvider) TrustBundle(ctx context.Context) (out *envelope.TrustStore, err error) {
	err = p.with(ctx, func(s *ledger) error { var e error; out, e = bundle(s, false); return e })
	return
}
func (p *FileProvider) VerifyRelease(ctx context.Context, raw []byte) (signed []byte, err error) {
	err = p.with(ctx, func(s *ledger) error {
		ts, e := bundle(s, true)
		if e != nil {
			return e
		}
		signed, _, e = envelope.Verify(raw, ts, envelope.PurposeRelease, p.now())
		return e
	})
	return
}

// PinReleaseRoot is a one-time out-of-band provisioning operation. No release
// private key is accepted by this provider, including during rotations.
func (p *FileProvider) PinReleaseRoot(ctx context.Context, k envelope.TrustedKey) error {
	return p.with(ctx, func(s *ledger) error {
		for _, old := range s.Keys {
			if old.Purpose == Release {
				return ErrUnavailable
			}
		}
		if len(k.Purposes) != 1 || k.Purposes[0] != envelope.PurposeRelease || k.Revoked {
			return ErrUnavailable
		}
		if _, err := envelope.NewTrustStore(k); err != nil {
			return err
		}
		s.Keys = append(s.Keys, KeyRecord{ID: k.KeyID, Purpose: Release, Public: k.PublicKey, State: "active", NotBefore: k.NotBefore, NotAfter: k.NotAfter})
		if err := advance(s); err != nil {
			return err
		}
		return p.save(*s)
	})
}

// ReleaseUpdate must be signed by the currently trusted release chain. Revision
// is monotonic; existing key windows may only narrow and revocations persist.
type ReleaseUpdate struct {
	Kind     string      `json:"kind"`
	Revision decstr.U64  `json:"revision"`
	Keys     []KeyRecord `json:"keys"`
}

func (p *FileProvider) UpdateReleaseTrust(ctx context.Context, raw []byte) error {
	return p.with(ctx, func(s *ledger) error {
		ts, err := bundle(s, true)
		if err != nil {
			return err
		}
		signed, _, err := envelope.Verify(raw, ts, envelope.PurposeRelease, p.now())
		if err != nil {
			return err
		}
		var u ReleaseUpdate
		d := json.NewDecoder(bytes.NewReader(signed))
		d.DisallowUnknownFields()
		if err = d.Decode(&u); err != nil || u.Kind != "fvpn-release-trust-v1" || u.Revision <= s.ReleaseRevision || len(u.Keys) == 0 {
			return ErrUnavailable
		}
		seen := map[string]bool{}
		for _, k := range u.Keys {
			if k.Purpose != Release || seen[k.ID] {
				return ErrUnavailable
			}
			seen[k.ID] = true
			found := false
			for i, old := range s.Keys {
				if old.ID != k.ID {
					continue
				}
				found = true
				if old.Purpose != Release || !bytes.Equal(old.Public, k.Public) || k.NotBefore.Before(old.NotBefore) || k.NotAfter.After(old.NotAfter) || (old.State == "revoked" && (k.State != "revoked" || k.RevokedAt == nil || !k.RevokedAt.Equal(*old.RevokedAt))) || (old.State == "retired" && k.State == "active") {
					return ErrUnavailable
				}
				s.Keys[i] = k
			}
			if !found {
				s.Keys = append(s.Keys, k)
			}
		}
		s.ReleaseRevision = u.Revision
		if err = advance(s); err != nil {
			return err
		}
		return p.save(*s)
	})
}

var _ Provider = (*FileProvider)(nil)

// DenyIdentity persists a terminal incarnation tombstone outside database
// restore scope. Write before DB revocation; a failed DB commit stays denied.
func (p *FileProvider) DenyIdentity(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrUnavailable
	}
	return p.with(ctx, func(s *ledger) error {
		for _, v := range s.DeniedIdentities {
			if v == id {
				return nil
			}
		}
		s.DeniedIdentities = append(s.DeniedIdentities, id)
		if err := advance(s); err != nil {
			return err
		}
		return p.save(*s)
	})
}
func (p *FileProvider) IdentityDenied(ctx context.Context, id string) (denied bool, err error) {
	err = p.with(ctx, func(s *ledger) error {
		for _, v := range s.DeniedIdentities {
			if v == id {
				denied = true
			}
		}
		return nil
	})
	return
}
