"""TEST ONLY: attribution cannot silently excuse a missing or racing packet."""
import json
from pathlib import Path
import tempfile
import unittest
import at12_analysis as a


class Attribution(unittest.TestCase):
    def fixture(self, root):
        for phase,start,end,value in [('before',100,110,0),('mid',200,210,97)]:
            counter=dict(packets=int(value!=0),bytes=value)
            doc=dict(counter_start_ns=start,counter_end_ns=end,counter_document={
                'counters':{a.COUNTER:counter},'samples':[dict(device_id=a.DEVICE,direction='upload',**counter)]})
            (root/(phase+'-observation.json')).write_text(json.dumps(doc))
        incoming=dict(packet=1,wall_ns='150',interface='fv-fh',packet_type=0,source='100.64.1.1',
                      destination='203.0.113.10',protocol=6,source_port=40000,destination_port=9001,
                      seq=10,ack=20,flags=0x18,ip_id=7,ip_length=97)
        outgoing=dict(incoming,packet=2,wall_ns='151',interface='fv-ih',packet_type=4,source='192.0.2.37')
        for role,rows in [('origin',[]),('node',[incoming,outgoing]),('responder',[])]:
            (root/('at12-all-'+role+'.jsonl')).write_text(''.join(json.dumps(r)+'\n' for r in rows))
            (root/('at12-all-'+role+'.meta.json')).write_text(json.dumps(dict(complete=True,read_packets=len(rows))))

    def test_exact_counter_and_egress(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);self.fixture(root)
            result=a.analyze(root)
            self.assertEqual(result['observed_egress_bytes'],97)
            self.assertEqual(result['counter_deltas'][a.COUNTER]['bytes'],97)
            self.assertFalse(result['upload_ledger'][0]['prior_origin_packets'])
            self.assertIn('No original',result['historical_f28_attribution'])

    def test_wrong_sample_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);self.fixture(root)
            p=root/'mid-observation.json';doc=json.loads(p.read_text())
            doc['counter_document']['samples'][0]['bytes']=96;p.write_text(json.dumps(doc))
            with self.assertRaises(ValueError):a.analyze(root)

    def test_missing_egress_is_visible(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);self.fixture(root)
            p=root/'at12-all-node.jsonl';p.write_text(p.read_text().splitlines()[0]+'\n')
            (root/'at12-all-node.meta.json').write_text(json.dumps(dict(complete=True,read_packets=1)))
            result=a.analyze(root)
            self.assertEqual(result['observed_egress_bytes'],0)
            self.assertEqual(result['counter_deltas'][a.COUNTER]['bytes'],97)

    def test_boundary_packet_is_marked(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);self.fixture(root)
            p=root/'at12-all-node.jsonl';rows=[json.loads(s) for s in p.read_text().splitlines()]
            rows[0]['wall_ns']='105';p.write_text(''.join(json.dumps(r)+'\n' for r in rows))
            self.assertEqual(a.analyze(root)['read_boundary_ingresses'],[1])

    def test_capture_loss_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);self.fixture(root)
            (root/'at12-all-origin.meta.json').write_text(json.dumps(dict(complete=False,read_packets=0)))
            with self.assertRaises(ValueError):a.analyze(root)
