#!/usr/bin/env python3
"""Connectivity probes for the lab. Prints one JSON document."""

import json
import os
import socket
import subprocess
import sys
import time
from concurrent.futures import ThreadPoolExecutor


def classify(exc: BaseException) -> str:
    if isinstance(exc, socket.timeout):
        return "timeout"
    text = str(exc).lower()
    if "timed out" in text:
        return "timeout"
    if "refused" in text:
        return "refused"
    if "unreachable" in text or "no route" in text:
        return "unreachable"
    return "error: " + str(exc).split("\n", 1)[0][:160]


def tcp_connect(dst: str, port: int, timeout: float, bind: str, family: int) -> str:
    sock = socket.socket(family, socket.SOCK_STREAM)
    try:
        sock.settimeout(timeout)
        if family == socket.AF_INET6:
            sock.setsockopt(socket.IPPROTO_IPV6, socket.IPV6_V6ONLY, 1)
        if bind:
            sock.bind((bind, 0))
        sock.connect((dst, port))
        return "connected"
    except OSError as exc:
        return classify(exc)
    finally:
        sock.close()


def udp_exchange(dst: str, port: int, timeout: float, bind: str, payload: bytes, family: int) -> str:
    sock = socket.socket(family, socket.SOCK_DGRAM)
    try:
        sock.settimeout(timeout)
        if bind:
            sock.bind((bind, 0))
        sock.sendto(payload, (dst, port))
        data, _addr = sock.recvfrom(4096)
        if data:
            return "reply"
        return "empty-reply"
    except OSError as exc:
        return classify(exc)
    finally:
        sock.close()


def ping(dst: str, timeout: float, v6: bool, bind: str) -> str:
    wait = max(1, int(timeout + 0.99))
    cmd = ["ping", "-c", "1", "-W", str(wait)]
    if v6:
        cmd.append("-6")
    if bind:
        cmd.extend(["-I", bind])
    cmd.append(dst)
    try:
        proc = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout + 2)
    except subprocess.TimeoutExpired:
        return "timeout"
    if proc.returncode == 0:
        return "reply"
    text = (proc.stderr + proc.stdout).lower()
    if "unreachable" in text or "no route" in text:
        return "unreachable"
    return "timeout"


def dns_query() -> bytes:
    # ID 0x0e01, recursion desired, one question, name "t.example", A IN.
    name = b"\x01t\x07example\x00"
    return b"\x0e\x01\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00" + name + b"\x00\x01\x00\x01"


def checksum(data: bytes) -> int:
    if len(data) % 2:
        data += b"\x00"
    total = 0
    for i in range(0, len(data), 2):
        total += (data[i] << 8) + data[i + 1]
    while total >> 16:
        total = (total & 0xFFFF) + (total >> 16)
    return (~total) & 0xFFFF


def ipv6_udp_frame(dst_mac: str, src_mac: str, src: str, dst: str, payload: bytes) -> bytes:
    def mac(text: str) -> bytes:
        return bytes(int(part, 16) for part in text.split(":"))

    src_b = socket.inet_pton(socket.AF_INET6, src)
    dst_b = socket.inet_pton(socket.AF_INET6, dst)
    udp_len = 8 + len(payload)
    udp = b"\x9c\x40\x00\x09" + udp_len.to_bytes(2, "big") + b"\x00\x00" + payload
    pseudo = src_b + dst_b + udp_len.to_bytes(4, "big") + b"\x00\x00\x00\x11" + udp
    csum = checksum(pseudo)
    if csum == 0:
        csum = 0xFFFF
    udp = udp[:6] + csum.to_bytes(2, "big") + udp[8:]
    ver = (6 << 28).to_bytes(4, "big")
    header = ver + len(udp).to_bytes(2, "big") + b"\x11\x40" + src_b + dst_b
    return mac(dst_mac) + mac(src_mac) + b"\x86\xdd" + header + udp


def ipv4_udp_frame(dst_mac: str, src_mac: str, src: str, dst: str, payload: bytes) -> bytes:
    def mac(text: str) -> bytes:
        return bytes(int(part, 16) for part in text.split(":"))

    src_b = socket.inet_aton(src)
    dst_b = socket.inet_aton(dst)
    udp_len = 8 + len(payload)
    udp = (40000).to_bytes(2, "big") + (9).to_bytes(2, "big") + udp_len.to_bytes(2, "big") + b"\x00\x00" + payload
    total = 20 + len(udp)
    ip = bytes((0x45, 0, total >> 8, total & 0xFF, 0, 0, 0, 0, 64, 17, 0, 0)) + src_b + dst_b
    csum = checksum(ip)
    ip = bytes((0x45, 0, total >> 8, total & 0xFF, 0, 0, 0, 0, 64, 17, csum >> 8, csum & 0xFF)) + src_b + dst_b
    return mac(dst_mac) + mac(src_mac) + b"\x08\x00" + ip + udp


def send_raw4(case: dict) -> str:
    frame = ipv4_udp_frame(case["dst_mac"], case["src_mac"], case["src"], case["dst"], b"v4")
    sock = socket.socket(socket.AF_PACKET, socket.SOCK_RAW)
    try:
        sock.bind((case.get("iface") or "fv-ff", 0))
        sock.send(frame)
        return "sent"
    except OSError as exc:
        return classify(exc)
    finally:
        sock.close()


def send_raw6(case: dict) -> str:
    frame = ipv6_udp_frame(case["dst_mac"], case["src_mac"], case["src"], case["dst"], b"v6")
    sock = socket.socket(socket.AF_PACKET, socket.SOCK_RAW)
    try:
        sock.bind((case.get("iface") or "fv-ff", 0))
        sock.send(frame)
        return "sent"
    except OSError as exc:
        return classify(exc)
    finally:
        sock.close()


def run_case(case: dict) -> dict:
    proto = case["proto"]
    timeout = float(case.get("timeout_ms") or 500) / 1000.0
    bind = case.get("bind") or ""
    start = time.monotonic()
    if proto == "tcp":
        obs = tcp_connect(case["dst"], int(case["port"]), timeout, bind, socket.AF_INET)
    elif proto == "tcp6":
        obs = tcp_connect(case["dst"], int(case["port"]), timeout, bind, socket.AF_INET6)
    elif proto == "udp":
        payload = b"u" * int(case.get("size") or 32)
        obs = udp_exchange(case["dst"], int(case["port"]), timeout, bind, payload, socket.AF_INET)
    elif proto == "dns":
        obs = udp_exchange(case["dst"], int(case.get("port") or 53), timeout, bind, dns_query(), socket.AF_INET)
    elif proto == "icmp":
        obs = ping(case["dst"], timeout, False, bind)
    elif proto == "icmp6":
        obs = ping(case["dst"], timeout, True, bind)
    elif proto == "raw4":
        obs = send_raw4(case)
    elif proto == "raw6":
        obs = send_raw6(case)
    else:
        obs = "error: unknown proto"
    elapsed = int((time.monotonic() - start) * 1000)
    expect = case["expect"]
    ok = (expect == "allow" and obs in ("connected", "reply")) or (expect == "deny" and obs not in ("connected", "reply")) or (expect == "sent" and obs == "sent")
    return {
        "name": case["name"],
        "ns": case.get("ns", ""),
        "proto": proto,
        "dst": case.get("dst", ""),
        "port": case.get("port", 0),
        "expect": expect,
        "observed": obs,
        "ok": ok,
        "elapsed_ms": elapsed,
    }


def run_file(path: str) -> None:
    with open(path, encoding="utf-8") as fh:
        doc = json.load(fh)
    cases = doc["cases"]
    workers = min(8, max(1, len(cases)))
    with ThreadPoolExecutor(max_workers=workers) as pool:
        results = list(pool.map(run_case, cases))
    json.dump({"results": results}, sys.stdout)
    sys.stdout.write("\n")
    if not all(item["ok"] for item in results):
        sys.exit(2)


def main() -> None:
    if len(sys.argv) != 2:
        sys.stderr.write("usage: probe.py cases.json\n")
        sys.exit(1)
    path = sys.argv[1]
    if os.environ.get("FV_INSIDE") == "1":
        run_file(path)
        return
    with open(path, encoding="utf-8") as fh:
        doc = json.load(fh)
    groups: dict[str, list] = {}
    for case in doc["cases"]:
        groups.setdefault(case["ns"], []).append(case)
    results = []
    procs = []
    token = f"{os.getpid()}-{time.monotonic_ns()}"
    for ns, cases in groups.items():
        # denyloop and the test harness probe at the same time. A fixed path
        # lets one invocation read the other's partial output.
        part = f"/scratch/probe-{token}-{ns}.json"
        out = f"/scratch/probe-{token}-{ns}.out"
        with open(part, "w", encoding="utf-8") as fh:
            json.dump({"cases": cases}, fh)
        env = dict(os.environ)
        env["FV_INSIDE"] = "1"
        outfh = open(out, "w", encoding="utf-8")
        procs.append((ns, part, out, outfh, subprocess.Popen(
            ["ip", "netns", "exec", ns, "python3", "/opt/netlab/scripts/probe.py", part],
            stdout=outfh,
            stderr=subprocess.PIPE,
            env=env,
        )))
    failed = False
    for ns, part, out, outfh, proc in procs:
        err = proc.stderr.read().decode("utf-8", "replace") if proc.stderr else ""
        code = proc.wait()
        outfh.close()
        try:
            with open(out, encoding="utf-8") as fh:
                payload = json.load(fh)
            results.extend(payload.get("results", []))
        except (OSError, json.JSONDecodeError):
            failed = True
            results.append({
                "name": ns,
                "ns": ns,
                "proto": "",
                "dst": "",
                "port": 0,
                "expect": "",
                "observed": "error: probe failed " + err[-200:],
                "ok": False,
                "elapsed_ms": 0,
            })
        if code not in (0, 2):
            failed = True
        try:
            os.remove(part)
            os.remove(out)
        except OSError:
            pass
    json.dump({"results": results}, sys.stdout)
    sys.stdout.write("\n")
    if failed or not all(item["ok"] for item in results):
        sys.exit(2)


if __name__ == "__main__":
    main()
