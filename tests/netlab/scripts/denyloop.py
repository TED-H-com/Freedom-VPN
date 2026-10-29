#!/usr/bin/env python3
"""Repeat a deny matrix until /scratch/denyloop.stop exists.

Unexpected allows are appended to /scratch/deny-hits.log. The lab test reads
that file. This process must not be the thing that opens forwarding.
"""

import json
import os
import subprocess
import sys
import time

CASES = sys.argv[1]
HITS = "/scratch/deny-hits.log"
STOP = "/scratch/denyloop.stop"
COUNT = "/scratch/denyloop.count"


def main() -> None:
    with open("/scratch/denyloop.pid", "w", encoding="utf-8") as fh:
        fh.write(str(os.getpid()) + "\n")
    n = 0
    while not os.path.exists(STOP):
        n += 1
        proc = subprocess.run(
            ["python3", "/opt/netlab/scripts/probe.py", CASES],
            capture_output=True,
            text=True,
        )
        try:
            doc = json.loads(proc.stdout)
        except json.JSONDecodeError:
            with open(HITS, "a", encoding="utf-8") as fh:
                fh.write("probe-failed\n")
            time.sleep(0.2)
            continue
        with open(HITS, "a", encoding="utf-8") as fh:
            for item in doc.get("results", []):
                if not item.get("ok"):
                    fh.write(json.dumps(item) + "\n")
        with open(COUNT, "w", encoding="utf-8") as fh:
            fh.write(str(n) + "\n")
        time.sleep(0.05)


if __name__ == "__main__":
    main()
