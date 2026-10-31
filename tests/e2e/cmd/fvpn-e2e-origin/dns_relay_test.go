package main

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
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
			got, err := relayDNS(network, addr, payload)
			if err != nil || string(got) != string(payload) {
				t.Fatalf("relay %x: %v", got, err)
			}
		})
	}
}
