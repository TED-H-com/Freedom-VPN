package dnsrelay

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

func TestAT12DNSRelayPreservesTCPAndUDPPayloads(t *testing.T) {
	for _, network := range []string{"tcp", "udp"} {
		t.Run(network, func(t *testing.T) {
			payload := []byte{0x12, 0x34, 0x81, 0x80, 0, 0, 0, 0, 0, 0, 0, 0}
			var addr string
			if network == "udp" {
				pc, err := net.ListenPacket("udp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				defer pc.Close()
				addr = pc.LocalAddr().String()
				go func() {
					buf := make([]byte, 512)
					n, peer, err := pc.ReadFrom(buf)
					if err == nil {
						_, _ = pc.WriteTo(buf[:n], peer)
					}
				}()
			} else {
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				defer ln.Close()
				addr = ln.Addr().String()
				go func() {
					c, err := ln.Accept()
					if err != nil {
						return
					}
					defer c.Close()
					var sz [2]byte
					if _, err = io.ReadFull(c, sz[:]); err != nil {
						return
					}
					buf := make([]byte, int(binary.BigEndian.Uint16(sz[:])))
					if _, err = io.ReadFull(c, buf); err == nil {
						_, _ = c.Write(append(sz[:], buf...))
					}
				}()
			}
			got, err := Exchange(network, addr, payload)
			if err != nil || string(got) != string(payload) {
				t.Fatalf("relay %x: %v", got, err)
			}
		})
	}
}

func TestAT12DNSRelayRefusesOtherSources(t *testing.T) {
	tcp, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	udp, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		tcp.Close()
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _ = serve(tcp, udp, "127.0.0.1:1", "192.0.2.53"); close(done) }()
	defer func() { tcp.Close(); udp.Close(); <-done }()
	for _, tc := range []struct{ network, addr string }{{"tcp", tcp.Addr().String()}, {"udp", udp.LocalAddr().String()}} {
		c, err := net.DialTimeout(tc.network, tc.addr, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		_ = c.SetDeadline(time.Now().Add(100 * time.Millisecond))
		_, _ = c.Write([]byte{0, 1, 0})
		var b [16]byte
		n, err := c.Read(b[:])
		c.Close()
		if err == nil || n != 0 {
			t.Fatalf("%s unauthorized response", tc.network)
		}
	}
}
