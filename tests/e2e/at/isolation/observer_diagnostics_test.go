//go:build ignore

package main

import (
	"encoding/json"
	"errors"
	"net"
	"strings"
	"syscall"
	"testing"
	"time"
)

type probeStub struct {
	writeError error
	reply      string
	readError  error
	reads      int
	writes     int
	deadline   time.Time
}

func (p *probeStub) SetDeadline(at time.Time) error { p.deadline = at; return nil }
func (p *probeStub) Write(b []byte) (int, error)    { p.writes++; return len(b), p.writeError }
func (p *probeStub) Read(b []byte) (int, error)     { p.reads++; return copy(b, p.reply), p.readError }

func TestDeadlineProbeOccupiedPortRecordsUnderlyingErrno(t *testing.T) {
	occupied, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	conn, dialErr := net.DialUDP("udp4", occupied.LocalAddr().(*net.UDPAddr), &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9})
	if conn != nil {
		conn.Close()
		t.Fatal("occupied source port unexpectedly opened")
	}
	if dialErr == nil {
		t.Fatal("expected bind failure")
	}
	result, diagnostic := deadlineProbeExchange(nil, dialErr)
	if result != "socket_error" || diagnostic["stage"] != "dial" || diagnostic["errno"] != int(syscall.EADDRINUSE) {
		t.Fatalf("result=%s diagnostic=%v", result, diagnostic)
	}
	raw, _ := json.Marshal(diagnostic)
	if strings.Contains(string(raw), "127.0.0.1") {
		t.Fatal("endpoint leaked")
	}
}

func TestDeadlineProbeWriteFailureAndSuccessfulExchange(t *testing.T) {
	p := &probeStub{writeError: &net.OpError{Op: "write", Err: syscall.ENETUNREACH}}
	before := time.Now()
	result, diagnostic := deadlineProbeExchange(p, nil)
	if result != "send_error" || diagnostic["stage"] != "write" || diagnostic["errno"] != int(syscall.ENETUNREACH) || p.reads != 0 || p.writes != 1 {
		t.Fatalf("result=%s diagnostic=%v probe=%+v", result, diagnostic, p)
	}
	if p.deadline.Before(before.Add(80*time.Millisecond)) || p.deadline.After(time.Now().Add(80*time.Millisecond)) {
		t.Fatal("deadline changed")
	}
	for _, tc := range []struct {
		reply string
		err   error
		want  string
	}{
		{"TEST ONLY", nil, "ok"}, {"wrong", nil, "timeout"}, {"", syscall.ETIMEDOUT, "timeout"},
	} {
		p = &probeStub{reply: tc.reply, readError: tc.err}
		got, diagnostic := deadlineProbeExchange(p, nil)
		if got != tc.want || diagnostic != nil || p.reads != 1 || p.writes != 1 {
			t.Fatalf("got %s %v", got, diagnostic)
		}
	}
}

func TestDeadlineProbeDiagnosticsNeverCopyArbitraryErrorText(t *testing.T) {
	for _, err := range []error{
		errors.New("TEST ONLY password=fixture https://secret.example.invalid/path"),
		&net.OpError{Op: "write", Net: "udp4", Addr: &net.UDPAddr{IP: net.IPv4(192, 0, 2, 1), Port: 8080}, Err: syscall.EPERM},
	} {
		diagnostic := deadlineProbeError("write", err)
		raw, _ := json.Marshal(diagnostic)
		for _, forbidden := range []string{"password", "https", "192.0.2.1", "8080", "secret"} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("leaked diagnostic %s", raw)
			}
		}
		if len(diagnostic) != 3 {
			t.Fatal("unexpected diagnostic fields")
		}
	}
	d := deadlineProbeError("write", errors.New("TEST ONLY arbitrary error"))
	if d["errno"] != nil || d["timeout"] != false {
		t.Fatalf("unknown cause fabricated: %v", d)
	}
	d = deadlineProbeError("write", syscall.ETIMEDOUT)
	if d["timeout"] != true {
		t.Fatalf("timeout lost: %v", d)
	}
}
