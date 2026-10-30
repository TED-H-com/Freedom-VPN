#!/usr/bin/env python3
"""AT30: invoke the owning backup tools, verify an isolated PITR, then replay.

Requires a current successful TestAT30_LivePITRSource on our running lab.
Never creates a parallel backup implementation or restores the live database.
"""
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[4]
PRIVATE = ROOT / '.agent/secops'
PUBLIC = ROOT / 'docs/e2e/at/secops'
STATE = ROOT / '.agent/e2e-state-w4-secops'
SOURCE = 'fvpn-w4-secops-postgres'
REPO = 'fvpn-w4-secops-pgbackrest-repo'
COPY = 'fvpn-w4-secops-restore-pitr'
VOLUME = COPY + '-pg'


def main():
    os.umask(0o077)
    source = json.loads((PUBLIC / 'pitr-source.json').read_text())
    if source.get('go_test_failed') or not source.get('physical_source'):
        raise SystemExit('successful current physical source run required')
    manifest = json.loads((PRIVATE / 'pitr-manifest.json').read_text())
    target = dt.datetime.fromisoformat(manifest['Target']).astimezone(dt.timezone.utc).isoformat().replace('+00:00', 'Z')
    commands = []
    with (PRIVATE / 'pitr-restore-private.log').open('wb') as log:
        def run(argv, data=None, expected=0, timeout=180):
            commands.append([str(a).replace(str(ROOT), '<worktree>') for a in argv])
            result = subprocess.run(argv, input=data, cwd=ROOT, stdout=subprocess.PIPE,
                                    stderr=log, timeout=timeout, check=False)
            if result.returncode != expected:
                log.write(result.stdout)
                log.flush()
                raise RuntimeError('PITR command failed; protected diagnostic retained')
            return result.stdout

        for obj in (COPY, COPY+'-init'):
            if run(['docker', 'ps', '-a', '--filter', 'name=^/'+obj+'$', '--format', '{{.Names}}']).strip():
                raise SystemExit('restore container occupied; refusing reuse')
        if run(['docker', 'volume', 'ls', '--filter', 'name=^'+VOLUME+'$', '--format', '{{.Name}}']).strip():
            raise SystemExit('restore volume occupied; refusing reuse')
        # The shared lab deliberately uses neutral immutable image caches.
        # Reference the running source's image ID; create only our own objects.
        image = run(['docker', 'inspect', '--format', '{{.Image}}', SOURCE]).decode().strip()
        binary = PRIVATE / 'restore.test'
        run(['python3', 'tests/e2e/at/secops/diskexec.py', 'go', 'test', '-c', '-tags', 'e2e,integration', '-o', str(binary), './tests/e2e/at/secops'])
        argv = ['deploy/backup/restore-pitr.sh', '--image', image, '--repo-volume', REPO,
                '--target-volume', VOLUME, '--primary-container', SOURCE,
                '--db-user', 'fvpn', '--db-name', 'fvpn', '--target-time', target]
        # A real negative guard: fail before touching the live primary.
        run(argv + ['--isolated-container', SOURCE], expected=2)
        started = time.monotonic()
        try:
            run(argv + ['--isolated-container', COPY])
            restore_seconds = time.monotonic() - started
            inspect = json.loads(run(['docker', 'inspect', COPY]))[0]
            mounts = inspect['Mounts']
            if (inspect['HostConfig']['NetworkMode'] != 'none' or inspect['NetworkSettings']['Ports']
                    or not any(m['Name'] == REPO and not m['RW'] for m in mounts)
                    or not any(m['Name'] == VOLUME and m['Destination'] == '/var/lib/postgresql/data' for m in mounts)):
                raise RuntimeError('restore isolation invariant failed')
            checklist = run(['docker', 'exec', '-i', '--user', 'postgres', COPY, 'psql', '-U', 'fvpn', '-d', 'fvpn',
                             '-v', 'ON_ERROR_STOP=1', '-v', 'restore_point='+target],
                            (ROOT / 'deploy/backup/post-restore-checklist.sql').read_bytes())
            log.write(checklist)
            sections = sum(('['+str(n)+'/6]').encode() in checklist for n in range(1, 7))
            if sections != 6:
                raise RuntimeError('post-restore checklist incomplete')
            # The vault is current external state, copied independently of PITR.
            # All copied secrets and retained bodies stay in private disk storage.
            with tempfile.TemporaryDirectory(prefix='pitr-vault-', dir=PRIVATE) as scratch:
                vault = Path(scratch) / 'vault'
                run(['docker', 'cp', 'fvpn-w4-secops-control:/var/lib/fvpn/vault', str(vault)])
                run(['docker', 'exec', COPY, 'mkdir', '-p', '/fixture', '/output'])
                for path, name in ((vault, 'vault'), (STATE/'secrets/postgres.password', 'password'),
                                   (PRIVATE/'retained.json', 'retained.json'),
                                   (PRIVATE/'pitr-manifest.json', 'pitr-manifest.json'), (binary, 'restore.test')):
                    run(['docker', 'cp', str(path), COPY+':/fixture/'+name])
                output = run(['docker', 'exec', '-e', 'FVPN_SECOPS_RESTORE=network-none', COPY,
                              '/fixture/restore.test', '-test.v', '-test.run', '^TestAT30_IsolatedRestoreReplayer$', '-test.timeout', '2m'])
                log.write(output)
                result = json.loads(run(['docker', 'exec', COPY, 'cat', '/output/restore.json']))
            result.update(source=source['physical_source'], network='none', published_ports=[],
                          repository_read_only=True, primary_overwrite_refused=True,
                          restore_seconds=restore_seconds, isolated_subset_seconds=time.monotonic()-started,
                          db_rpo_seconds=source['physical_source']['db_rpo_seconds'],
                          full_service_rpo_seconds=None, full_service_rto_seconds=None,
                          checklist_sections=sections, checklist_sha256=hashlib.sha256(checklist).hexdigest(),
                          restore_image_digest=image, verifier_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),
                          policy_revision=source.get('policy_revision'), node_incarnation=source.get('node_incarnation'),
                          image_digests=source.get('image_digests'), source_revision=source.get('source_revision'),
                          expected='ACK 7; earlier grant/audit retained; post-target writes absent; replay 3 accepted/7 duplicates; repeat 10 duplicates',
                          actual='all physical boundary and replay assertions passed', commands=commands,
                          timing_scope='dev-host isolated copy promotion/checklist/service replay; not service RTO or hardware capacity')
            (PUBLIC/'pitr.json').write_text(json.dumps(result, indent=2)+'\n')
            print('physical PITR: ACK 7 -> 10; 3 accepted, 7 duplicates; repeat 10 duplicates; grant/audit boundary verified')
        finally:
            # These names were absent before this invocation. Never touch other prefixes.
            subprocess.run(['docker', 'rm', '-fv', COPY, COPY+'-init'], stdout=log, stderr=log, check=False)
            subprocess.run(['docker', 'volume', 'rm', VOLUME], stdout=log, stderr=log, check=False)


if __name__ == '__main__':
    main()
