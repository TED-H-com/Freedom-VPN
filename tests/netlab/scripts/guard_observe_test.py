"""TEST ONLY synthetic NFLOG ABI parser cases."""
import struct
import unittest
from at12_observe_test import ip4
import guard_observe as g


def fixture(prefix=b'f29g1001-iifname\0'):
    tcp=struct.pack('!HHIIBBHHH',40000,9001,123,456,5<<4,0x18,1000,0,0)+bytes(5)
    attrs={1:struct.pack('!HBB',0x800,2,0),3:struct.pack('!QQ',20,123),9:ip4(6,tcp),10:prefix,8:bytes(12),16:bytes(14)}
    pkt=struct.pack('!BBH',2,0,29)
    for k,v in attrs.items():
        length=len(v)+4;pkt+=struct.pack('<HH',length,k)+v+bytes((-length)%4)
    return b'\x4d\x3c\xb2\xa1'+struct.pack('<HHIIII',2,4,0,0,262144,239)+struct.pack('<IIII',20,124000,len(pkt),len(pkt))+pkt


class NFLog(unittest.TestCase):
    def test_exact_rule_and_headers(self):
        row=g.decode(fixture())[0]
        self.assertEqual((row['generation'],row['rule'],row['seq'],row['data_length']),('1001','iifname',123,5))
        self.assertEqual(row['kernel_wall_ns'],'20000123000')
        self.assertNotIn('payload',row)
        self.assertNotIn('mac',row)

    def test_microsecond_device(self):
        raw=bytearray(fixture())
        raw[:4]=b'\xd4\xc3\xb2\xa1'
        struct.pack_into('<I',raw,28,124)
        row=g.decode(bytes(raw))[0]
        self.assertEqual(row['pcap_wall_ns'],'20000124000')
        self.assertEqual(row['pcap_precision_ns'],1000)

    def test_bad_prefix(self):
        with self.assertRaises(ValueError):g.decode(fixture(b'unexpected\0'))

    def test_truncated(self):
        for size in (0,20,30,len(fixture())-1):
            with self.assertRaises(ValueError):g.decode(fixture()[:size])
