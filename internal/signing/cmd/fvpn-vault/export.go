package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

type exportedKey struct {
	ID           string          `json:"key_id"`
	Purpose      signing.Purpose `json:"purpose"`
	NotBefore    time.Time       `json:"not_before"`
	NotAfter     time.Time       `json:"not_after"`
	Revoked      bool            `json:"revoked"`
	PublicSHA256 string          `json:"public_sha256"`
}
type exportedPurpose struct {
	Purpose              signing.Purpose `json:"purpose"`
	CurrentNonRevokedKey bool            `json:"current_non_revoked_key"`
}
type exportReceipt struct {
	Kind                string            `json:"kind"`
	Provider            string            `json:"provider"`
	ExportedAt          time.Time         `json:"exported_at"`
	LedgerRevision      decstr.U64        `json:"ledger_revision"`
	ReleaseRevision     decstr.U64        `json:"release_revision"`
	Output              string            `json:"output"`
	SHA256              string            `json:"sha256"`
	KeyUseChecked       bool              `json:"key_use_checked"`
	PrivateKeyFilesRead bool              `json:"private_key_files_read"`
	Keys                []exportedKey     `json:"keys"`
	Purposes            []exportedPurpose `json:"purposes"`
}

func prepareExport(s signing.FileSnapshot, now time.Time, name string) ([]byte, exportReceipt, error) {
	r := exportReceipt{Kind: "fvpn-file-vault-export-trust-v1", Provider: "file", ExportedAt: now,
		LedgerRevision: s.Revision, ReleaseRevision: s.ReleaseRevision, Output: name,
		KeyUseChecked: false, Keys: []exportedKey{}, Purposes: []exportedPurpose{}}
	if s.NodeTrust == nil || s.NodeTrust.Len() == 0 {
		return nil, r, errors.New("empty bundle")
	}
	// Preserve the initializer's byte serialization, including its final newline.
	raw, err := document(s.NodeTrust)
	if err != nil {
		return nil, r, err
	}
	if _, err = envelope.ParseNodeTrustStoreJSON(raw); err != nil {
		return nil, r, err
	}
	r.SHA256 = digest(raw)
	for _, k := range sortedKeys(s.Keys) {
		trusted, ok := s.NodeTrust.Lookup(k.ID)
		if !ok {
			continue
		}
		r.Keys = append(r.Keys, exportedKey{ID: k.ID, Purpose: k.Purpose, NotBefore: trusted.NotBefore.UTC(), NotAfter: trusted.NotAfter.UTC(), Revoked: trusted.Revoked, PublicSHA256: digest(trusted.PublicKey)})
	}
	for _, purpose := range []signing.Purpose{signing.DesiredState, signing.Lease, signing.Recovery, signing.Release} {
		status := exportedPurpose{Purpose: purpose}
		for _, k := range r.Keys {
			if k.Purpose == purpose && !k.Revoked && !now.Before(k.NotBefore) && now.Before(k.NotAfter) {
				status.CurrentNonRevokedKey = true
			}
		}
		r.Purposes = append(r.Purposes, status)
	}
	return raw, r, nil
}

// Inject only individual IO boundaries in tests, preserving the real publish
// and cleanup flow. All operations are relative to an opened destination root.
type publicationOps struct {
	write func(*os.File, []byte) (int, error)
	sync  func(*os.File) error
	close func(*os.File) error
	link  func(*os.Root, string, string) error
	read  func(*os.Root, string) ([]byte, error)
}

func (o publicationOps) defaults() publicationOps {
	if o.write == nil {
		o.write = (*os.File).Write
	}
	if o.sync == nil {
		o.sync = (*os.File).Sync
	}
	if o.close == nil {
		o.close = (*os.File).Close
	}
	if o.link == nil {
		o.link = (*os.Root).Link
	}
	if o.read == nil {
		o.read = (*os.Root).ReadFile
	}
	return o
}
func resolved(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}
func publish(vault, path string, raw []byte, ops publicationOps) error {
	// Resolve both directories before making even a temporary name. Final
	// symlinks (including dangling ones) are refused by Lstat and by Link.
	vaultDir, err := resolved(vault)
	if err != nil {
		return err
	}
	parent, err := resolved(filepath.Dir(path))
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(vaultDir, parent)
	if err != nil {
		return err
	}
	if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("destination in vault")
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer root.Close()
	name := filepath.Base(path)
	if _, err = root.Lstat(name); !errors.Is(err, os.ErrNotExist) {
		return errors.New("destination not new")
	}
	temp := ".fvpn-vault-" + rand.Text()
	f, err := root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer root.Remove(temp) // Best effort; no power-loss durability promise.
	defer f.Close()
	if err = f.Chmod(0600); err != nil {
		return err
	}
	ops = ops.defaults()
	if n, err := ops.write(f, raw); err != nil {
		return err
	} else if n != len(raw) {
		return io.ErrShortWrite
	}
	if err = ops.sync(f); err != nil {
		return err
	}
	if err = ops.close(f); err != nil {
		return err
	}
	if err = ops.link(root, temp, name); err != nil {
		return err
	}
	got, err := ops.read(root, name)
	if err != nil || !bytes.Equal(got, raw) {
		// Only our successful exclusive link is eligible for removal. Never remove
		// a pre-existing target after a failed Link.
		removeErr := root.Remove(name)
		return errors.Join(errors.New("read-back failed"), removeErr)
	}
	return nil
}
