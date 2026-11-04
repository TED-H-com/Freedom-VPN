#!/usr/bin/env python3
"""D-160 command admission. Receipts never share a tool's stdout/stderr.

Use `admission.py run <command...>` to wrap a foreground command tree. A timed
section is admitted by the Go harness before it exports its section name.
Children record that name and execute without another check or admission wait.
No reservation is taken. No arguments or private admission text are published.
"""
import datetime
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile
import time


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def record(row):
    path = os.environ.get('FVPN_E2E_ADMISSION_LOG')
    if not path:
        raise RuntimeError('FVPN_E2E_ADMISSION_LOG is required for admission')
    data = (json.dumps(row, separators=(',', ':')) + '\n').encode()
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600)
    try:
        os.write(fd, data)
    finally:
        os.close(fd)


def admit(label, check=None, sleep=time.sleep, monotonic=time.monotonic):
    if check is None:
        check = lambda: subprocess.run(['bash', os.environ['FVPN_HOLD_SH']],
                                       capture_output=True, text=True)
    begin = monotonic()
    while True:
        at = now()
        result = check()
        free = result.returncode == 0 and result.stdout.strip() == 'free'
        record({'event': 'admission', 'at': at, 'answered_at': now(),
                'label': label, 'answer': 'free' if free else 'held',
                'exit_code': result.returncode})
        if free:
            return
        if monotonic() - begin >= 7200:
            raise SystemExit(125)
        sleep(min(60, 7200 - (monotonic() - begin)))


def tool(name, paths):
    section = os.environ.get('FVPN_E2E_TIMED_SECTION')
    if section:
        record({'event': 'command_in_section', 'at': now(), 'command': name,
                'section': section, 'admission_checked': False})
    else:
        admit(name)
    # argv[0] stays PATH-relative: recursive GNU Make must use this wrapper.
    os.execve(paths[name], [name] + sys.argv[1:], os.environ)


def run(command):
    if not os.environ.get('FVPN_HOLD_SH') or os.environ.get('FVPN_E2E_ADMISSION_WRAPPED'):
        return subprocess.call(command)
    with tempfile.TemporaryDirectory(prefix='e2e-admission-', dir=os.environ['TMPDIR']) as directory:
        bindir = pathlib.Path(directory)
        paths = {name: shutil.which(name) for name in ('go', 'make', 'docker', 'npm')}
        if not all(paths.values()):
            raise RuntimeError('missing command for admission wrappers')
        (bindir / 'paths.json').write_text(json.dumps(paths))
        for name in paths:
            (bindir / name).symlink_to(pathlib.Path(__file__).resolve())
        env = os.environ.copy()
        env['PATH'] = str(bindir) + os.pathsep + env['PATH']
        env['FVPN_E2E_ADMISSION_WRAPPED'] = '1'
        return subprocess.call(command, env=env)


if __name__ == '__main__':
    name = pathlib.Path(sys.argv[0]).name
    if name in ('go', 'make', 'docker', 'npm'):
        paths = json.loads((pathlib.Path(sys.argv[0]).parent / 'paths.json').read_text())
        tool(name, paths)
    elif sys.argv[1] == 'check':
        if os.environ.get('FVPN_E2E_TIMED_SECTION'):
            raise RuntimeError('admission check forbidden inside a timed section')
        admit(sys.argv[2])
    elif sys.argv[1] == 'run':
        raise SystemExit(run(sys.argv[2:]))
    else:
        raise SystemExit('expected check or run')
