"""D-146 discriminator tests: retained successful bytes alone cannot qualify."""
import copy
import json
from pathlib import Path
import unittest

import d146_evidence as d146

ROOT = Path(__file__).resolve().parents[4]
BASE = ROOT/'docs/e2e/at/members/history/follow-up-17-captures/81b638e8-be59-4996-9594-14dd402ebb86'


def pair(direction, device):
    meta = lambda k, v: {'match': {'op': '==', 'left': {'meta': {'key': k}}, 'right': v}}
    common = [meta('iifname', 'fvmb0' if direction == 'upload' else 'fvup0'), meta('oifname', 'fvup0' if direction == 'upload' else 'fvmb0'),
              {'match': {'op': '==', 'left': {'payload': {'protocol': 'ip', 'field': 'saddr' if direction == 'upload' else 'daddr'}}, 'right': '100.64.1.1'}}]
    key = {'tcp option': {'name': 'maxseg', 'field': 'size'}}
    expr = common + [
        {'match': {'op': '==', 'left': {'&': [{'payload': {'protocol': 'tcp', 'field': 'flags'}}, 'syn']}, 'right': 'syn'}},
        {'match': {'op': '>', 'left': key, 'right': 1160}}, {'mangle': {'key': key, 'value': 1160}}]
    accept = common + [{'counter': ('u_' if direction == 'upload' else 'd_') + device.replace('-', '')},
                       {'mangle': {'key': {'meta': {'key': 'priority'}}, 'value': '1:2f'}}, {'accept': None}]
    return dict(clamp_index=1, accept_index=2, clamp=dict(chain='forward', expr=expr), accept=dict(chain='forward', expr=accept))


def fixture():
    capture = json.loads((BASE/'matrix1280-original-capture.json').read_text())
    doc = json.loads((BASE/'matrix1280.json').read_text())
    notes = {n['name']: json.loads(n['detail']) for n in doc['observations']}
    v = notes['transfer']
    device = '019a0000-0000-7000-8000-0000000000a1'
    data = dict(capture=capture, run_id='TEST ONLY', source_commit='TEST ONLY', row='TEST ONLY historical headers, synthetic rules and MSS',
                selected_before=v['selected_exit_before'], selected_after=v['selected_exit_after'], device_id=device,
                rules=dict(available=True, serving=True, ended_at='2026-11-03T08:52:50Z', pairs={d: pair(d, device) for d in ('upload', 'download')}),
                input_applied_at='2026-11-03T08:52:48Z', bulk_started_at='2026-11-03T08:52:55Z', bulk_ended_at='2026-11-03T08:52:56Z',
                transfer=v['bulk'], original_byte_predicate=True, upload_predicate=True)
    decoded = dict(zip(('member', 'node', 'origin'), notes['decode']))
    packets = {r: [json.loads(line) for line in (BASE/'headers/matrix1280'/(r+'.headers.jsonl')).read_text().splitlines()] for r in decoded}
    return data, packets, decoded


def emulate_clamp(packets):
    # Synthetic mutation only, in memory. Historical records stay unchanged.
    for role in ('member', 'origin'):
        for p in packets[role]:
            if 'tcp' in p and p['tcp']['flags'] & 2 and p['packet_type'] != 4:
                for o in p['tcp']['options']:
                    if 'mss' in o:
                        o['mss'] = min(o['mss'], 1160)


class D146Tests(unittest.TestCase):
    def test_original_successful_bytes_stay_unqualified(self):
        data, packets, decoded = fixture()
        result = d146.qualify(data, packets, decoded)
        self.assertFalse(result['qualified'])
        self.assertIn('both_advertisements_clamped', result['reasons'])
        self.assertTrue(result['checks']['both_byte_predicates'])

    def test_synthetic_complete_evidence_then_independent_failures(self):
        data, packets, decoded = fixture()
        emulate_clamp(packets)
        result = d146.qualify(data, packets, decoded)
        self.assertTrue(result['qualified'], result['reasons'])
        mutations = {
            'new_handshake': lambda d, p, c: d['rules'].update(ended_at='2026-11-03T08:52:56Z'),
            'upload': lambda d, p, c: d.update(upload_predicate=False),
            'original_bytes': lambda d, p, c: d.update(original_byte_predicate=False),
            'serving': lambda d, p, c: d['rules'].update(serving=False),
            'clamp_placement': lambda d, p, c: d['rules']['pairs']['download'].update(accept_index=3),
            'capture_drop': lambda d, p, c: d['capture']['points']['origin'].update(kernel_drops=1),
            'family': lambda d, p, c: d['selected_after'].update(chosen_family='ipv6'),
            'ptb': lambda d, p, c: p['node'][0].update(protocol=58, icmp={'type': 2, 'code': 0}),
        }
        for name, mutate in mutations.items():
            with self.subTest(name=name):
                d, p, c = copy.deepcopy((data, packets, decoded))
                mutate(d, p, c)
                self.assertFalse(d146.qualify(d, p, c)['qualified'])
        # Keep record counts valid while a selected reverse datagram is missing.
        d, p, c = copy.deepcopy((data, packets, decoded))
        lost = result['reverse']['receiver_packets'][0]['packet']
        p['member'] = [row for row in p['member'] if row['packet'] != lost]
        c['member']['records'] -= 1
        d['capture']['points']['member']['captured'] -= 1
        changed = d146.qualify(d, p, c)
        self.assertTrue(changed['checks']['complete_captures'])
        self.assertFalse(changed['qualified'])
        self.assertTrue(changed['reverse']['absent_sender_packets'])
        # A no-op clamp (already-small advertised MSS) cannot qualify.
        for p in packets['member']:
            if p.get('tcp', {}).get('flags', 0) & 2:
                for o in p['tcp']['options']:
                    if 'mss' in o:
                        o['mss'] = 1000
        self.assertFalse(d146.qualify(data, packets, decoded)['qualified'])


if __name__ == '__main__':
    unittest.main()
