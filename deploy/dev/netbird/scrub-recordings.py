#!/usr/bin/env python3
"""Promote only allowlisted, scrubbed v0.82.0 API shapes to TEST ONLY fixtures."""
import json
from pathlib import Path

root = Path(__file__).resolve().parents[3]
source = root / 'deploy/dev/netbird/.state/recordings'
target = root / 'internal/netbird/testdata'
ids = {}
addresses = {}


def scrub(value, field=''):
    if isinstance(value, dict):
        return {k: scrub(v, k) for k, v in value.items() if k not in ('key', 'password', 'plain_token')}
    if isinstance(value, list):
        return [scrub(v, field) for v in value]
    if isinstance(value, str) and value:
        if field in ('id', 'user_id', 'peer', 'groups', 'peer_groups', 'access_control_groups', 'source_posture_checks', 'network_id'):
            if value not in ids:
                ids[value] = 'TEST-ID-' + str(len(ids) + 1)
            return ids[value]
        if field in ('last_seen', 'last_login', 'created_at', 'updated_at', 'expires'):
            return '2026-01-01T00:00:00Z'
        if field in ('name', 'description'):
            return 'TEST ONLY ' + field
        if field in ('hostname', 'dns_label'):
            return 'test-peer.example.invalid' if field == 'hostname' else 'test-peer'
        if field in ('ip', 'ipv6', 'connection_ip'):
            if value not in addresses:
                addresses[value] = len(addresses) + 11
            suffix = addresses[value]
            return {'ip': f'100.64.0.{suffix}', 'ipv6': f'2001:db8::{suffix}', 'connection_ip': f'192.0.2.{suffix}'}[field]
        if field in ('mac', 'mac_address'):
            return '02:00:00:00:00:01'
        if field in ('serial_number', 'os', 'kernel_version', 'city_name', 'country_code'):
            return 'TEST ONLY'
        if field in ('net', 'network_address', 'net_ip'):
            return '192.0.2.0/24'
    return value


for name in ('group', 'route', 'peer', 'policy', 'nameserver'):
    if name == 'policy':
        policies = json.loads((source / 'policies.json').read_text())
        value = next(p for p in policies if p['rules'][0]['protocol'] == 'icmp')
    else:
        value = json.loads((source / (name + '.json')).read_text())
    value = scrub(value)
    value['_fixture'] = 'TEST ONLY - NOT A SECRET; recorded NetBird CE 0.82.0, identifiers/addresses/metadata replaced'
    (target / (name + '.json')).write_text(json.dumps(value, indent=2, sort_keys=True) + '\n')
print('Scrubbed five response fixtures. Review every field before committing.')
