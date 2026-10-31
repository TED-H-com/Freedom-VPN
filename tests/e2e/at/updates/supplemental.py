#!/usr/bin/env python3
"""Same-run transport readback and retired-fixture uncached seed, before teardown."""
from datetime import datetime, timezone
import hashlib
import ipaddress
import json
from pathlib import Path
import subprocess
import sys
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[4]
EVIDENCE = ROOT / 'docs/e2e/at/updates'
CONTAINER = 'fvpn-w4-updates-installer-node'
CONFIG = '/etc/vpn-node/updater.json'
UPDATER = '/opt/vpn-node/current/fvpn-updater'
STAGING = '/var/lib/vpn-node/update-staging/'


def write(name, value):
    (EVIDENCE / name).write_text(json.dumps(value, indent=2) + '\n')


def command(*args):
    p = subprocess.run(['docker', 'exec', CONTAINER, *args],
                       capture_output=True, text=True, timeout=180, check=False)
    return dict(command=list(args), exit=p.returncode,
                stdout=p.stdout.strip(), stderr=p.stderr.strip())


def read(*args):
    out = command(*args)
    if out['exit']:
        raise RuntimeError('owned fixture readback failed: ' + args[0])
    return out['stdout']


def instant(value):
    return datetime.fromisoformat(value.replace('Z', '+00:00')).timestamp()


def transport(rec, obs):
    p = subprocess.run(['docker', 'logs', 'fvpn-w4-updates-control'],
                       capture_output=True, text=True, timeout=30, check=True)
    rows = []
    for line in (p.stdout + '\n' + p.stderr).splitlines():
        try:
            row = json.loads(line)
        except ValueError:
            continue
        if (row.get('route') == 'POST /api/v1/agent/update-results'
                and instant(row['time']) <= instant(rec['at'])):
            rows.append({k: row[k] for k in ('time', 'msg', 'request_id', 'route', 'status')})
    rows.sort(key=lambda row: row['time'])
    names = ('success_submission', 'keyed_exact_replay', 'new_key_exact_replay',
             'modified_replay', 'wrong_job_id', 'failure_submission',
             'rollback_correction', 'first_receipt_after_correction',
             'correction_exact_replay', 'correction_new_key_replay', 'third_outcome_conflict')
    manual = {name: obs[name]['status'] for name in names}
    flush = obs['generated_repeat_flush_route_counts']
    automatic = len(rows) - len(manual)
    counts = {status: sum(row['status'] == status for row in rows) for status in (202, 409)}
    ok = (automatic == 2 and counts == {202: 10, 409: 3}
          and flush == {'before': 1, 'after': 1}
          and obs['generated_delivery_counts'] == {'history': 1, 'outbox': 0})
    write('installer-delivery-transport.json', dict(
        run_id=rec['run_id'], source_commit=rec['git'], at=rec['at'],
        scope='Route logs through installer completion, before regular enrollment; only installer result producer',
        logs=rows, manual_contract_stimuli=manual, automatic_result_posts=automatic,
        automatic_success_posts=flush['before'], automatic_failure_posts=automatic-flush['before'],
        local_counts_before_manual_replays=obs['generated_delivery_counts'],
        success_ack=obs['generated_local_ack'], repeat_flush_route_counts=flush,
        no_repeat_flush_post=flush == {'before': 1, 'after': 1}, passed=ok,
        capture='docker logs fvpn-w4-updates-control; whitelist time/msg/request_id/route/status; cutoff is installer evidence timestamp',
        explanation='11 explicit curl stimuli plus one automatic success and one automatic stopped-producer failure; repeated product flush adds no POST.'))
    return ok


def origin(rec, obs):
    if obs['fixture_retired']['lifecycle_state'] != 'retired':
        raise RuntimeError('refusing uncached probe on an active installer fixture')
    current = read('readlink', '/opt/vpn-node/current')
    release = current.rsplit('/', 1)[1]
    manifest = STAGING + release + '.manifest.json'
    envelope = json.loads(read('cat', manifest))
    artifact = next(a for a in envelope['signed']['artifacts'] if a['architecture'] == 'amd64')
    cache = STAGING + artifact['sha256'] + '.tar.zst'
    held = cache + '.f12-origin-held'
    config_hash = read('sha256sum', CONFIG)
    cache_hash = read('sha256sum', cache)
    before = json.loads(read(UPDATER, 'status', '--config', CONFIG))
    hostname = urlsplit(artifact['url']).hostname
    address = read('getent', 'ahostsv4', hostname).split()[0]
    ip = ipaddress.ip_address(address)
    private = any(ip in ipaddress.ip_network(cidr) for cidr in
                  ('10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16'))
    cgnat = ip in ipaddress.ip_network('100.64.0.0/10')
    if command('test', '!', '-e', held)['exit']:
        raise RuntimeError('held archive already exists')
    if command('mv', cache, held)['exit']:
        raise RuntimeError('could not withhold cache')
    try:
        seed = command(UPDATER, 'seed', '--config', CONFIG, '--manifest', manifest, '--tree', current)
    finally:
        if command('mv', held, cache)['exit']:
            raise RuntimeError('could not restore archive')
    after = json.loads(read(UPDATER, 'status', '--config', CONFIG))
    restored = read('sha256sum', cache) == cache_hash
    unchanged_config = read('sha256sum', CONFIG) == config_hash
    unchanged_current = read('readlink', '/opt/vpn-node/current') == current
    changed = sorted(k for k in before.keys() | after.keys() if before.get(k) != after.get(k))
    ok = (seed['exit'] == 0 and restored and unchanged_config and unchanged_current
          and changed == ['seeded_baseline'] and not private and not cgnat
          and after['seeded_baseline']['release_id'] == release)
    write('artifact-origin.json', dict(
        run_id=rec['run_id'], source_commit=rec['git'], at=datetime.now(timezone.utc).isoformat(),
        scope='TEST ONLY post-handoff uncached seed of retired installer fixture; no apply or central acceptance claim',
        artifact_url=artifact['url'], config=obs['generated_config'],
        cache_sha256_before=cache_hash.split()[0], current_before=current,
        status_before=before, seed_without_cache=seed, cache_restored_identical=restored,
        generated_config_unchanged=unchanged_config, status_after=after,
        status_unchanged=before == after, current_unchanged=unchanged_current,
        changed_status_fields=changed, status_except_seed_unchanged=changed == ['seeded_baseline'],
        passed=ok, resolved_origin=dict(hostname=hostname, is_rfc1918=private, is_cgnat=cgnat,
            classification='controlled Docker origin; address omitted',
            address_sha256=hashlib.sha256(address.encode()).hexdigest()),
        decision='No rejection and no new origin input: allowed_artifact_origins grants private/CGNAT exceptions, not a universal host restriction. Cache restored; only verified seeded_baseline added.',
        stimulus=dict(cache_path=cache, held_path=held,
            steps=['Read retained signed manifest for current and record status/config/cache hashes.',
                   'Move only the retained archive to a held sibling; run recorded seed with unchanged generated config.',
                   'Restore the archive in Python finally; compare hashes, pointer and transaction state.'],
            cleanup='Exact archive restored; runner subsequently tears down owned fixture.')))
    return ok


def main():
    rec = json.loads((EVIDENCE / 'installer-handoff.json').read_text())
    if instant(rec['at']) < float(sys.argv[1]):
        raise RuntimeError('refusing stale installer evidence')
    obs = {o['name']: json.loads(o['detail']) for o in rec['observations']}
    if rec['status'] != 'pass':
        for name in ('installer-delivery-transport.json', 'artifact-origin.json'):
            write(name, dict(run_id=rec['run_id'], source_commit=rec['git'],
                             status='blocked', reason='Shared-wrapper handoff incomplete; no fallback attempted'))
        return 1
    transport_ok = transport(rec, obs)
    origin_ok = origin(rec, obs)
    print(f'installer supplements: transport={transport_ok}, uncached_origin={origin_ok}')
    return 0 if transport_ok and origin_ok else 1


if __name__ == '__main__':
    sys.exit(main())
