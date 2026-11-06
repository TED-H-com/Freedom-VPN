package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestAT20_DNSHelperLocalUDP(t *testing.T) {
	for _, respond := range []bool{false, true} {
		t.Run(fmt.Sprint(respond), func(t *testing.T) {
			server, err := net.ListenPacket("udp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			server.SetDeadline(time.Now().Add(5 * time.Second))
			done := make(chan error, 1)
			go func() {
				buf := make([]byte, 1500)
				n, addr, err := server.ReadFrom(buf)
				if err == nil && respond {
					_, err = server.WriteTo(dnsTestResponse(buf[:n]), addr)
				}
				done <- err
			}()
			var stdout, stderr bytes.Buffer
			code := runDNSOutput(server.LocalAddr().String(), "test-secret.example.invalid", false, &stdout, &stderr)
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if respond {
				if code != 0 || stdout.String() != "rcode=0 answers=1\n" || stderr.Len() != 0 {
					t.Fatalf("success changed: code=%d stdout=%q stderr=%q", code, &stdout, &stderr)
				}
			} else if code != 1 || stdout.Len() != 0 || stderr.String() != "fvpn-e2e-member-session: dns read class=timeout\n" {
				t.Fatalf("timeout classification lost: code=%d stdout=%q stderr=%q", code, &stdout, &stderr)
			}
		})
	}
}

func TestAT20_DNSHelperLocalTCP(t *testing.T) {
	for _, response := range []string{"eof", "partial_eof", "success"} {
		t.Run(response, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			if err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					done <- err
					return
				}
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(5 * time.Second))
				var header [2]byte
				_, err = io.ReadFull(conn, header[:])
				query := make([]byte, int(binary.BigEndian.Uint16(header[:])))
				if err == nil {
					_, err = io.ReadFull(conn, query)
				}
				if err == nil {
					switch response {
					case "partial_eof":
						_, err = conn.Write([]byte{0, 12})
					case "success":
						reply := dnsTestResponse(query)
						_, err = conn.Write(append([]byte{byte(len(reply) >> 8), byte(len(reply))}, reply...))
					}
				}
				done <- err
			}()
			var stdout, stderr bytes.Buffer
			code := runDNSOutput(listener.Addr().String(), "test-secret.example.invalid", true, &stdout, &stderr)
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if response == "success" {
				if code != 0 || stdout.String() != "rcode=0 answers=1\n" || stderr.Len() != 0 {
					t.Fatalf("success changed: code=%d stdout=%q stderr=%q", code, &stdout, &stderr)
				}
			} else if code != 1 || stdout.Len() != 0 || stderr.String() != "fvpn-e2e-member-session: dns read class=eof\n" {
				t.Fatalf("read classification lost: code=%d stdout=%q stderr=%q", code, &stdout, &stderr)
			}
		})
	}
}

func TestAT20_DNSHelperDialAndWriteFailures(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runDNSOutput("TEST-SECRET-INVALID-ADDRESS", "test-secret.example.invalid", false, &stdout, &stderr); code != 1 || stderr.String() != "fvpn-e2e-member-session: dns dial class=io_error\n" || stdout.Len() != 0 {
		t.Fatalf("unsafe dial failure: code=%d stdout=%q stderr=%q", code, &stdout, &stderr)
	}
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	conn, err := net.Dial("udp", server.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	query, id, err := dnsQuery("test-secret.example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	if code := exchangeDNS(conn, query, id, false, &stdout, &stderr); code != 1 || stderr.String() != "fvpn-e2e-member-session: dns write class=closed\n" || stdout.Len() != 0 {
		t.Fatalf("unsafe write failure: code=%d stdout=%q stderr=%q", code, &stdout, &stderr)
	}
}

func TestAT20_DNSHelperWrappedErrorClasses(t *testing.T) {
	for _, tc := range []struct {
		err   error
		class string
	}{
		{context.Canceled, "canceled"},
		{context.DeadlineExceeded, "timeout"},
		{os.ErrDeadlineExceeded, "timeout"},
		{syscall.ETIMEDOUT, "timeout"},
		{io.EOF, "eof"},
		{io.ErrUnexpectedEOF, "eof"},
		{net.ErrClosed, "closed"},
		{syscall.ECONNREFUSED, "refused"},
		{syscall.ECONNRESET, "reset"},
		{syscall.EHOSTUNREACH, "unreachable"},
		{syscall.ENETUNREACH, "unreachable"},
		{errors.New("TEST-SECRET-RAW-ERROR"), "io_error"},
	} {
		t.Run(tc.class+fmt.Sprint(tc.err), func(t *testing.T) {
			err := &net.OpError{Op: "TEST-SECRET-OP", Net: "TEST-SECRET-NET", Err: fmt.Errorf("TEST-SECRET-WRAPPER: %w", tc.err)}
			var out bytes.Buffer
			dnsFailure(&out, "read", err)
			if out.String() != "fvpn-e2e-member-session: dns read class="+tc.class+"\n" || strings.Contains(out.String(), "TEST-SECRET") {
				t.Fatalf("unsafe/incorrect classification: %q", &out)
			}
		})
	}
}

func dnsTestResponse(query []byte) []byte {
	// One TEST ONLY documentation A answer; copy only the transaction ID.
	reply := make([]byte, 12)
	copy(reply[:2], query[:2])
	reply[2] = 0x80
	reply[7] = 1
	return append(reply, 0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 4, 192, 0, 2, 1)
}
