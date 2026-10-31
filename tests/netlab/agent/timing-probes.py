"""TEST ONLY: dual-phase 80 ms UDP probes and independent node packet reads.

Invoked inside the isolated agent netlab container. Times are decimal strings
on CLOCK_MONOTONIC; packet read times are capture observations, not commits.
"""
import json
import os
from pathlib import Path
import select
import signal
import socket
import struct
import subprocess
import sys
import time
from datetime import datetime, timezone

PORT = 40099


def anchor():
    before = time.monotonic_ns()
    wall = datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z')
    return dict(mono_before_ns=str(before), wall=wall,
                mono_after_ns=str(time.monotonic_ns()))


def client(mode, deadline, output):
    now = time.monotonic_ns()
    if mode == 'renewal':
        start, stop = now + 100_000_000, now + 76_000_000_000
    else:
        start = deadline - 101_000_000
        start -= ((start - max(now + 100_000_000, deadline - 5_000_000_000)) // 100_000_000) * 100_000_000
        stop = deadline + 600_000_000
    schedule = sorted((at + offset, phase) for at in range(start, stop, 100_000_000)
                      for offset, phase in [(0, 'phased'), (37_000_000, 'arbitrary_37ms')])
    rows, pending = [], {}
    for_at = 0
    while for_at < len(schedule) or pending:
        now = time.monotonic_ns()
        while for_at < len(schedule) and schedule[for_at][0] <= now:
            at, phase = schedule[for_at]
            s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            s.setblocking(False)
            s.connect(('203.0.113.10', PORT))
            payload = struct.pack('!I', for_at) + b'TEST ONLY'
            row = dict(packet_id=for_at, phase=phase, scheduled_ns=str(at),
                       sent_ns=str(time.monotonic_ns()), result='timeout')
            s.send(payload)
            pending[s] = (row, int(row['sent_ns']) + 80_000_000, payload)
            rows.append(row)
            for_at += 1
        wake = min([v[1] for v in pending.values()] +
                   ([schedule[for_at][0]] if for_at < len(schedule) else []))
        ready, _, _ = select.select(list(pending), [], [], max(0, (wake-time.monotonic_ns())/1e9))
        for s in ready:
            row, _, payload = pending[s]
            try:
                if s.recv(2048) == payload:
                    row['result'] = 'ok'
                    row['read_completion_ns'] = str(time.monotonic_ns())
                    s.close()
                    del pending[s]
            except (BlockingIOError, ConnectionRefusedError):
                pass
        now = time.monotonic_ns()
        for s, (row, end, _) in list(pending.items()):
            if now >= end:
                row['read_completion_ns'] = str(now)
                s.close()
                del pending[s]
    Path(output).write_text(json.dumps(rows))


def trace(mode, deadline, output):
    if not Path('/.dockerenv').exists():
        raise RuntimeError('lab container required')
    clocks = [anchor()]
    reads = []
    packets = {}
    for dev in ['fv-fh', 'fv-ih']:
        s = socket.socket(socket.AF_PACKET, socket.SOCK_RAW, socket.htons(3))
        s.setsockopt(socket.SOL_SOCKET, socket.SO_RCVBUF, 4*1024*1024)
        s.bind((dev, 0))
        s.setblocking(False)
        reads.append(s)
    server = subprocess.Popen(['ip', 'netns', 'exec', 'fv-inet', sys.executable, '-u', __file__, 'serve'], stdout=subprocess.PIPE)
    client_path = output + '.client.json'
    restart = None
    try:
        if server.stdout.readline() != b'ready\n':
            raise RuntimeError('echo server did not start')
        proc = subprocess.Popen(['ip', 'netns', 'exec', 'fv-a1', sys.executable, __file__,
                                 'client', mode, str(deadline), client_path])
        while proc.poll() is None:
            if mode == 'expiry_restart' and restart is None and time.monotonic_ns() >= deadline-3_000_000_000:
                restart = subprocess.Popen([sys.executable, __file__, 'restart', output+'.restart.json'])
            ready, _, _ = select.select(reads, [], [], .01)
            for s in ready:
                while True:
                    try:
                        packet = s.recv(65535)
                    except BlockingIOError:
                        break
                    at = time.monotonic_ns()
                    if len(packet) >= 55 and packet[12:14] == b'\x08\x00' and packet[23] == 17 and packet[36:38] == struct.pack('!H', PORT) and packet[46:55] == b'TEST ONLY':
                        seq = struct.unpack('!I', packet[42:46])[0]
                        packets.setdefault(seq, {})['ingress_ns' if s == reads[0] else 'egress_ns'] = str(at)
        if proc.wait() != 0:
            raise RuntimeError('probe process failed')
        if mode == 'expiry_restart':
            if restart is None or restart.wait() != 0:
                raise RuntimeError('restart did not complete')
        rows = json.loads(Path(client_path).read_text())
        for row in rows:
            row.update(packets.get(row['packet_id'], {}))
        stats = [dict(zip(['packets', 'drops'], struct.unpack('II', s.getsockopt(263, 6, 8)))) for s in reads]
        clocks.append(anchor())
        result = dict(mode=mode, clock='CLOCK_MONOTONIC', boot_id=Path('/proc/sys/kernel/random/boot_id').read_text().strip(),
                      anchors=clocks, deadline_ns=str(deadline), packets=rows, capture_stats=stats)
        Path(output).write_text(json.dumps(result, indent=2)+'\n')
        if any(s['drops'] for s in stats):
            raise RuntimeError('capture dropped packets')
    finally:
        if 'proc' in locals() and proc.poll() is None:
            proc.terminate()
            proc.wait()
        server.terminate()
        server.wait()
        for s in reads:
            s.close()


def restart_agent(output):
    # Only the harness-owned process in this lab container; topology is intact.
    if not Path('/.dockerenv').exists():
        raise RuntimeError('lab container required')
    pidfile = Path('/scratch/w2/agent.pid')
    pid = int(pidfile.read_text())
    evidence = dict(start_ns=str(time.monotonic_ns()))
    os.kill(pid, signal.SIGTERM)
    until = time.monotonic()+2
    while time.monotonic() < until:
        try:
            if Path(f'/proc/{pid}/stat').read_text().split()[2] == 'Z':
                break
        except FileNotFoundError:
            break
        time.sleep(.005)
    else:
        raise RuntimeError('agent failed to stop gracefully')
    evidence['stopped_ns'] = str(time.monotonic_ns())
    with open('/scratch/w2/agent.log','ab') as log:
        child = subprocess.Popen(['fvpn-agent','run','-config','/scratch/w2/agent.json'],stdout=log,stderr=log,start_new_session=True)
    pidfile.write_text(str(child.pid)+'\n')
    evidence['started_ns'] = str(time.monotonic_ns())
    Path(output).write_text(json.dumps(evidence,indent=2)+'\n')


if __name__ == '__main__':
    if sys.argv[1] == 'restart':
        restart_agent(sys.argv[2])
    elif sys.argv[1] == 'serve':
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        s.bind(('203.0.113.10', PORT))
        print('ready', flush=True)
        while True:
            data, peer = s.recvfrom(2048)
            s.sendto(data, peer)
    elif sys.argv[1] == 'client':
        client(sys.argv[2], int(sys.argv[3]), sys.argv[4])
    else:
        trace(sys.argv[2], int(sys.argv[3]), sys.argv[4])
