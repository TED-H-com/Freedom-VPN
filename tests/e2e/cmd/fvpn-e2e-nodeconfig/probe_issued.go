package main

// probe-issued watches one health challenge inside the node and prints only
// true or false. The configured probe is read from stdin and is not printed.
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

func probeIssued() error {
	var configured map[string]any
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 8192))
	if err != nil || json.Unmarshal(raw, &configured) != nil || configured == nil {
		return fmt.Errorf("probe config")
	}
	stop := "/var/lib/vpn-node/e2e/probe-issued-stop"
	_ = os.Remove(stop)
	deadline := time.Now().Add(4 * time.Minute)
	for time.Now().Before(deadline) {
		if _, err = os.Stat(stop); err == nil {
			break
		}
		body, err := os.ReadFile("/var/lib/vpn-node/update-journal/health-challenge.json")
		if err == nil && challengeCarriesProbe(body, configured) {
			fmt.Println("true")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println("false")
	return nil
}

func challengeCarriesProbe(raw []byte, configured map[string]any) bool {
	if len(raw) == 0 || len(raw) > 8192 {
		return false
	}
	var doc struct {
		Probe json.RawMessage `json:"probe"`
	}
	if json.Unmarshal(raw, &doc) != nil || len(doc.Probe) == 0 || string(doc.Probe) == "null" {
		return false
	}
	want, err := json.Marshal(configured)
	if err != nil {
		return false
	}
	var left, right any
	if json.Unmarshal(doc.Probe, &left) != nil || json.Unmarshal(want, &right) != nil {
		return false
	}
	a, e1 := json.Marshal(left)
	b, e2 := json.Marshal(right)
	return e1 == nil && e2 == nil && bytes.Equal(a, b)
}
