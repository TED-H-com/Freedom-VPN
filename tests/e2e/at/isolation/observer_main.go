//go:build ignore

package main

// TEST ONLY transparent mTLS receiver. Bodies are retained only for observations
// and lease renewal; authentication headers and credential responses are excluded.
import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	_ "modernc.org/sqlite"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func observeProxy() {
	roots := func(path string) *x509.CertPool {
		raw, e := os.ReadFile(path)
		must(e)
		p := x509.NewCertPool()
		if !p.AppendCertsFromPEM(raw) {
			os.Exit(1)
		}
		return p
	}
	cert, e := tls.LoadX509KeyPair("/var/lib/vpn-node/identity/cert.pem", "/var/lib/vpn-node/identity/key.pem")
	must(e)
	target, _ := url.Parse("https://agent.example.invalid:9443")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots("/etc/vpn-node/ca.pem"), Certificates: []tls.Certificate{cert}}, ResponseHeaderTimeout: 15 * time.Second}
	file, e := os.OpenFile("/var/lib/vpn-node/e2e/isolation-observations.jsonl", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	must(e)
	defer file.Close()
	var mu sync.Mutex
	var queue []map[string]any
	go func() {
		for range time.NewTicker(100 * time.Millisecond).C {
			mu.Lock()
			batch := queue
			queue = nil
			mu.Unlock()
			for _, row := range batch {
				_ = json.NewEncoder(file).Encode(row)
			}
			_ = file.Sync()
		}
	}()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !((r.Method == "POST" && (r.URL.Path == "/api/v1/agent/observations" || strings.Contains(r.URL.Path, "lease"))) || (r.Method == "GET" && strings.Contains(r.URL.Path, "desired-state"))) {
			proxy.ServeHTTP(w, r)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
		if err != nil {
			http.Error(w, "capture read failed", 500)
			return
		}
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(raw))
		row := map[string]any{"received_at": time.Now().UTC().Format(time.RFC3339Nano), "path": r.URL.Path}
		if json.Valid(raw) {
			row["body"] = json.RawMessage(raw)
		}
		capture := &captureResponse{ResponseWriter: w}
		proxy.ServeHTTP(capture, r)
		row["response_at"] = time.Now().UTC().Format(time.RFC3339Nano)
		row["status"] = capture.status
		if json.Valid(capture.body.Bytes()) {
			row["response"] = json.RawMessage(capture.body.Bytes())
		}
		mu.Lock()
		queue = append(queue, row)
		mu.Unlock()
	})
	srv := &http.Server{Addr: "127.0.0.1:9444", Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, ClientCAs: roots("/etc/vpn-node/agent-ca.pem"), ClientAuth: tls.RequireAndVerifyClientCert}}
	must(srv.ListenAndServeTLS("/etc/vpn-node/fault-proxy.crt", "/etc/vpn-node/fault-proxy.key"))
}

type captureResponse struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *captureResponse) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
func (w *captureResponse) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func configureObservation() {
	p := "/etc/vpn-node/agent-run.json"
	raw, e := os.ReadFile(p)
	must(e)
	var cfg map[string]any
	must(json.Unmarshal(raw, &cfg))
	for _, k := range []string{"heartbeat_seconds", "poll_seconds", "observation_seconds"} {
		cfg[k] = 1
	}
	// Agent-core 27's opt-in, detached producer timing; no product wire change.
	cfg["test_timing_path"] = "/var/lib/vpn-node/e2e/isolation-operations.jsonl"
	if cfg["test_mode"] != true {
		panic("producer timing requires the existing TEST ONLY node")
	}
	raw, e = json.MarshalIndent(cfg, "", "  ")
	must(e)
	must(os.WriteFile(p, raw, 0600))
	raw, e = json.Marshal(clockIdentity())
	must(e)
	must(os.WriteFile("/var/lib/vpn-node/e2e/isolation-clock.json", raw, 0600))
}

func clockIdentity() map[string]string {
	boot, e := os.ReadFile("/proc/sys/kernel/random/boot_id")
	must(e)
	offsets, e := os.ReadFile("/proc/self/timens_offsets")
	must(e)
	namespace, e := os.Readlink("/proc/self/ns/time")
	must(e)
	return map[string]string{"boot_id": strings.TrimSpace(string(boot)), "time_namespace": namespace, "offsets": string(offsets), "clock": "CLOCK_MONOTONIC"}
}

// Read the live agent's clock namespace after an ordinary container restart.
func producerClock() {
	row := clockIdentity()
	row["sample_start"] = time.Now().UTC().Format(time.RFC3339Nano)
	raw, e := exec.Command("pgrep", "-x", "fvpn-agent").Output()
	must(e)
	pid := strings.TrimSpace(string(raw))
	_, e = strconv.Atoi(pid)
	must(e)
	row["agent_pid"] = pid
	row["time_namespace"], e = os.Readlink("/proc/" + pid + "/ns/time")
	must(e)
	for name, file := range map[string]string{"offsets": "timens_offsets", "process_stat": "stat"} {
		raw, e = os.ReadFile("/proc/" + pid + "/" + file)
		must(e)
		row[name] = strings.TrimSpace(string(raw))
	}
	raw, e = os.ReadFile("/var/lib/vpn-node/e2e/isolation-operations.jsonl")
	must(e)
	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	var last map[string]any
	must(json.Unmarshal(lines[len(lines)-1], &last))
	row["measurement_run_id"], _ = last["run_id"].(string)
	row["latest_sequence"], _ = last["sequence"].(string)
	row["scope"] = "Read-only live source agent clock after ordinary container restart; no clock or namespace mutation."
	row["sample_end"] = time.Now().UTC().Format(time.RFC3339Nano)
	must(json.NewEncoder(os.Stdout).Encode(row))
}

// All timestamps use the host CLOCK_MONOTONIC shared by these containers.
func monoNS() int64 {
	var ts unix.Timespec
	must(unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts))
	return ts.Nano()
}
func clockSample() map[string]string {
	return map[string]string{"mono_ns": strconv.FormatInt(monoNS(), 10), "wall": time.Now().UTC().Format(time.RFC3339Nano)}
}
func deadlineProbes() {
	identity := clockIdentity()
	deadline, e := strconv.ParseInt(os.Args[3], 10, 64)
	must(e)
	f, e := os.Create(os.Args[4])
	must(e)
	defer f.Close()
	// The main cadence is phased at D-(100ms+1ms). A second cadence has
	// an arbitrary 37ms offset and shares the explicit D-101ms boundary sample.
	type scheduled struct {
		at    int64
		phase string
	}
	var schedule []scheduled
	firstN, lastN := int64(-80), int64(20)
	if os.Args[1] == "renewal-probes" {
		deadline = monoNS() + 70000000000
		firstN, lastN = -698, -1
	}
	for n := firstN; n <= lastN; n++ {
		schedule = append(schedule, scheduled{deadline - 101000000 + n*100000000, "phased"})
		schedule = append(schedule, scheduled{deadline - 64000000 + n*100000000, "arbitrary_37ms"})
	}
	// D-140: one planning checkpoint after the original cadence has supplied
	// latency samples. It is not a packet and changes neither original cadence.
	if os.Args[1] == "deadline-probes" {
		schedule = append(schedule, scheduled{deadline - 1000000000, "plan_dense"})
	}
	sort.Slice(schedule, func(i, j int) bool { return schedule[i].at < schedule[j].at })
	var sent = map[int]int64{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	id := 0
	for index := 0; index < len(schedule); index++ {
		s := schedule[index]
		if s.phase == "dense_boundary" {
			// Avoid timer coalescing at sub-millisecond spacing. This affects only
			// added probes; send/read behavior and both old cadences stay intact.
			if delay := s.at - monoNS() - 1500000; delay > 0 {
				time.Sleep(time.Duration(delay))
			}
			for monoNS() < s.at {
				runtime.Gosched()
			}
		} else if delay := s.at - monoNS(); delay > 0 {
			time.Sleep(time.Duration(delay))
		}
		if s.phase == "plan_dense" {
			plan := map[string]any{"planned_at": clockSample(), "deadline_mono_ns": fmt.Sprint(deadline), "qualified": false,
				"scope": "D-140 added schedule only; same-run original-cadence capture-read latency, not exact kernel timestamps"}
			var captures []map[string]any
			if len(os.Args) > 5 {
				raw, err := os.ReadFile(os.Args[5])
				if err == nil {
					for _, line := range bytes.Split(raw, []byte("\n")) {
						var row map[string]any
						if json.Unmarshal(line, &row) == nil {
							captures = append(captures, row)
						}
					}
				}
			}
			ingress, egress := map[int]int64{}, map[int]int64{}
			for _, r := range captures {
				n, _ := r["packet_id"].(float64)
				stamp, _ := r["capture_read"].(map[string]any)
				v, _ := stamp["mono_ns"].(string)
				at, _ := strconv.ParseInt(v, 10, 64)
				if r["interface"] == "fvmb0" && ingress[int(n)] == 0 {
					ingress[int(n)] = at
				}
				if r["interface"] == "fvup0" && egress[int(n)] == 0 {
					egress[int(n)] = at
				}
			}
			var latencies []int64
			var samples []map[string]any
			var maxTransit int64
			mu.Lock()
			for n, in := range ingress {
				out, send := egress[n], sent[n]
				if send <= 0 || in < send || out < in {
					continue
				}
				latencies = append(latencies, in-send)
				maxTransit = max(maxTransit, out-in)
				samples = append(samples, map[string]any{"packet_id": n, "send_mono_ns": fmt.Sprint(send), "ingress_mono_ns": fmt.Sprint(in), "egress_mono_ns": fmt.Sprint(out), "latency_ns": fmt.Sprint(in - send), "transit_ns": fmt.Sprint(out - in)})
			}
			mu.Unlock()
			sort.Slice(samples, func(i, j int) bool { return samples[i]["packet_id"].(int) < samples[j]["packet_id"].(int) })
			plan["samples"] = samples
			if len(latencies) >= 20 {
				sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
				lo, hi, median := latencies[0], latencies[len(latencies)-1], latencies[len(latencies)/2]
				// Include arrival-latency spread: a fast following packet can catch
				// a slow predecessor even when send spacing exceeds node transit.
				// Round upward to 0.1ms; actual noninterleaving still needs capture.
				spacing := ((max(int64(500000), (hi-lo)+2*maxTransit+100000) + 99999) / 100000) * 100000
				margin := hi - lo + 3*spacing
				first, last := deadline-101000000-margin, deadline+100000000+margin
				plan["qualified"] = true
				plan["latency_min_ns"], plan["latency_median_ns"], plan["latency_max_ns"] = fmt.Sprint(lo), fmt.Sprint(median), fmt.Sprint(hi)
				plan["latency_spread_ns"], plan["max_transit_ns"], plan["spacing_ns"] = fmt.Sprint(hi-lo), fmt.Sprint(maxTransit), fmt.Sprint(spacing)
				plan["predicted_first_ingress_ns"], plan["predicted_last_ingress_ns"] = fmt.Sprint(first), fmt.Sprint(last)
				plan["spacing_reason"] = "max(0.5ms, measured latency spread + twice measured maximum transit + 0.1ms), rounded up to 0.1ms; avoid existing cadence sends within one spacing; verify actual ingress/egress order"
				var added []string
				for predicted := first; predicted <= last; predicted += spacing {
					at := predicted - median
					clear := true
					for _, old := range schedule[index+1:] {
						if old.phase == "dense_boundary" {
							continue
						}
						delta := old.at - at
						if delta > -spacing && delta < spacing {
							clear = false
							break
						}
					}
					if clear {
						schedule = append(schedule, scheduled{at, "dense_boundary"})
						added = append(added, fmt.Sprint(at))
					}
				}
				plan["added_scheduled_mono_ns"] = added
				sort.Slice(schedule[index+1:], func(i, j int) bool { return schedule[index+1+i].at < schedule[index+1+j].at })
			}
			raw, err := json.MarshalIndent(plan, "", "  ")
			must(err)
			must(os.WriteFile(os.Args[4]+".schedule.json", append(raw, '\n'), 0600))
			continue
		}
		id++
		wg.Add(1)
		go func(id int, s scheduled) {
			defer wg.Done()
			port := 42000 + id
			dst := &net.UDPAddr{IP: net.ParseIP(os.Args[2]), Port: 8080}
			c, e := net.DialUDP("udp4", &net.UDPAddr{Port: port}, dst)
			row := map[string]any{"trace_clock": identity, "packet_id": id, "source_port": port, "phase": s.phase, "scheduled_mono_ns": fmt.Sprint(s.at), "deadline_mono_ns": fmt.Sprint(deadline), "send": clockSample()}
			mu.Lock()
			sent[id], _ = strconv.ParseInt(row["send"].(map[string]string)["mono_ns"], 10, 64)
			mu.Unlock()
			result := "socket_error"
			if e == nil {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(80 * time.Millisecond))
				_, e = c.Write([]byte("TEST ONLY"))
				if e == nil {
					b := make([]byte, 9)
					n, err := c.Read(b)
					result = "timeout"
					if err == nil && n == 9 && string(b) == "TEST ONLY" {
						result = "ok"
					}
				} else {
					result = "send_error"
				}
			}
			row["read_completion"] = clockSample()
			row["result"] = result
			mu.Lock()
			_ = json.NewEncoder(f).Encode(row)
			mu.Unlock()
		}(id, s)
	}
	wg.Wait()
}
func packetClock() {
	seconds, e := strconv.Atoi(os.Args[2])
	must(e)
	f, e := os.Create(os.Args[3])
	must(e)
	defer f.Close()
	// SOCK_DGRAM strips link headers on both WireGuard and Ethernet. Retain
	// only UDP headers for TEST ONLY source-port packet IDs, never payload.
	fd, e := unix.Socket(unix.AF_PACKET, unix.SOCK_DGRAM|unix.SOCK_NONBLOCK, int(0x0300))
	must(e)
	defer unix.Close(fd)
	ifaces, e := net.Interfaces()
	must(e)
	names := map[int]string{}
	for _, it := range ifaces {
		names[it.Index] = it.Name
	}
	_ = json.NewEncoder(f).Encode(map[string]any{"event": "ready", "clock": clockSample(), "trace_clock": clockIdentity()})
	end := monoNS() + int64(seconds)*1000000000
	b := make([]byte, 65536)
	for monoNS() < end {
		n, sa, e := unix.Recvfrom(fd, b, 0)
		stamp := clockSample()
		if e == unix.EAGAIN {
			time.Sleep(200 * time.Microsecond)
			continue
		}
		if e != nil {
			must(e)
		}
		link, ok := sa.(*unix.SockaddrLinklayer)
		if !ok || n < 28 || b[0]>>4 != 4 || b[9] != 17 {
			continue
		}
		ihl := int(b[0]&15) * 4
		if ihl < 20 || n < ihl+8 {
			continue
		}
		src, dst := int(binary.BigEndian.Uint16(b[ihl:])), int(binary.BigEndian.Uint16(b[ihl+2:]))
		id := src - 42000
		if dst != 8080 || id < 1 || id > 2000 {
			continue
		}
		name := names[link.Ifindex]
		if name != "fvmb0" && name != "fvup0" {
			continue
		}
		_ = json.NewEncoder(f).Encode(map[string]any{"packet_id": id, "interface": name, "packet_type": link.Pkttype, "capture_read": stamp, "source_port": src, "destination_port": dst})
	}
	stats, e := unix.GetsockoptTpacketStats(fd, unix.SOL_PACKET, unix.PACKET_STATISTICS)
	row := map[string]any{"event": "complete", "clock": clockSample(), "statistics_available": e == nil}
	if e == nil {
		row["packets"] = stats.Packets
		row["drops"] = stats.Drops
	}
	_ = json.NewEncoder(f).Encode(row)
}

// Passive netlink monitor: timestamps are userspace receipt bounds, never
// mislabelled as renderer start or command completion.
func nftClock() {
	f, e := os.OpenFile("/var/lib/vpn-node/e2e/isolation-nft-events.jsonl", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	must(e)
	defer f.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "nft", "-j", "monitor")
	pipe, e := cmd.StdoutPipe()
	must(e)
	must(cmd.Start())
	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 65536), 8<<20)
	for scanner.Scan() {
		at := clockSample()
		line := scanner.Bytes()
		var doc map[string]any
		if json.Unmarshal(line, &doc) != nil {
			continue
		}
		entries, _ := doc["nftables"].([]any)
		if len(entries) == 0 {
			entries = []any{doc}
		}
		for _, entry := range entries {
			actions, _ := entry.(map[string]any)
			for action, obj := range actions {
				kinds, _ := obj.(map[string]any)
				for kind, body := range kinds {
					b, _ := body.(map[string]any)
					if b["table"] != "fvpn" {
						continue
					}
					name, _ := b["name"].(string)
					chain, _ := b["chain"].(string)
					if name != "fv_deadline_guard" && name != "allowed_v4" && kind != "rule" {
						continue
					}
					row := map[string]any{"receipt": at, "action": action, "kind": kind, "name": name, "chain": chain}
					if comment, ok := b["comment"].(string); ok && strings.HasPrefix(comment, "fvpn_generation=") {
						row["generation_comment"] = comment
					}
					_ = json.NewEncoder(f).Encode(row)
				}
			}
		}
	}
	_ = cmd.Wait()
}

// Read the live SQLite outbox without copying a WAL in flight or changing it.
func observationPending() {
	db, err := sql.Open("sqlite", "file:/var/lib/vpn-node/state/agent.db?mode=ro")
	must(err)
	defer db.Close()
	var pending int
	must(db.QueryRow("SELECT count(*) FROM outbox WHERE status = 'pending'").Scan(&pending))
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{"pending": pending, "at": time.Now().UTC().Format(time.RFC3339Nano)}))
}

// TEST ONLY terminal stop sample inside the lab. Preserve the supervisor pause
// contract, signal the exact process through pidfd, and observe its exit. The
// supervisor retains its 15-second policy. The matching parent wait receipt
// supplies exit status; elapsed time alone never establishes graceful exit.
func agentStopSample() {
	raw, err := os.ReadFile("/var/lib/vpn-node/e2e/agent.pid")
	must(err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	must(err)
	fd, err := unix.PidfdOpen(pid, 0)
	must(err)
	defer unix.Close(fd)
	launchRaw, err := os.ReadFile("/var/lib/vpn-node/e2e/agent-process-id")
	must(err)
	launch := strings.TrimSpace(string(launchRaw))
	if launch == "" || launch == "." || filepath.Base(launch) != launch {
		panic("invalid launch ID")
	}
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	must(err)
	fields := strings.Fields(string(stat)[strings.LastIndex(string(stat), ")")+1:])
	if len(fields) < 20 {
		panic("short process stat")
	}
	startTicks := fields[19]
	row := map[string]any{"pid": pid, "launch_id": launch, "start_ticks": startTicks, "agent_budget_seconds": 15, "scope": "TEST ONLY terminal stop; pidfd SIGTERM send and exit readiness brackets; supervisor pause and escalation retained", "step_cancel": "not observable here"}
	must(os.WriteFile("/var/lib/vpn-node/e2e/pause-agent", nil, 0600))
	start := monoNS()
	row["signal_start"] = clockSample()
	err = unix.PidfdSendSignal(fd, unix.SIGTERM, nil, 0)
	row["signal_return"] = clockSample()
	row["signal_ok"] = err == nil
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	lastAlive := start
	exited := false
	for monoNS()-start < int64(15*time.Second) {
		n, e := unix.Poll(fds, 10)
		if e == unix.EINTR {
			continue
		}
		if e != nil {
			break
		}
		if n > 0 && fds[0].Revents&unix.POLLIN != 0 {
			exited = true
			break
		}
		lastAlive = monoNS()
	}
	end := monoNS()
	row["exit_observed"], row["exit_sample"] = exited, clockSample()
	row["signal_to_exit_upper_ns"], row["last_alive_since_signal_ns"] = strconv.FormatInt(end-start, 10), strconv.FormatInt(lastAlive-start, 10)
	row["within_15s"] = err == nil && exited && end-start <= int64(15*time.Second)
	cleanup, e := exec.Command("agent-service.sh", "stop").CombinedOutput()
	row["supervisor_stop_ok"], row["supervisor_stop_output"] = e == nil, strings.TrimSpace(string(cleanup))
	row["parent_wait_available"] = false
	for i := 0; i < 200; i++ {
		raw, e := os.ReadFile("/var/lib/vpn-node/e2e/agent-exits/" + launch + ".json")
		if e == nil {
			var receipt struct {
				ID         string `json:"id"`
				PID        int    `json:"pid"`
				StartTicks string `json:"start_ticks"`
				ExitCode   *int   `json:"exit_code"`
				Signal     *int   `json:"signal"`
			}
			if json.Unmarshal(raw, &receipt) == nil {
				matched := receipt.ID == launch && receipt.PID == pid && receipt.StartTicks == startTicks
				row["parent_wait_receipt"], row["parent_wait_available"], row["parent_identity_matched"] = json.RawMessage(raw), true, matched
				row["cooperative_exit"] = matched && receipt.Signal == nil && receipt.ExitCode != nil && *receipt.ExitCode == 0
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if raw, e := os.ReadFile("/var/lib/vpn-node/e2e/agent-exits/" + launch + ".sigkill.json"); e == nil && json.Valid(raw) {
		row["attempted_escalation"] = json.RawMessage(raw)
	}
	row["outcome_source"] = "parent wait receipt; escalation sidecar is intent only; pidfd timestamps bound observation, not exact kernel exit"
	must(json.NewEncoder(os.Stdout).Encode(row))
}
