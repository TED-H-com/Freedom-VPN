#!/usr/bin/env python3
"""Hold one TCP echo and one UDP echo. Append timestamps to a log file."""

import os
import socket
import sys
import time


def main() -> None:
    dst = sys.argv[1]
    tcp_port = int(sys.argv[2])
    udp_port = int(sys.argv[3])
    path = sys.argv[4]
    with open(path + ".pid", "w", encoding="utf-8") as pidf:
        pidf.write(str(os.getpid()) + "\n")
    logf = open(path, "a", encoding="utf-8")

    def log(msg: str) -> None:
        logf.write(f"{time.time():.3f} {msg}\n")
        logf.flush()

    tcp = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    tcp.settimeout(1.5)
    tcp.connect((dst, tcp_port))
    udp = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    udp.settimeout(1.5)
    log("up")
    n = 0
    while True:
        n += 1
        token = f"t{n}\n".encode()
        try:
            tcp.sendall(token)
            data = tcp.recv(128)
            if not data:
                log("fail tcp-eof")
                time.sleep(0.2)
                continue
            udp.sendto(token, (dst, udp_port))
            udp.recvfrom(128)
            log("ok")
        except OSError as exc:
            log("fail " + type(exc).__name__)
        time.sleep(0.2)


if __name__ == "__main__":
    main()
