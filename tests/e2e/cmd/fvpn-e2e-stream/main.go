// Command fvpn-e2e-stream emits a bounded TEST ONLY native member stream.
// Only synthetic sequence IDs and socket-write bounds are exported.
package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"net"
	"os"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture"
)

const magic = "FVPN TEST ONLY 27"

type attempt struct {
	Sequence uint64              `json:"sequence,string"`
	Before   nativecapture.Clock `json:"before"`
	After    nativecapture.Clock `json:"after"`
	Sent     bool                `json:"sent"`
}

func main() {
	addr := flag.String("addr", "", "owned origin IPv4:40002")
	duration := flag.Int("seconds", 90, "bounded stream duration")
	flag.Parse()
	target, err := net.ResolveUDPAddr("udp4", *addr)
	if err != nil || target.Port != 40002 || *duration < 1 || *duration > 180 {
		os.Exit(2)
	}
	// Existing devstack slots use only these isolated controlled origin IPs.
	ip := target.IP.To4()
	if ip == nil || ip[0] != 192 || ip[1] != 0 || ip[2] < 3 || ip[2] > 12 || ip[3] != 10 {
		os.Exit(2)
	}
	c, err := net.ListenUDP("udp4", &net.UDPAddr{Port: 40001})
	if err != nil {
		os.Exit(1)
	}
	defer c.Close()
	rows := []attempt{}
	end := time.Now().Add(time.Duration(*duration) * time.Second)
	for sequence := uint64(0); time.Now().Before(end); sequence++ {
		body := binary.BigEndian.AppendUint64([]byte(magic), sequence)
		before, e1 := nativecapture.Now()
		n, e := c.WriteToUDP(body, target)
		after, e2 := nativecapture.Now()
		if e1 != nil || e2 != nil {
			os.Exit(1)
		}
		rows = append(rows, attempt{sequence, before, after, e == nil && n == len(body)})
		time.Sleep(20 * time.Millisecond)
	}
	if json.NewEncoder(os.Stdout).Encode(rows) != nil {
		os.Exit(1)
	}
}
