#!/usr/bin/env python3
"""TEST ONLY collector prerequisite, separate from the one diagnostic window."""
import json
import os
from pathlib import Path
import subprocess
import sys
sys.dont_write_bytecode=True
import host
name=sys.argv[1];prefix=host.PREFIX
out=host.DEST/name;out.mkdir()
code=1
try:
    code=subprocess.run(['make','-f','mk/netlab.mk','netlab-up','FVPN_LAB_PREFIX='+prefix]).returncode
    if code==0:
        code=subprocess.run(['docker','exec',prefix,'python3','-B','/work/docs/poc/G2-followup30-evidence/collector-check.py']).returncode
finally:
    for suffix in ('.err','.jsonl','.meta.json','-rules.nft','-injected.json','-readback.json','-result.json'):
        p=subprocess.run(['docker','exec',prefix,'cat','/scratch/collector'+suffix],capture_output=True)
        if p.returncode==0:(out/('collector'+suffix)).write_bytes(p.stdout)
    subprocess.run(['make','-f','mk/netlab.mk','netlab-down','FVPN_LAB_PREFIX='+prefix],check=True)
    subprocess.run([sys.executable,'-B',str(host.DEST/'cleanup.py'),name+'-cleanup'],check=True)
sys.exit(code)
