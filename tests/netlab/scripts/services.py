#!/usr/bin/env python3
"""Lab listeners. They log accepts only (no payloads, no query names)."""

import socket
import sys
import threading

ROLE = sys.argv[1] if len(sys.argv) > 1 else "inet"
LOG = f"/scratch/svc-{ROLE}.log"
READY = f"/scratch/svc-{ROLE}.ready"


def note(msg: str) -> None:
    with open(LOG, "a", encoding="utf-8") as fh:
        fh.write(msg + "\n")


def peer(conn: socket.socket) -> str:
    try:
        host, port = conn.getpeername()[:2]
        return f"{host}:{port}"
    except OSError:
        return "?"


def local(conn: socket.socket) -> str:
    try:
        host, port = conn.getsockname()[:2]
        return f"{host}:{port}"
    except OSError:
        return "?"


def handle_http(conn: socket.socket, port: int) -> None:
    note(f"accept tcp {peer(conn)} -> {local(conn)}")
    try:
        conn.recv(2048)
        body = b"ok\n"
        conn.sendall(b"HTTP/1.0 200 OK\r\nContent-Length: 3\r\nConnection: close\r\n\r\n" + body)
    except OSError:
        pass
    finally:
        conn.close()


def handle_echo(conn: socket.socket, port: int) -> None:
    note(f"accept tcp {peer(conn)} -> {local(conn)}")
    try:
        while True:
            data = conn.recv(4096)
            if not data:
                break
            conn.sendall(data)
    except OSError:
        pass
    finally:
        conn.close()


def handle_dns_tcp(conn: socket.socket, port: int) -> None:
    note(f"accept tcp {peer(conn)} -> {local(conn)}")
    try:
        hdr = conn.recv(2)
        if len(hdr) < 2:
            return
        n = int.from_bytes(hdr, "big")
        query = b""
        while len(query) < n:
            chunk = conn.recv(n - len(query))
            if not chunk:
                break
            query += chunk
        resp = dns_response(query)
        conn.sendall(len(resp).to_bytes(2, "big") + resp)
    except OSError:
        pass
    finally:
        conn.close()


def dns_response(query: bytes) -> bytes:
    if len(query) < 12:
        return b""
    flags = b"\x81\x80"
    qd = query[4:6]
    header = query[:2] + flags + qd + b"\x00\x01\x00\x00\x00\x00"
    i = 12
    while i < len(query):
        lab = query[i]
        if lab == 0:
            i += 5
            break
        i += 1 + lab
    question = query[12:i]
    # Name pointer, A, IN, TTL 30, 203.0.113.10. No qname is logged.
    answer = b"\xc0\x0c\x00\x01\x00\x01\x00\x00\x00\x1e\x00\x04" + bytes((203, 0, 113, 10))
    return header + question + answer


def serve_tcp(port: int, handler) -> None:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.bind(("0.0.0.0", port))
    sock.listen(64)

    def loop() -> None:
        while True:
            conn, _addr = sock.accept()
            conn.settimeout(20)
            threading.Thread(target=handler, args=(conn, port), daemon=True).start()

    threading.Thread(target=loop, daemon=True).start()


def serve_udp(port: int, kind: str) -> None:
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.bind(("0.0.0.0", port))

    def loop() -> None:
        while True:
            data, addr = sock.recvfrom(4096)
            note(f"udp {addr[0]}:{addr[1]} -> 0.0.0.0:{port} bytes={len(data)}")
            if kind == "echo":
                sock.sendto(data, addr)
            elif kind == "dns":
                sock.sendto(dns_response(data), addr)

    threading.Thread(target=loop, daemon=True).start()


def main() -> None:
    open(LOG, "w", encoding="utf-8").close()
    if ROLE == "inet":
        serve_tcp(80, handle_http)
        serve_tcp(9001, handle_echo)
        serve_tcp(53, handle_dns_tcp)
        serve_tcp(8080, handle_http)
        serve_udp(9000, "echo")
        serve_udp(53, "dns")
    elif ROLE == "lan":
        serve_tcp(80, handle_http)
    elif ROLE == "host":
        serve_tcp(2222, handle_http)
        serve_tcp(8443, handle_http)
    elif ROLE == "member":
        serve_tcp(8080, handle_http)
    else:
        sys.stderr.write(f"unknown role {ROLE}\n")
        sys.exit(1)
    open(READY, "w", encoding="utf-8").write("ok\n")
    threading.Event().wait()


if __name__ == "__main__":
    main()
