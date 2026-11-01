#!/usr/bin/env python3
"""Hold one TCP echo and one UDP echo. Append timestamps to a log file."""

import json
import os
import socket
import sys
import time


class TraceEnd:
    """One serial successor cycle, never a retry of a failed datagram.

    At a loop boundary every UDP send has its read. Earlier losses already
    have successors, so only the last read can need one. A failed successor
    exhausts the bound: keep both its failure and the resulting missing tail.
    """
    def __init__(self):
        self.last_udp_failed = False
        self.extra_cycles = 0

    def udp_read(self, outcome, response_matches=False):
        self.last_udp_failed = outcome != "ok" or response_matches is not True

    def boundary(self, requested):
        if not requested:
            return None
        if not self.last_udp_failed:
            return "complete"
        if self.extra_cycles == 1:
            return "missing"
        self.extra_cycles += 1
        return None


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
    ending = TraceEnd()

    def event(kind: str, **fields) -> None:
        if not recording:
            return
        # TEST ONLY metadata. Never record payloads or query names.
        trace.write(json.dumps(dict(event=kind, sequence=n,
                                    wall_ns=str(time.time_ns()),
                                    mono_ns=str(time.monotonic_ns()), **fields)) + "\n")
        trace.flush()
        if kind == "udp_read":
            ending.udp_read(fields.get("outcome"), fields.get("response_matches"))

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
        end_reason = ending.boundary(recording and os.path.exists(path + ".trace-stop"))
        if end_reason:
            # finish_stream requests this only after the final guard closes.
            # The count bound preserves a failed/missing successor as failure.
            # Keep the acceptance stream running for the later revoke test.
            event("trace_end", end_reason=end_reason, extra_cycles=ending.extra_cycles)
            recording = False
            with open(path + ".trace-done", "w", encoding="utf-8") as done:
                json.dump(dict(end_reason=end_reason, extra_cycles=ending.extra_cycles,
                               sequence=n), done)
                done.write("\n")
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
