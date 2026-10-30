package isolation

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Explicit offline review of existing evidence; never issues network traffic or
// replaces the original live measurement. Keep that record in history first.
func TestAT13_ReviewCapturedExpiry(t *testing.T) {
	dir := os.Getenv("FVPN_AT_REVIEW_DIR")
	if dir == "" {
		t.Skip("set FVPN_AT_REVIEW_DIR to review saved expiry captures")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "AT13.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rec Record
	if err = json.Unmarshal(raw, &rec); err != nil {
		t.Fatal(err)
	}
	found := false
	for i := range rec.Checks {
		c := &rec.Checks[i]
		if c.Name != "lease member expiry under control loss" {
			continue
		}
		found = true
		inputs, outputs, requests := 0, 0, 0
		valid, ingress, denied := true, true, true
		var before, after string
		var end, stopped time.Time
		for _, o := range c.Observations {
			switch o.Name {
			case "negative_node_ingress":
				n, ok := CaptureCount(o.Detail)
				inputs++
				valid = valid && ok
				ingress = ingress && n > 0
			case "negative_node_egress":
				n, ok := CaptureCount(o.Detail)
				outputs++
				valid = valid && ok
				denied = denied && n == 0
			case "post_deadline_bytes_before":
				before = o.Detail
			case "post_deadline_bytes_after":
				after = o.Detail
			case "grant_deadline":
				end, _ = time.Parse(time.RFC3339, o.Detail)
			case "control_stopped_at":
				stopped, _ = time.Parse(time.RFC3339, o.Detail)
			default:
				if strings.HasPrefix(o.Name, "post_deadline_") {
					requests++
					denied = denied && strings.Contains(o.Detail, "result=timeout;") && strings.Contains(o.Detail, "dev fvnb0")
				}
			}
		}
		b, e1 := strconv.ParseUint(before, 10, 64)
		a, e2 := strconv.ParseUint(after, 10, 64)
		at, e3 := time.Parse(time.RFC3339, c.At)
		ran := valid && ingress && inputs == 5 && outputs == 5 && requests == 5 && e1 == nil && e2 == nil && e3 == nil && !end.IsZero() && !stopped.IsZero() && stopped.Before(end) && at.After(end)
		c.Status = NegativeStatus(c.Positive, ran, denied && a == b)
		c.Observations = append(c.Observations, Observation{Name: "offline_capture_review", Detail: fmt.Sprintf("singular/plural parser: ingress=%d, egress=%d, requests=%d, valid=%t; no new traffic; reviewed UTC %s", inputs, outputs, requests, valid, time.Now().UTC().Format(time.RFC3339))})
		t.Logf("saved expiry: %s; ingress=%d egress=%d requests=%d bytes=%d/%d", c.Status, inputs, outputs, requests, b, a)
	}
	if !found {
		t.Fatal("saved expiry check absent")
	}
	h := sha256.New()
	for _, name := range []string{"capture.go", "capture_review_test.go"} {
		body, e := os.ReadFile(name)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = h.Write(body)
	}
	rec.Observations = append(rec.Observations, Observation{Name: "offline_evaluator", Detail: fmt.Sprintf("SHA256(concatenated capture.go, capture_review_test.go)=%x; build identifies the unchanged live measurement; original AT13 saved under history/follow-up-2-live", h.Sum(nil))})
	rec.Status = ChecksStatus(rec.Checks)
	if err = Write(dir, rec); err != nil {
		t.Fatal(err)
	}
}
