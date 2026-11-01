"""TEST ONLY checks for loss accounting and content-free stream decoding."""
import struct
import json
from pathlib import Path
import tempfile
import unittest

from analyze_followup27 import interval, sequence_ranges, handshake_groups
from decode_capture import decode_ip
from archive_followup27 import archive
from followup27_io import read, INDEX


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

    def test_lossless_archive_rejects_tampering_and_document_indexes(self):
        with tempfile.TemporaryDirectory() as scratch:
            run=Path(scratch)/'followup27'/'test-only-run'
            run.mkdir(parents=True)
            (run/'measurement.json').write_text(json.dumps(dict(id='measurement',run_id='test-only-run',status='pass',source_commit='0'*40)))
            raw=b'Status: fail\n'+b'TEST ONLY\n'*110000
            path=run/'capture.md'
            path.write_bytes(raw)
            archive(run)
            self.assertTrue(path.read_text().startswith(INDEX))
            self.assertIn('**fail**',path.read_text())
            with self.assertRaisesRegex(AssertionError,'document index'):read(path)
            archived=path.with_suffix('.md.gz')
            self.assertEqual(read(archived),raw)
            saved=archived.read_bytes()
            archived.write_bytes(saved+b'changed')
            with self.assertRaisesRegex(AssertionError,'compressed receipt'):read(archived)
            archived.write_bytes(saved)
            manifest=run/'archive.json'
            doc=json.loads(manifest.read_text())
            entries=json.loads(doc['observations'][0]['detail'])
            entries['files'][0]['original']['sha256']='0'*64
            doc['observations'][0]['detail']=json.dumps(entries)
            manifest.write_text(json.dumps(doc))
            with self.assertRaisesRegex(AssertionError,'original receipt'):read(archived)


if __name__=='__main__':
    unittest.main()
