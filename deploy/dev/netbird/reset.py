#!/usr/bin/env python3
"""Explicit reset of this disposable TEST ONLY stack and its generated secrets."""
import os
from pathlib import Path
import shutil
import subprocess

root = Path(__file__).resolve().parent
prefix = os.environ.get('FVPN_NB_PREFIX', 'fvpn-dev-netbird')
state = root / '.state'
marker = state / 'prefix'
if not prefix.startswith('fvpn-') or (marker.exists() and marker.read_text().strip() != prefix):
    raise SystemExit('Refusing reset: invalid prefix or state belongs to another stack')
subprocess.run(['docker', 'compose', '-f', str(root / 'compose.yml'), '--profile', 'clients',
                'down', '-v', '--rmi', 'all'], check=True)
if state.is_symlink():
    raise SystemExit('Refusing symlinked state directory')
if state.exists():
    shutil.rmtree(state)
print('Disposable NetBird stack, owned images/volumes and generated secrets removed.')
