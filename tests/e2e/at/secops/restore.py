#!/usr/bin/env python3
"""Restore only the task's logical backup into a network-none PostgreSQL copy.

No source database is reset. No host ports or other stack objects are used.
This is a measured subset of AT30, not a complete WAL/PITR drill.
"""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[4]
PRIVATE = ROOT / '.agent/secops'
STATE = ROOT / '.agent/e2e-state-w4-secops'
PUBLIC = ROOT / 'docs/e2e/at/secops'
CONTAINER = 'fvpn-w4-secops-restore-postgres'
VOLUME = 'fvpn-w4-secops-restore-data'


def main():
    if not (PRIVATE / 'restore.dump').is_file() or not (PRIVATE / 'retained.json').is_file():
        raise SystemExit('live backup and retained batch required')
    os.umask(0o077)
    log = (PRIVATE / 'restore-private.log').open('wb')
    commands = []

    def run(argv, data=None, timeout=180):
        # Public command paths are placeholders; no credential is in argv.
        commands.append([a.replace(str(ROOT), '<worktree>') for a in argv])
        result = subprocess.run(argv, input=data, cwd=ROOT, stdout=subprocess.PIPE,
                                stderr=log, timeout=timeout, check=False)
        if result.returncode:
            log.write(result.stdout)
            log.flush()
            raise RuntimeError("isolated restore command failed; see private log")
        return result.stdout

    # Refuse pre-existing resources, even under this prefix.
    occupied = run(['docker', 'ps', '-a', '--filter', 'name=^'+CONTAINER+'$', '--format', '{{.Names}}'])
    volume = run(['docker', 'volume', 'ls', '--filter', 'name=^'+VOLUME+'$', '--format', '{{.Name}}'])
    if occupied.strip() or volume.strip():
        raise SystemExit('restore namespace occupied; refusing reuse')
    image = run(['docker', 'inspect', '--format', '{{.Config.Image}}', 'fvpn-w4-secops-postgres']).decode().strip()
    if not image.startswith('fvpn-w4-secops-'):
        raise SystemExit('source image outside assigned namespace')
    binary = PRIVATE / 'restore.test'
    run(['python3', 'tests/e2e/at/secops/diskexec.py', 'go', 'test', '-c', '-tags', 'e2e,integration', '-o', str(binary), './tests/e2e/at/secops'])
    vault_scratch = tempfile.TemporaryDirectory(prefix="restore-vault-", dir=PRIVATE)
    vault = Path(vault_scratch.name) / "vault"
    run(['docker', 'cp', 'fvpn-w4-secops-control:/var/lib/fvpn/vault', str(vault)])
    started = time.monotonic()
    try:
        run(['docker', 'volume', 'create', VOLUME])
        run(['docker', 'run', '-d', '--name', CONTAINER, '--network', 'none',
             '-e', 'POSTGRES_USER=fvpn', '-e', 'POSTGRES_DB=fvpn',
             '-e', 'POSTGRES_PASSWORD_FILE=/fixture/password',
             '-e', 'FVPN_SECOPS_RESTORE=network-none', '-e', 'TMPDIR=/tmp',
             '-v', VOLUME+':/var/lib/postgresql/data',
             '-v', str(STATE / 'secrets/postgres.password')+':/fixture/password:ro',
             '-v', str(vault)+':/fixture/vault',
             '-v', str(PRIVATE / 'retained.json')+':/fixture/retained.json:ro',
             '-v', str(binary)+':/fixture/restore.test:ro',
             '-v', str(PUBLIC)+':/output', image])
        for _ in range(60):
            ready = subprocess.run(['docker', 'exec', CONTAINER, 'pg_isready', '-h', '127.0.0.1', '-U', 'fvpn'],
                                   stdout=log, stderr=log, check=False)
            if ready.returncode == 0:
                break
            time.sleep(1)
        else:
            raise RuntimeError('isolated postgres readiness deadline')
        restore_start = time.monotonic()
        run(['docker', 'exec', '-i', CONTAINER, 'pg_restore', '--exit-on-error', '--no-owner', '-U', 'fvpn', '-d', 'fvpn'],
            (PRIVATE / 'restore.dump').read_bytes())
        restore_seconds = time.monotonic()-restore_start
        output = run(['docker', 'exec', '--user', f'{os.getuid()}:{os.getgid()}', CONTAINER, '/fixture/restore.test', '-test.v', '-test.run', '^TestAT30_IsolatedRestoreReplayer$', '-test.timeout', '2m'])
        log.write(output)
        result = json.loads((PUBLIC / 'restore.json').read_text())
        result.update(restore_seconds=restore_seconds, isolated_subset_seconds=time.monotonic()-started,
                      network='none', published_ports=[], commands=commands,
                      full_service_rto_seconds=None, full_service_rpo_seconds=None)
        (PUBLIC / 'restore.json').write_text(json.dumps(result, indent=2)+'\n')
        print('isolated logical restore: ACK 5 -> 10; five accepted, five duplicate; replay again ten duplicate')
    finally:
        run(['docker', 'rm', '-fv', CONTAINER])
        run(['docker', 'volume', 'rm', VOLUME])
        log.close()
        vault_scratch.cleanup()


if __name__ == '__main__':
    main()
