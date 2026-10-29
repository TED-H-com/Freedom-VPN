#!/usr/bin/env python3
"""Send a counted transfer. Prints JSON payload byte totals, not IP lengths."""

import json
import socket
import sys


def main() -> None:
    proto, dst, port_s, count_s, size_s = sys.argv[1:6]
    port, count, size = int(port_s), int(count_s), int(size_s)
    payload = b"x" * size
    sent = 0
    got = 0
    if proto == "udp":
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(0.5)
        for _ in range(count):
            sock.sendto(payload, (dst, port))
            sent += size
            try:
                data, _addr = sock.recvfrom(65535)
                got += len(data)
            except socket.timeout:
                pass
        sock.close()
    elif proto == "tcp":
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(15)
        sock.connect((dst, port))
        blob = payload * count
        while blob:
            n = sock.send(blob[:8192])
            blob = blob[n:]
            sent += n
        sock.shutdown(socket.SHUT_WR)
        while True:
            data = sock.recv(8192)
            if not data:
                break
            got += len(data)
        sock.close()
    else:
        sys.stderr.write("proto must be udp or tcp\n")
        sys.exit(1)
    json.dump({"sent": sent, "got": got}, sys.stdout)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
