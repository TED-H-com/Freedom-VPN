#!/usr/bin/env python3
"""AT13 D-147 per-source-set and overlap verification inside the G2 netlab container."""

import contextlib
import ipaddress
import json
import os
import pathlib
import socket
import struct
import subprocess
import sys
import tempfile
import time

MEMBER_NS = "fv-a1"
MEMBER_IP = "100.64.1.1"
SNAT_IP = "192.0.2.37"
ORIGIN_NS = "fv-f34origin"
ORIGIN_IP = "203.0.113.34"
ORIGIN_PORT = 8080
PATH_MTU = 1000
UPLOAD_BYTES = 64064
DEV_A1 = "019a00000000700080000000000000a1"
ASSIGNED_CLASS = "1:2f"
ASSIGNED_PRIORITY = "0x0001002f"
ETHERNET_L2_LEN = 14

COUNTERS = (
    f"u_{DEV_A1}",
    f"d_{DEV_A1}",
    f"du_{DEV_A1}",
    f"dd_{DEV_A1}",
    "ctdns",
    "ctdeny",
    "cthost",
    "ctdenyd",
    "ctinv",
    "ctnodown",
    "ctunkd",
)

ROWS = (
    {
        "id": "resolver_v4_only",
        "source_ip": "203.0.113.53",
        "expected_sets": ["resolver_v4"],
        "upload_drop_counter": "ctdns",
    },
    {
        "id": "denied_v4_only",
        "source_ip": "192.168.34.2",
        "expected_sets": ["denied_v4"],
        "upload_drop_counter": "ctdeny",
    },
    {
        "id": "host_v4_only",
        "source_ip": "203.0.113.33",
        "expected_sets": ["host_v4"],
        "upload_drop_counter": "cthost",
    },
    {
        "id": "denied_v4_and_host_v4_overlap",
        "source_ip": "192.168.34.1",
        "expected_sets": ["denied_v4", "host_v4"],
        "upload_drop_counter": "ctdeny",
    },
)


def run(*args, check=True):
    return subprocess.run(args, check=check, text=True, capture_output=True)


def checksum(data: bytes) -> int:
    if len(data) % 2:
        data += b"\x00"
    total = sum(struct.unpack(f"!{len(data) // 2}H", data))
    total = (total >> 16) + (total & 0xFFFF)
    total += total >> 16
    return (~total) & 0xFFFF


def ipv4_header(src: str, dst: str, proto: int, payload_len: int, ttl: int = 64, df: bool = True) -> bytes:
    ver_ihl = 0x45
    total_len = 20 + payload_len
    flags_frag = 0x4000 if df else 0
    head = struct.pack(
        "!BBHHHBBH4s4s",
        ver_ihl,
        0,
        total_len,
        0x3412,
        flags_frag,
        ttl,
        proto,
        0,
        socket.inet_aton(src),
        socket.inet_aton(dst),
    )
    csum = checksum(head)
    return head[:10] + struct.pack("!H", csum) + head[12:]


def parse_pcap(path: pathlib.Path) -> dict:
    raw = path.read_bytes() if path.exists() else b""
    if len(raw) < 24:
        return {
            "packets": 0,
            "ip_bytes": 0,
            "tcp_packets": 0,
            "tcp_ip_bytes": 0,
            "icmp_packets": 0,
            "icmp_ip_bytes": 0,
            "max_ip_len": 0,
            "max_tcp_payload": 0,
            "syn_mss": [],
            "icmp": [],
        }
    magic = raw[:4]
    endian = "<" if magic in (b"\xd4\xc3\xb2\xa1", b"\x4d\x3c\xb2\xa1") else ">"
    _, _, _, _, _, _, linktype = struct.unpack(endian + "IHHIIII", raw[:24])
    off = 24
    packets = 0
    ip_bytes = 0
    tcp_packets = 0
    tcp_ip_bytes = 0
    icmp_packets = 0
    icmp_ip_bytes = 0
    max_ip_len = 0
    max_tcp_payload = 0
    syn_mss = []
    icmp_rows = []
    while off + 16 <= len(raw):
        _, _, incl_len, _ = struct.unpack(endian + "IIII", raw[off : off + 16])
        off += 16
        frame = raw[off : off + incl_len]
        off += incl_len
        if linktype == 1:
            if len(frame) < 14 or struct.unpack("!H", frame[12:14])[0] != 0x0800:
                continue
            pkt = frame[14:]
        elif linktype in (12, 101):
            pkt = frame
        elif linktype == 113:
            if len(frame) < 16 or struct.unpack("!H", frame[14:16])[0] != 0x0800:
                continue
            pkt = frame[16:]
        else:
            continue
        if len(pkt) < 20 or (pkt[0] >> 4) != 4:
            continue
        ihl = (pkt[0] & 0x0F) * 4
        total_len = struct.unpack("!H", pkt[2:4])[0]
        proto = pkt[9]
        src = socket.inet_ntoa(pkt[12:16])
        dst = socket.inet_ntoa(pkt[16:20])
        packets += 1
        ip_bytes += total_len
        max_ip_len = max(max_ip_len, total_len)
        if proto == 6 and len(pkt) >= ihl + 20:
            tcp_packets += 1
            tcp_ip_bytes += total_len
            th = pkt[ihl:]
            doff = (th[12] >> 4) * 4
            flags = th[13]
            payload = max(0, total_len - ihl - doff)
            max_tcp_payload = max(max_tcp_payload, payload)
            if flags & 0x02 and len(th) >= doff:
                opts = th[20:doff]
                idx = 0
                while idx < len(opts):
                    kind = opts[idx]
                    if kind == 0:
                        break
                    if kind == 1:
                        idx += 1
                        continue
                    if idx + 1 >= len(opts):
                        break
                    olen = opts[idx + 1]
                    if olen < 2 or idx + olen > len(opts):
                        break
                    if kind == 2 and olen == 4:
                        syn_mss.append(struct.unpack("!H", opts[idx + 2 : idx + 4])[0])
                    idx += olen
        elif proto == 1 and len(pkt) >= ihl + 8:
            icmp_packets += 1
            icmp_ip_bytes += total_len
            ic = pkt[ihl:]
            icmp_type, icmp_code, _, rest = struct.unpack("!BBHI", ic[:8])
            icmp_rows.append(
                {
                    "src": src,
                    "dst": dst,
                    "icmp_type": icmp_type,
                    "icmp_code": icmp_code,
                    "mtu": rest & 0xFFFF,
                    "ip_len": total_len,
                }
            )
    return {
        "packets": packets,
        "ip_bytes": ip_bytes,
        "tcp_packets": tcp_packets,
        "tcp_ip_bytes": tcp_ip_bytes,
        "icmp_packets": icmp_packets,
        "icmp_ip_bytes": icmp_ip_bytes,
        "max_ip_len": max_ip_len,
        "max_tcp_payload": max_tcp_payload,
        "syn_mss": syn_mss,
        "icmp": icmp_rows,
    }


@contextlib.contextmanager
def capture(ns: str | None, iface: str, expr: str, direction: str | None = None):
    fd, raw_path = tempfile.mkstemp(prefix="d147-", suffix=".pcap", dir="/scratch")
    os.close(fd)
    path = pathlib.Path(raw_path)
    path.unlink(missing_ok=True)
    cmd = []
    if ns:
        cmd.extend(["ip", "netns", "exec", ns])
    cmd.extend(["tcpdump", "-ni", iface, "-s", "0", "-U", "-w", str(path)])
    if direction:
        cmd.append(f"--direction={direction}")
    cmd.append(expr)
    proc = subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True)
    time.sleep(0.2)
    if proc.poll() is not None:
        err = proc.stderr.read() if proc.stderr else ""
        raise RuntimeError(f"tcpdump failed on {ns or 'root'}:{iface}: {err.strip()}")
    try:
        yield path
    finally:
        time.sleep(0.15)
        proc.terminate()
        try:
            proc.wait(timeout=2)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait(timeout=2)


def read_nft_table() -> dict:
    return json.loads(run("nft", "-j", "list", "table", "inet", "fvpn").stdout)


def read_counters() -> dict:
    doc = read_nft_table()
    found = {}
    for item in doc.get("nftables", []):
        c = item.get("counter")
        if c and c.get("name") in COUNTERS:
            found[c["name"]] = {"packets": int(c["packets"]), "bytes": int(c["bytes"])}
    return {name: found.get(name, {"packets": 0, "bytes": 0}) for name in COUNTERS}


def diff_counters(before: dict, after: dict) -> dict:
    return {
        name: {
            "packets": after[name]["packets"] - before[name]["packets"],
            "bytes": after[name]["bytes"] - before[name]["bytes"],
        }
        for name in COUNTERS
    }


def elem_contains_ip(elem, target: ipaddress.IPv4Address) -> bool:
    if isinstance(elem, dict) and "elem" in elem:
        elem = elem["elem"].get("val", elem["elem"])
    if isinstance(elem, str):
        return ipaddress.ip_address(elem) == target
    if isinstance(elem, dict):
        if "prefix" in elem:
            p = elem["prefix"]
            net = ipaddress.ip_network(f"{p['addr']}/{p['len']}", strict=False)
            return target in net
        if "range" in elem:
            lo = ipaddress.ip_address(elem["range"][0])
            hi = ipaddress.ip_address(elem["range"][1])
            return int(lo) <= int(target) <= int(hi)
    return False


def verify_set_membership(addr_str: str) -> dict:
    target = ipaddress.ip_address(addr_str)
    doc = read_nft_table()
    sets = {"resolver_v4": False, "denied_v4": False, "host_v4": False}
    for item in doc.get("nftables", []):
        s = item.get("set")
        if not s or s.get("name") not in sets:
            continue
        elems = s.get("elem", [])
        if not isinstance(elems, list):
            elems = [elems]
        sets[s["name"]] = any(elem_contains_ip(e, target) for e in elems)
    matched = [name for name in ("resolver_v4", "denied_v4", "host_v4") if sets[name]]
    return {
        "in_resolver_v4": sets["resolver_v4"],
        "in_denied_v4": sets["denied_v4"],
        "in_host_v4": sets["host_v4"],
        "matched_sets": matched,
    }


def read_tc_classes(iface: str) -> dict:
    doc = json.loads(run("tc", "-j", "-s", "class", "show", "dev", iface).stdout)
    out = {}
    for item in doc:
        handle = item.get("handle")
        stats = item.get("stats", {})
        if handle:
            out[handle] = {
                "packets": int(stats.get("packets", 0)),
                "bytes": int(stats.get("bytes", 0)),
                "drops": int(stats.get("drops", 0)),
            }
    return out


def diff_tc_classes(before: dict, after: dict) -> dict:
    handles = sorted(set(before) | set(after))
    out = {}
    for h in handles:
        b = before.get(h, {"packets": 0, "bytes": 0, "drops": 0})
        a = after.get(h, {"packets": 0, "bytes": 0, "drops": 0})
        dp = a["packets"] - b["packets"]
        db = a["bytes"] - b["bytes"]
        dd = a["drops"] - b["drops"]
        if dp or db or dd or h in (ASSIGNED_CLASS, "1:fffe"):
            out[h] = {"packets": dp, "bytes": db, "drops": dd}
    return out


def route_pmtu(ns: str, dst: str) -> int | None:
    res = run("ip", "-n", ns, "-j", "route", "get", dst, check=False)
    if res.returncode != 0 or not res.stdout.strip():
        return None
    for row in json.loads(res.stdout):
        if "mtu" in row:
            return int(row["mtu"])
        for metric in row.get("metrics", []):
            if "mtu" in metric:
                return int(metric["mtu"])
    return None


def send_unsolicited_frag_needed(src_ip: str) -> None:
    code = f"""
import socket, struct
def csum(data):
    if len(data) % 2:
        data += b"\\x00"
    s = sum(struct.unpack(f"!{{len(data)//2}}H", data))
    s = (s >> 16) + (s & 0xffff)
    s += s >> 16
    return (~s) & 0xffff
inner_ip = struct.pack("!BBHHHBBH4s4s", 0x45, 0, 40, 0x2222, 0x4000, 64, 6, 0,
    socket.inet_aton("{SNAT_IP}"), socket.inet_aton("{ORIGIN_IP}"))
inner_ip = inner_ip[:10] + struct.pack("!H", csum(inner_ip)) + inner_ip[12:]
inner_tcp = struct.pack("!HHII", 54321, {ORIGIN_PORT}, 1, 0)
icmp_wo = struct.pack("!BBHHH", 3, 4, 0, 0, {PATH_MTU}) + inner_ip + inner_tcp
icmp = icmp_wo[:2] + struct.pack("!H", csum(icmp_wo)) + icmp_wo[4:]
outer = struct.pack("!BBHHHBBH4s4s", 0x45, 0, 20 + len(icmp), 0x3333, 0x4000, 64, 1, 0,
    socket.inet_aton("{src_ip}"), socket.inet_aton("{MEMBER_IP}"))
outer = outer[:10] + struct.pack("!H", csum(outer)) + outer[12:]
s = socket.socket(socket.AF_INET, socket.SOCK_RAW, socket.IPPROTO_RAW)
s.setsockopt(socket.IPPROTO_IP, socket.IP_HDRINCL, 1)
s.sendto(outer + icmp, ("{MEMBER_IP}", 0))
"""
    run("ip", "netns", "exec", "fv-inet", "python3", "-B", "-c", code)


def tcp_upload(ns: str, dst: str, port: int, total_bytes: int, timeout: float = 12.0) -> dict:
    code = f"""
import json, socket, time
t0 = time.monotonic()
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.settimeout({timeout})
try:
    s.connect(("{dst}", {port}))
    body = b"U" * {total_bytes}
    req = (
        f"POST /upload HTTP/1.1\\r\\nHost: {dst}\\r\\nContent-Length: {total_bytes}\\r\\nConnection: close\\r\\n\\r\\n"
    ).encode() + body
    s.sendall(req)
    chunks = []
    while True:
        data = s.recv(4096)
        if not data:
            break
        chunks.append(data)
    raw = b"".join(chunks).decode("latin1", errors="replace")
    head, _, payload = raw.partition("\\r\\n\\r\\n")
    print(json.dumps({{"ok": payload.startswith("ok:"), "status": head.splitlines()[0] if head else "", "body": payload.strip(), "elapsed_ms": round((time.monotonic() - t0) * 1000, 2)}}))
except Exception as exc:
    print(json.dumps({{"ok": False, "error": str(exc), "elapsed_ms": round((time.monotonic() - t0) * 1000, 2)}}))
finally:
    s.close()
"""
    out = run("ip", "netns", "exec", ns, "python3", "-B", "-c", code, check=False)
    try:
        res = json.loads(out.stdout.strip().splitlines()[-1])
    except Exception:
        res = {"ok": False, "error": (out.stderr or out.stdout).strip()}
    res["pmtu"] = route_pmtu(ns, dst)
    return res


def setup_paths():
    run("ip", "netns", "del", ORIGIN_NS, check=False)
    run("ip", "-n", "fv-inet", "link", "del", "fv-f34r", check=False)
    run("ip", "netns", "add", ORIGIN_NS)
    run("ip", "-n", ORIGIN_NS, "link", "set", "lo", "up")
    run("ip", "link", "add", "fv-f34r", "type", "veth", "peer", "name", "fv-f34o")
    run("ip", "link", "set", "fv-f34r", "netns", "fv-inet")
    run("ip", "link", "set", "fv-f34o", "netns", ORIGIN_NS)

    run("ip", "-n", "fv-inet", "addr", "add", "198.18.34.1/30", "dev", "fv-f34r")
    run("ip", "-n", "fv-inet", "link", "set", "fv-f34r", "mtu", str(PATH_MTU), "up")
    run("ip", "netns", "exec", "fv-inet", "ethtool", "-K", "fv-f34r", "gro", "off", "gso", "off", "tso", "off", check=False)
    run("ip", "-n", "fv-inet", "route", "replace", f"{ORIGIN_IP}/32", "via", "198.18.34.2", "dev", "fv-f34r")
    for row in ROWS:
        run("ip", "-n", "fv-inet", "addr", "replace", f"{row['source_ip']}/32", "dev", "lo")
    run(
        "ip",
        "netns",
        "exec",
        "fv-inet",
        "sysctl",
        "-wq",
        "net.ipv4.ip_forward=1",
        "net.ipv4.icmp_ratelimit=0",
        "net.ipv4.icmp_msgs_per_sec=100000",
        "net.ipv4.icmp_msgs_burst=100000",
    )
    run(
        "ip",
        "netns",
        "exec",
        "fv-inet",
        "sh",
        "-c",
        "for f in /proc/sys/net/ipv4/conf/*/rp_filter; do echo 0 > $f; done",
    )

    run("ip", "-n", ORIGIN_NS, "addr", "add", "198.18.34.2/30", "dev", "fv-f34o")
    run("ip", "-n", ORIGIN_NS, "addr", "add", f"{ORIGIN_IP}/32", "dev", "lo")
    run("ip", "-n", ORIGIN_NS, "link", "set", "fv-f34o", "mtu", "1500", "up")
    run("ip", "netns", "exec", ORIGIN_NS, "ethtool", "-K", "fv-f34o", "gro", "off", "gso", "off", "tso", "off", check=False)
    run("ip", "-n", ORIGIN_NS, "route", "replace", "default", "via", "198.18.34.1", "dev", "fv-f34o")
    run(
        "ip",
        "netns",
        "exec",
        ORIGIN_NS,
        "sh",
        "-c",
        "for f in /proc/sys/net/ipv4/conf/*/rp_filter; do echo 0 > $f; done",
    )

    run("ip", "route", "replace", "192.168.34.1/32", "via", "192.0.2.38", "dev", "fv-ih")
    run("ip", "route", "replace", "192.168.34.2/32", "via", "192.0.2.38", "dev", "fv-ih")

    proc = subprocess.Popen(
        ["ip", "netns", "exec", ORIGIN_NS, "python3", "-B", "/opt/netlab/scripts/services.py", "inet"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    probe = (
        f"import socket\ns=socket.socket()\ns.settimeout(0.4)\n"
        f"s.connect(('{ORIGIN_IP}', {ORIGIN_PORT}))\ns.close()\n"
    )
    ready = False
    for _ in range(25):
        res = run("ip", "netns", "exec", "fv-inet", "python3", "-B", "-c", probe, check=False)
        if res.returncode == 0:
            ready = True
            break
        time.sleep(0.1)
    if not ready:
        proc.terminate()
        raise RuntimeError("f34 origin listener did not become ready")
    return proc


def cleanup_paths(proc):
    if proc is not None:
        proc.terminate()
        try:
            proc.wait(timeout=2)
        except subprocess.TimeoutExpired:
            proc.kill()
    run("ip", "-n", "fv-inet", "route", "replace", "192.0.2.36/30", "dev", "fv-ii", "proto", "kernel", "scope", "link", "src", "192.0.2.38", check=False)
    for row in ROWS:
        run("ip", "-n", "fv-inet", "addr", "del", f"{row['source_ip']}/32", "dev", "lo", check=False)
    run("ip", "route", "del", "192.168.34.1/32", "via", "192.0.2.38", "dev", "fv-ih", check=False)
    run("ip", "route", "del", "192.168.34.2/32", "via", "192.0.2.38", "dev", "fv-ih", check=False)
    run("ip", "-n", "fv-inet", "route", "del", f"{ORIGIN_IP}/32", "via", "198.18.34.2", "dev", "fv-f34r", check=False)
    run("ip", "-n", "fv-inet", "link", "del", "fv-f34r", check=False)
    run("ip", "netns", "del", ORIGIN_NS, check=False)
    run("ip", "-n", MEMBER_NS, "route", "flush", "cache", check=False)


def run_row(row_spec: dict) -> dict:
    row_id = row_spec["id"]
    src_ip = row_spec["source_ip"]
    expected_sets = row_spec["expected_sets"]
    upload_drop_counter = row_spec["upload_drop_counter"]

    membership = verify_set_membership(src_ip)
    membership_ok = membership["matched_sets"] == expected_sets

    # Route fv-inet's return link to node with preferred source src_ip so the
    # kernel router emits ICMP fragmentation-needed and time-exceeded errors
    # with outer source src_ip.
    run(
        "ip",
        "-n",
        "fv-inet",
        "route",
        "replace",
        "192.0.2.36/30",
        "dev",
        "fv-ii",
        "proto",
        "kernel",
        "scope",
        "link",
        "src",
        src_ip,
    )
    run("ip", "-n", MEMBER_NS, "route", "flush", "cache")
    pmtu_before = route_pmtu(MEMBER_NS, ORIGIN_IP)

    # 1. Positive: genuine router ICMP frag-needed (type 3, code 4, MTU 1000)
    # on an admitted member's 64,064-byte TCP upload.
    with (
        capture("fv-inet", "fv-ii", f"icmp and src host {src_ip}") as router_pcap,
        capture(None, "fv-ih", f"ip and src host {SNAT_IP} and dst host {ORIGIN_IP}", "out") as up_pcap,
        capture(None, "fv-fh", f"ip and dst host {MEMBER_IP} and (src host {ORIGIN_IP} or src host {src_ip})", "out") as down_pcap,
        capture(MEMBER_NS, "fv-wg0", f"(tcp and host {ORIGIN_IP}) or (icmp and src host {src_ip})") as mem_pcap,
    ):
        c_before = read_counters()
        tc_up_before = read_tc_classes("fv-ih")
        tc_down_before = read_tc_classes("fv-fh")
        transfer = tcp_upload(MEMBER_NS, ORIGIN_IP, ORIGIN_PORT, UPLOAD_BYTES)
        time.sleep(0.2)
    c_after = read_counters()
    tc_up_after = read_tc_classes("fv-ih")
    tc_down_after = read_tc_classes("fv-fh")

    router_cap = parse_pcap(router_pcap)
    up_cap = parse_pcap(up_pcap)
    down_cap = parse_pcap(down_pcap)
    mem_cap = parse_pcap(mem_pcap)
    for p in (router_pcap, up_pcap, down_pcap, mem_pcap):
        p.unlink(missing_ok=True)

    pos_counters = diff_counters(c_before, c_after)
    tc_up_diff = diff_tc_classes(tc_up_before, tc_up_after)
    tc_down_diff = diff_tc_classes(tc_down_before, tc_down_after)

    up_usage = pos_counters[f"u_{DEV_A1}"]
    down_usage = pos_counters[f"d_{DEV_A1}"]
    up_leaf = tc_up_diff.get(ASSIGNED_CLASS, {"packets": 0, "bytes": 0, "drops": 0})
    down_leaf = tc_down_diff.get(ASSIGNED_CLASS, {"packets": 0, "bytes": 0, "drops": 0})

    frag_router = [x for x in router_cap["icmp"] if x["icmp_type"] == 3 and x["icmp_code"] == 4 and x["mtu"] == PATH_MTU]
    frag_down = [x for x in down_cap["icmp"] if x["icmp_type"] == 3 and x["icmp_code"] == 4 and x["mtu"] == PATH_MTU]
    frag_mem = [x for x in mem_cap["icmp"] if x["icmp_type"] == 3 and x["icmp_code"] == 4 and x["mtu"] == PATH_MTU]

    pos_ok = (
        membership_ok
        and pmtu_before is None
        and transfer.get("ok") is True
        and transfer.get("body") == f"ok:{UPLOAD_BYTES}"
        and transfer.get("pmtu") == PATH_MTU
        and len(frag_router) >= 1
        and len(frag_down) == len(frag_router)
        and len(frag_mem) == len(frag_router)
        and down_cap["icmp_ip_bytes"] > 0
        and pos_counters["ctdenyd"]["packets"] == 0
        and pos_counters["ctinv"]["packets"] == 0
        and pos_counters["ctdns"]["packets"] == 0
        and pos_counters["ctdeny"]["packets"] == 0
        and pos_counters["cthost"]["packets"] == 0
        and up_usage == {"packets": up_cap["packets"], "bytes": up_cap["ip_bytes"]}
        and down_usage == {"packets": down_cap["packets"], "bytes": down_cap["ip_bytes"]}
        and down_usage["bytes"] == down_cap["tcp_ip_bytes"] + down_cap["icmp_ip_bytes"]
        and up_leaf == {
            "packets": up_usage["packets"],
            "bytes": up_usage["bytes"] + ETHERNET_L2_LEN * up_usage["packets"],
            "drops": 0,
        }
        and down_leaf == {
            "packets": down_usage["packets"],
            "bytes": down_usage["bytes"] + ETHERNET_L2_LEN * down_usage["packets"],
            "drops": 0,
        }
    )

    positive_subcase = {
        "name": "related-frag-needed",
        "passed": pos_ok,
        "pmtu_before": pmtu_before,
        "transfer": transfer,
        "captures": {
            "router_fv_ii": router_cap,
            "node_up_egress_fv_ih": up_cap,
            "node_down_egress_fv_fh": down_cap,
            "member_fv_wg0": mem_cap,
        },
        "counters": pos_counters,
        "usage_reconciliation": {
            "device_id": DEV_A1,
            "assigned_priority": ASSIGNED_PRIORITY,
            "assigned_tc_class": ASSIGNED_CLASS,
            "upload_counter_matches_capture": up_usage == {"packets": up_cap["packets"], "bytes": up_cap["ip_bytes"]},
            "download_counter_matches_capture": down_usage == {"packets": down_cap["packets"], "bytes": down_cap["ip_bytes"]},
            "download_tcp_ip_bytes": down_cap["tcp_ip_bytes"],
            "download_frag_needed_icmp_packets": down_cap["icmp_packets"],
            "download_frag_needed_icmp_ip_bytes": down_cap["icmp_ip_bytes"],
            "download_total_ip_bytes": down_usage["bytes"],
            "tc_upload_leaf": up_leaf,
            "tc_download_leaf": down_leaf,
            "tc_upload_all": tc_up_diff,
            "tc_download_all": tc_down_diff,
        },
    }

    # 2. Negative 1: time-exceeded (ICMP type 11, code 0) from src_ip is dropped at ctdenyd.
    with (
        capture("fv-inet", "fv-ii", f"icmp and src host {src_ip}") as te_router_pcap,
        capture(None, "fv-ih", f"icmp and src host {src_ip}", "in") as te_node_in_pcap,
        capture(None, "fv-fh", f"icmp and src host {src_ip}", "out") as te_node_out_pcap,
        capture(MEMBER_NS, "fv-wg0", f"icmp and src host {src_ip}") as te_mem_pcap,
    ):
        te_before = read_counters()
        run("ip", "netns", "exec", MEMBER_NS, "ping", "-n", "-t", "3", "-c", "1", "-W", "1", ORIGIN_IP, check=False)
        time.sleep(0.2)
    te_after = read_counters()
    te_router_cap = parse_pcap(te_router_pcap)
    te_node_in_cap = parse_pcap(te_node_in_pcap)
    te_node_out_cap = parse_pcap(te_node_out_pcap)
    te_mem_cap = parse_pcap(te_mem_pcap)
    for p in (te_router_pcap, te_node_in_pcap, te_node_out_pcap, te_mem_pcap):
        p.unlink(missing_ok=True)
    te_counters = diff_counters(te_before, te_after)
    te_ok = (
        any(x["icmp_type"] == 11 and x["icmp_code"] == 0 for x in te_router_cap["icmp"])
        and any(x["icmp_type"] == 11 and x["icmp_code"] == 0 for x in te_node_in_cap["icmp"])
        and te_node_in_cap["packets"] == 1
        and te_node_out_cap["packets"] == 0
        and te_mem_cap["packets"] == 0
        and te_counters["ctdenyd"]["packets"] == 1
        and te_counters[f"d_{DEV_A1}"]["packets"] == 0
        and te_counters[f"d_{DEV_A1}"]["bytes"] == 0
    )
    time_exceeded_subcase = {
        "name": "time-exceeded-dropped",
        "passed": te_ok,
        "dropped_at_counter": "ctdenyd",
        "captures": {
            "router_fv_ii": te_router_cap,
            "node_ingress_fv_ih": te_node_in_cap,
            "node_egress_fv_fh": te_node_out_cap,
            "member_fv_wg0": te_mem_cap,
        },
        "counters": te_counters,
    }

    # 3. Negative 2: unsolicited fragmentation-needed (untracked quoted tuple, valid checksum)
    # from src_ip is dropped at ctinv.
    with (
        capture(None, "fv-ih", f"icmp and src host {src_ip}", "in") as unsol_node_in_pcap,
        capture(None, "fv-fh", f"icmp and src host {src_ip}", "out") as unsol_node_out_pcap,
        capture(MEMBER_NS, "fv-wg0", f"icmp and src host {src_ip}") as unsol_mem_pcap,
    ):
        unsol_before = read_counters()
        send_unsolicited_frag_needed(src_ip)
        time.sleep(0.2)
    unsol_after = read_counters()
    unsol_node_in_cap = parse_pcap(unsol_node_in_pcap)
    unsol_node_out_cap = parse_pcap(unsol_node_out_pcap)
    unsol_mem_cap = parse_pcap(unsol_mem_pcap)
    for p in (unsol_node_in_pcap, unsol_node_out_pcap, unsol_mem_pcap):
        p.unlink(missing_ok=True)
    unsol_counters = diff_counters(unsol_before, unsol_after)
    unsol_ok = (
        any(x["icmp_type"] == 3 and x["icmp_code"] == 4 and x["mtu"] == PATH_MTU for x in unsol_node_in_cap["icmp"])
        and unsol_node_in_cap["packets"] == 1
        and unsol_node_out_cap["packets"] == 0
        and unsol_mem_cap["packets"] == 0
        and unsol_counters["ctinv"]["packets"] == 1
        and unsol_counters["ctdenyd"]["packets"] == 0
        and unsol_counters[f"d_{DEV_A1}"]["packets"] == 0
        and unsol_counters[f"d_{DEV_A1}"]["bytes"] == 0
    )
    unsolicited_subcase = {
        "name": "unsolicited-frag-needed-dropped",
        "passed": unsol_ok,
        "dropped_at_counter": "ctinv",
        "captures": {
            "node_ingress_fv_ih": unsol_node_in_cap,
            "node_egress_fv_fh": unsol_node_out_cap,
            "member_fv_wg0": unsol_mem_cap,
        },
        "counters": unsol_counters,
    }

    # 4. Negative 3: member upload to src_ip is dropped at the upload destination-deny
    # rule before u_<device>, with 0 upload usage.
    with (
        capture(None, "fv-fh", f"icmp and src host {MEMBER_IP} and dst host {src_ip}", "in") as up_node_in_pcap,
        capture(None, "fv-ih", f"icmp and dst host {src_ip}", "out") as up_node_out_pcap,
        capture("fv-inet", "fv-ii", f"icmp and dst host {src_ip}") as up_router_pcap,
    ):
        up_before = read_counters()
        run("ip", "netns", "exec", MEMBER_NS, "ping", "-n", "-c", "1", "-W", "1", src_ip, check=False)
        time.sleep(0.2)
    up_after = read_counters()
    up_node_in_cap = parse_pcap(up_node_in_pcap)
    up_node_out_cap = parse_pcap(up_node_out_pcap)
    up_router_cap = parse_pcap(up_router_pcap)
    for p in (up_node_in_pcap, up_node_out_pcap, up_router_pcap):
        p.unlink(missing_ok=True)
    up_counters = diff_counters(up_before, up_after)
    other_upload_drop_counters = [c for c in ("ctdns", "ctdeny", "cthost") if c != upload_drop_counter]
    up_ok = (
        up_node_in_cap["packets"] == 1
        and up_node_out_cap["packets"] == 0
        and up_router_cap["packets"] == 0
        and up_counters[upload_drop_counter]["packets"] == 1
        and all(up_counters[c]["packets"] == 0 for c in other_upload_drop_counters)
        and up_counters[f"u_{DEV_A1}"]["packets"] == 0
        and up_counters[f"u_{DEV_A1}"]["bytes"] == 0
        and up_counters[f"du_{DEV_A1}"]["packets"] == 0
        and up_counters[f"du_{DEV_A1}"]["bytes"] == 0
    )
    upload_subcase = {
        "name": "member-upload-to-source-dropped",
        "passed": up_ok,
        "dropped_at_counter": upload_drop_counter,
        "captures": {
            "node_ingress_fv_fh": up_node_in_cap,
            "node_egress_fv_ih": up_node_out_cap,
            "router_fv_ii": up_router_cap,
        },
        "counters": up_counters,
    }

    row_passed = membership_ok and pos_ok and te_ok and unsol_ok and up_ok
    return {
        "id": row_id,
        "source_ip": src_ip,
        "expected_sets": expected_sets,
        "set_membership": membership,
        "set_membership_verified": membership_ok,
        "upload_drop_counter": upload_drop_counter,
        "passed": row_passed,
        "subcases": [
            positive_subcase,
            time_exceeded_subcase,
            unsolicited_subcase,
            upload_subcase,
        ],
    }


def main() -> int:
    proc = None
    try:
        proc = setup_paths()
        rows = [run_row(spec) for spec in ROWS]
        passed = all(r["passed"] for r in rows)
        out = {
            "passed": passed,
            "member_ns": MEMBER_NS,
            "member_ip": MEMBER_IP,
            "device_id": DEV_A1,
            "snat_ip": SNAT_IP,
            "origin_ip": ORIGIN_IP,
            "origin_port": ORIGIN_PORT,
            "path_mtu": PATH_MTU,
            "upload_bytes": UPLOAD_BYTES,
            "assigned_priority": ASSIGNED_PRIORITY,
            "assigned_tc_class": ASSIGNED_CLASS,
            "rows": rows,
        }
        print(json.dumps(out, indent=2))
        return 0 if passed else 1
    finally:
        cleanup_paths(proc)


if __name__ == "__main__":
    sys.exit(main())
