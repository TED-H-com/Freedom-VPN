#!/usr/bin/env python3
"""Foreground D-116 / AT10 agent measurement with harness-owned teardown.

Call through disktmp.py. The caller checks the shared-host suite limit first.
"""
import os
from pathlib import Path
import subprocess
import sys

prefix = os.environ.get('FVPN_LAB_PREFIX', '')
if not prefix.startswith('fvpn-w1-netlab-'):
    raise SystemExit('set FVPN_LAB_PREFIX=fvpn-w1-netlab-<run>')
root = Path(__file__).resolve().parents[3]
make = ['make', '-f', 'mk/netlab.mk']
status = 1
try:
    subprocess.run(make + ['netlab-up'], cwd=root, check=True)
    for binary, package in [('fvpn-agent', './node/agent/cmd/fvpn-agent'),
                            ('fvpn-labcontrol', './node/internal/fakecontrol/cmd/fvpn-labcontrol')]:
        path = Path(os.environ['GOTMPDIR']) / binary
        subprocess.run(['go', 'build', '-o', str(path), package], cwd=root, check=True)
        subprocess.run(['docker', 'cp', str(path), prefix + ':/usr/local/bin/' + binary], check=True)
    env = dict(os.environ, FVPN_NETLAB='1', FVPN_D116_MARKER='1')
    status = subprocess.run(['go', 'test', '-tags', 'netlab', '-count=1', '-timeout=8m', '-v',
                             './tests/netlab/agent', '-run', '^TestD116_AT10_InstallerMarkerEnrollment$'],
                            cwd=root, env=env, check=False).returncode
finally:
    subprocess.run(make + ['netlab-down'], cwd=root, check=False)
sys.exit(status)
