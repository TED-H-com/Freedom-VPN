package installer

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// rejectTokenArgv refuses a token on the command line (§18.2). The value
// would be stored in shell history. Call this before flag parsing.
// RejectTokenArgv refuses a token on the command line (§18.2).
func RejectTokenArgv(args []string) error {
	for _, a := range args {
		if a == "--token" || a == "-token" || strings.HasPrefix(a, "--token=") || strings.HasPrefix(a, "-token=") {
			return errf(ExitUsage, "usage", "enrollment token must not be passed as an argument")
		}
	}
	return nil
}

// checkTokenFile requires a regular file, not a symlink, mode 0600.
func checkTokenFile(path string) error {
	st, err := os.Lstat(path)
	if err != nil {
		return errf(ExitUsage, "usage", "token file is not readable")
	}
	if st.Mode()&os.ModeSymlink != 0 {
		return errf(ExitUsage, "usage", "token file must not be a symlink")
	}
	if !st.Mode().IsRegular() {
		return errf(ExitUsage, "usage", "token file must be a regular file")
	}
	if st.Mode().Perm() != 0o600 {
		return errf(ExitUsage, "usage", "token file mode must be 0600")
	}
	return nil
}

// shredFile overwrites the file with zeros, syncs, and removes it.
// A missing file is success. The contents are never logged (§6.1).
func shredFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	if st.Size() > 0 {
		zeros := make([]byte, st.Size())
		if _, err := f.WriteAt(zeros, 0); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Remove(path)
}

// promptToken reads one line from /dev/tty with echo off (§18.2).
// A pipe is refused. The caller writes the line to a 0600 file and shreds it.
func promptToken() (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", errf(ExitUsage, "usage", "token file is required when there is no terminal")
	}
	defer tty.Close()
	termios, err := unix.IoctlGetTermios(int(tty.Fd()), unix.TCGETS)
	if err != nil {
		return "", errf(ExitUsage, "usage", "token file is required when there is no terminal")
	}
	old := *termios
	termios.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(int(tty.Fd()), unix.TCSETS, termios); err != nil {
		return "", errf(ExitUsage, "usage", "token file is required when there is no terminal")
	}
	defer unix.IoctlSetTermios(int(tty.Fd()), unix.TCSETS, &old)
	if _, err := io.WriteString(tty, "enrollment token: "); err != nil {
		return "", err
	}
	line, err := bufio.NewReader(tty).ReadString('\n')
	_, _ = io.WriteString(tty, "\n")
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return "", errf(ExitUsage, "usage", "empty enrollment token")
	}
	return line, nil
}

// writeTokenTemp stores a prompted token in a 0600 file under dir.
func writeTokenTemp(dir, token string) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, ".enroll-token-*")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return "", err
	}
	if _, err := io.WriteString(f, token); err != nil {
		_ = f.Close()
		_ = shredFile(name)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = shredFile(name)
		return "", err
	}
	return name, nil
}
