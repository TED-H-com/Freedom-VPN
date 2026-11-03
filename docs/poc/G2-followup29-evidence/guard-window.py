#!/usr/bin/env python3
"""TEST ONLY one foreground 100-reload diagnostic; cleanup even on failure."""
import os
import subprocess
import sys

prefix='fvpn-w1-netlab-f29'
status=1
try:
    status=subprocess.run(['make','-f','mk/netlab.mk','netlab-up','FVPN_LAB_PREFIX='+prefix]).returncode
    if status==0:
        status=subprocess.run(['go','test','-tags','netlab','-count=1','-timeout','10m','-p','1','-parallel','1',
            '-v','./tests/netlab','-run','^TestAT10_AT13_ReloadRevokeBaseline$'],env=dict(os.environ,FVPN_NETLAB='1')).returncode
finally:
    cleanup=subprocess.run(['make','-f','mk/netlab.mk','netlab-down','FVPN_LAB_PREFIX='+prefix]).returncode
    if cleanup:status=status or cleanup
sys.exit(status)
