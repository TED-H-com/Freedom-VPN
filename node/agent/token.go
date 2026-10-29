package agent

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
)

// ReadToken reads an enrollment token from a file, or from stdin when path
// is "-". The token is never taken from argv or the environment (§6.1).
func ReadToken(path string) (string, error) {
	var b []byte
	var err error
	if path == "-" {
		b, err = io.ReadAll(io.LimitReader(os.Stdin, 4096))
	} else {
		if path == "" {
			return "", errors.New("agent: token file is required")
		}
		b, err = os.ReadFile(path)
	}
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(b))
	if token == "" || strings.ContainsAny(token, "\r\n") {
		return "", errors.New("agent: enrollment token is empty")
	}
	return token, nil
}

// NeutralizeToken overwrites a token file with zeros, syncs it, and removes
// it. Stdin ("-") has nothing to delete. A failed enroll must not call this.
func NeutralizeToken(path string) error {
	if path == "" || path == "-" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	zeros := bytes.Repeat([]byte{0}, int(info.Size()))
	if len(zeros) == 0 {
		zeros = []byte{0}
	}
	_, werr := f.Write(zeros)
	serr := f.Sync()
	cerr := f.Close()
	rerr := os.Remove(path)
	if werr != nil {
		return werr
	}
	if serr != nil {
		return serr
	}
	if cerr != nil {
		return cerr
	}
	return rerr
}
