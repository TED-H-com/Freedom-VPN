#!/usr/bin/python3
"""TEST ONLY common A/B collector; dispatch unchanged binaries, write after exit.

Installed as fvpn-netlab in the owned container; real binary keeps the same basename in a separate directory. Only synthetic lab Apply inputs/results are retained. No shell, no
network operations, no command changes and no output filtering.
"""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time

args = sys.argv[1:]
real = '/usr/local/lib/f22/fvpn-netlab'
if not args or args[0] not in ('apply', 'hold'):
    os.execv(real, [real] + args)
body = None
if '-f' not in args or args[args.index('-f') + 1] == '-':
    body = sys.stdin.buffer.read()
    model = body
else:
    model = Path(args[args.index('-f') + 1]).read_bytes()
start = time.monotonic_ns()
p = subprocess.run([real] + args, input=body, capture_output=True, check=False)
end = time.monotonic_ns()
sys.stdout.buffer.write(p.stdout)
sys.stderr.buffer.write(p.stderr)
root = Path('/scratch/f22-command-results')
root.mkdir(exist_ok=True)
record = dict(test_only=True, invocation=args[0], start_ns=str(start), end_ns=str(end), exit=p.returncode,
              input_sha256=hashlib.sha256(model).hexdigest(), input_model=json.loads(model))
try:
    record['result'] = json.loads(p.stdout)
except (ValueError, UnicodeDecodeError):
    record['result_available'] = False
(root / (str(start) + '.json')).write_text(json.dumps(record, indent=2)+'\n')
sys.exit(p.returncode)
