"""TEST ONLY D-104 mechanism trace; run inside the isolated netlab container."""
import json
import os
import re
import select
import socket
import struct
import subprocess
import sys
import time


def run(*args, **kwargs):
    return subprocess.check_output(args, text=True, **kwargs)


def duration(s):
    units = {"d": 86400000, "h": 3600000, "m": 60000, "s": 1000, "ms": 1}
    return sum(int(n) * units[u] for n, u in re.findall(r"(\d+)(ms|[dhms])", s))


model = json.load(sys.stdin)
entries = ", ".join("192.0.2.%d timeout %dms expires %dms" % (i, 1000+i, 700+i) for i in range(1, 9))
script = "add set inet fvpn fv_granularity { type ipv4_addr; flags timeout; }\nadd element inet fvpn fv_granularity { " + entries + " }\n"
subprocess.run(["nft", "-f", "-"], input=script, text=True, check=True)
try:
    precision_start = time.monotonic_ns()
    text_listing = run("nft", "list", "set", "inet", "fvpn", "fv_granularity")
    precision_end = time.monotonic_ns()
    json_listing = json.loads(run("nft", "-j", "list", "set", "inet", "fvpn", "fv_granularity"))
finally:
    run("nft", "delete", "set", "inet", "fvpn", "fv_granularity")
values = [(duration(a), duration(b)) for a, b in re.findall(r"timeout (\w+) expires (\w+)", text_listing)]
assert len(values) == 8 and [a for a, _ in values] == list(range(1001, 1009)), values
assert all(values[i+1][1]-values[i][1] == 1 for i in range(7)), values

# Open the peer transmitter in the routing namespace, then keep its socket
# while returning to root for both captures. No tc injection or third-party I/O.
root = os.open("/proc/self/ns/net", os.O_RDONLY)
peer = os.open("/run/netns/fv-fwd", os.O_RDONLY)
os.setns(peer, 0)
send = socket.socket(socket.AF_PACKET, socket.SOCK_RAW, socket.htons(3))
send.bind(("fv-ff", 0))
peer_mac = send.getsockname()[4]
os.setns(root, 0)
os.close(peer)
os.close(root)
root_mac = bytes.fromhex(open("/sys/class/net/fv-fh/address").read().strip().replace(":", ""))
reads = []
for dev in ["fv-fh", "fv-ih"]:
    s = socket.socket(socket.AF_PACKET, socket.SOCK_RAW, socket.htons(3))
    s.bind((dev, 0))
    s.setblocking(False)
    reads.append(s)

applied = json.loads(run("fvpn-netlab", "apply", "-deadline-window", "2900ms", input=json.dumps(model)))
assert applied["verified"], applied
D = int(applied["leases"][0]["deadline_mono_ns"])


def checksum(b):
    s = sum(struct.unpack("!%dH" % (len(b)//2), b))
    s = (s >> 16)+(s & 65535)
    s += s >> 16
    return (~s) & 65535


src = socket.inet_aton("100.64.1.1")
dst = socket.inet_aton("203.0.113.10")
rows = []
# Phase the 100 ms cadence at D-(bound+quantum). This provides a sample at
# the requested continuity boundary without hiding a cadence-sized gap.
for seq in range(33):
    target = D - 2601000000 + seq*100000000
    time.sleep(max(0, (target-time.monotonic_ns())/1e9))
    payload = struct.pack("!I", seq)+b"TEST ONLY"
    udp = struct.pack("!HHHH", 40098, 40099, len(payload)+8, 0)+payload
    ip = struct.pack("!BBHHHBBH4s4s", 69, 0, len(udp)+20, seq, 0, 64, 17, 0, src, dst)
    ip = ip[:10]+struct.pack("!H", checksum(ip))+ip[12:]
    row = {"packet_id": seq, "target_ns": str(target), "sent_ns": str(time.monotonic_ns()), "ingress_ns": "0", "egress_ns": "0"}
    send.send(root_mac+peer_mac+b"\x08\x00"+ip+udp)
    end = int(row["sent_ns"])+80000000
    while time.monotonic_ns() < end:
        ready, _, _ = select.select(reads, [], [], max(0, (end-time.monotonic_ns())/1e9))
        for s in ready:
            packet = s.recv(65535)
            at = time.monotonic_ns()
            if len(packet) >= 46 and packet[12:14] == b"\x08\x00" and packet[23] == 17 and packet[36:38] == struct.pack("!H", 40099) and packet[42:46] == struct.pack("!I", seq):
                row["ingress_ns" if s == reads[0] else "egress_ns"] = str(at)
    row["end_ns"] = str(time.monotonic_ns())
    rows.append(row)

print(json.dumps({
    "test_only": True, "clock": "CLOCK_MONOTONIC", "cadence_ms": 100, "observation_window_ms": 80,
    "kernel": run("uname", "-r").strip(), "nft": run("nft", "--version").strip(),
    "nft_package": run("dpkg-query", "-W", "-f=${Version}", "nftables"),
    "precision": {"read_start_ns": str(precision_start), "read_end_ns": str(precision_end), "timeout_expires_ms": values, "text": text_listing, "json": json_listing},
    "deadline_ns": str(D), "apply": applied, "packets": rows,
}, indent=2))
