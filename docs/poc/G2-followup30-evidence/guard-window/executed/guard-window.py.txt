#!/usr/bin/env python3
"""TEST ONLY one foreground 100-reload diagnostic; cleanup even on failure."""
from pathlib import Path
import os
import subprocess
import sys

prefix='fvpn-w1-netlab-f30'
status=1
try:
    status=subprocess.run(['make','-f','mk/netlab.mk','netlab-up','FVPN_LAB_PREFIX='+prefix]).returncode
    if status==0:
        status=subprocess.run(['go','test','-tags','netlab','-count=1','-timeout','10m','-p','1','-parallel','1',
            '-v','./tests/netlab','-run','^TestAT10_AT13_ReloadRevokeBaseline$'],env=dict(os.environ,FVPN_NETLAB='1')).returncode
finally:
    # Even failed readiness/cleanup keeps the derived log and capture errors.
    destination=Path(os.environ['FVPN_NETLAB_EVIDENCE_DIR'])/'TestAT10_AT13_ReloadRevokeBaseline'
    destination.mkdir(parents=True,exist_ok=True)
    for suffix in ('.err','.meta.json','.jsonl'):
        name='at10-guard'+suffix
        if (destination/name).exists():continue
        result=subprocess.run(['docker','exec',prefix,'cat','/scratch/'+name],capture_output=True)
        if result.returncode==0:(destination/name).write_bytes(result.stdout)
    cleanup=subprocess.run(['make','-f','mk/netlab.mk','netlab-down','FVPN_LAB_PREFIX='+prefix]).returncode
    if cleanup:status=status or cleanup
sys.exit(status)
