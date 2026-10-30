package healthattest

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Input is what the updater knows when it issues a challenge.
type Input struct {
	TxnID         string
	ReleaseID     string
	NodeID        string
	IncarnationID string
	BootID        string
	Now           time.Time
	TTL           time.Duration
	Components    Components
}

// Issue builds a challenge with a fresh 256-bit nonce. The caller writes it.
func Issue(in Input) (Challenge, error) {
	if in.TxnID == "" || in.ReleaseID == "" || in.NodeID == "" || in.IncarnationID == "" || in.BootID == "" {
		return Challenge{}, errors.New("healthattest: incomplete challenge")
	}
	if !canonicalID(in.TxnID) || !canonicalID(in.ReleaseID) || !canonicalID(in.NodeID) || !canonicalID(in.IncarnationID) || !canonicalID(in.BootID) {
		return Challenge{}, errors.New("healthattest: id is not a canonical uuid")
	}
	if in.TTL <= 0 || in.Now.IsZero() {
		return Challenge{}, errors.New("healthattest: clock or ttl")
	}
	if in.Components.NodeAgent == "" || in.Components.Netbird == "" || in.Components.NodeUpdater == "" || in.Components.Enforcement == "" {
		return Challenge{}, errors.New("healthattest: component version")
	}
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return Challenge{}, err
	}
	now := in.Now.UTC()
	return Challenge{
		TxnID:         in.TxnID,
		Nonce:         hex.EncodeToString(buf[:]),
		ReleaseID:     in.ReleaseID,
		NodeID:        in.NodeID,
		IncarnationID: in.IncarnationID,
		BootID:        in.BootID,
		IssuedAt:      now,
		ExpiresAt:     now.Add(in.TTL),
		Components:    in.Components,
	}, nil
}

// Same reports whether two challenges are the same issuance. Times compare
// as instants, so a JSON round trip still matches.
func (c Challenge) Same(o Challenge) bool {
	return c.TxnID == o.TxnID && c.Nonce == o.Nonce && c.ReleaseID == o.ReleaseID &&
		c.NodeID == o.NodeID && c.IncarnationID == o.IncarnationID && c.BootID == o.BootID &&
		c.IssuedAt.Equal(o.IssuedAt) && c.ExpiresAt.Equal(o.ExpiresAt) &&
		c.Components == o.Components && c.Consumed == o.Consumed
}

// WriteChallenge writes the challenge atomically, mode 0600.
func WriteChallenge(dir string, c Challenge) error {
	return writeJSON(filepath.Join(dir, ChallengeFile), c)
}

// WriteAttestation writes the agent's answer atomically, mode 0600.
func WriteAttestation(dir string, a Attestation) error {
	return writeJSON(filepath.Join(dir, AttestationFile), a)
}

// ReadChallenge reads the live challenge. Symlinks and any mode other than
// 0600 are refused.
func ReadChallenge(dir string) (Challenge, os.FileInfo, error) {
	var c Challenge
	info, err := readJSON(filepath.Join(dir, ChallengeFile), &c)
	return c, info, err
}

// ReadAttestation reads the agent's answer. Symlinks and any mode other than
// 0600 are refused.
func ReadAttestation(dir string) (Attestation, os.FileInfo, error) {
	var a Attestation
	info, err := readJSON(filepath.Join(dir, AttestationFile), &a)
	return a, info, err
}

// AttestationPrecedes reports that the attestation timestamp is not strictly
// after the challenge timestamp. Equal timestamps count as precedes.
func AttestationPrecedes(challengeMod, attestMod time.Time) bool {
	return !attestMod.After(challengeMod)
}

// Observed is one look at the attestation path. The updater takes it before
// writing the challenge. A later look that still matches was not rewritten.
// Inode and content are the signal: on this host two immediate writes can
// share one file timestamp (待實測 on the node filesystem).
type Observed struct {
	Present bool
	Inode   uint64
	Mod     time.Time
	Size    int64
	Sum     [32]byte
}

// ObserveAttestation stats the attestation path. A missing file is an empty
// Observed and a nil error. Symlinks are refused.
func ObserveAttestation(dir string) (Observed, error) {
	path := filepath.Join(dir, AttestationFile)
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Observed{}, nil
		}
		return Observed{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Observed{}, &rejectError{Reason: ReasonSymlink}
	}
	if !info.Mode().IsRegular() {
		return Observed{}, &rejectError{Reason: ReasonMalformed}
	}
	if info.Size() > MaxFileBytes {
		return Observed{}, &rejectError{Reason: ReasonTooLarge}
	}
	f, err := os.Open(path)
	if err != nil {
		return Observed{}, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return Observed{}, err
	}
	var sum [32]byte
	copy(sum[:], h.Sum(nil))
	return Observed{Present: true, Inode: inodeOf(info), Mod: info.ModTime(), Size: info.Size(), Sum: sum}, nil
}

// Rewritten reports whether cur is a different file from the snapshot taken
// before the challenge existed. A missing file is not a rewrite.
func (cur Observed) Rewritten(prior Observed) bool {
	if !cur.Present {
		return false
	}
	if !prior.Present {
		return true
	}
	if cur.Inode != 0 && prior.Inode != 0 && cur.Inode != prior.Inode {
		return true
	}
	if cur.Mod.After(prior.Mod) || cur.Size != prior.Size || cur.Sum != prior.Sum {
		return true
	}
	return false
}

func inodeOf(info os.FileInfo) uint64 {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	return st.Ino
}

// Consume removes the challenge and the attestation. A missing file is not
// an error. The updater calls this after it has recorded health.
func Consume(dir string) error {
	var err error
	for _, name := range []string{ChallengeFile, AttestationFile} {
		if e := os.Remove(filepath.Join(dir, name)); e != nil && !os.IsNotExist(e) {
			err = e
		}
	}
	return err
}

// MarkConsumed rewrites the challenge so a later read fails closed. The
// updater uses this when removing the file fails.
func MarkConsumed(dir string, c Challenge) error {
	c.Consumed = true
	return WriteChallenge(dir, c)
}

func writeJSON(path string, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	body = append(body, '\n')
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = tmp.Close()
			_ = os.Remove(name)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(body); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	ok = true
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func readJSON(path string, dest any) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return info, &rejectError{Reason: ReasonSymlink}
	}
	if !info.Mode().IsRegular() {
		return info, &rejectError{Reason: ReasonMalformed}
	}
	if info.Mode().Perm() != 0o600 {
		return info, &rejectError{Reason: ReasonBadMode}
	}
	if info.Size() > MaxFileBytes {
		return info, &rejectError{Reason: ReasonTooLarge}
	}
	f, err := os.Open(path)
	if err != nil {
		return info, err
	}
	defer f.Close()
	buf := make([]byte, info.Size())
	if _, err := io.ReadFull(f, buf); err != nil {
		return info, err
	}
	dec := json.NewDecoder(bytes.NewReader(buf))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return info, fmt.Errorf("healthattest: %w", err)
	}
	if dec.More() {
		return info, &rejectError{Reason: ReasonMalformed}
	}
	return info, nil
}

// rejectError is a typed refusal from a reader, as opposed to a missing file.
type rejectError struct{ Reason Reason }

func (e *rejectError) Error() string { return "healthattest: " + string(e.Reason) }

// ReasonOf returns a reader refusal, or empty when err is not one.
func ReasonOf(err error) Reason {
	var r *rejectError
	if errors.As(err, &r) {
		return r.Reason
	}
	return ""
}
