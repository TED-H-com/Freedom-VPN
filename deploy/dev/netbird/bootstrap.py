#!/usr/bin/env python3
"""TEST ONLY NetBird bootstrap, using public APIs exclusively; no secret output."""
import fcntl
import json
import os
from pathlib import Path
import secrets
import time
import urllib.error
import urllib.request

STATE = Path(__file__).resolve().parent / '.state'
API = f"http://127.0.0.1:{int(os.environ.get('FVPN_NB_API_PORT', '18080'))}/api"


def save(name, value):
    os.umask(0o077)
    temporary = STATE / (name + '.tmp')
    temporary.write_text(value)
    temporary.chmod(0o600)
    temporary.replace(STATE / name)


def call(method, path, data=None, token=None):
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['Authorization'] = 'Token ' + token
    request = urllib.request.Request(API + path, data=None if data is None else json.dumps(data).encode(),
                                     headers=headers, method=method)
    try:
        with urllib.request.urlopen(request, timeout=15) as response:
            body = response.read()
            return json.loads(body) if body else None
    except urllib.error.HTTPError as error:
        # Upstream response bodies may contain secrets. Never print them.
        raise RuntimeError(f'NetBird {method} failed with HTTP {error.code}') from None
    except urllib.error.URLError:
        raise RuntimeError('NetBird unavailable') from None


def bootstrap():
    os.umask(0o077)
    STATE.mkdir(mode=0o700, exist_ok=True)
    for attempt in range(60):
        try:
            status = call('GET', '/instance')
            break
        except RuntimeError:
            if attempt == 59:
                raise
            time.sleep(1)
    if status['setup_required']:
        password = secrets.token_urlsafe(32)
        save('admin-password', password)
        owner = call('POST', '/setup', {'email': 'admin@example.invalid', 'name': 'TEST ONLY Admin',
                    'password': password, 'create_pat': True, 'pat_expire_in': 1})
        save('owner-token', owner['personal_access_token'])
    # A lost setup response never triggers another owner. Recover through the
    # dashboard with the saved password, or reset this disposable dev instance.
    if not (STATE / 'owner-token').exists():
        raise RuntimeError('Owner token unavailable: use saved admin password to issue a replacement PAT, or follow README reset')
    owner_token = (STATE / 'owner-token').read_text().strip()
    # IPv4-only G1 lab (§8.3): v0.82 defaults new accounts to IPv6 for All.
    accounts = call('GET', '/accounts', token=owner_token)
    if len(accounts) != 1:
        raise RuntimeError('Expected exactly one dev account')
    account = accounts[0]
    settings = account['settings']
    desired = {'ipv6_enabled_groups': [], 'auto_update_version': 'disabled',
               'metrics_push_enabled': False, 'jwt_groups_enabled': False,
               'groups_propagation_enabled': False}
    if any(settings.get(k) != v for k, v in desired.items()):
        settings.update(desired)
        call('PUT', '/accounts/' + account['id'], {'settings': settings}, owner_token)
    observed = call('GET', '/accounts', token=owner_token)
    if any(observed[0]['settings'].get(k) != v for k, v in desired.items()):
        raise RuntimeError('IPv4-only account setting not confirmed')
    # Remove the default before creating any non-admin identity or starting peers.
    groups = call('GET', '/groups', token=owner_token)
    all_id = next(g['id'] for g in groups if g['name'] == 'All')
    for policy in call('GET', '/policies', token=owner_token):
        broad = any(all_id in [g['id'] for g in r.get(side, [])]
                    for r in policy['rules'] for side in ('sources', 'destinations'))
        if broad:
            if policy['name'] != 'Default':
                raise RuntimeError('Unexpected broad policy: bootstrap stopped')
            call('DELETE', '/policies/' + policy['id'], token=owner_token)
    if any(p['name'] == 'Default' for p in call('GET', '/policies', token=owner_token)):
        raise RuntimeError('Default policy deletion not confirmed')
    users = call('GET', '/users', token=owner_token)
    service = [u for u in users if u['name'] == 'fvpn-adapter' and u.get('is_service_user')]
    if len(service) > 1:
        raise RuntimeError('Duplicate adapter service identities')
    if not service:
        # CE network_admin cannot create setup keys or delete peers. Admin is the
        # least built-in role covering the adapter's complete operation set.
        service = [call('POST', '/users', {'name': 'fvpn-adapter', 'role': 'admin',
                                         'is_service_user': True, 'auto_groups': []}, owner_token)]
    user = service[0]
    if user['role'] != 'admin' or user['auto_groups']:
        raise RuntimeError('Unexpected adapter service permissions')
    token_path = STATE / 'adapter-token'
    if not token_path.exists():
        # Lost token response: revoke the unusable named token before replacing.
        path = '/users/' + user['id'] + '/tokens'
        for token in call('GET', path, token=owner_token) or []:
            if token['name'] == 'fvpn-adapter-dev':
                call('DELETE', path + '/' + token['id'], token=owner_token)
        pat = call('POST', path, {'name': 'fvpn-adapter-dev', 'expires_in': 30}, owner_token)
        save('adapter-token', pat['plain_token'])
    call('GET', '/groups', token=token_path.read_text().strip())
    for label in ('a', 'b'):
        name = 'TEST ONLY Member ' + label
        match = [u for u in call('GET', '/users', token=owner_token) if u['name'] == name]
        if len(match) > 1:
            raise RuntimeError('Duplicate TEST ONLY member')
        if not match:
            u = call('POST', '/users', {'name': name, 'email': f'member-{label}@example.invalid',
                     'role': 'user', 'is_service_user': False, 'auto_groups': []}, owner_token)
            save('member-' + label + '.json', json.dumps(u))
        elif not (STATE / ('member-' + label + '.json')).exists():
            raise RuntimeError('Member password unavailable: follow README reset; no duplicate user created')
    print('Bootstrap verified: owner, adapter service token, two TEST ONLY users; default policy absent. Secrets withheld.')


if __name__ == '__main__':
    try:
        os.umask(0o077)
        STATE.mkdir(mode=0o700, exist_ok=True)
        with (STATE / 'bootstrap.lock').open('a') as lock:
            fcntl.flock(lock, fcntl.LOCK_EX)
            bootstrap()
    except Exception as error:
        # No traceback or exception payload can accidentally include a response.
        print(str(error) if isinstance(error, RuntimeError) else 'Bootstrap state unavailable; see README recovery')
        raise SystemExit(1)
