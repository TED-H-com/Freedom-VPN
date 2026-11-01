"""TEST ONLY checks for loss accounting and content-free stream decoding."""
import struct
import unittest

from analyze_followup27 import interval, sequence_ranges, handshake_groups
from decode_capture import decode_ip


class Followup27Analysis(unittest.TestCase):
    def test_loss_ranges_do_not_waive_isolated_or_terminal_loss(self):
        self.assertEqual(sequence_ranges([9, 0, 2, 3, 4, 9]), [[0, 0], [2, 4], [9, 9]])

    def test_socket_write_interval_keeps_uncertainty(self):
        send = {'before': {'mono_before_ns': '100'}, 'after': {'mono_after_ns': '150'}}
        self.assertEqual(interval(send, {'mono_lower_ns': '140', 'mono_upper_ns': '160'}),
                         {'lower_ns': '-10', 'upper_ns': '60'})

    def test_stream_sequence_is_explicit_and_closed(self):
        ip = bytearray(20)
        ip[0], ip[9] = 0x45, 17
        ip[12:16], ip[16:20] = bytes([100, 64, 0, 1]), bytes([192, 0, 2, 10])
        body = b'FVPN TEST ONLY 27' + struct.pack('!Q', 2**53+9)
        udp = struct.pack('!HHHH', 40001, 40002, 8+len(body), 0)
        struct.pack_into('!H', ip, 2, len(ip)+len(udp)+len(body))
        packet = bytes(ip)+udp+body
        self.assertNotIn('fixture_sequence', decode_ip(packet, lambda _: 'test'))
        out = decode_ip(packet, lambda _: 'test', fixture_sequences=True)
        self.assertEqual(out['fixture_sequence'], str(2**53+9))
        self.assertNotIn('payload', out)
        changed = packet[:-8]+b'12345678'
        self.assertEqual(decode_ip(changed, lambda _: 'test', fixture_sequences=True)['fixture_sequence'],
                         str(struct.unpack('!Q', b'12345678')[0]))
        unknown = packet[:28]+b'UNRELATED DATA!!'+packet[-8:]
        self.assertNotIn('fixture_sequence',decode_ip(unknown,lambda _: 'test',fixture_sequences=True))

    def test_same_handshake_bytes_group_across_hops(self):
        p = {'packet':1,'timestamp':'1.0','interface':'lo','packet_type':4,
             'wireguard':{'type':2,'message_sha256':'test-only-hash','source_loopback':True}}
        groups = handshake_groups({'member':[p], 'node':[dict(p,packet=2)], 'origin':[]})
        self.assertEqual(len(groups),1)
        self.assertEqual({h['role'] for h in groups[0]['hops']},{'member','node'})


if __name__=='__main__':
    unittest.main()
