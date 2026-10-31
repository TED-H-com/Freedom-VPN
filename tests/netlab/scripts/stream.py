#!/usr/bin/env python3
"""Hold one TCP echo and one UDP echo. Append timestamps to a log file."""

import json
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

    trace = open(path + ".jsonl", "a", encoding="utf-8")
    for suffix in (".trace-stop", ".trace-done"):
        try:
            os.unlink(path + suffix)
        except FileNotFoundError:
            pass
    recording = True

    def event(kind: str, **fields) -> None:
        if not recording:
            return
        # TEST ONLY metadata. Never record payloads or query names.
        trace.write(json.dumps(dict(event=kind, sequence=n,
                                    wall_ns=str(time.time_ns()),
                                    mono_ns=str(time.monotonic_ns()), **fields)) + "\n")
        trace.flush()

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
        if recording and os.path.exists(path + ".trace-stop"):
            # The preceding read is complete. Keep the acceptance stream
            # running for revoke tests, but end this measurement sidecar.
            recording = False
            with open(path + ".trace-done", "w", encoding="utf-8") as done:
                done.write("ok\n")
        n += 1
        token = f"t{n}\n".encode()
        stage = "tcp"
        event("tcp_send")
        try:
            tcp.sendall(token)
            data = tcp.recv(128)
            event("tcp_read", size=len(data))
            if not data:
                log("fail tcp-eof")
                time.sleep(0.2)
                continue
            stage = "udp"
            sent_wall, sent_mono = time.time_ns(), time.monotonic_ns()
            udp.sendto(token, (dst, udp_port))
            event("udp_send", send_wall_ns=str(sent_wall), send_mono_ns=str(sent_mono),
                  source_port=udp.getsockname()[1], size=len(token))
            data, _ = udp.recvfrom(128)
            event("udp_read", outcome="ok", response_matches=data == token)
            log("ok")
        except OSError as exc:
            event(stage + "_read", outcome=type(exc).__name__)
            log("fail " + type(exc).__name__)
        time.sleep(0.2)


if __name__ == "__main__":
    main()
