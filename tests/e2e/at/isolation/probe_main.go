//go:build ignore

// TEST ONLY controlled socket fixture; never run on the host network.
package main

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(2)
	}
	switch os.Args[1] {
	case "deadline-probes", "renewal-probes":
		deadlineProbes()
	case "nft-clock":
		nftClock()
	case "packet-clock":
		packetClock()
	case "clock":
		_ = json.NewEncoder(os.Stdout).Encode(clockSample())
	case "observe-proxy":
		observeProxy()
	case "configure-observation":
		configureObservation()
	case "producer-clock":
		producerClock()
	case "timed":
		seconds, _ := strconv.Atoi(os.Args[4])
		end := time.Now().Add(time.Duration(seconds) * time.Second)
		f, e := os.Create(os.Args[5])
		must(e)
		defer f.Close()
		for time.Now().Before(end) {
			start := time.Now()
			c, e := net.DialTimeout("tcp4", net.JoinHostPort(os.Args[2], os.Args[3]), 150*time.Millisecond)
			result := "timeout"
			if e == nil {
				result = "connected"
				c.Close()
			} else if strings.Contains(e.Error(), "refused") {
				result = "refused"
			}
			_ = json.NewEncoder(f).Encode(map[string]any{"start": start.UTC().Format(time.RFC3339Nano), "end": time.Now().UTC().Format(time.RFC3339Nano), "result": result})
			time.Sleep(50 * time.Millisecond)
		}
	case "pulse":
		seconds, _ := strconv.Atoi(os.Args[3])
		end := time.Now().Add(time.Duration(seconds) * time.Second)
		c, e := net.Dial("udp4", net.JoinHostPort(os.Args[2], "9"))
		must(e)
		defer c.Close()
		for time.Now().Before(end) {
			_, _ = c.Write([]byte("TEST ONLY"))
			time.Sleep(50 * time.Millisecond)
		}
	case "loop":
		seconds, _ := strconv.Atoi(os.Args[3])
		end := time.Now().Add(time.Duration(seconds) * time.Second)
		var attempts, delivered atomic.Uint64
		var workers sync.WaitGroup
		start := time.Now()
		for range 8 {
			workers.Go(func() {
				for time.Now().Before(end) {
					c, e := net.DialTimeout("tcp4", net.JoinHostPort(os.Args[2], "8080"), 100*time.Millisecond)
					attempts.Add(1)
					if e == nil {
						delivered.Add(1)
						c.Close()
					} else if strings.Contains(e.Error(), "refused") {
						delivered.Add(1)
					}
					time.Sleep(20 * time.Millisecond)
				}
			})
		}
		workers.Wait()
		raw, _ := json.Marshal(map[string]any{"attempts": attempts.Load(), "delivered": delivered.Load(), "seconds": time.Since(start).Seconds()})
		must(os.WriteFile(os.Args[4], raw, 0600))
	case "serve":
		udp, err := net.ListenPacket("udp4", ":8080")
		must(err)
		go func() {
			b := make([]byte, 64)
			for {
				n, a, e := udp.ReadFrom(b)
				if e != nil {
					return
				}
				_, _ = udp.WriteTo(b[:n], a)
			}
		}()
		ln, err := net.Listen("tcp4", ":8080")
		must(err)
		for {
			c, e := ln.Accept()
			if e != nil {
				return
			}
			go func() { defer c.Close(); _, _ = io.Copy(c, c) }()
		}
	case "connect":
		c, e := net.DialTimeout("tcp", net.JoinHostPort(os.Args[2], os.Args[3]), 500*time.Millisecond)
		result := "timeout"
		if e == nil {
			result = "connected"
			c.Close()
		} else if strings.Contains(e.Error(), "refused") {
			result = "refused"
		} else if ne, ok := e.(net.Error); !ok || !ne.Timeout() {
			result = "unreachable"
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"result": result})
	case "stream":
		seconds, _ := strconv.Atoi(os.Args[3])
		path := os.Args[4]
		tcp, e := net.DialTimeout("tcp4", net.JoinHostPort(os.Args[2], "8080"), time.Second)
		must(e)
		defer tcp.Close()
		udp, e := net.Dial("udp4", net.JoinHostPort(os.Args[2], "8080"))
		must(e)
		defer udp.Close()
		stats := map[string]any{"started_at": time.Now().UTC().Format(time.RFC3339Nano), "tcp_ok": 0, "udp_ok": 0, "tcp_first_fail": "", "udp_first_fail": ""}
		end := time.Now().Add(time.Duration(seconds) * time.Second)
		for time.Now().Before(end) {
			for name, c := range map[string]net.Conn{"tcp": tcp, "udp": udp} {
				_ = c.SetDeadline(time.Now().Add(250 * time.Millisecond))
				_, err := c.Write([]byte("fixture"))
				b := make([]byte, 7)
				if err == nil {
					_, err = io.ReadFull(c, b)
				}
				if err == nil && string(b) == "fixture" {
					stats[name+"_ok"] = stats[name+"_ok"].(int) + 1
				} else if stats[name+"_first_fail"] == "" {
					stats[name+"_first_fail"] = time.Now().UTC().Format(time.RFC3339Nano)
				}
			}
			stats["sample_at"] = time.Now().UTC().Format(time.RFC3339Nano)
			raw, _ := json.Marshal(stats)
			_ = os.WriteFile(path+".tmp", raw, 0600)
			_ = os.Rename(path+".tmp", path)
			time.Sleep(100 * time.Millisecond)
		}
	default:
		os.Exit(2)
	}
}
func must(err error) {
	if err != nil {
		os.Exit(1)
	}
}
