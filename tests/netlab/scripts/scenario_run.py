#!/usr/bin/env python3
"""Fresh-topology focused foreground run; Python owns unconditional teardown.

Caller checks the shared-host suite slot first. Invoke through disktmp.py.
"""
import os
import subprocess
import sys

prefix = os.environ['FVPN_LAB_PREFIX']
if not prefix.startswith('fvpn-w1-netlab-'):
    raise SystemExit('this launcher requires a lane-owned object prefix')
make = ['make', '-f', 'mk/netlab.mk']
status = 1
try:
    subprocess.run(make + ['netlab-up'], check=True)
    env = dict(os.environ, FVPN_NETLAB='1')
    status = subprocess.run(['go', 'test', '-tags', 'netlab', '-count=1', '-timeout=10m',
                             '-v', './tests/netlab', '-run', sys.argv[1]], env=env).returncode
finally:
    teardown = subprocess.run(make + ['netlab-down'])
    if teardown.returncode:
        status = teardown.returncode
sys.exit(status)
