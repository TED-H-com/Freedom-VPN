"""TEST ONLY owner of the lab member veth, never an agent route writer.

Restore saved addresses/peer routes only while Apply has raised the link. The
explicit ISO-F6 recovery test leaves this owner stopped to test the topology hold.
"""
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time

base = Path("/scratch/w2")
pidfile = base / "topology-owner.pid"
config = base / "topology-owner.json"
iface = "fv-fh"


def read(*args):
    return json.loads(subprocess.check_output(["ip", "-j", *args]))


def snapshot():
    link = read("addr", "show", "dev", iface)[0]
    return {"up": "UP" in link["flags"], "addresses": link.get("addr_info", []),
            "routes": {f: read(f, "route", "show", "dev", iface) for f in ["-4", "-6"]}}


def stop():
    if pidfile.exists():
        pid = int(pidfile.read_text())
        cmd = Path(f"/proc/{pid}/cmdline")
        if cmd.exists() and b"topology-owner.py\x00run" in cmd.read_bytes():
            os.kill(pid, signal.SIGTERM)
        for _ in range(20):
            if not pidfile.exists():
                break
            time.sleep(0.05)
        pidfile.unlink(missing_ok=True)


def run():
    wanted = json.loads(config.read_text())
    active = True

    def end(*_):
        nonlocal active
        active = False

    signal.signal(signal.SIGTERM, end)
    pidfile.write_text(str(os.getpid()))
    try:
        while active:
            got = snapshot()
            if got["up"]:
                for address in wanted["addresses"]:
                    if address["scope"] != "global":
                        continue
                    if any(a["family"] == address["family"] and a["local"] == address["local"] for a in got["addresses"]):
                        continue
                    family = "-4" if address["family"] == "inet" else "-6"
                    args = ["ip", family, "addr", "replace", f'{address["local"]}/{address["prefixlen"]}', "dev", iface]
                    if family == "-6":
                        args.append("nodad")
                    subprocess.run(args, check=False, stderr=subprocess.DEVNULL)
                for family, routes in wanted["routes"].items():
                    for route in routes:
                        if "gateway" not in route or any(r.get("dst") == route["dst"] and r.get("gateway") == route["gateway"] for r in got["routes"][family]):
                            continue
                        subprocess.run(["ip", family, "route", "replace", route["dst"], "via", route["gateway"], "dev", iface], check=False, stderr=subprocess.DEVNULL)
            time.sleep(0.1)
    finally:
        pidfile.unlink(missing_ok=True)


if sys.argv[1] == "capture":
    config.write_text(json.dumps(snapshot()))
elif sys.argv[1] == "run":
    run()
elif sys.argv[1] == "stop":
    stop()
