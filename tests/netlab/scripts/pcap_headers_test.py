import socket
import struct
import tempfile
from pathlib import Path
import unittest

from pcap_headers import compare_udp, decode


class CaptureTests(unittest.TestCase):
    def test_missing_large_reverse_packet_and_duplicate_receipts(self):
        sent=[dict(packet=1,ip_length=1360,udp_sha256='large'),dict(packet=2,ip_length=1152,udp_sha256='small'),dict(packet=3,ip_length=1360,udp_sha256='large')]
        got=compare_udp(sent,[dict(udp_sha256='small'),dict(udp_sha256='large')])
        self.assertEqual(got['sizes']['1360'],dict(sent=2,arrived=1,absent=1))
        self.assertEqual(got['absent_sender_packets'],[3])
        self.assertEqual(got['unmatched_receiver_packets'],0)

    def test_ipv6_ptb_and_udp_headers_without_payload_export(self):
        def ipv6(proto,body):
            ip=struct.pack('!IHBB',6<<28,len(body),proto,64)+socket.inet_pton(socket.AF_INET6,'2001:db8::1')+socket.inet_pton(socket.AF_INET6,'2001:db8::2')
            sll=struct.pack('!HHIHBB8s',0x86dd,0,7,1,4,6,b'\0'*8)
            return sll+ip+body
        secret=b'packet-content-must-not-be-exported'
        packets=[ipv6(17,struct.pack('!HHHH',51820,40000,len(secret)+8,0)+secret),ipv6(58,struct.pack('!BBHI',2,0,0,1280)+b'\0'*48)]
        raw=struct.pack('<IHHIIII',0xa1b2c3d4,2,4,0,0,65535,276)
        for i,p in enumerate(packets):
            raw+=struct.pack('<IIII',10,i,len(p),len(p))+p
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp)/'test.pcap'
            path.write_bytes(raw)
            rows=decode(path)
            self.assertEqual(rows[0]['ip_length'],40+8+len(secret))
            self.assertEqual(rows[0]['ifindex'],7)
            self.assertEqual(rows[1]['ptb_mtu'],1280)
            self.assertNotIn(secret.decode(),str(rows))
            path.write_bytes(raw[:-1])
            with self.assertRaises(ValueError):
                decode(path)


if __name__=='__main__':
    unittest.main()
