package counterread

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// PrivateDirectory is inside the owned container's disk-backed run state.
// Each failed sample owns a UUID directory. The harness exports it before teardown.
const PrivateDirectory = "/var/lib/vpn-node/e2e/counterread"

// ParseFailure is safe to publish. The exact parser error can contain object
// names/values from nft output, so it lives only in parser-error.txt alongside
// output.raw. Neither is a public log. Class is a structural hint, not a cause.
type ParseFailure struct {
	ID              string `json:"id"`
	Class           string `json:"class"`
	ParserErrorKind string `json:"parser_error_kind"`
	SHA256          string `json:"sha256"`
	Bytes           int    `json:"bytes,string"`
	Lines           int    `json:"lines"`
	Retained        bool   `json:"private_retained"`
	RetentionError  string `json:"retention_error,omitempty"`
	Exported        bool   `json:"private_exported"`
	ExportError     string `json:"export_error,omitempty"`
}

func parseClass(raw []byte, err error) (string, string) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return "empty", "empty_document"
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return "truncated", "unexpected_eof"
	}
	// CombinedOutput can contain a diagnostic line plus JSON. Recognize only
	// an otherwise complete JSON document after removing explicit warning/error
	// lines. Do not strip arbitrary lines or repair the parser input.
	lines := bytes.Split(raw, []byte{'\n'})
	kept := make([][]byte, 0, len(lines))
	removed := false
	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(string(line)))
		if strings.HasPrefix(lower, "warning:") || strings.HasPrefix(lower, "error:") {
			removed = true
		} else {
			kept = append(kept, line)
		}
	}
	if removed && json.Valid(bytes.Join(kept, []byte{'\n'})) {
		return "interleaved", "diagnostic_and_json"
	}
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		return "unexpected_format", "json_syntax"
	}
	var typ *json.UnmarshalTypeError
	if errors.As(err, &typ) {
		return "unexpected_format", "json_type"
	}
	return "unexpected_format", "parser_rejected"
}

func retainParseFailure(directory string, raw []byte, parseErr error) *ParseFailure {
	class, kind := parseClass(raw, parseErr)
	lines := bytes.Count(raw, []byte{'\n'})
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		lines++
	}
	d := &ParseFailure{ID: uuid.NewString(), Class: class, ParserErrorKind: kind, SHA256: fmt.Sprintf("%x", sha256.Sum256(raw)), Bytes: len(raw), Lines: lines}
	dir := filepath.Join(directory, d.ID)
	if err := os.MkdirAll(directory, 0700); err != nil {
		d.RetentionError = "directory_create"
		return d
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		d.RetentionError = "sample_create"
		return d
	}
	for _, f := range []struct {
		name string
		data []byte
	}{{"output.raw", raw}, {"parser-error.txt", []byte(parseErr.Error())}} {
		// Exclusive files prevent accidental overwrite of retained evidence.
		file, err := os.OpenFile(filepath.Join(dir, f.name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			d.RetentionError = "file_create"
			return d
		}
		_, writeErr := file.Write(f.data)
		syncErr := file.Sync()
		closeErr := file.Close()
		if errors.Join(writeErr, syncErr, closeErr) != nil {
			d.RetentionError = "file_write"
			return d
		}
	}
	d.Retained = true
	return d
}
