//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// ContainerExport describes the complete, atomically published stdout stream.
// Bytes is a decimal string; SHA256 covers exactly the destination file.
type ContainerExport struct {
	Bytes  string `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// ExportContainerFile streams a file out of an owned slot, without the 1 MiB
// lab.docker diagnostic limit. Quiesce the producer first for a closed capture.
// The destination is mode 0600 and is replaced only after a successful command.
// This is raw private scratch: callers must redact before publishing evidence.
func (s *Stack) ExportContainerFile(ctx context.Context, slot, source, destination string) (ContainerExport, error) {
	switch slot {
	case "node", "node2", "node3", "node-clone", "installer-node", "origin", "member":
	default:
		return ContainerExport{}, fmt.Errorf("unknown export slot")
	}
	if !filepath.IsAbs(source) || filepath.Clean(source) != source {
		return ContainerExport{}, fmt.Errorf("absolute clean source path required")
	}
	cmd := exec.CommandContext(ctx, "docker", "exec", s.lab.prefix+"-"+slot, "cat", "--", source)
	return exportCommand(cmd, destination)
}

func exportCommand(cmd *exec.Cmd, destination string) (result ContainerExport, err error) {
	if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return result, err
	}
	f, err := os.CreateTemp(filepath.Dir(destination), ".capture-*")
	if err != nil {
		return result, err
	}
	defer func() { _ = f.Close(); _ = os.Remove(f.Name()) }()
	hash := sha256.New()
	var stderr diagnosticBuffer
	cmd.Stdout, cmd.Stderr = io.MultiWriter(f, hash), &stderr
	if err = cmd.Run(); err != nil {
		return result, fmt.Errorf("container export failed: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		return result, err
	}
	if err = f.Sync(); err != nil {
		return result, err
	}
	if err = f.Close(); err != nil {
		return result, err
	}
	if err = os.Rename(f.Name(), destination); err != nil {
		return result, err
	}
	return ContainerExport{Bytes: fmt.Sprint(info.Size()), SHA256: fmt.Sprintf("%x", hash.Sum(nil))}, nil
}

type diagnosticBuffer struct {
	bytes.Buffer
	overflow bool
}

func (b *diagnosticBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := (1 << 20) - b.Len()
	if len(p) > remaining {
		b.overflow = true
		p = p[:remaining]
	}
	_, _ = b.Buffer.Write(p)
	return n, nil
}
