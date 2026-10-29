#!/usr/bin/env python3
"""Generate TEST ONLY configuration; never print secrets. Standard library only."""
import base64
import json
import os
from pathlib import Path
import secrets

os.umask(0o077)
state = Path(__file__).resolve().parent / '.state'
state.mkdir(mode=0o700, exist_ok=True)
port = int(os.environ.get('FVPN_NB_API_PORT', '18080'))
config = state / 'config.yaml'
if not config.exists():
    # JSON is valid YAML. All stores are owned by NetBird; we never open them.
    config.write_text(json.dumps({'server': {
        'listenAddress': ':8080',
        'exposedAddress': f'http://netbird.example.invalid:{port}',
        'stunPorts': [3478], 'metricsPort': 9090,
        'healthcheckAddress': ':9000', 'logLevel': 'warn', 'logFile': 'console',
        'authSecret': secrets.token_hex(32), 'dataDir': '/var/lib/netbird',
        'disableAnonymousMetrics': True, 'disableGeoliteUpdate': True,
        'auth': {'issuer': f'http://netbird.example.invalid:{port}/oauth2',
                 'dashboardRedirectURIs': [f'http://netbird.example.invalid:{port}/nb-auth',
                                          f'http://netbird.example.invalid:{port}/nb-silent-auth'],
                 'cliRedirectURIs': ['http://localhost:53000/']},
        'store': {'engine': 'sqlite', 'encryptionKey': base64.b64encode(secrets.token_bytes(32)).decode()}
    }}, indent=2) + '\n')
    config.chmod(0o600)
print('TEST ONLY configuration ready in deploy/dev/netbird/.state (secrets withheld).')
