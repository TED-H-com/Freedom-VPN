package main

import (
	"encoding/binary"
	"io"
	"net"
	"time"
)

// serveDNSRelay is TEST ONLY plumbing from the isolated origin network to the
// product-managed filtering resolver. No query names or packet data are logged.
func serveDNSRelay(addr, upstream string) error {
	tcp, err := net.Listen("tcp4", addr)
	if err != nil {
		return err
	}
	defer tcp.Close()
	udp, err := net.ListenPacket("udp4", addr)
	if err != nil {
		return err
	}
	defer udp.Close()
	slots := make(chan struct{}, 64)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, peer, err := udp.ReadFrom(buf)
			if err != nil {
				return
			}
			packet := append([]byte(nil), buf[:n]...)
			select {
			case slots <- struct{}{}:
				go func() {
					defer func() { <-slots }()
					if response, err := relayDNS("udp", upstream, packet); err == nil {
						_, _ = udp.WriteTo(response, peer)
					}
				}()
			default:
			}
		}
	}()
	for {
		conn, err := tcp.Accept()
		if err != nil {
			return err
		}
		select {
		case slots <- struct{}{}:
			go func() {
				defer func() { <-slots }()
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				var size [2]byte
				if _, err := io.ReadFull(conn, size[:]); err != nil {
					return
				}
				n := int(binary.BigEndian.Uint16(size[:]))
				if n > 4096 {
					return
				}
				packet := make([]byte, n)
				if _, err := io.ReadFull(conn, packet); err != nil {
					return
				}
				response, err := relayDNS("tcp", upstream, packet)
				if err != nil {
					return
				}
				binary.BigEndian.PutUint16(size[:], uint16(len(response)))
				_, _ = conn.Write(append(size[:], response...))
			}()
		default:
			conn.Close()
		}
	}
}
func relayDNS(network, upstream string, packet []byte) ([]byte, error) {
	conn, err := net.DialTimeout(network, upstream, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if network == "tcp" {
		packet = append([]byte{byte(len(packet) >> 8), byte(len(packet))}, packet...)
	}
	if _, err = conn.Write(packet); err != nil {
		return nil, err
	}
	if network == "tcp" {
		var size [2]byte
		if _, err = io.ReadFull(conn, size[:]); err != nil {
			return nil, err
		}
		n := int(binary.BigEndian.Uint16(size[:]))
		if n > 4096 {
			return nil, io.ErrShortBuffer
		}
		buf := make([]byte, n)
		_, err = io.ReadFull(conn, buf)
		return buf, err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	return buf[:n], err
}
