//go:build linux

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const siblingStop = "/tmp/fvpn-members-sibling-stop"

var siblingMagic = []byte("FVPN TEST ONLY 12")

func monoNS() int64 {
	var t unix.Timespec
	if unix.ClockGettime(unix.CLOCK_MONOTONIC, &t) != nil {
		return 0
	}
	return t.Nano()
}
func siblingStopped() bool { _, e := os.Stat(siblingStop); return e == nil }

type siblingPacket struct {
	Sequence  uint64 `json:"sequence,string"`
	At        int64  `json:"mono_ns,string"`
	Interface string `json:"interface"`
}

func siblingSequence(b []byte) (uint64, bool) {
	if len(b) < 20 || b[0]>>4 != 4 || b[9] != 17 {
		return 0, false
	}
	h := int(b[0]&15) * 4
	if h < 20 || len(b) < h+8+len(siblingMagic)+8 || binary.BigEndian.Uint16(b[h+2:h+4]) != 40002 || !bytes.Equal(b[h+8:h+8+len(siblingMagic)], siblingMagic) {
		return 0, false
	}
	return binary.BigEndian.Uint64(b[h+8+len(siblingMagic):]), true
}

func siblingSend() {
	c, e := net.ListenUDP("udp4", &net.UDPAddr{Port: 40001})
	if e != nil {
		os.Exit(1)
	}
	defer c.Close()
	fmt.Println("ready")
	attempts, errors := 0, 0
	start := monoNS()
	for until := time.Now().Add(5 * time.Minute); time.Now().Before(until) && !siblingStopped(); time.Sleep(20 * time.Millisecond) {
		b := append([]byte{}, siblingMagic...)
		b = binary.BigEndian.AppendUint64(b, uint64(attempts))
		attempts++
		if _, e := c.WriteToUDP(b, &net.UDPAddr{IP: net.ParseIP(controlledOrigin), Port: 40002}); e != nil {
			errors++
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"start_ns": fmt.Sprint(start), "end_ns": fmt.Sprint(monoNS()), "attempts": attempts, "send_errors": errors})
}

func siblingObserve() {
	fd, e := unix.Socket(unix.AF_PACKET, unix.SOCK_DGRAM|unix.SOCK_NONBLOCK, int(htons(unix.ETH_P_ALL)))
	if e != nil {
		os.Exit(1)
	}
	defer unix.Close(fd)
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, 4<<20)
	names := map[int]string{}
	for _, name := range []string{"fvmb0", "fvup0"} {
		i, e := net.InterfaceByName(name)
		if e != nil {
			os.Exit(1)
		}
		names[i.Index] = name
	}
	boot, e := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if e != nil {
		os.Exit(1)
	}
	start := monoNS()
	packets := []siblingPacket{}
	readErrors := 0
	fmt.Println("ready")
	buf := make([]byte, 2048)
	stopAt := time.Time{}
	for until := time.Now().Add(5*time.Minute + time.Second); time.Now().Before(until); {
		if siblingStopped() && stopAt.IsZero() {
			stopAt = time.Now().Add(200 * time.Millisecond)
		}
		if !stopAt.IsZero() && time.Now().After(stopAt) {
			break
		}
		n, sa, e := unix.Recvfrom(fd, buf, 0)
		if e == unix.EAGAIN || e == unix.EWOULDBLOCK {
			time.Sleep(time.Millisecond)
			continue
		}
		if e != nil {
			readErrors++
			break
		}
		at := monoNS()
		link, ok := sa.(*unix.SockaddrLinklayer)
		if !ok {
			continue
		}
		iface := names[link.Ifindex]
		if iface == "" {
			continue
		}
		if iface == "fvmb0" && link.Pkttype == unix.PACKET_OUTGOING {
			continue
		}
		if iface == "fvup0" && link.Pkttype != unix.PACKET_OUTGOING {
			continue
		}
		seq, ok := siblingSequence(buf[:n])
		if ok {
			packets = append(packets, siblingPacket{seq, at, iface})
		}
	}
	stats, e := unix.GetsockoptTpacketStats(fd, unix.SOL_PACKET, unix.PACKET_STATISTICS)
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"boot_id": strings.TrimSpace(string(boot)), "clock": "CLOCK_MONOTONIC", "start_ns": fmt.Sprint(start), "end_ns": fmt.Sprint(monoNS()), "packets": packets, "read_errors": readErrors, "stats_available": e == nil, "capture_stats": stats})
}
