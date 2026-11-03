"""TEST ONLY synthetic packet parser regressions."""
import struct
import unittest
import at12_observe as obs


def ip4(proto, body):
    return struct.pack('!BBHHHBBH4s4s', 0x45, 0, 20+len(body), 7, 0, 64, proto, 0,
                       bytes((100,64,1,1)), bytes((203,0,113,10))) + body


class Headers(unittest.TestCase):
    def test_tcp_97_bytes(self):
        tcp=struct.pack('!HHIIBBHHH', 40000, 9001, 123, 456, 8 << 4, 0x18, 1024, 0, 0)+bytes(12)+bytes(45)
        row=obs.headers(ip4(6,tcp),0x800)
        self.assertEqual((row['ip_length'],row['data_length'],row['seq']), (97,45,123))
        self.assertNotIn('payload',row)

    def test_combined_97_bytes(self):
        rows=[]
        for head,size,flags in ((32,5,0x18),(20,0,4)):
            tcp=struct.pack('!HHIIBBHHH',40000,9001,123,456,(head//4)<<4,flags,1024,0,0)+bytes(head-20+size)
            rows.append(obs.headers(ip4(6,tcp),0x800))
        self.assertEqual([r['ip_length'] for r in rows],[57,40])

    def test_udp(self):
        row=obs.headers(ip4(17,struct.pack('!HHHH',40001,9000,12,0)+bytes(4)),0x800)
        self.assertEqual(row['ip_length'],32)

    def test_icmp(self):
        row=obs.headers(ip4(1,bytes((8,0,0,0,0,7,0,1))+bytes(56)),0x800)
        self.assertEqual((row['ip_length'],row['icmp_sequence']),(84,1))

    def test_ipv6(self):
        packet=struct.pack('!IHBB16s16s',6 << 28,8,58,64,bytes(16),bytes(16))+bytes(8)
        self.assertEqual(obs.headers(packet,0x86dd)['ip_length'],48)

    def test_no_link_addresses(self):
        self.assertEqual(obs.headers(bytes(28),0x806),dict(ethertype=0x806,captured_length=28))

    def test_reject_truncation(self):
        for packet in (b'',ip4(6,bytes(8)),ip4(17,struct.pack('!HHHH',1,2,100,0))):
            with self.assertRaises(ValueError): obs.headers(packet,0x800)
