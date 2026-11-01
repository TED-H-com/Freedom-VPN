package members

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEvidenceGzipLosslessAndPins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AT02.json")
	raw := bytes.Repeat([]byte("0123456789\n"), evidenceGzipThreshold/11+1)
	if err := writeEvidence(path, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("oversize plain evidence retained")
	}
	got, err := readEvidence(path)
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatal("lossless round trip", err)
	}
	meta, err := os.ReadFile(path + ".gz.sha256.json")
	if err != nil {
		t.Fatal(err)
	}
	var pin evidenceDigest
	if err = json.Unmarshal(meta, &pin); err != nil {
		t.Fatal(err)
	}
	pin.UncompressedSHA256 = "wrong"
	bad, _ := json.Marshal(pin)
	if err = os.WriteFile(path+".gz.sha256.json", bad, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = readEvidence(path); err == nil {
		t.Fatal("incorrect uncompressed pin accepted")
	}
	if err = os.WriteFile(path+".gz.sha256.json", meta, 0644); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path+".gz", []byte("corrupt"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = readEvidence(path); err == nil {
		t.Fatal("incorrect compressed pin accepted")
	}
	if err = writeEvidence(path, []byte("small\n")); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path + ".gz"); !os.IsNotExist(err) {
		t.Fatal("stale compressed evidence retained")
	}
}
