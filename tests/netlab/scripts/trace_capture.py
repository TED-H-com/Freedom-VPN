#!/usr/bin/env python3
"""TEST ONLY header traces; node any is the independent forwarding record."""
import os
from pathlib import Path
import signal
import subprocess
import sys
import time


def paths(base):
    return [("origin", "fv-a1"), ("node", None), ("responder", "fv-inet")]


def start(base):
    for role, ns in paths(base):
        path = Path(base + "-" + role)
        cmd = (["ip", "netns", "exec", ns] if ns else []) + [
            "tcpdump", "--immediate-mode", "-nn", "-tt", "-v", "-l", "-s", "96",
            "-i", "any", "udp port 9000"]
        with path.with_suffix(".txt").open("w") as out, path.with_suffix(".err").open("w") as err:
            proc = subprocess.Popen(cmd, stdout=out, stderr=err, start_new_session=True)
        path.with_suffix(".pid").write_text(str(proc.pid))
        end = time.monotonic() + 5
        while "listening on" not in path.with_suffix(".err").read_text():
            if proc.poll() is not None or time.monotonic() > end:
                raise RuntimeError("trace capture did not start: " + role)
            time.sleep(.01)


def stop(base):
    for role, _ in paths(base):
        path = Path(base + "-" + role)
        pid = int(path.with_suffix(".pid").read_text())
        os.kill(pid, signal.SIGINT)
        end = time.monotonic() + 5
        while "packets dropped by kernel" not in path.with_suffix(".err").read_text():
            if time.monotonic() > end:
                raise RuntimeError("trace capture did not stop: " + role)
            time.sleep(.01)


if __name__ == "__main__":
    {"start": start, "stop": stop}[sys.argv[1]](sys.argv[2])
