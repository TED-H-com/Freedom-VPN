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
	Probe         *ChallengeProbe
	ProbeOmitted  *ProbeOmission
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
	if in.Probe != nil && in.TTL > 300*time.Second {
		return Challenge{}, errors.New("healthattest: probe ttl exceeds limit")
	}
	if in.ProbeOmitted != nil && (in.Probe != nil || !in.ProbeOmitted.Valid()) {
		return Challenge{}, errors.New("healthattest: invalid probe omission")
	}
	if in.Components.NodeAgent == "" || in.Components.Netbird == "" || in.Components.NodeUpdater == "" || in.Components.Enforcement == "" {
		return Challenge{}, errors.New("healthattest: component version")
	}
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return Challenge{}, err
	}
	now := in.Now.UTC()
	var probe *ChallengeProbe
	var omitted *ProbeOmission
	if in.ProbeOmitted != nil {
		copy := *in.ProbeOmitted
		omitted = &copy
	}
	if in.Probe != nil {
		copy := *in.Probe
		copy.Destinations = append([]ChallengeProbeDest(nil), in.Probe.Destinations...)
		probe = &copy
	}
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
		Probe:         probe,
		ProbeOmitted:  omitted,
	}, nil
}

// Same reports whether two challenges are the same issuance. Times compare
// as instants, so a JSON round trip still matches.
func (c Challenge) Same(o Challenge) bool {
	return c.TxnID == o.TxnID && c.Nonce == o.Nonce && c.ReleaseID == o.ReleaseID &&
		c.NodeID == o.NodeID && c.IncarnationID == o.IncarnationID && c.BootID == o.BootID &&
		c.IssuedAt.Equal(o.IssuedAt) && c.ExpiresAt.Equal(o.ExpiresAt) &&
		c.Components == o.Components && c.Consumed == o.Consumed && probeSame(c.Probe, o.Probe) && omissionSame(c.ProbeOmitted, o.ProbeOmitted)
}

func omissionSame(a, b *ProbeOmission) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func probeSame(a, b *ChallengeProbe) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if a.PeerID != b.PeerID || a.SourceV4 != b.SourceV4 || a.ResolverDNS != b.ResolverDNS || len(a.Destinations) != len(b.Destinations) {
		return false
	}
	for i := range a.Destinations {
		if a.Destinations[i] != b.Destinations[i] {
			return false
		}
	}
	return true
}

// WriteChallenge writes the challenge atomically, mode 0600.
func WriteChallenge(dir string, c Challenge) error {
	if c.ProbeOmitted != nil && (c.Probe != nil || !c.ProbeOmitted.Valid()) {
		return &rejectError{Reason: ReasonMalformed}
	}
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
	if err == nil && c.ProbeOmitted != nil && (c.Probe != nil || !c.ProbeOmitted.Valid()) {
		err = &rejectError{Reason: ReasonMalformed}
	}
	return c, info, err
}

// ReadAttestation reads the agent's answer. Symlinks and any mode other than
// 0600 are refused.
func ReadAttestation(dir string) (Attestation, os.FileInfo, error) {
	var a Attestation
	info, err := readJSON(filepath.Join(dir, AttestationFile), &a)
	return a, info, err
}

// ReadObservedAttestation decodes and fingerprints the same bounded read.
// This avoids a path replacement between a freshness observation and decoding.
func ReadObservedAttestation(dir string) (Attestation, Observed, error) {
	var a Attestation
	body, info, err := readFile(filepath.Join(dir, AttestationFile))
	if err != nil {
		return a, Observed{}, err
	}
	if err := decodeJSON(body, &a); err != nil {
		return a, Observed{}, err
	}
	return a, observed(info, body), nil
}

// CheckDir validates an existing private journal directory without following
// a symlink or repairing unsafe permissions. Layout setup uses the same check
// as challenge and attestation I/O.
func CheckDir(dir string) error { return checkDir(dir) }

func checkDir(dir string) error {
	// A trailing separator makes Lstat follow a directory symlink on Unix.
	// Match filepath.Join's normalization before checking the directory itself.
	info, err := os.Lstat(filepath.Clean(dir))
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return &rejectError{Reason: ReasonSymlink}
	}
	if !info.IsDir() {
		return &rejectError{Reason: ReasonMalformed}
	}
	if info.Mode().Perm()&0o077 != 0 {
		return &rejectError{Reason: ReasonBadMode}
	}
	return checkOwner(info)
}

func checkOwner(info os.FileInfo) error {
	if st, ok := info.Sys().(*syscall.Stat_t); ok && uint64(st.Uid) != uint64(os.Geteuid()) {
		return &rejectError{Reason: ReasonBadOwner}
	}
	return nil
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
	if err := checkDir(dir); err != nil {
		return Observed{}, err
	}
	body, info, err := readFile(filepath.Join(dir, AttestationFile))
	if err != nil {
		if os.IsNotExist(err) {
			return Observed{}, nil
		}
		return Observed{}, err
	}
	return observed(info, body), nil
}

func observed(info os.FileInfo, body []byte) Observed {
	return Observed{Present: true, Inode: inodeOf(info), Mod: info.ModTime(), Size: int64(len(body)), Sum: sha256.Sum256(body)}
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
	if err := checkDir(dir); err != nil {
		return err
	}
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
	if err := checkDir(filepath.Dir(path)); err != nil {
		return err
	}
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if len(body) > MaxFileBytes {
		return &rejectError{Reason: ReasonTooLarge}
	}
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
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func readJSON(path string, dest any) (os.FileInfo, error) {
	body, info, err := readFile(path)
	if err != nil {
		return info, err
	}
	return info, decodeJSON(body, dest)
}

func readFile(path string) ([]byte, os.FileInfo, error) {
	if err := checkDir(filepath.Dir(path)); err != nil {
		return nil, nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, info, &rejectError{Reason: ReasonSymlink}
	}
	if !info.Mode().IsRegular() {
		return nil, info, &rejectError{Reason: ReasonMalformed}
	}
	if info.Mode().Perm() != 0o600 {
		return nil, info, &rejectError{Reason: ReasonBadMode}
	}
	if err := checkOwner(info); err != nil {
		return nil, info, err
	}
	if info.Size() > MaxFileBytes {
		return nil, info, &rejectError{Reason: ReasonTooLarge}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, info, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, info, err
	}
	if !os.SameFile(info, opened) || info.Mode() != opened.Mode() {
		return nil, info, &rejectError{Reason: ReasonMalformed}
	}
	if err := checkOwner(opened); err != nil {
		return nil, info, err
	}
	buf, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
	if err != nil {
		return nil, info, err
	}
	if len(buf) > MaxFileBytes {
		return nil, info, &rejectError{Reason: ReasonTooLarge}
	}
	return buf, info, nil
}

func decodeJSON(buf []byte, dest any) error {
	dec := json.NewDecoder(bytes.NewReader(buf))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return fmt.Errorf("healthattest: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return &rejectError{Reason: ReasonMalformed}
	}
	return nil
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
