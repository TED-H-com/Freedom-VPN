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
	dir              string
	now              func() time.Time
	conditionalHooks conditionalHooks
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
	return OpenFileProviderContext(context.Background(), dir)
}

// OpenFileProviderContext opens existing state without creating or repairing it.
// The context bounds lock contention, including the initial validation read.
func OpenFileProviderContext(ctx context.Context, dir string) (*FileProvider, error) {
	p := &FileProvider{dir: dir, now: time.Now}
	err := p.with(ctx, func(*ledger) error { return nil })
	return p, err
}

// FileSnapshot is public metadata from one locked ledger read. It contains no
// private material or denied identity identifiers (§16.2, §17.2).
type FileSnapshot struct {
	Revision            decstr.U64
	ReleaseRevision     decstr.U64
	Keys                []KeyRecord
	DeniedIdentityCount int
	NodeTrust           *envelope.TrustStore
}

// Snapshot returns independent copies and the existing node bundle from the
// same revision. It never opens private key files or changes the vault.
func (p *FileProvider) Snapshot(ctx context.Context) (out FileSnapshot, err error) {
	err = p.with(ctx, func(s *ledger) error {
		trust, e := bundle(s, false)
		if e != nil {
			return e
		}
		out = FileSnapshot{Revision: s.Revision, ReleaseRevision: s.ReleaseRevision,
			DeniedIdentityCount: len(s.DeniedIdentities), NodeTrust: trust,
			Keys: make([]KeyRecord, len(s.Keys))}
		for i, k := range s.Keys {
			k.Public = bytes.Clone(k.Public)
			if k.RevokedAt != nil {
				revoked := *k.RevokedAt
				k.RevokedAt = &revoked
			}
			out.Keys[i] = k
		}
		return nil
	})
	return
}

// with uses a cross-process lock: cached provider handles never cache key state.
func (p *FileProvider) with(ctx context.Context, fn func(*ledger) error, lockedChecks ...func(*os.File) error) error {
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
	for _, check := range lockedChecks {
		if err := check(f); err != nil {
			return err
		}
	}
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
		if k.Purpose == Release {
			if k.ID == "" || len(k.ID) > 128 {
				return ErrUnavailable
			}
		} else if _, err := uuid.Parse(k.ID); err != nil {
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
	_, err := p.saveTracked(s, conditionalHooks{})
	return err
}

// saveTracked distinguishes failures before replacement from errors at/after
// rename. Even a rename error is uncertain: do not infer filesystem state from
// an error return. Existing callers retain the same writes and error behavior.
func (p *FileProvider) saveTracked(s ledger, hooks conditionalHooks) (bool, error) {
	if err := validateLedger(s); err != nil {
		return false, err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return false, err
	}
	f, err := os.CreateTemp(p.dir, ".state-")
	if err != nil {
		return false, err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return false, err
	}
	if closeErr != nil {
		return false, closeErr
	}
	if hooks.beforeRename != nil {
		if err = hooks.beforeRename(); err != nil {
			return false, err
		}
	}
	if err = os.Rename(f.Name(), filepath.Join(p.dir, "state.json")); err != nil {
		return true, err
	}
	if hooks.afterRename != nil {
		if err = hooks.afterRename(); err != nil {
			return true, err
		}
	}
	d, err := os.Open(p.dir)
	if err != nil {
		return true, err
	}
	defer d.Close()
	return true, d.Sync()
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
		if !purpose.Valid() || k.Purpose == LedgerEpoch || (releaseOnly && k.Purpose != Release) {
			continue
		}
		keys = append(keys, envelope.TrustedKey{KeyID: k.ID, PublicKey: k.Public, Purposes: []envelope.Purpose{purpose}, NotBefore: k.NotBefore, NotAfter: k.NotAfter, Revoked: k.State == "revoked"})
	}
	return envelope.NewNodeTrustStore(keys...)
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

// ChangeOutcome reports only what a conditional mutation knows. An error return
// instead means no mutation ran (unavailable ledger, invalid lock or deadline).
type ChangeOutcome uint8

const (
	ChangeRefused ChangeOutcome = iota + 1
	ChangeApplied
	ChangeAlreadyRevoked
	ChangeNotApplied
	ChangeUncertain
)

// ChangeResult contains public facts only. On ChangeNotApplied/ChangeUncertain,
// Key and Previous, when present, describe an attempted replacement, not a
// confirmed ledger. CreatedKeyID is
// set only after exclusive key-file creation succeeds, including write failures.
// Previous is the retired record after rotation, or the pre-revocation record.
// A refused result discloses no vault contents.
type ChangeResult struct {
	Outcome      ChangeOutcome
	OldRevision  decstr.U64
	NewRevision  decstr.U64
	Key          *KeyRecord
	Previous     *KeyRecord
	CreatedKeyID string
}

// LockValidationError means the file held locked was not the regular, private
// entry named lock when checked. It is distinct from preconditions and deadlines.
type LockValidationError struct{}

func (*LockValidationError) Error() string { return "signing: invalid conditional lock" }

// Per-handle, unexported fault seams; ordinary provider methods never use them.
// Configure before a test starts calls, never concurrently with a call.
type conditionalHooks struct {
	locked       func()
	beforeKey    func(string) error
	beforeRename func() error
	afterRename  func() error
}

func (p *FileProvider) checkConditionalLock(f *os.File) error {
	if p.conditionalHooks.locked != nil {
		p.conditionalHooks.locked()
	}
	held, err := f.Stat()
	if err != nil {
		return &LockValidationError{}
	}
	entry, err := os.Lstat(filepath.Join(p.dir, "lock"))
	if err != nil || !held.Mode().IsRegular() || !entry.Mode().IsRegular() ||
		held.Mode().Perm()&0077 != 0 || entry.Mode().Perm()&0077 != 0 || !os.SameFile(held, entry) {
		return &LockValidationError{}
	}
	return nil
}

// RotateConditional switches only the four central Ed25519 purposes (§16.2).
// expectedActive == "" explicitly expects no active record (expiry alone does
// not remove an active record). All preconditions precede key generation inside
// the lock. Context cancellation only bounds acquiring the lock, never the save.
func (p *FileProvider) RotateConditional(ctx context.Context, purpose Purpose, expectedRevision decstr.U64, expectedActive string, validity, overlap time.Duration) (out ChangeResult, err error) {
	err = p.with(ctx, func(s *ledger) error {
		out.Outcome = ChangeRefused
		if s.Revision != expectedRevision || !central(purpose) || validity <= 0 || overlap < 0 {
			return nil
		}
		previous := -1
		for i, k := range s.Keys {
			if k.Purpose == purpose && k.State == "active" {
				previous = i
				break
			}
		}
		if (previous < 0 && expectedActive != "") || (previous >= 0 && s.Keys[previous].ID != expectedActive) {
			return nil
		}
		out.Outcome = ChangeNotApplied
		out.OldRevision = s.Revision
		// Exhaustion is a no-change failure, never revision wraparound (§17.2).
		if e := advance(s); e != nil {
			return nil
		}
		id, e := uuid.NewV7()
		if e != nil {
			return nil
		}
		pub, key, e := ed25519.GenerateKey(rand.Reader)
		if e != nil {
			return nil
		}
		raw, e := x509.MarshalPKCS8PrivateKey(key)
		if e != nil {
			return nil
		}
		if p.conditionalHooks.beforeKey != nil {
			if e = p.conditionalHooks.beforeKey(id.String()); e != nil {
				return nil
			}
		}
		f, e := os.OpenFile(filepath.Join(p.dir, id.String()+".key"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return nil
		}
		out.CreatedKeyID = id.String()
		// Restore owner access even under a restrictive umask; never broaden
		// group/other access. This applies only to the newly created file.
		e = f.Chmod(0600)
		if e == nil {
			_, e = f.Write(raw)
		}
		if e == nil {
			e = f.Sync()
		}
		closeErr := f.Close()
		if e != nil || closeErr != nil {
			return nil
		}
		now := p.now().UTC()
		k := KeyRecord{ID: id.String(), Purpose: purpose, Public: pub, State: "active", NotBefore: now, NotAfter: now.Add(validity)}
		if previous >= 0 {
			old := &s.Keys[previous]
			old.State = "retired"
			if now.Add(overlap).Before(old.NotAfter) {
				old.NotAfter = now.Add(overlap)
			}
			if !old.NotBefore.Before(old.NotAfter) {
				old.NotAfter = old.NotBefore.Add(time.Nanosecond)
			}
			copy := *old
			out.Previous = &copy
		}
		s.Keys = append(s.Keys, k)
		out.Key, out.NewRevision = &k, s.Revision
		out.Outcome = p.saveChange(*s)
		return nil
	}, p.checkConditionalLock)
	return
}

// RevokeConditional preserves terminal revocation, including its timestamp
// (§16.2, §17.4). Already-revoked wins over a stale revision, but never over a
// missing key, wrong purpose or forbidden purpose. Active revocation is opt-in.
func (p *FileProvider) RevokeConditional(ctx context.Context, id string, purpose Purpose, expectedRevision decstr.U64, revokeActive bool) (out ChangeResult, err error) {
	err = p.with(ctx, func(s *ledger) error {
		out.Outcome = ChangeRefused
		for i := range s.Keys {
			k := &s.Keys[i]
			if k.ID != id {
				continue
			}
			if k.Purpose != purpose || !central(purpose) {
				return nil
			}
			if k.State == "revoked" {
				out = ChangeResult{Outcome: ChangeAlreadyRevoked, OldRevision: s.Revision, NewRevision: s.Revision, Key: k}
				return nil
			}
			if s.Revision != expectedRevision || (k.State == "active" && !revokeActive) {
				return nil
			}
			out.Outcome, out.OldRevision = ChangeNotApplied, s.Revision
			if e := advance(s); e != nil {
				return nil
			}
			old := *k
			now := p.now().UTC()
			k.State, k.RevokedAt = "revoked", &now
			out.Key, out.Previous, out.NewRevision = k, &old, s.Revision
			out.Outcome = p.saveChange(*s)
			return nil
		}
		return nil
	}, p.checkConditionalLock)
	return
}

func (p *FileProvider) saveChange(s ledger) ChangeOutcome {
	replaceAttempted, err := p.saveTracked(s, p.conditionalHooks)
	if err == nil {
		return ChangeApplied
	}
	if replaceAttempted {
		return ChangeUncertain
	}
	return ChangeNotApplied
}
