package main

// TEST ONLY read-only observation of the real updater/agent handshake. These
// files are evidence copies; they are never read by either product process.
import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func upgradeObserve() error {
	const root = "/var/lib/vpn-node/"
	f, err := os.OpenFile(root+"e2e/upgrade-observe.jsonl", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	read := func(path string) string {
		b, e := os.ReadFile(path)
		if e != nil {
			return "absent"
		}
		return strings.TrimSpace(string(b))
	}
	enc := json.NewEncoder(f)
	last := ""
	for until := time.Now().Add(3 * time.Minute); time.Now().Before(until); time.Sleep(20 * time.Millisecond) {
		if _, e := os.Stat(root + "e2e/upgrade-observe.done"); e == nil {
			return f.Sync()
		}
		waits, _ := filepath.Glob(root + "update-journal/meter-wait-*")
		notes := map[string]string{}
		for _, path := range waits {
			notes[filepath.Base(path)] = read(path)
		}
		checkpoint := read(root + "state/meter.checkpoint")
		latch := read(root + "update-journal/forwarding.blocked")
		key, _ := json.Marshal([]any{notes, checkpoint, latch})
		if string(key) == last {
			continue
		}
		last = string(key)
		row := map[string]any{"at": time.Now().UTC(), "boot_id": read("/proc/sys/kernel/random/boot_id"), "wait_notes": notes, "checkpoint": checkpoint, "latch": latch, "heartbeat": read("/run/vpn-node/heartbeat")}
		nft, e := exec.Command("nft", "-j", "list", "table", "inet", "fvpn").Output()
		row["nft"] = string(nft)
		row["nft_read_ok"] = e == nil
		control, controlErr := exec.Command("curl", "--silent", "--max-time", "2", "--cacert", "/etc/vpn-node/ca.pem", "--cert", root+"identity/cert.pem", "--key", root+"identity/key.pem", "-o", "/dev/null", "-w", "%{http_code}", "https://agent.example.invalid:9443/api/v1/agent/desired-state").Output()
		row["control_http_status"] = string(control)
		row["control_read_ok"] = controlErr == nil
		row["read_finished_at"] = time.Now().UTC()

		if err = enc.Encode(row); err != nil {
			return err
		}
		if err = f.Sync(); err != nil {
			return err
		}
	}
	return nil
}
