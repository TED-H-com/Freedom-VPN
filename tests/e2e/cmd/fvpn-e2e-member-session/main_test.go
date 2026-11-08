package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
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

// Stub connections exercise startup failures without depending on kernel timing.
type startupConn struct {
	net.Conn
	read              *strings.Reader
	readErr, writeErr error
	closed            bool
	written           string
	readDeadline      time.Time
}

func (c *startupConn) Read(p []byte) (int, error) {
	if c.readErr != nil {
		return 0, c.readErr
	}
	return c.read.Read(p)
}
func (c *startupConn) Write(p []byte) (int, error) {
	c.written += string(p)
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	return len(p), nil
}
func (c *startupConn) Close() error                      { c.closed = true; return nil }
func (c *startupConn) SetReadDeadline(d time.Time) error { c.readDeadline = d; return nil }

func TestAT13_SessionStartupStageAndOriginalSockets(t *testing.T) {
	for _, stage := range []string{"", "tcp_dial", "tcp_hello_write", "tcp_hello_read", "tcp_hello_response", "udp_dial"} {
		t.Run(stage, func(t *testing.T) {
			tcp := &startupConn{read: strings.NewReader("OK\nECHO ping\n")}
			udp := &startupConn{}
			cause := &net.OpError{Op: "TEST ONLY private operation", Net: "TEST ONLY private destination", Err: syscall.ECONNRESET}
			if stage == "tcp_hello_write" {
				tcp.writeErr = cause
			}
			if stage == "tcp_hello_read" {
				tcp.readErr = cause
			}
			if stage == "tcp_hello_response" {
				tcp.read = strings.NewReader("TEST ONLY private peer response\n")
			}
			calls := []string{}
			before := time.Now().UTC()
			gotTCP, gotUDP, reader, failure := startSession("TEST ONLY tcp address", "TEST ONLY udp address", uuid.New(), func(network, addr string, timeout time.Duration) (net.Conn, error) {
				calls = append(calls, network)
				if timeout != 5*time.Second {
					t.Fatal("dial deadline changed")
				}
				if stage == network+"_dial" {
					return nil, cause
				}
				if network == "tcp" {
					return tcp, nil
				}
				return udp, nil
			})
			after := time.Now().UTC()
			if len(calls) < 1 || len(calls) > 2 || calls[0] != "tcp" || (len(calls) == 2 && calls[1] != "udp") {
				t.Fatalf("changed dial sequence: %v", calls)
			}
			if stage == "" {
				if failure != nil || gotTCP != tcp || gotUDP != udp || tcp.closed || udp.closed {
					t.Fatalf("changed original sockets: %+v", failure)
				}
				line, err := reader.ReadString('\n')
				if err != nil || line != "ECHO ping\n" {
					t.Fatalf("buffered reply lost: %q %v", line, err)
				}
			} else {
				class := "reset"
				protocol := "tcp"
				if stage == "tcp_hello_response" {
					class = "protocol"
				}
				if stage == "udp_dial" {
					protocol = "udp"
				}
				if failure == nil || failure.Stage != stage || failure.Class != class || failure.Error != protocol || failure.Running || failure.TCPEchoes != 0 || failure.UDPReplies != 0 {
					t.Fatalf("wrong failure: %+v", failure)
				}
				if failure.Started.Before(before) || failure.Failed.Before(failure.Started) || failure.Failed.After(after) || failure.Started.Location() != time.UTC || failure.Failed.Location() != time.UTC {
					t.Fatalf("wrong failure time: %+v", failure)
				}
				if gotTCP != nil || gotUDP != nil || reader != nil || (stage != "tcp_dial" && !tcp.closed) {
					t.Fatal("failed startup leaked connection")
				}
				raw, err := json.Marshal(failure)
				if err != nil || strings.Contains(string(raw), "TEST ONLY") {
					t.Fatalf("private data leaked: %s %v", raw, err)
				}
			}
			if stage != "tcp_dial" && tcp.written == "" {
				t.Fatal("missing member hello")
			}
			if stage != "tcp_dial" && stage != "tcp_hello_write" && (tcp.readDeadline.Before(before.Add(5*time.Second)) || tcp.readDeadline.After(after.Add(5*time.Second))) {
				t.Fatal("hello read deadline changed")
			}
			wantDials := 1
			if stage == "" || stage == "udp_dial" {
				wantDials = 2
			}
			if len(calls) != wantDials {
				t.Fatal("retried failed startup")
			}
		})
	}
}
