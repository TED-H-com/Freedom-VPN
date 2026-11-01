package enforce

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Executor is the privileged boundary. Scripts are nft or tc input, not shell.
// The real implementation pipes them to nft -f - and tc -batch -. Tests use a fake.
type Executor interface {
	NftBatch(ctx context.Context, script string) error
	TCBatch(ctx context.Context, script string) error
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
	// LookPath reports where name would be found. It does not run the binary.
	LookPath(name string) (string, error)
}

// CmdError is a failed nft, tc or conntrack invocation. Out is the combined
// output and is safe to log: these commands do not receive secrets.
type CmdError struct {
	Cmd string
	Err error
	Out string
}

func (e *CmdError) Error() string {
	msg := strings.TrimSpace(e.Out)
	if msg == "" {
		return fmt.Sprintf("%s: %v", e.Cmd, e.Err)
	}
	return fmt.Sprintf("%s: %v: %s", e.Cmd, e.Err, msg)
}

func (e *CmdError) Unwrap() error { return e.Err }

// System runs the host binaries inside the network namespace of this process.
// The lab invokes it only inside the lab container.
type System struct{}

func (System) NftBatch(ctx context.Context, script string) error {
	return runStdin(ctx, script, "nft", "-f", "-")
}

func (System) TCBatch(ctx context.Context, script string) error {
	script = stripBatchComments(script)
	if strings.TrimSpace(script) == "" {
		return nil
	}
	return runStdin(ctx, script, "tc", "-batch", "-")
}

// stripBatchComments drops blank lines and '#' comments. nft -f accepts
// comments; tc -batch does not. The rendered script keeps the comments so a
// human can read it.
func stripBatchComments(script string) string {
	var b strings.Builder
	for _, line := range strings.Split(script, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func (System) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (System) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return nil, &CmdError{Cmd: name + " " + strings.Join(args, " "), Err: err, Out: buf.String()}
	}
	return buf.Bytes(), nil
}

func runStdin(ctx context.Context, script, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(script)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return &CmdError{Cmd: name, Err: err, Out: buf.String()}
	}
	return nil
}

func isNoTable(err error) bool {
	var ce *CmdError
	if !errors.As(err, &ce) {
		return false
	}
	msg := ce.Out + ce.Error()
	return strings.Contains(msg, "No such file") || strings.Contains(msg, "does not exist")
}
