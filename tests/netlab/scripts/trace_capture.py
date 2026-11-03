#!/usr/bin/env python3
"""TEST ONLY header traces; node any is the independent forwarding record."""
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time

from reload_analysis import guards, json_rows


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


def final_guard(apply_path):
    windows = guards(json_rows(Path(apply_path)))
    if [w['generation'] for w in windows] != [str(g) for g in range(1000, 1101)]:
        raise ValueError("missing final reload guard bracket (requires 100 reloads)")
    last = windows[-1]
    if (not last['verified'] or
            int(last['release_end']['mono_ns']) < int(last['install_end']['mono_ns']) or
            int(last['apply_end']['mono_ns']) < int(last['release_end']['mono_ns'])):
        raise ValueError("final reload guard bracket not closed")
    return last


def finish_stream(path, apply_path):
    # This check precedes the stop request; an incomplete campaign never gets
    # a successful measurement boundary. Aborted traces are still retained.
    last = final_guard(apply_path)
    Path(path + ".trace-finish.json").write_text(json.dumps(dict(
        final_guard=last, requested_mono_ns=str(time.monotonic_ns()),
        max_extra_cycles=1, watchdog_seconds=15), indent=2) + "\n")
    Path(path + ".trace-stop").touch()
    # Pending cycle + one successor: each has up to four 1.5 s socket
    # operations and a 0.2 s pause (12.4 s total); 15 s leaves overhead.
    # A watchdog expiry is an evidence failure, never a fabricated read/end.
    end = time.monotonic() + 15
    while not Path(path + ".trace-done").exists():
        if time.monotonic() > end:
            raise RuntimeError("stream successor missing: finish watchdog expired")
        time.sleep(.01)


if __name__ == "__main__":
    {"start": start, "stop": stop, "finish-stream": finish_stream}[sys.argv[1]](*sys.argv[2:])
