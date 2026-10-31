// Command fvpn-e2e-member-session keeps one member TCP session and one
// authenticated UDP echo open, or sends one DNS query. TEST ONLY.
// The UDP key is read from a mode-0600 file and is never printed.
package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth"
	"github.com/google/uuid"
)

func main() {
	dns := flag.String("dns", "", "one UDP DNS query to host:port, then exit")
	dnsTCP := flag.Bool("dns-tcp", false, "use TCP for the DNS query")
	name := flag.String("name", "", "DNS name for -dns")
	tcpAddr := flag.String("tcp", "", "member TCP echo address")
	udpAddr := flag.String("udp", "", "member UDP echo address")
	device := flag.String("device", "", "device UUID")
	keyFile := flag.String("key-file", "", "mode-0600 UDP key file")
	pidFile := flag.String("pid-file", "", "pid file")
	statusFile := flag.String("status-file", "", "status file")
	interval := flag.Duration("interval", 500*time.Millisecond, "echo interval")
	flag.Parse()
	if *dns != "" {
		os.Exit(runDNS(*dns, *name, *dnsTCP))
	}
	os.Exit(runSession(*tcpAddr, *udpAddr, *device, *keyFile, *pidFile, *statusFile, *interval))
}

func runDNS(addr, name string, tcp bool) int {
	if addr == "" || name == "" || strings.ContainsAny(name, " \t\r\n") {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: dns address and name are required")
		return 2
	}
	query, id, err := dnsQuery(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: dns name")
		return 2
	}
	network := "udp"
	if tcp {
		network = "tcp"
	}
	conn, err := net.DialTimeout(network, addr, 3*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: dns dial")
		return 1
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if tcp {
		query = append([]byte{byte(len(query) >> 8), byte(len(query))}, query...)
	}
	if _, err = conn.Write(query); err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: dns write")
		return 1
	}
	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if tcp && err == nil {
		if n < 2 {
			return 1
		}
		size := int(binary.BigEndian.Uint16(buf[:2]))
		if size > len(buf)-2 {
			return 1
		}
		for n < size+2 {
			more, e := conn.Read(buf[n:])
			if e != nil {
				return 1
			}
			n += more
		}
		buf = buf[2:n]
		n -= 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: dns read")
		return 1
	}
	rcode, answers := dnsAnswers(buf[:n], id)
	if rcode < 0 {
		return 1
	}
	fmt.Printf("rcode=%d answers=%d\n", rcode, len(answers))
	return 0
}

func runSession(tcpAddr, udpAddr, device, keyFile, pidFile, statusFile string, interval time.Duration) int {
	if tcpAddr == "" || udpAddr == "" || pidFile == "" || statusFile == "" || interval <= 0 {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: tcp, udp, pid-file and status-file are required")
		return 2
	}
	id, err := uuid.Parse(device)
	if err != nil || id == uuid.Nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: device")
		return 2
	}
	info, err := os.Stat(keyFile)
	if err != nil || info.Mode().Perm() != 0o600 || !info.Mode().IsRegular() {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: key file mode")
		return 64
	}
	key, err := os.ReadFile(keyFile)
	if err != nil || len(strings.TrimSpace(string(key))) == 0 {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: key file")
		return 1
	}
	key = []byte(strings.TrimSpace(string(key)))
	defer os.Remove(keyFile)
	defer os.Remove(pidFile)
	if err = os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: pid file")
		return 1
	}
	tcp, err := net.DialTimeout("tcp", tcpAddr, 5*time.Second)
	if err != nil {
		writeStatus(statusFile, 0, 0, "tcp")
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: tcp dial")
		return 1
	}
	defer tcp.Close()
	if _, err = tcp.Write([]byte("MEMBER " + id.String() + "\n")); err != nil {
		writeStatus(statusFile, 0, 0, "tcp")
		return 1
	}
	reader := bufio.NewReader(tcp)
	_ = tcp.SetReadDeadline(time.Now().Add(5 * time.Second))
	hello, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(hello) != "OK" {
		writeStatus(statusFile, 0, 0, "tcp")
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: tcp hello")
		return 1
	}
	udp, err := net.DialTimeout("udp", udpAddr, 5*time.Second)
	if err != nil {
		writeStatus(statusFile, 0, 0, "udp")
		fmt.Fprintln(os.Stderr, "fvpn-e2e-member-session: udp dial")
		return 1
	}
	defer udp.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	// Both transports continue independently on their original sockets after a cutoff.
	// Failed TCP never reconnects; UDP continues attempting on the same tuple.
	ctx, timeout := context.WithTimeout(ctx, 20*time.Minute)
	defer timeout()
	var tcpN, udpN, tcpAttempts, udpAttempts atomic.Int64
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				tcpAttempts.Add(1)
				_ = tcp.SetDeadline(time.Now().Add(time.Second))
				if _, err := fmt.Fprintln(tcp, "ping"); err != nil {
					continue
				}
				if line, err := reader.ReadString('\n'); err == nil && line == "ECHO ping\n" {
					tcpN.Add(1)
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				udpAttempts.Add(1)
				_ = udp.SetDeadline(time.Now().Add(300 * time.Millisecond))
				if _, err := udp.Write(probeauth.Seal(key, id, false)); err != nil {
					continue
				}
				buf := make([]byte, probeauth.Size)
				if n, err := udp.Read(buf); err == nil {
					if got, ok := probeauth.Open(key, buf[:n], true); ok && got == id {
						udpN.Add(1)
					}
				}
			}
		}
	}()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	write := func(running bool) {
		writeSessionStatus(statusFile, map[string]any{"tcp_echoes": tcpN.Load(), "udp_replies": udpN.Load(), "tcp_attempts": tcpAttempts.Load(), "udp_attempts": udpAttempts.Load(), "running": running})
	}
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			write(false)
			return 0
		case <-tick.C:
			write(true)
		}
	}
}

func writeStatus(path string, tcpN, udpN int, problem string) {
	writeSessionStatus(path, map[string]any{"tcp_echoes": tcpN, "udp_replies": udpN, "error": problem, "running": false})
}
func writeSessionStatus(path string, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, append(body, '\n'), 0o644) == nil {
		_ = os.Rename(tmp, path)
	}
}

func dnsQuery(name string) ([]byte, uint16, error) {
	labels := strings.Split(strings.TrimSuffix(name, "."), ".")
	q := make([]byte, 12)
	id := uint16(0x0e2e)
	binary.BigEndian.PutUint16(q[0:2], id)
	binary.BigEndian.PutUint16(q[2:4], 0x0100)
	binary.BigEndian.PutUint16(q[4:6], 1)
	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return nil, 0, fmt.Errorf("label")
		}
		q = append(q, byte(len(label)))
		q = append(q, label...)
	}
	q = append(q, 0, 0, 1, 0, 1)
	return q, id, nil
}

func dnsAnswers(raw []byte, id uint16) (int, []string) {
	if len(raw) < 12 || binary.BigEndian.Uint16(raw[0:2]) != id || raw[2]&0x80 == 0 {
		return -1, nil
	}
	rcode := int(raw[3] & 0x0f)
	qd := int(binary.BigEndian.Uint16(raw[4:6]))
	an := int(binary.BigEndian.Uint16(raw[6:8]))
	off := 12
	for i := 0; i < qd && off < len(raw); i++ {
		off = skipName(raw, off)
		off += 4
	}
	var answers []string
	for i := 0; i < an && off+12 <= len(raw); i++ {
		off = skipName(raw, off)
		if off+10 > len(raw) {
			break
		}
		typ := binary.BigEndian.Uint16(raw[off : off+2])
		rdlen := int(binary.BigEndian.Uint16(raw[off+8 : off+10]))
		off += 10
		if off+rdlen > len(raw) {
			break
		}
		if typ == 1 && rdlen == 4 {
			answers = append(answers, net.IP(raw[off:off+4]).String())
		}
		off += rdlen
	}
	return rcode, answers
}

func skipName(raw []byte, off int) int {
	for off < len(raw) {
		n := int(raw[off])
		if n == 0 {
			return off + 1
		}
		if n&0xc0 == 0xc0 {
			return off + 2
		}
		off += 1 + n
	}
	return len(raw)
}
