#!/usr/bin/env python3
"""TEST ONLY real NetBird fixture; runs only inside the isolated netlab container.

All services and peer traffic stay inside its namespaces. Secret-bearing API
responses and daemon logs stay in disk scratch and are never evidence output.
"""
import base64
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import secrets
import subprocess
import sys
import time
import urllib.request

ROOT = Path('/scratch/real-netbird')
API = 'http://192.0.2.1:8080'


def run(*args, ns=None):
    if ns:
        args = ('ip', 'netns', 'exec', ns, *args)
    return subprocess.run(args, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE).stdout


def spawn(label, *args, ns=None, env=None):
    if ns:
        args = ('ip', 'netns', 'exec', ns, *args)
    log = (ROOT / (label + '.log')).open('wb')
    proc = subprocess.Popen(args, stdout=log, stderr=log, env=env)
    (ROOT / (label + '.pid')).write_text(str(proc.pid))
    return proc


def call(method, path, data=None, token=None):
    headers = {'Content-Type': 'application/json'}
    if token:
        headers['Authorization'] = 'Token ' + token
    req = urllib.request.Request(API + '/api' + path, headers=headers, method=method,
                                 data=None if data is None else json.dumps(data).encode())
    with urllib.request.urlopen(req, timeout=5) as response:
        body = response.read()
        return json.loads(body) if body else None


def wait(fn, timeout=60):
    end = time.monotonic() + timeout
    while time.monotonic() < end:
        try:
            value = fn()
            if value:
                return value
        except Exception:
            pass
        time.sleep(.2)
    raise RuntimeError('fixture readiness timed out; private logs remain in scratch')


def cli(label, *args):
    return run('netbird', '--daemon-addr', 'unix://' + str(ROOT / (label + '.sock')),
               '--log-level', 'error', *args, ns='fv-member' if label == 'member' else None)


def setup():
    if not Path('/.dockerenv').exists():
        raise RuntimeError('must run inside netlab container')
    os.umask(0o077)
    ROOT.mkdir(parents=True, exist_ok=True)
    run('ip', 'netns', 'add', 'fv-member')
    run('ip', 'link', 'add', 'fv-nbhost', 'type', 'veth', 'peer', 'name', 'fv-nbpeer')
    run('ip', 'link', 'set', 'fv-nbpeer', 'netns', 'fv-member')
    for ns, iface, addr in [(None, 'fv-nbhost', '192.0.2.1/30'), ('fv-member', 'fv-nbpeer', '192.0.2.2/30')]:
        run('ip', 'addr', 'add', addr, 'dev', iface, ns=ns)
        run('ip', 'link', 'set', iface, 'up', ns=ns)
        run('ip', 'link', 'set', 'lo', 'up', ns=ns)
    run('ip', 'netns', 'add', 'fv-inet')
    run('ip', 'link', 'add', 'fv-ih', 'type', 'veth', 'peer', 'name', 'fv-ia')
    run('ip', 'link', 'set', 'fv-ia', 'netns', 'fv-inet')
    for ns, iface, addr in [(None, 'fv-ih', '192.0.2.37/30'), ('fv-inet', 'fv-ia', '192.0.2.38/30')]:
        run('ip', 'addr', 'add', addr, 'dev', iface, ns=ns)
        run('ip', 'link', 'set', iface, 'up', ns=ns)
        run('ip', 'link', 'set', 'lo', 'up', ns=ns)
    run('sysctl', '-qw', 'net.ipv4.ip_forward=1')
    cfg = {'server': {'listenAddress': ':8080', 'exposedAddress': API,
           'stunPorts': [3478], 'metricsPort': 9090, 'healthcheckAddress': ':9000',
           'logLevel': 'warn', 'logFile': 'console', 'authSecret': secrets.token_hex(32),
           'dataDir': str(ROOT / 'server'), 'disableAnonymousMetrics': True,
           'disableGeoliteUpdate': True, 'auth': {'issuer': API + '/oauth2',
           'dashboardRedirectURIs': [API + '/nb-auth'], 'cliRedirectURIs': ['http://localhost:53000/']},
           'store': {'engine': 'sqlite', 'encryptionKey': base64.b64encode(secrets.token_bytes(32)).decode()}}}
    config = ROOT / 'server.json'
    config.write_text(json.dumps(cfg))
    (ROOT / 'server').mkdir()
    env = dict(os.environ, NB_SETUP_PAT_ENABLED='true', NB_DISABLE_GEOLOCATION='true')
    spawn('server', 'netbird-server', '--config', str(config), env=env)
    wait(lambda: call('GET', '/instance'))
    owner = call('POST', '/setup', {'email': 'admin@example.invalid', 'name': 'TEST ONLY',
                  'password': secrets.token_urlsafe(32), 'create_pat': True, 'pat_expire_in': 1})
    token = owner['personal_access_token']
    account = call('GET', '/accounts', token=token)[0]
    settings = account['settings']
    settings.update(ipv6_enabled_groups=[], auto_update_version='disabled', metrics_push_enabled=False)
    call('PUT', '/accounts/' + account['id'], {'settings': settings}, token)
    key = call('POST', '/setup-keys', {'name': 'TEST ONLY', 'type': 'reusable',
               'expires_in': 3600, 'auto_groups': [], 'usage_limit': 2}, token)['key']
    (ROOT / 'setup-key').write_text(key)
    for label in ['node', 'member']:
        config = ROOT / (label + '.json')
        config.write_text(json.dumps({'ManagementURL': {'Scheme': 'http', 'Host': '192.0.2.1:8080'},
            'AdminURL': {'Scheme': 'http', 'Host': '192.0.2.1:8080'}, 'PrivateKey': '',
            'WgIface': 'fvnb0', 'DisableDNS': True, 'DisableIPv6': True}))
        spawn(label, 'netbird', 'service', 'run', '--config', str(config), '--daemon-addr',
              'unix://' + str(ROOT / (label + '.sock')), '--log-level', 'warn',
              ns='fv-member' if label == 'member' else None)
        wait(lambda: (ROOT / (label + '.sock')).exists())
        cli(label, 'up', '--management-url', API, '--setup-key-file', str(ROOT / 'setup-key'),
            '--interface-name', 'fvnb0', '--disable-dns', '--disable-ipv6', '--disable-firewall',
            '--hostname', 'fv-' + label, '--network-monitor=false')
    peers = wait(lambda: (p if len(p := call('GET', '/peers', token=token)) == 2 else None))
    addresses = {p['name']: p['ip'] for p in peers}
    (ROOT / 'peers.json').write_text(json.dumps(addresses))
    wait(lambda: run('ping', '-c', '1', '-W', '1', addresses['fv-node'], ns='fv-member'))
    wait(lambda: json.loads(cli('node', 'status', '--json'))['peers']['connected'] == 1)
    if not json.loads(cli('node', 'status', '--json'))['usesKernelInterface']:
        raise RuntimeError('real NetBird client is not using kernel WireGuard')
    print('measured_at=' + datetime.now(timezone.utc).isoformat())
    print(run('netbird', 'version').decode().strip())
    print(run('uname', '-srm').decode().strip())
    print('real NetBird fixture ready: kernel peer pair, IPv4, network-monitor=false, firewall disabled', flush=True)


if __name__ == '__main__':
    try:
        setup()
    except Exception as exc:
        print('real NetBird fixture failed: ' + type(exc).__name__, file=sys.stderr)
        sys.exit(1)
