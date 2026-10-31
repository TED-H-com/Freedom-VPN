package agent

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/google/uuid"
)

// ForwardingBlocked reports the updater latch (§13.5). The file
// forwarding.blocked lives in the updater journal directory. The agent also
// treats `fvpn-updater recover` exit status 2 as a block. Either one keeps
// member forwarding off.
func ForwardingBlocked(journalDir string) bool {
	if journalDir == "" {
		return false
	}
	_, blocked := meterBarrierIdentity(journalDir)
	return blocked
}

// Presence or an unreadable/malformed latch always blocks. The UUID spelling
// is opaque: strip only the updater's final newline, never normalize the ID.
func meterBarrierIdentity(journalDir string) (string, bool) {
	if journalDir == "" {
		return "", false
	}
	raw, err := readBarrierFile(filepath.Join(journalDir, "forwarding.blocked"))
	if errors.Is(err, os.ErrNotExist) {
		return "", false
	}
	id := strings.TrimSuffix(string(raw), "\n")
	if err != nil || uuid.Validate(id) != nil {
		return "", true
	}
	return id, true
}

func readBarrierFile(path string) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > 4096 {
		return nil, errors.New("agent: invalid barrier file")
	}
	raw, err := io.ReadAll(io.LimitReader(f, 4097))
	if len(raw) > 4096 {
		return nil, errors.New("agent: oversized barrier file")
	}
	return raw, err
}
