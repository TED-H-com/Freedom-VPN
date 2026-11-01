package members

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const evidenceGzipThreshold = 20_000_000

type evidenceDigest struct {
	CompressedSHA256   string `json:"compressed_sha256"`
	UncompressedSHA256 string `json:"uncompressed_sha256"`
	CompressedBytes    string `json:"compressed_bytes"`
	UncompressedBytes  string `json:"uncompressed_bytes"`
}

// Historical files are never rewritten. New large files carry both byte pins.
func writeEvidence(path string, raw []byte) error {
	if len(raw) <= evidenceGzipThreshold {
		if err := os.WriteFile(path, raw, 0644); err != nil {
			return err
		}
		for _, suffix := range []string{".gz", ".gz.sha256.json"} {
			if err := os.Remove(path + suffix); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		return nil
	}
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	pin := evidenceDigest{fmt.Sprintf("%x", sha256.Sum256(b.Bytes())), fmt.Sprintf("%x", sha256.Sum256(raw)), fmt.Sprint(b.Len()), fmt.Sprint(len(raw))}
	meta, err := json.MarshalIndent(pin, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(path+".gz", b.Bytes(), 0644); err != nil {
		return err
	}
	if err = os.WriteFile(path+".gz.sha256.json", append(meta, '\n'), 0644); err != nil {
		return err
	}
	if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func readEvidence(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) && !strings.HasSuffix(path, ".gz") {
		return readEvidence(path + ".gz")
	}
	if err != nil || !strings.HasSuffix(path, ".gz") {
		return raw, err
	}
	meta, err := os.ReadFile(path + ".sha256.json")
	if err != nil {
		return nil, err
	}
	var pin evidenceDigest
	if err = json.Unmarshal(meta, &pin); err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != pin.CompressedSHA256 || fmt.Sprint(len(raw)) != pin.CompressedBytes {
		return nil, fmt.Errorf("compressed evidence pin mismatch")
	}
	r, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(plain)) != pin.UncompressedSHA256 || fmt.Sprint(len(plain)) != pin.UncompressedBytes {
		return nil, fmt.Errorf("uncompressed evidence pin mismatch")
	}
	return plain, nil
}
