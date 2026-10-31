package main

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth"
	"github.com/google/uuid"
)

func TestMemberEchoRequiresTCPRegistration(t *testing.T) {
	echo := newMemberEcho()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	go echo.serveTCP(ln)
	key := []byte("TEST ONLY - NOT A SECRET")
	go echo.serveUDP(pc, key)

	device := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	other := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err = conn.Write([]byte("MEMBER " + device.String() + "\n")); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	okLine, err := readLimitedLine(reader, 16)
	if err != nil || okLine != "OK" {
		t.Fatalf("hello %q %v", okLine, err)
	}
	if _, err = conn.Write([]byte("ping\n")); err != nil {
		t.Fatal(err)
	}
	echoed, err := readLimitedLine(reader, 32)
	if err != nil || echoed != "ECHO ping" {
		t.Fatalf("echo %q %v", echoed, err)
	}

	udp, err := net.Dial("udp", pc.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer udp.Close()
	if _, err = udp.Write(probeauth.Seal(key, other, false)); err != nil {
		t.Fatal(err)
	}
	_ = udp.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	buf := make([]byte, probeauth.Size)
	if _, err = udp.Read(buf); err == nil {
		t.Fatal("unregistered device was answered")
	}
	if _, err = udp.Write(probeauth.Seal(key, device, false)); err != nil {
		t.Fatal(err)
	}
	_ = udp.SetReadDeadline(time.Now().Add(time.Second))
	n, err := udp.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := probeauth.Open(key, buf[:n], true)
	if !ok || got != device {
		t.Fatal("registered echo")
	}
	// A correctly shaped request with the wrong MAC must never receive a reply.
	_, _ = udp.Write(probeauth.Seal([]byte("TEST ONLY wrong key"), device, false))
	_ = udp.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	if _, err = udp.Read(buf); err == nil {
		t.Fatal("invalid MAC was answered")
	}
	_ = conn.Close()
	deadline := time.Now().Add(time.Second)
	for echo.registered(device) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if echo.registered(device) {
		t.Fatal("closed TCP registration retained")
	}
	_, _ = udp.Write(probeauth.Seal(key, device, false))
	_ = udp.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	if _, err = udp.Read(buf); err == nil {
		t.Fatal("closed session was answered")
	}

}
