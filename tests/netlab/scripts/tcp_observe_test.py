"""TEST ONLY parser regressions use invented headers, never live traffic."""
import struct
import unittest

import tcp_observe as obs


def packet(flags=0x12, seq=4294967294, ack=123, options=b'', data=b''):
    tcp = struct.pack('!HHIIBBHHH', 40000, 9001, seq, ack, ((20+len(options))//4)<<4, flags, 512, 0, 0) + options + data
    ip = struct.pack('!BBHHHBBH4s4s', 0x45, 0, 20+len(tcp), 7, 0x4000, 64, 6, 0, bytes((100,64,1,1)), bytes((203,0,113,10))) + tcp
    frame = struct.pack('!HHIHBB8s', 0x800, 0, 8, 1, 4, 6, b'\x00'*8) + ip
    return struct.pack('<IIII', 100, 123456789, len(frame), len(frame)) + frame


def pcap(*packets):
    return b'\x4d\x3c\xb2\xa1' + struct.pack('<HHIIII', 2, 4, 0, 0, 65535, 276) + b''.join(packets)


class TCPObservationTests(unittest.TestCase):
    def test_absolute_sequence_flags_options_and_no_payload_or_mac(self):
        opts = b'\x01\x01\x08\x0a' + struct.pack('!II', 77, 66)
        rows = obs.decode(pcap(packet(options=opts, data=b'private-data')), {'8':'fv-fh'})
        self.assertEqual(len(rows), 1)
        r = rows[0]
        self.assertEqual((r['seq'], r['ack'], r['flags_text']), (4294967294, 123, ['SYN','ACK']))
        self.assertEqual((r['wall_ns'], r['data_length'], r['tsval'], r['tsecr']), ('100123456789', 12, 77, 66))
        self.assertEqual((r['interface'], r['direction']), ('fv-fh','out'))
        self.assertNotIn('private-data', str(r))
        self.assertNotIn('mac', r)

    def test_every_segment_including_retransmission_and_ack(self):
        rows = obs.decode(pcap(packet(), packet(), packet(flags=16)), {'8':'fv-fh'})
        self.assertEqual([r['packet'] for r in rows], [1,2,3])
        self.assertEqual(rows[0]['seq'], rows[1]['seq'])
        self.assertEqual(rows[2]['flags_text'], ['ACK'])

    def test_reject_truncation_at_every_byte(self):
        raw = pcap(packet())
        for end in range(len(raw)):
            if end == 24: continue  # a complete empty pcap
            with self.subTest(end=end), self.assertRaises(ValueError):
                obs.decode(raw[:end], {'8':'fv-fh'})

    def test_bad_headers_and_options_fail_closed(self):
        raw = pcap(packet())
        cases = []
        for offset, value in [(0,0), (20,113), (31,255), (40,0), (50,3), (60,0x65), (69,17), (92,0x10)]:
            bad = bytearray(raw); bad[offset] = value; cases.append(bytes(bad))
        cases += [pcap(packet(options=b'\x08\x0a\x00\x00')), pcap(packet(options=b'\x02\x03\x00\x00')), pcap(packet(options=b'\x05\x04\x00\x00'))]
        for i, value in enumerate(cases):
            with self.subTest(case=i), self.assertRaises(ValueError):
                obs.decode(value, {'8':'fv-fh'})
        with self.assertRaises(ValueError): obs.decode(raw, {})

    def test_capture_count_and_drop_failures(self):
        good = '3 packets captured\n3 packets received by filter\n0 packets dropped by kernel\n'
        self.assertEqual(obs.validate_stats(good, 3)['dropped_by_kernel'], 0)
        for text, count in [(good,2), (good.replace('0 packets','1 packets'),3), ('',0), (good.replace('3 packets received by filter',''),3)]:
            with self.assertRaises(ValueError): obs.validate_stats(text,count)

    def test_conntrack_presence_absence_and_bad_rows(self):
        line = 'ipv4 2 tcp 6 431999 ESTABLISHED src=100.64.1.1 dst=203.0.113.10 sport=40000 dport=9001 src=203.0.113.10 dst=203.0.113.1 sport=9001 dport=40000 [ASSURED] mark=0 use=1 id=123'
        self.assertEqual(obs.conntrack(''), [])
        self.assertEqual(obs.conntrack(line)[0]['state'], 'ESTABLISHED')
        for bad in [line.replace('id=123',''), line.replace('dport=9001','dport=80'), line+'\n'+line, 'partial']:
            with self.assertRaises(ValueError): obs.conntrack(bad)


if __name__ == '__main__': unittest.main()
