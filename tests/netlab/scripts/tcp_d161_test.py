"""D-161: synthetic timelines using follow-up 35's real TCP header shape.

Addresses, absolute ISNs, flag/options syntax and NAT paths come from live3;
intervals are shortened to a three-byte TEST ONLY request. Times/ids are synthetic.
"""
from copy import deepcopy
import unittest

from reload_analysis import classify_d120
import reload_analysis_test as udp_fixture
from tcp_d161 import checked_tcp, tcp_facts, ROUTES


class D161Test(unittest.TestCase):
    def fixture(self, loss_direction='request'):
        blocks = {role: [] for role in ('origin', 'node', 'responder')}
        def packet(direction, at, ident, start, size=0, flags='P.', hops=4, port=39090):
            for index, (role, interface, way) in enumerate(ROUTES[direction][:hops]):
                client = '100.64.1.1' if (direction == 'request' and index < 2 or
                                           direction == 'echo' and index >= 2) else '192.0.2.37'
                source, dest = f'{client}.{port}', '203.0.113.10.9001'
                if direction == 'echo':
                    source, dest = dest, source
                ns = at + index * 50_000
                seq = f'{start}:{start+size}' if size else str(start)
                blocks[role].append(
                    f'{ns//10**9}.{ns%10**9:09d} {interface} {way} IP (tos 0x0, ttl 63, id {ident}, offset 0, flags [DF], proto TCP (6), length {52+size})\n'
                    f'    {source} > {dest}: Flags [{flags}], cksum 0x84ef (correct), seq {seq}, ack 1, win 64, options [nop,nop,TS val {ident} ecr 1], length {size}\n')
        # Real absolute sequence bases, with a complete handshake path.
        packet('request', 900_000_000, 1, 2565188531, flags='S')
        packet('echo', 901_000_000, 2, 672356393, flags='S.')
        if loss_direction == 'request':
            packet('request', 1_015_000_000, 3, 2565188532, 3, hops=2)
            packet('request', 2_700_000_000, 4, 2565188532, 3)
            packet('echo', 2_701_000_000, 5, 672356394, 3)
        else:
            packet('request', 1_014_000_000, 3, 2565188532, 3)
            packet('echo', 1_015_000_000, 4, 672356394, 3, hops=2)
            packet('echo', 2_700_000_000, 5, 672356394, 3)
        events = [dict(event='tcp_send', sequence=1, wall_ns='1013000000'),
                  dict(event='tcp_read', sequence=1, wall_ns='2514000000', outcome='TimeoutError'),
                  # A later read confirms the delayed echo, on the same stream.
                  dict(event='tcp_read', sequence=2, wall_ns='2702000000', size=3,
                       response_sequence=1, echo_status='stale'),
                  dict(event='trace_end', sequence=2, wall_ns='2900000000')]
        return blocks, events, packet

    def facts(self, blocks, events):
        captures, errors = {}, []
        for role, chunks in blocks.items():
            try:
                captures[role] = checked_tcp(''.join(chunks),
                    f'{len(chunks)} packets captured\n0 packets dropped by kernel\n')
            except ValueError as exc:
                errors.append(str(exc))
        return tcp_facts(events, captures, errors)

    def verdict(self, facts, windows=None, udp_losses=0):
        fixture = udp_fixture.D120Test()
        ds = [fixture.datagram(i+1, i % 2 == 0) for i in range(udp_losses * 2)]
        ds = ds or [fixture.datagram(loss=False)]
        return classify_d120(ds, windows or [fixture.window()], ['1:TimeoutError'], tcp_facts=facts)

    def assert_fails(self, facts, reason, windows=None):
        verdict = self.verdict(facts, windows)
        self.assertFalse(verdict['passed'], verdict)
        self.assertTrue(any('(iv)' in e and reason in e for e in verdict['errors']), verdict['errors'])

    def test_all_conditions_hold_both_directions_and_stall_from_captures(self):
        for direction in ROUTES:
            blocks, events, _ = self.fixture(direction)
            facts = self.facts(blocks, events)
            verdict = self.verdict(facts)
            self.assertTrue(verdict['passed'], verdict)
            self.assertEqual(verdict['tcp_reads'], 2)
            self.assertEqual(verdict['tcp_timeouts'], 1)
            self.assertEqual(len(verdict['guard_attributed_timeouts']), 1)
            self.assertEqual(verdict['tcp_guard_losses'][0]['direction'], direction)
            self.assertEqual(verdict['tcp_guard_losses'][0]['generation'], '1001')
            self.assertEqual(verdict['longest_tcp_stall']['stall_ns'],
                             (2_701_150_000 - 1_015_000_000) if direction == 'request' else
                             (2_700_150_000 - 1_014_000_000))

    def test_every_lost_transmission_must_qualify(self):
        for at, reason in [(1_200_000_000, 'outside'), (1_015_000_000, 'unobserved')]:
            blocks, events, packet = self.fixture()
            packet('request', at, 9, 2565188532, 3, hops=2 if reason == 'outside' else 1)
            self.assert_fails(self.facts(blocks, events), reason)

    def test_two_brackets_unverified_and_each_bound(self):
        blocks, events, _ = self.fixture()
        facts = self.facts(blocks, events)
        w = udp_fixture.D120Test().window()
        self.assert_fails(facts, 'ambiguous', [w, dict(w, generation='1002')])
        self.assert_fails(facts, 'not verified', [dict(w, verified=False)])
        for key in ('upper_ns', 'render_to_installed_ns'):
            self.assert_fails(facts, '(ii)', [dict(w, **{key: 100_000_001})])
        # Same inclusive outer edge and 2us margin as position() and UDP.
        for at in (999_998_000, 1_030_002_000):
            f = deepcopy(facts)
            f['requests'][0]['lost_transmissions'][0]['node_entry_ns'] = at
            self.assertTrue(self.verdict(f)['passed'])

    def test_bytes_never_delivered_or_echo_not_confirmed(self):
        blocks, events, _ = self.fixture()
        for role in blocks:
            blocks[role] = [b for b in blocks[role] if 'id 4,' not in b and 'id 5,' not in b]
        self.assert_fails(self.facts(blocks, events), 'not delivered')
        blocks, events, _ = self.fixture()
        events[2]['response_sequence'] = 2
        self.assert_fails(self.facts(blocks, events), 'not delivered')

    def test_old_distinct_socket_does_not_relabel_measured_connection(self):
        # Full-lab f35 gate2 contains retransmissions and RST/FIN from sockets
        # opened before capture. Their distinct ports are not the stream.
        blocks, events, packet = self.fixture()
        packet('request', 1_400_000_000, 12, 123456, 3, port=51346)
        packet('echo', 1_401_000_000, 13, 654321, flags='F.', port=51346)
        packet('request', 1_402_000_000, 14, 123459, flags='R', port=51346)
        facts = self.facts(blocks, events)
        self.assertTrue(self.verdict(facts)['passed'])
        self.assertEqual(facts['connection']['client_port'], 39090)
        self.assertEqual(facts['ignored_other_connection_packets'],
                         {'origin': 3, 'node': 6, 'responder': 3})

    def test_reset_fin_second_connection_fail_even_without_timeout(self):
        for flags, port, reason in [('R.', 39090, 'reset'), ('F.', 39090, 'FIN'),
                                     ('S', 39091, 'second connection'), ('S', 39090, 'second connection')]:
            blocks, events, packet = self.fixture()
            packet('request', 2_800_000_000, 7, 12345, flags=flags, port=port)
            facts = self.facts(blocks, events)
            self.assert_fails(facts, reason)
            d = udp_fixture.D120Test().datagram(loss=False)
            self.assertFalse(classify_d120([d], [udp_fixture.D120Test().window()], tcp_facts=facts)['passed'])

    def test_missing_incomplete_unparsed_and_kernel_drop(self):
        blocks, events, _ = self.fixture()
        for reason in ('missing TCP capture', 'incomplete combined capture', 'unparsed TCP header'):
            self.assert_fails(tcp_facts(events, {}, [reason]), reason)
        text = ''.join(blocks['origin'])
        n = len(blocks['origin'])
        for badtext, stats in [(text, f'{n+1} packets captured\n0 packets dropped by kernel'),
                               (text, f'{n} packets captured\n1 packets dropped by kernel'),
                               (text.replace('seq 2565188532:2565188535', 'seq bad'),
                                f'{n} packets captured\n0 packets dropped by kernel')]:
            with self.assertRaises(ValueError):
                checked_tcp(badtext, stats)
        blocks['origin'][2] = blocks['origin'][2].replace('seq 2565188532:2565188535', 'seq bad')
        self.assert_fails(self.facts(blocks, events), 'unparsed')

    def test_eof_and_no_loss_timeout_fail(self):
        blocks, events, _ = self.fixture()
        facts = self.facts(blocks, events)
        verdict = classify_d120([udp_fixture.D120Test().datagram(loss=False)],
                                [udp_fixture.D120Test().window()], ['1:tcp-eof'], tcp_facts=facts)
        self.assertFalse(verdict['passed'])
        self.assertIn('tcp-eof', verdict['errors'][0])
        facts['requests'][0]['lost_transmissions'] = []
        self.assert_fails(facts, 'no captured lost transmission')

    def test_loss_after_node_is_not_guard_loss(self):
        blocks, events, _ = self.fixture()
        # An egress observation with no far-side arrival fails condition 1.
        facts = self.facts(blocks, events)
        facts['requests'][0]['lost_transmissions'][0]['node_exit_ns'] = 1_015_100_000
        self.assert_fails(facts, 'not dropped at node')

    def test_four_udp_plus_one_pass_five_plus_one_fail(self):
        blocks, events, _ = self.fixture()
        facts = self.facts(blocks, events)
        self.assertTrue(self.verdict(facts, udp_losses=4)['passed'])
        verdict = self.verdict(facts, udp_losses=5)
        self.assertFalse(verdict['passed'])
        self.assertEqual(verdict['total_losses'], 6)
        self.assertTrue(any('(v)' in e for e in verdict['errors']))

    def test_missing_hop_or_ambiguous_identity_fails_closed(self):
        blocks, events, _ = self.fixture()
        blocks['node'] = [b for b in blocks['node'] if not ('id 4,' in b and ' Out ' in b)]
        self.assert_fails(self.facts(blocks, events), 'missing intermediate')
        blocks, events, _ = self.fixture()
        blocks['origin'].append(blocks['origin'][2])
        self.assert_fails(self.facts(blocks, events), 'ambiguous')


if __name__ == '__main__':
    unittest.main()
